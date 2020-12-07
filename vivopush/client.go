package vivopush

import (
	"push_sdks/clients"
	"push_sdks/config"
	"push_sdks/common"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/bitly/go-simplejson"
	log "github.com/sirupsen/logrus"
)

const TOKEN_EXPIRES = 86400

type VivoTokenPar struct {
	AppId     string `json:"appId"`
	AppKey    string `json:"appKey"`
	Timestamp int64  `json:"timestamp"`
	Sign      string `json:"sign"`
}

type AuthToken struct {
	token      string
	valid_time int64
}

type VivoPush struct {
	host       string
	cfg        *config.PushServerCfg
	pushMod    int
	httpClient *clients.HTTPClient
	authToken  atomic.Value

	systemMsgLimitTimestamp    int64
	operationMsgLimitTimestamp int64
}

func NewClient(cfg config.PushServerCfg) (*VivoPush, error) {
	ret := &VivoPush{
		cfg:        &cfg,
		host:       ProductionHost,
		httpClient: clients.NewHTTPClient(),
	}
	ret.GetToken()
	return ret, nil
}

//----------------------------------------Token----------------------------------------//
//获取token  返回的expiretime 秒  当过期的时候
func (vc *VivoPush) GetToken() (string, int, error) {
	now := time.Now().UnixNano() / 1e6
	md5Ctx := md5.New()
	n, err := md5Ctx.Write([]byte(vc.cfg.AppId + vc.cfg.AppKey + strconv.FormatInt(now, 10) + vc.cfg.AppSecret))
	if err != nil {
		log.WithError(err).WithField("n", n).Error("md5 error")
		return "", 0, err
	}
	sign := hex.EncodeToString(md5Ctx.Sum(nil))

	formData, err := json.Marshal(&VivoTokenPar{
		AppId:     vc.cfg.AppId,
		AppKey:    vc.cfg.AppKey,
		Timestamp: now,
		Sign:      sign,
	})
	if err != nil {
		return "", 0, err
	}

	req := &clients.Request{
		Method: http.MethodPost,
		URL:    ProductionHost + AuthURL,
		Body:   []byte(formData),
		Header: []clients.HTTPOption{clients.SetHeader("Content-Type", "application/json")},
	}
	resp, err := vc.httpClient.DoHttpRequest(nil, req)
	if err != nil {
		return "", 0, err
	}
	if resp.Status != http.StatusOK {
		return "", 0, fmt.Errorf("vivo get token statusCode :[%v] error:[%v]", resp.Status, string(resp.Body))
	}
	js, err := simplejson.NewJson(resp.Body)
	if err != nil {
		return "", 0, err
	}

	token, err := js.Get("authToken").String()
	if err != nil {
		return "", 0, err
	}

	vc.authToken.Store(AuthToken{
		token:      token,
		valid_time: time.Now().Unix() + TOKEN_EXPIRES,
	})

	return token, TOKEN_EXPIRES, nil
}

func (vc *VivoPush) NeedAccessToken() bool {
	return vc.cfg.NeedAccessToken
}

func (vc *VivoPush) Name() string {
	return vc.cfg.Name
}

