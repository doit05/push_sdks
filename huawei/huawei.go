/*

https://developer.huawei.com/consumer/cn/doc/development/HMSCore-Examples-V5/server-go-sample-code-0000001051066004-V5
*/
package huawei

import (
	"push_sdks/config"
	"push_sdks/common"
	model "push_sdks/common"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type HuaweiClient struct {
	cfg    *config.PushServerCfg
	client *HttpPushClient
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func NewClient(conf config.PushServerCfg) (*HuaweiClient, error) {
	pushClient = GetPushClient(&conf)
	return &HuaweiClient{client: pushClient, cfg: &conf}, nil
}

func (c *HuaweiClient) Name() string {
	return c.cfg.Name
}

func (c *HuaweiClient) NeedAccessToken() bool {
	return c.cfg.NeedAccessToken
}

func (c *HuaweiClient) GetToken() (string, int, error) {
	token, expire_in, err := c.client.authClient.GetAuthToken(context.Background())
	if err == nil {
		c.client.token = token
	}
	return token, expire_in, err
}

func (c *HuaweiClient) PushMsg(msg *common.Msg, tokens []string) (failsInfoMap map[string]*common.CallbackResponseItem, err error) {
	msgRequest, err := c.getMsgRequest(msg, tokens)
	if err != nil {
		log.Errorf("Failed to get message request! Error is %s\n", err.Error())
		return nil, err
	}

	resp, err := client.SendMessage(context.Background(), msgRequest)
	if err != nil {
		log.WithFields(log.Fields{
			"msg":    msg,
			"tokens": tokens,
		}).Infof("Failed to send message! Error is %s\n", err.Error())
		return nil, err
	}

	failsInfoMap = make(map[string]*common.CallbackResponseItem, len(tokens))
	status, err := strconv.ParseInt(resp.Code, 10, 64)
	if err != nil {
		status = common.UNKONW
	} else {
		status = formatStatus(status)
	}
	for _, token := range tokens {
		failsInfoMap[token] = &common.CallbackResponseItem{
			Status:       status,
			Description:  resp.Msg,
			RequestId:    resp.RequestId,
			Token:        token,
			DeviceVendor: c.cfg.Name,
			PackageName:  c.cfg.Package,
		}
	}

	if resp.Code != Success && status != common.CALLBACK_STATUS_INVALID_DEVICE_TOKEN {
		log.Errorf("Failed to send message! Response is %+v\n", resp)
		return failsInfoMap, fmt.Errorf("Failed to send message! Response is %+v", resp)
	}

	log.Debugf("Succeed to send message! Response is %+v\n", resp)
	return failsInfoMap, nil
}

func (c *HuaweiClient) getMsgRequest(msg *common.Msg, tokens []string) (*model.MessageRequest, error) {
	msgRequest := model.NewNotificationMsgRequest()
	msgRequest.Message.Data = "msgRequest.Message.Data"
	msgRequest.Message.Token = tokens
	msgRequest.Message.Android = model.GetDefaultAndroid()
	msgRequest.Message.Android.Notification = model.GetDefaultAndroidNotification()
	msgRequest.Message.Android.Data = "msgRequest.Message.Android.Data"
	msgRequest.Message.Android.Notification.Title = msg.MsgTitle
	msgRequest.Message.Android.Notification.NotifySummary = msg.SubMsgTile
	msgRequest.Message.Android.Notification.Body = msg.MsgBody
	msgRequest.Message.Android.Notification.Image = msg.ImgUrl
	msgRequest.Message.Android.Notification.DefaultSound = true
	msgRequest.Message.Android.BiTag = fmt.Sprintf("%d", msg.Id)
	msgRequest.Message.Android.Notification.Importance = ""
	msgRequest.Message.Android.Notification.ChannelId = msg.ChannelID

	msgRequest.Message.Android.Notification.ClickAction.Type = 1
	msgRequest.Message.Android.Notification.ClickAction.Intent = fmt.Sprintf("intent://%s/conversationlist?isFromPush=true#Intent;scheme=rong;launchFlags=0x4000000;S.options={\"appData\":\"%s\"};end", c.cfg.Package, msg.MsgAction)
	log.Debugf("Default message is %+v\n", msgRequest)
	return msgRequest, nil
}

type callbackRequest struct {
	Data []*common.CallbackResponseItem `json:"statuses"`
}

func (c *HuaweiClient) PushReciver(r *http.Request) (*common.CallbackResponse, map[string]interface{}, error) {
	res := map[string]interface{}{"errno": 0, "errmsg": "success"}
	var reciver callbackRequest
	requestBody, err := ioutil.ReadAll(r.Body)
	log.WithFields(log.Fields{
		"deviceVendor": c.cfg.Name,
		"url":          r.URL.RequestURI(),
		"header":       r.Header,
	}).Tracef("callback")
	err = json.Unmarshal(requestBody, &reciver)
	if err != nil {
		return nil, nil, err
	}
	var callbackResponse common.CallbackResponse
	for _, item := range reciver.Data {
		item.MsgId, err = strconv.ParseInt(item.BiTag, 10, 64)
		if err != nil {
			log.WithError(err).WithField("deviceVendor", c.cfg.Name).Errorf("invalid msg id: %v", item.BiTag)
			continue
		}
		status := formatStatus(item.Status)
		item.Status = status
		callbackResponse.Data = append(callbackResponse.Data, item)
	}
	return &callbackResponse, res, nil
}

func formatStatus(input int64) int64 {
	status := input
	switch input {
	case 0, 80000000:
		status = common.CALLBACK_STATUS_OK
	case 2:
		status = common.CALLBACK_STATUS_UNINSTALL_APP
	case 5, 80100000, 80300007:
		status = common.CALLBACK_STATUS_INVALID_DEVICE_TOKEN
	case 6:
		status = common.CALLBACK_STATUS_DISABLE_PUSH
	case 10:
		status = common.CALLBACK_STATUS_INACTIVE_DEVICE_TOKEN
	case 15: //离线用户消息覆盖 （目前不使用 ）
	case 27: //透传消息 目标应用进程不存在。（目前不使用 ）
	case 102:
		status = common.CALLBACK_STATUS_PUSH_RATE_LIMIT
	case 202:
		status = common.CALLBACK_STATUS_PUSH_TOTAL_LIMIT
	case 81000001:
		status = common.CALLBACK_STATUS_NEED_RETRY
	default:
		status = common.CALLBACK_STATUS_NEED_RETRY
	}
	return status
}

func (c *HuaweiClient) RegisterDeviceToken(deviceId string) (deviceToken string, err error) {
	return deviceId, nil
}
