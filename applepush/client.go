package applepush

import (
	"push_sdks/config"
	"push_sdks/common"
	"fmt"
	"net/http"
	"strings"

	"github.com/sideshow/apns2/payload"

	log "github.com/sirupsen/logrus"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
)

type Client struct {
	cfg    *config.PushServerCfg
	client *apns2.Client
}

type aps struct {
	Alter alter `json:"alter"`
}

type alter struct {
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	Body        string `json:"body"`
	LaunchImage string `json:"launch_image"`
}

func NewClient(cfg config.PushServerCfg) (*Client, error) {
	cert, err := certificate.FromP12File(cfg.ExtraConfigFile, cfg.ExtraConfigFilePassword)
	if err != nil {
		return nil, fmt.Errorf("%s NewClient error: [%v]", cfg.Name, err)
	}
	client := apns2.NewClient(cert).Production()
	if cfg.TestMod {
		client = apns2.NewClient(cert).Development()
	}
	return &Client{cfg: &cfg, client: client}, nil
}

func (c *Client) Name() string {
	return c.cfg.Name
}

func (c *Client) NeedAccessToken() bool {
	return c.cfg.NeedAccessToken
}

func (c *Client) GetToken() (token string, expire_in int, err error) {
	return "", 0, nil
}

func (c *Client) formatMsg(msg *common.Msg) *apns2.Notification {
	notification := &apns2.Notification{}
	notification.Priority = 10 //ios push 设置最高优先级
	notification.Topic = c.cfg.Package
	aps := aps{}
	aps.Alter.Title = msg.MsgTitle
	aps.Alter.Subtitle = msg.SubMsgTile
	if len(aps.Alter.Subtitle) > 10 {
		aps.Alter.Subtitle = aps.Alter.Subtitle[:10]
	}
	aps.Alter.Body = msg.MsgBody
	aps.Alter.LaunchImage = msg.ImgUrl
	payload := payload.NewPayload().Alert("hello").Badge(1).Custom("presentBadge", 1).Custom("presentSound", 1).Custom("appData", msg.MsgAction).Custom("presentAlert", 1)
	payload = payload.AlertBody(msg.MsgBody).AlertTitle(msg.MsgTitle)
	payload = payload.AlertAction("action").AlertActionLocKey("PLAY")

	notification.Payload = payload
	return notification
}

func (c *Client) PushMsg(msg *common.Msg, tokens []string) (failsInfoMap map[string]*common.CallbackResponseItem, err error) {
	notification := c.formatMsg(msg)
	for _, token := range tokens {
		notification.DeviceToken = token
		res, err := c.client.Push(notification)
		if res != nil {
			if failsInfoMap == nil {
				failsInfoMap = make(map[string]*common.CallbackResponseItem, 1)
			}
			failsInfoMap[token] = &common.CallbackResponseItem{
				Status:       formatStatus(res),
				Description:  res.Reason,
				RequestId:    res.ApnsID,
				Token:        token,
				DeviceVendor: c.cfg.Name,
				PackageName:  c.cfg.Package,
			}
		}
		if err != nil {
			log.WithError(err).WithField("result", res).Warnf("%s send msg Error:", c.cfg.Name)
			return failsInfoMap, err
		}
		log.WithField("result", res).WithField("msg", msg).Debugf("%s send msg", c.cfg.Name)
	}
	return failsInfoMap, nil
}

func (c *Client) PushReciver(r *http.Request) (*common.CallbackResponse, map[string]interface{}, error) {
	return nil, nil, nil
}

func (c *Client) RegisterDeviceToken(deviceId string) (deviceToken string, err error) {
	return deviceId, nil
}

func formatStatus(res *apns2.Response) int64 {
	status := res.StatusCode
	switch res.StatusCode {
	case 200: //Success
		status = common.CALLBACK_STATUS_OK
	case 400: //Bad request.
		if strings.Contains(res.Reason, "BadDeviceToken") {
			status = common.CALLBACK_STATUS_INVALID_DEVICE_TOKEN
		}
	case 403: //There was an error with the certificate or with the provider’s authentication token.
	case 405: // The request used an invalid :method value. Only POST requests are supported.
	case 410: //The device token is no longer active for the topic.
		status = common.CALLBACK_STATUS_INACTIVE_DEVICE_TOKEN
	case 413: //The notification payload was too large.
	case 429: //The server received too many requests for the same device token.
		status = common.CALLBACK_STATUS_PUSH_RATE_LIMIT
	case 500:
		status = common.CALLBACK_STATUS_NEED_RETRY
	case 503:
		status = common.CALLBACK_STATUS_NEED_RETRY
	default:
	}
	return int64(status)
}