func (vc *VivoPush) PushMsg(msg *common.Msg, tokens []string) (failsInfoMap map[string]*common.CallbackResponseItem, err error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	systemMsgLimit, operatoinMsgLimit := vc.IsTodayLimit()
	if systemMsgLimit && operatoinMsgLimit {
		failsInfoMap = make(map[string]*common.CallbackResponseItem, len(tokens))
		for _, token := range tokens {
			failsInfoMap[token] = &common.CallbackResponseItem{
				Status:      common.CALLBACK_STATUS_PUSH_TOTAL_LIMIT,
				Description: common.GetCallBackMsg(common.CALLBACK_STATUS_PUSH_TOTAL_LIMIT),
				MsgId:       msg.Id,
				Token:       token,
				Timestamp:   time.Now().Unix(),
			}
		}
		log.WithFields(log.Fields{
			"msgInfo": msg,
			"tokens":  tokens,
			"status":  common.CALLBACK_STATUS_PUSH_TOTAL_LIMIT,
		}).Debugf("vivo push msg result limit")
		return
	}
	if len(tokens) == 1 {
		formatMsg := NewVivoMessage(msg.MsgTitle, msg.MsgBody)
		formatMsg.PushMode = 0
		if vc.pushMod == 1 {
			formatMsg.PushMode = 1
		}
		formatMsg.NotificationChannel = msg.ChannelID
		formatMsg.SkipType = 1
		formatMsg.SkipContent = msg.MsgAction
		if vc.cfg.Redirect != "" {
			formatMsg.Extra = make(map[string]string, 2)
			formatMsg.Extra["callback"] = vc.cfg.Redirect + fmt.Sprintf("?deviceVendor=%s", vc.Name())
			params := common.CallbackParam{MsgId: msg.Id, Package: vc.cfg.Package}
			paramsData, err := json.Marshal(params)
			if err != nil {
				return nil, err
			}
			formatMsg.Extra["callback.param"] = string(paramsData)
		}

		formatMsg.Classification = 1 //系统消息
		if systemMsgLimit {
			formatMsg.Classification = 0
		}

		formatMsg.RequestId = fmt.Sprintf("%d", msg.Id)
		var result *SendResult
		for i := 0; i < 2; i++ {
			result, err = vc.send(formatMsg, tokens[0])
			if result != nil {
				//VIVO正式应用发送的title及content里面不能是纯数字、纯英文、纯符号、符号加数字，包含“测试”字样、大括号、中括号 。
				if result.Result == 10104 || result.Result == 10085 {
					formatMsg.Title = "Hi," + formatMsg.Title
					continue
				}
			}
			break
		}

		//暂时屏蔽vivo regId不合法和发送超出时间限制, 运营消息总量超出， 系统消息总量超出
		if result != nil && (result.Result == 10302 || result.Result == 10071 || result.Result == 10070 || result.Result == 10073) {
			err = nil
		}

		failsInfoMap = vc.ResultItemFormat(result, tokens)
		if err != nil {
			log.WithField("msgInfo", msg).WithError(err).Warnf("vivo push msg result: [%v]", result)
			return failsInfoMap, err
		}
		if result.Result > 0 {
			log.WithField("msgInfo", msg).WithField("token", tokens[0]).Infof("vivo push msg result: [%v]", result)
		}

		return failsInfoMap, nil

	}

	formatMsg := NewListPayloadMessage(msg.MsgTitle, msg.MsgBody)
	formatMsg.SkipType = 1
	formatMsg.SkipContent = msg.MsgAction
	formatMsg.RequestId = fmt.Sprintf("%d", msg.Id)

	result, err := vc.sendList(formatMsg, tokens)
	//暂时屏蔽vivo regId不合法和发送超出时间限制, 运营消息总量超出， 系统消息总量超出
	if result != nil && (result.Result == 10302 || result.Result == 10071 || result.Result == 10070 || result.Result == 10073) {
		err = nil
	}
	failsInfoMap = vc.ResultItemFormat(result, tokens)
	if err != nil {
		log.WithField("msgInfo", msg).WithError(err).Errorf("vivo push msg result: [%v]", result)
		return failsInfoMap, err
	}
	log.WithField("msgInfo", msg).Infof("vivo push msg result: [%v] ", result)
	return failsInfoMap, nil
}

