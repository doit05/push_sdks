package xiaomipush

import (
	"push_go/config"
	"push_go/push_sdks/common"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

//消息payload，根据业务自定义
type Payload struct {
	PushTitle    string `json:"push_title"`
	PushBody     string `json:"push_body"`
	IsShowNotify string `json:"is_show_notify"`
	Ext          string `json:"ext"`
}

type Client struct {
	cfg    *config.PushServerCfg
	mipush *MiPush
}

//获取实例
func NewClient(config config.PushServerCfg) (*Client, error) {
	if config.Package == "" || config.AppSecret == "" {
		return nil, errors.New("请检查配置")
	}
	xm := &Client{
		cfg:    &config,
		mipush: NewMiPushClient(config.AppSecret, []string{config.Package}),
	}

	return xm, nil
}

func (m *Client) Name() string {
	return m.cfg.Name
}

func (m *Client) NeedAccessToken() bool {
	return m.cfg.NeedAccessToken
}

func (m *Client) GetToken() (string, int, error) {
	return "", 0, nil
}

func (m *Client) PushMsg(msg *common.Msg, tokens []string) (failsInfoMap map[string]*common.CallbackResponseItem, err error) {
	msg1 := NewAndroidMessage(msg.MsgTitle, msg.MsgBody).SetPayload(msg.MsgAction).SetNotifyID(msg.Id).SetTimeToSend(time.Now().Unix() * 1000)
	params := common.CallbackParam{MsgId: msg.Id, Package: m.cfg.Package}
	paramsData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("?deviceVendor=%s", m.Name())
	msg1 = msg1.SetCallback(m.cfg.Redirect+query, string(paramsData))
	if msg.ImgUrl != "" {
		result, err := m.mipush.UploadImg(context.TODO(), msg.ImgUrl)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"imgUrl": msg.ImgUrl,
				"result": result,
			}).Errorf("%s upload img error", m.cfg.Name)
		}
		if result != nil && result.Data.PicUrl != "" {
			msg1.Extra["notification_bigPic_uri"] = result.Data.PicUrl
			msg1.Extra["notification_style_type"] = "2"
		}
	}
	msg1.Extra["channel_id"] = msg.ChannelID
	res, err := m.mipush.SendToList(context.Background(), msg1, tokens)
	if res != nil {
		if failsInfoMap == nil {
			failsInfoMap = make(map[string]*common.CallbackResponseItem, len(tokens))
		}
		for _, token := range tokens {
			failsInfoMap[token] = &common.CallbackResponseItem{
				Status:       res.Code,
				Description:  res.Reason,
				RequestId:    res.MessageID,
				Token:        token,
				DeviceVendor: m.cfg.Name,
				PackageName:  m.cfg.Package,
			}
		}
	}
	if err != nil {
		return failsInfoMap, fmt.Errorf("%s broadcast msg error:[%v]", m.cfg.Name, err)
	}
	log.WithFields(log.Fields{"name": m.Name(), "msgInfo": *msg1, "tokens": tokens, "err": err}).Debugf("push msg result: %v", res)
	return failsInfoMap, err
}

type CallBackItem struct {
	Param     string                 `json:"param"`
	Status    int64                  `json:"type"`
	Targets   string                 `json:"targets"`
	Jobkey    string                 `json:"jobkey"`
	BarStatus string                 `json:"barStatus"`
	TimeStamp int64                  `json:"timeStamp"`
	Extra     map[string]interface{} `json:"extra"`
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
	if data = form.Get("data"); data == "" {
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
		item.Token = val.Targets
		item.RequestId = requestId
		item.Timestamp = val.TimeStamp
		item.PackageName = params.Package
		status := val.Status
		switch val.Status {
		case 1:
			status = common.CALLBACK_STATUS_OK
		case 2:
			status = common.CALLBACK_STATUS_OK
		case 3:
			status = common.CALLBACK_STATUS_OK
		case 16:
			status = common.CALLBACK_STATUS_INVALID_DEVICE_TOKEN
		case 32:
			status = common.CALLBACK_STATUS_DISABLE_PUSH
		case 64: //64：目标设备不符合过滤条件（包括网络条件不符合、地理位置不符合、App版本不符合、机型不符合、地区语言不符合等）。
		case 128:
			status = common.CALLBACK_STATUS_PUSH_TOTAL_LIMIT
		default:
			status = common.CALLBACK_STATUS_NEED_RETRY
		}
		item.Status = status
		callbackResponse.Data = append(callbackResponse.Data, item)
	}
	return &callbackResponse, res, nil
}

func (c *Client) RegisterDeviceToken(deviceId string) (deviceToken string, err error) {
	return deviceId, nil
}
