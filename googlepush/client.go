package googlepush

import (
	"push_sdks/config"
	"push_sdks/common"
	"fmt"
	"net/http"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/google/martian/log"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

//Client
type Client struct {
	cfg       *config.PushServerCfg
	msgClient *messaging.Client
}

//NewClient resturns an instance of messaging.Client
func NewClient(cfg config.PushServerCfg) (*Client, error) {
	opt := option.WithCredentialsFile(cfg.ExtraConfigFile)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("%s initializing app error : [%v]", cfg.Name, err)
	}

	ctx := context.Background()
	msgClient, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s initializing msg client error : [%v]", cfg.Name, err)
	}

	return &Client{cfg: &cfg, msgClient: msgClient}, nil
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
	message := &messaging.MulticastMessage{
		Data: map[string]string{
			"score": "850",
		},
		Tokens: tokens,
	}
	message.Notification = &messaging.Notification{}
	message.Notification.Title = msg.MsgTitle
	message.Notification.Body = msg.MsgBody
	message.Notification.ImageURL = msg.ImgUrl
	message.Android = &messaging.AndroidConfig{}
	message.Android.Notification = &messaging.AndroidNotification{
		Title:       msg.MsgTitle,
		Body:        msg.MsgBody,
		ClickAction: "first_open",
		Visibility:  messaging.VisibilityPrivate,
	}

	br, err := c.msgClient.SendMulticast(context.Background(), message)
	if err != nil {
		return nil, fmt.Errorf("%s push msg error:[%v] message:[%v]", c.cfg.Name, err, message)
	}
	log.Infof("%s push msg results:[%v] message:[%v]", c.cfg.Name, br, message)
	return nil, err
}

func (c *Client) PushReciver(r *http.Request) (*common.CallbackResponse, map[string]interface{}, error) {
	return nil, nil, nil
}

func (c *Client) RegisterDeviceToken(deviceId string) (deviceToken string, err error) {
	return deviceId, nil
}