func (vc *VivoPush) ResultItemFormat(result *SendResult, tokens []string) map[string]*common.CallbackResponseItem {
	if result == nil {
		return nil
	}
	var status int64
	switch result.Result {
	case 10302:
		status = common.CALLBACK_STATUS_INVALID_DEVICE_TOKEN
	case 10054:
		status = common.CALLBACK_STATUS_INVALID_DEVICE_TOKEN
	case 10070:
		status = common.CALLBACK_STATUS_PUSH_TOTAL_LIMIT
		vc.setOperationMsgLimit()
	case 10073:
		status = common.CALLBACK_STATUS_PUSH_TOTAL_LIMIT
		vc.setSystemMsgLimit()
	case 10252:
		status = common.CALLBACK_STATUS_PUSH_RATE_LIMIT
	case 10071:
		status = common.CALLBACK_STATUS_INACTIVE_DEVICE_TOKEN
	case 10040:
		status = common.CALLBACK_STATUS_NEED_RETRY
	default:
		status = int64(result.Result)
	}
	failsInfoMap := make(map[string]*common.CallbackResponseItem, len(tokens))
	for _, token := range tokens {
		res := &common.CallbackResponseItem{
			Status:       status,
			Description:  result.Desc,
			RequestId:    result.TaskId,
			Token:        token,
			DeviceVendor: vc.cfg.Name,
			PackageName:  vc.cfg.Package,
		}
		failsInfoMap[token] = res
	}
	return failsInfoMap
}

type CallBackItem struct {
	Param   string `json:"param"`
	Targets string `json:"targets"`
}

func (vc *VivoPush) PushReciver(r *http.Request) (*common.CallbackResponse, map[string]interface{}, error) {
	res := map[string]interface{}{"errno": 0, "errmsg": "success"}
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"deviceVendor": vc.cfg.Name,
			"url":          r.URL.RequestURI(),
			"header":       r.Header,
		}).Errorf("invalid request body")
		return nil, nil, err
	}
	log.WithFields(log.Fields{
		"deviceVendor": vc.cfg.Name,
		"url":          r.URL.RequestURI(),
		"header":       r.Header,
	}).Tracef("callback")

	var reciver map[string]CallBackItem
	err = json.Unmarshal(requestBody, &reciver)
	if err != nil {
		log.WithFields(log.Fields{
			"deviceVendor": vc.cfg.Name,
			"url":          r.URL.RequestURI(),
			"header":       r.Header,
			"form":         string(requestBody),
		}).Errorf("empty params data")
		return nil, nil, err
	}
	var callbackResponse common.CallbackResponse
	for requestId, val := range reciver {
		item := &common.CallbackResponseItem{}
		params := common.CallbackParam{}
		err := json.Unmarshal([]byte(val.Param), &params)
		if err != nil {
			log.WithError(err).WithField("deviceVendor", vc.cfg.Name).Errorf("invalid callback param: %v", val.Param)
			continue
		}
		item.MsgId = params.MsgId
		item.Token = val.Targets
		item.RequestId = requestId
		item.Timestamp = time.Now().UnixNano() / 1000
		item.PackageName = params.Package
		item.Status = common.CALLBACK_STATUS_OK
		callbackResponse.Data = append(callbackResponse.Data, item)
	}
	return &callbackResponse, res, nil
}

func (vc *VivoPush) UploadIcon(picPath string) (*common.CallbackResponse, map[string]interface{}, error) {
	return nil, nil, nil
}

func (c *VivoPush) RegisterDeviceToken(deviceId string) (deviceToken string, err error) {
	return deviceId, nil
}

//----------------------------------------Sender----------------------------------------//
// 根据regID，发送消息到指定设备上
func (v *VivoPush) send(msg *Message, regID string) (*SendResult, error) {
	params := v.assembleSendParams(msg, regID)
	res, err := v.doPost(v.host+SendURL, params)
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if result.Result != 0 {
		return &result, errors.New(result.Desc)
	}
	return &result, nil
}

// 保存群推消息公共体接口
func (v *VivoPush) SaveListPayload(msg *MessagePayload) (*SendResult, error) {
	res, err := v.doPost(v.host+SaveListPayloadURL, msg.JSON())
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if result.Result != 0 {
		return &result, errors.New(result.Desc)
	}
	return &result, nil
}

