package huawei

import (
	"push_go/clients"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"push_go/config"
)

type AuthClient struct {
	endpoint  string
	appId     string
	appSecret string
	client    *clients.HTTPClient
}

type TokenMsg struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	Scope            string `json:"scope"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// NewMQClient creates a instance of the huawei cloud auth client
// It's contained in huawei cloud app and provides service through huawei cloud app
// If AuthUrl is null using default auth url address
func NewAuthClient(conf *config.PushServerCfg) (*AuthClient, error) {
	if conf.AppId == "" || conf.AppSecret == "" {
		return nil, errors.New("appId or appSecret is null")
	}

	c := clients.NewHTTPClient()

	return &AuthClient{
		endpoint:  conf.AuthUrl,
		appId:     conf.AppId,
		appSecret: conf.AppSecret,
		client:    c,
	}, nil
}

// GetAuthToken gets token from huawei cloud
// the developer can access the app by using this token
func (ac *AuthClient) GetAuthToken(ctx context.Context) (string, int, error) {
	u, _ := url.Parse(ac.appSecret)
	body := fmt.Sprintf("grant_type=client_credentials&client_secret=%s&client_id=%s", u.String(), ac.appId)

	request := &clients.Request{
		Method: http.MethodPost,
		URL:    ac.endpoint,
		Body:   []byte(body),
		Header: []clients.HTTPOption{clients.SetHeader("Content-Type", "application/x-www-form-urlencoded")},
	}

	resp, err := ac.client.DoHttpRequest(ctx, request)
	if err != nil {
		return "", 0, err
	}

	var token TokenMsg
	if resp.Status == 200 {
		err = json.Unmarshal(resp.Body, &token)
		if err != nil {
			return "", 0, err
		}
		return token.AccessToken, token.ExpiresIn, nil
	} else {
		return "", 0, errors.New(string(resp.Body))
	}
	return "", 0, nil
}
