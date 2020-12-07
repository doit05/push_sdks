package oppopush

import (
	"push_go/clients"
	"push_go/config"
	"push_go/push_sdks/common"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const MaxTotalPerBatch = 1000
const PICTURE_TTL = 30 * 86400

type OppoPush struct {
	cfg           *config.PushServerCfg
	AccessToken   string
	IconId        string
	IconUpdatedAt int64
	httpClient    *clients.HTTPClient
}

func NewClient(cfg config.PushServerCfg) (*OppoPush, error) {
	httpClient := clients.NewHTTPClient()
	return &OppoPush{cfg: &cfg, httpClient: httpClient}, nil
}

func (c *OppoPush) NeedAccessToken() bool {
	return c.cfg.NeedAccessToken
}
func (c *OppoPush) Name() string {
	return c.cfg.Name
}

func (c *OppoPush) GetToken() (string, int, error) {
	tokenInfo, err := GetToken(c.cfg.AppKey, c.cfg.AppSecret)
	if err != nil {
		return "", 0, err
	}
	c.AccessToken = tokenInfo.AccessToken
	return tokenInfo.AccessToken, MaxTimeToLive, err
}

func (c *OppoPush) PushMsg(msg *common.Msg, tokens []string) (failsInfoMap map[string]*common.CallbackResponseItem, err error) {
	//保存通知栏消息内容体
	msg0 := NewSaveMessageContent(msg.MsgTitle, msg.MsgBody).
		SetSubTitle(msg.SubMsgTile)
	msg0.ActionParameters = fmt.Sprintf("{\"appData\": \"%s\"}", msg.MsgAction)
	msg0.AppMessageID = fmt.Sprintf("%v_%v", msg.Id, time.Now().UnixNano())
	if len(msg0.Title) > 50 {
		msg0.Title = msg0.Title[:50]
	}
	if len(msg0.SubTitle) > 10 {
		msg0.SubTitle = msg0.SubTitle[:10]
	}
	//your channel id
	msg0.ChannelID = msg.ChannelID

	msg0.CallBackURL = c.cfg.Redirect + fmt.Sprintf("?deviceVendor=%s", c.Name())
	params := common.CallbackParam{MsgId: msg.Id, Package: c.cfg.Package}
	paramsData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	msg0.CallBackParameter = string(paramsData)
	if msg.ImgUrl != "" {
		picId, err := c.GetImgId(msg.ImgUrl)
		if err != nil {
			log.WithError(err).Errorf("%s upload pic err", c.cfg.Name)
		}
		if picId != "" {
			msg0.Style = 3
			msg0.BigPictureId = picId
			if len(msg0.Content) > 50 {
				msg0.Content = msg0.Content[:50]
			}
		}
	}

	log.WithFields(log.Fields{"msg": msg, "push msg": msg0}).Tracef("%s send msg", c.cfg.Name)
	result, err := c.saveMessageContent(msg0)
	if err != nil {
		return nil, fmt.Errorf("%s saveMessageContent error:[%v]", c.cfg.Name, err)
	}
	if result.Data.MessageID == "" {
		log.Errorf("%s saveMessageContent result:[%v]", c.cfg.Name, result)
		return nil, fmt.Errorf("%s saveMessageContent result:[%v]", c.cfg.Name, result)
	}
	//广播推送-通知栏消息
	broadcast := NewBroadcast(result.Data.MessageID).
		SetTargetType(2).
		SetTargetValue(strings.Join(tokens, ";"))
	res, err := c.broadcast(broadcast)
	if res != nil {
		if failsInfoMap == nil {
			failsInfoMap = make(map[string]*common.CallbackResponseItem, len(tokens))
		}
		for _, token := range tokens {
			failsInfoMap[token] = &common.CallbackResponseItem{
				Status:       formatStatus(res.Code),
				Description:  res.Message,
				RequestId:    res.Data.MessageID,
				Token:        token,
				DeviceVendor: c.cfg.Name,
				PackageName:  c.cfg.Package,
			}
		}
	}
	if err != nil {
		return failsInfoMap, fmt.Errorf("%s broadcast msg error:[%v]", c.cfg.Name, err)
	}
	log.WithField("result", res).Debugf("%s broadcast msg success", c.cfg.Name)
	return nil, nil
}

type CallBackItem struct {
	AppId     string `json:"appId"`
	Param     string `json:"param"`
	Targets   string `json:"registrationIds"`
	TaseId    string `json:"taskId"`
	MessageId string `json:"messageId"`
	EventType string `json:"eventType"`
}

func (c *OppoPush) PushReciver(r *http.Request) (*common.CallbackResponse, map[string]interface{}, error) {
	res := map[string]interface{}{"errno": 0, "errmsg": "success"}
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"deviceVendor": c.cfg.Name,
			"url":          r.URL.RequestURI(),
			"header":       r.Header,
		}).Errorf("invalid request body")
		return nil, nil, err
	}
	log.WithFields(log.Fields{
		"deviceVendor": c.cfg.Name,
		"url":          r.URL.RequestURI(),
		"header":       r.Header,
	}).Tracef("callback")

	var reciver []CallBackItem
	err = json.Unmarshal(requestBody, &reciver)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"deviceVendor": c.cfg.Name,
			"url":          r.URL.RequestURI(),
			"header":       r.Header,
			"form":         string(requestBody),
		}).Errorf("empty params data")
		return nil, nil, err
	}
	var callbackResponse common.CallbackResponse
	for _, val := range reciver {
		item := &common.CallbackResponseItem{}
		params := common.CallbackParam{}
		err := json.Unmarshal([]byte(val.Param), &params)
		if err != nil {
			log.WithError(err).WithField("deviceVendor", c.cfg.Name).Errorf("invalid callback param: %v", val.Param)
			continue
		}
		item.MsgId = params.MsgId
		item.Token = val.Targets
		item.RequestId = val.MessageId
		item.Timestamp = time.Now().UnixNano() / 1000
		item.PackageName = params.Package
		item.Status = common.CALLBACK_STATUS_OK
		if val.EventType != "push_arrive" {
			item.Status = common.UNKONW
		}
		callbackResponse.Data = append(callbackResponse.Data, item)
	}
	return &callbackResponse, res, nil
}

