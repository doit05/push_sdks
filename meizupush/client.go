package meizupush

import (
	"push_sdks/clients"
	"push_sdks/config"
	"push_sdks/common"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

//Client
type Client struct {
	cfg    *config.PushServerCfg
	client *clients.HTTPClient
}

//NewClient resturns an instance of messaging.Client
func NewClient(cfg config.PushServerCfg) (*Client, error) {
	return &Client{cfg: &cfg, client: clients.NewHTTPClient()}, nil
}

//NeedAccessToken returns bool
func (c *Client) NeedAccessToken() bool {
	return c.cfg.NeedAccessToken
}

func (c *Client) Name() string {
	return c.cfg.Name
}

func (c *Client) GetToken() (token string, expire_in int, err error) {
	return token, 0, err
}

func (c *Client) PushMsg(msg *common.Msg, tokens []string) (failsInfoMap map[string]*common.CallbackResponseItem, err error) {
	msgData := BuildNotificationMessage().
		noticeBarType(2).
		noticeTitle(msg.MsgTitle).
		noticeContent(msg.MsgBody)
	msgData.ClickTypeInfo.Parameters = map[string]interface{}{
		"pushParams": msg.MsgAction, //跟客户端协商字段pathParams
	}
	if c.cfg.Redirect != "" {
		msgData.Extra = map[string]interface{}{}
		msgData.Extra["callback"] = c.cfg.Redirect + fmt.Sprintf("?deviceVendor=%s", c.Name())
		params := common.CallbackParam{MsgId: msg.Id, Package: c.cfg.Package}
		paramsData, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		msgData.Extra["callback.param"] = string(paramsData)
		msgData.Extra["callback.type"] = 3
	}

	pushid := strings.Join(tokens, ",")
	res := c.PushNotificationMessageByPushId(c.cfg.AppId, pushid, msgData.toJson(), c.cfg.AppSecret)
	log.WithFields(log.Fields{"appid": c.cfg.AppId, "sercret": c.cfg.AppSecret, "res": res, "push msg": msgData, "pushid": pushid}).Tracef("%s send msg", c.cfg.Name)
	if failsInfoMap == nil {
		failsInfoMap = make(map[string]*common.CallbackResponseItem, len(tokens))
	}

	for _, token := range tokens {
		failsInfoMap[token] = &common.CallbackResponseItem{
			Status:       formatStatus(res.Code),
			Description:  res.Message,
			MsgId:        msg.Id,
			RequestId:    res.MsgId,
			Token:        token,
			DeviceVendor: c.cfg.Name,
			PackageName:  c.cfg.Package,
		}
	}

	if res.Code != 200 {
		log.WithFields(log.Fields{"res": res, "push msg": msgData}).Infof("%s send msg error", c.cfg.Name)
		return failsInfoMap, fmt.Errorf("%s push msg msg:[%v]", c.cfg.Name, res.Message)
	}
	log.WithField("result", res).Debugf("meizu end send msg")
	return failsInfoMap, nil
}

type CallBackItem struct {
	Param   string   `json:"param"`
	Status  int64    `json:"type"`
	Targets []string `json:"targets"`
}

func (c *Client) PushReciver(r *http.Request) (*common.CallbackResponse, map[string]interface{}, error) {
	res := map[string]interface{}{"errno": 0, "errmsg": "success"}
	form := r.Form
	log.WithFields(log.Fields{
		"deviceVendor": c.cfg.Name,
		"url":          r.URL.RequestURI(),
		"header":       r.Header,
		"form":         form,
	}).Tracef("callback")
	data := ""
	if data = form.Get("cb"); data == "" {
		log.WithFields(log.Fields{
			"deviceVendor": c.cfg.Name,
			"url":          r.URL.RequestURI(),
			"header":       r.Header,
			"form":         form,
		}).Errorf("empty params data")
		return nil, nil, fmt.Errorf("empty params data")
	}
	var reciver map[string]CallBackItem
	err := json.Unmarshal([]byte(data), &reciver)
	if err != nil {
		log.WithFields(log.Fields{
			"deviceVendor": c.cfg.Name,
			"url":          r.URL.RequestURI(),
			"header":       r.Header,
			"form":         form,
		}).Errorf("empty params data")
		return nil, nil, err
	}
	var callbackResponse common.CallbackResponse
	for requestId, val := range reciver {
		item := &common.CallbackResponseItem{}
		params := common.CallbackParam{}
		err := json.Unmarshal([]byte(val.Param), &params)
		if err != nil {
			log.WithError(err).WithField("deviceVendor", c.cfg.Name).Errorf("invalid callback param: %v", val.Param)
			continue
		}
		item.MsgId = params.MsgId
		item.Token = strings.Join(val.Targets, ",")
		item.RequestId = requestId
		item.Timestamp = time.Now().UnixNano() / 1000
		item.PackageName = params.Package
		status := val.Status
		switch val.Status {
		case 1:
			status = common.CALLBACK_STATUS_OK
		case 2:
			status = common.CALLBACK_STATUS_OK
		case 3:
			status = common.CALLBACK_STATUS_OK
		default:
			status = common.CALLBACK_STATUS_NEED_RETRY
		}
		item.Status = status
		callbackResponse.Data = append(callbackResponse.Data, item)
	}
	return &callbackResponse, res, nil
}

func formatStatus(input int) int64 {
	status := input
	switch input {
	case 200:
		status = common.CALLBACK_STATUS_OK
	case 500: //其他异常
	case 1001:
		status = common.CALLBACK_STATUS_NEED_RETRY //系统错误
	case 1003: //服务器繁忙
		status = common.CALLBACK_STATUS_NEED_RETRY
	case 110010:
		status = common.CALLBACK_STATUS_PUSH_TOTAL_RATE_LIMIT
	case 110053:
		status = common.CALLBACK_STATUS_PUSH_RATE_LIMIT
	default:
		status = common.CALLBACK_STATUS_NEED_RETRY
	}
	return int64(status)
}