// 群推
func (v *VivoPush) sendList(msg *MessagePayload, regIds []string) (*SendResult, error) {
	if len(regIds) < 2 || len(regIds) > 1000 {
		return nil, errors.New("regIds个数必须大于等于2,小于等于 1000")
	}
	res, err := v.SaveListPayload(msg)
	if err != nil {
		return res, err
	}
	if res.Result != 0 {
		return res, errors.New(res.Desc)
	}
	msgList := NewListMessage(regIds, res.TaskId)
	msgList.PushMode = v.pushMod

	bytes, err := json.Marshal(msgList)
	if err != nil {
		return nil, err
	}
	//推送
	res2, err := v.doPost(v.host+PushToListURL, bytes)
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res2, &result)
	if err != nil {
		return nil, err
	}
	if result.Result != 0 {
		return &result, errors.New(result.Desc)
	}
	return &result, nil
}

// 全量推送
func (v *VivoPush) SendAll(msg *MessagePayload) (*SendResult, error) {
	res2, err := v.doPost(v.host+PushToAllURL, msg.JSON())
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res2, &result)
	if err != nil {
		return nil, err
	}
	if result.Result != 0 {
		return nil, errors.New(result.Desc)
	}
	return &result, nil
}

//----------------------------------------Tracer----------------------------------------//
// 获取指定消息的状态。
func (v *VivoPush) GetMessageStatusByJobKey(jobKey string) (*BatchStatusResult, error) {
	params := v.assembleStatusByJobKeyParams(jobKey)
	res, err := v.doGet(v.host+MessagesStatusURL, params)
	if err != nil {
		return nil, err
	}
	var result BatchStatusResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (v *VivoPush) assembleSendParams(msg *Message, regID string) []byte {
	msg.RegId = regID
	jsondata := msg.JSON()
	return jsondata
}

func (v *VivoPush) assembleStatusByJobKeyParams(jobKey string) string {
	form := url.Values{}
	form.Add("taskIds", jobKey)
	return "?" + form.Encode()
}

func handleResponse(response *http.Response) ([]byte, error) {
	if response == nil {
		return nil, errors.New("empty response")
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (v *VivoPush) doPost(url string, formData []byte) ([]byte, error) {
	var err error
	req := &clients.Request{
		Method: http.MethodPost,
		URL:    url,
		Body:   formData,
		Header: []clients.HTTPOption{clients.SetHeader("Content-Type", "application/json"), clients.SetHeader("authToken", v.authToken.Load().(AuthToken).token)},
	}
	resp, err := v.httpClient.DoHttpRequest(nil, req)

	for tryTime := 0; tryTime < PostRetryTimes; tryTime++ {
		resp, err = v.httpClient.DoHttpRequest(context.Background(), req)
		if err == nil {
			break
		}

		if tryTime > PostRetryTimes {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}
	if resp.Status != http.StatusOK {
		return nil, fmt.Errorf("server error status:[%d] body:[%s]", resp.Status, string(resp.Body))
	}
	return resp.Body, nil
}

func (v *VivoPush) doGet(url string, params string) ([]byte, error) {
	var result []byte
	var req *http.Request
	var resp *http.Response
	var err error
	req, err = http.NewRequest("GET", url+params, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authToken", v.authToken.Load().(AuthToken).token)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	result, err = handleResponse(resp)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getTodayZeroTime() int64 {
	t := time.Now()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()
}

func (v *VivoPush) IsTodayLimit() (systemMsgLimit bool, operatoinMsgLimit bool) {
	zeroTime := getTodayZeroTime()
	systemMsgLimit = atomic.LoadInt64(&v.systemMsgLimitTimestamp) == zeroTime
	operatoinMsgLimit = atomic.LoadInt64(&v.operationMsgLimitTimestamp) == zeroTime
	return systemMsgLimit, operatoinMsgLimit
}

func (v *VivoPush) setSystemMsgLimit() {
	atomic.StoreInt64(&v.systemMsgLimitTimestamp, getTodayZeroTime())
}

func (v *VivoPush) setOperationMsgLimit() {
	atomic.StoreInt64(&v.operationMsgLimitTimestamp, getTodayZeroTime())
}