func (c *OppoPush) RegisterDeviceToken(deviceId string) (deviceToken string, err error) {
	return deviceId, nil
}

func (c *OppoPush) GetImgId(imgUrl string) (string, error) {
	fileName := common.GetImgNameFromUrl(imgUrl)
	imgUrl += "?x-oss-process=image/quality,q_100/resize,limit_0,m_fill,w_876,h_324"
	path := os.TempDir()
	filePath, err := common.DowlodPic(imgUrl, path, c.cfg.Name, fileName)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"url": imgUrl,
		}).Error("download img err ", c.cfg.Name)
		return "", err
	}
	imgId, err := c.UploadPic(filePath)
	return imgId, err
}

//图片要求尺寸876*324 px,文件大小1M以内，格式为PNG/JPG/JPEG
func (c *OppoPush) UploadPic(imgPath string) (string, error) {
	params := map[string]string{"auth_token": tokenInstance.AccessToken, "picture_ttl": strconv.Itoa(PICTURE_TTL)}
	res, err := doUpload(MediaHost+UploadBigPicURL, imgPath, "icon"+filepath.Ext(imgPath), params)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"body":    string(res),
			"imgpath": imgPath,
		}).Errorf("%s upload img err ", c.cfg.Name)
		return "", err
	}

	var result SmallPictureResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"body":    string(res),
			"imgpath": imgPath,
		}).Errorf("%s upload img err ", c.cfg.Name)
		return "", err
	}

	if result.Code > 0 {
		err = errors.New(result.Message)
		log.WithError(err).WithFields(log.Fields{
			"body":    string(res),
			"imgpath": imgPath,
		}).Errorf("%s upload img err ", c.cfg.Name)
		return "", err
	}

	return result.Data.BigPictureId, nil
}

//图片要求尺寸144*144 px，文件大小为50k以内,格式为PNG/JPG/JPEG
func (c *OppoPush) UploadIcon(iconPath string) (string, error) {
	params := map[string]string{"auth_token": tokenInstance.AccessToken, "picture_ttl": strconv.Itoa(PICTURE_TTL)}
	res, err := doUpload(MediaHost+UploadSmallPicURL, iconPath, "icon"+filepath.Ext(iconPath), params)
	if err != nil {
		return "", err
	}

	var result SmallPictureResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"body": string(res),
			"icon": iconPath,
		}).Error("upload icon err", c.cfg.Name)
		return "", err
	}

	if result.Code > 0 {
		err = errors.New(result.Message)
		log.WithError(err).WithFields(log.Fields{
			"body": string(res),
			"icon": iconPath,
		}).Error("upload icon err", c.cfg.Name)
		return "", err
	}

	if len(result.Data.SmallPictureId) > 0 {
		c.IconId = result.Data.SmallPictureId
		c.IconUpdatedAt = time.Now().Unix()
	}
	return result.Data.SmallPictureId, nil
}

// 保存通知栏消息内容体
func (c *OppoPush) saveMessageContent(msg *NotificationMessage) (*SaveSendResult, error) {
	tokenInstance, err := GetToken(c.cfg.AppKey, c.cfg.AppSecret)
	if err != nil {
		return nil, err
	}
	params := defaultForm(msg)
	params.Add("auth_token", tokenInstance.AccessToken)
	bytes, err := doPost(PushHost+SaveMessageContentURL, params)
	if err != nil {
		return nil, err
	}
	var result SaveSendResult
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// 广播推送-通知栏消息
func (c *OppoPush) broadcast(broadcast *Broadcast) (*BroadcastSendResult, error) {
	tokenInstance, err := GetToken(c.cfg.AppKey, c.cfg.AppSecret)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Add("message_id", broadcast.MessageID)
	params.Add("target_type", strconv.Itoa(broadcast.TargetType))
	params.Add("target_value", broadcast.TargetValue)
	params.Add("auth_token", tokenInstance.AccessToken)
	bytes, err := doPost(PushHost+MessageBroadcastURL, params)
	if err != nil {
		return nil, err
	}
	var result BroadcastSendResult
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return &result, errors.New(result.Message)
	}
	return &result, nil
}

// 单推-通知栏消息推送
func (c *OppoPush) unicast(accesstoken string, message *Message) (*UnicastSendResult, error) {
	params := url.Values{}
	params.Add("message", message.String())
	params.Add("auth_token", accesstoken)
	bytes, err := doPost(PushHost+MessageUnicastURL, params)
	if err != nil {
		return nil, err
	}
	var result UnicastSendResult
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, errors.New(result.Message)
	}
	return &result, nil
}

// 批量单推-通知栏消息推送
func (c *OppoPush) unicastBatch(accesstoken string, messages []Message) (*UnicastBatchSendResult, error) {
	jsons, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add("messages", string(jsons))
	params.Add("auth_token", accesstoken)
	bytes, err := doPost(PushHost+MessageUnicastBatchURL, params)
	if err != nil {
		return nil, err
	}
	var result UnicastBatchSendResult
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, errors.New(result.Message)
	}
	return &result, nil
}

// 获取失效的registration_id列表
func (c *OppoPush) fetchInvalidRegidList(accesstoken string) (*FetchInvalidRegidListSendResult, error) {
	params := url.Values{}
	params.Add("auth_token", accesstoken)
	bytes, err := doGet(FeedbackHost+FetchInvalidRegidListURL, "?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var result FetchInvalidRegidListSendResult
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, errors.New(result.Message)
	}
	return &result, nil
}

func defaultForm(msg *NotificationMessage) url.Values {
	form := url.Values{}
	if msg.AppMessageID != "" {
		form.Add("app_message_id", msg.AppMessageID)
	}
	form.Add("title", msg.Title)
	if msg.SubTitle != "" {
		form.Add("sub_title", msg.SubTitle)
	}
	form.Add("content", msg.Content)
	if msg.ClickActionType > 0 {
		form.Add("click_action_type", strconv.Itoa(msg.ClickActionType))
	}
	if msg.ClickActionType == 1 || msg.ClickActionType == 4 {
		form.Add("click_action_activity", msg.ClickActionActivity)
	}
	if msg.ClickActionType == 2 || msg.ClickActionType == 5 {
		form.Add("click_action_url", msg.ClickActionURL)
	}
	if msg.ActionParameters != "" {
		form.Add("action_parameters", msg.ActionParameters)
	}
	if msg.ShowTimeType > 0 {
		form.Add("show_time_type", strconv.Itoa(msg.ShowTimeType))
	}
	if msg.ShowTimeType > 0 {
		form.Add("show_start_time", strconv.FormatInt(msg.ShowStartTime, 10))
	}
	if msg.ShowTimeType > 0 {
		form.Add("show_end_time", strconv.FormatInt(msg.ShowEndTime, 10))
	}
	if !msg.OffLine {
		form.Add("off_line", strconv.FormatBool(msg.OffLine))
	}
	if msg.OffLine && msg.OffLineTTL > 0 {
		form.Add("off_line_ttl", strconv.Itoa(msg.OffLineTTL))
	}
	if msg.PushTimeType > 0 {
		form.Add("push_time_type", strconv.Itoa(msg.PushTimeType))
	}
	if msg.PushTimeType > 0 {
		form.Add("push_start_time", strconv.FormatInt(msg.PushStartTime, 10))
	}
	if msg.TimeZone != "" {
		form.Add("time_zone", msg.TimeZone)
	}
	if msg.FixSpeed {
		form.Add("fix_speed", strconv.FormatBool(msg.FixSpeed))
	}
	if msg.FixSpeed {
		form.Add("fix_speed_rate", strconv.FormatInt(msg.FixSpeedRate, 10))
	}
	if msg.NetworkType > 0 {
		form.Add("network_type", strconv.Itoa(msg.NetworkType))
	}
	if msg.CallBackURL != "" {
		form.Add("call_back_url", msg.CallBackURL)
	}
	if msg.CallBackParameter != "" {
		form.Add("call_back_parameter", msg.CallBackParameter)
	}
	if msg.ChannelID != "" {
		form.Add("channel_id", msg.ChannelID)
	}
	if msg.Style > 0 {
		form.Add("style", strconv.FormatInt(msg.Style, 10))
	}
	if msg.BigPictureId != "" {
		form.Add("big_picture_id", msg.BigPictureId)
	}
	return form
}

func (u *Message) String() string {
	bytes, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func formatStatus(input int) int64 {
	status := input
	switch input {
	case -2: //服务器流量控制
		status = common.CALLBACK_STATUS_PUSH_TOTAL_RATE_LIMIT
	case -1:
		status = common.CALLBACK_STATUS_NEED_RETRY
	case 0:
		status = common.CALLBACK_STATUS_OK
	case 13:
		status = common.CALLBACK_STATUS_PUSH_TOTAL_LIMIT
	case 33:
		status = common.CALLBACK_STATUS_PUSH_TOTAL_LIMIT
	default:
	}
	return int64(status)
}
