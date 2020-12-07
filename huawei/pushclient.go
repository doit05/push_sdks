package huawei

import (
	"push_go/clients"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"reflect"

	"push_go/config"
)

type HttpPushClient struct {
	endpoint   string
	appId      string
	token      string
	authClient *AuthClient
	client     *clients.HTTPClient
}

// NewMQClient creates a instance of the huawei cloud common client
// It's contained in huawei cloud app and provides service through huawei cloud app
func NewHttpClient(c *config.PushServerCfg) (*HttpPushClient, error) {
	if c.AppId == "" {
		return nil, errors.New("appId can't be empty")
	}

	client := clients.NewHTTPClient()

	authClient, err := NewAuthClient(c)
	if err != nil {
		return nil, err
	}

	return &HttpPushClient{
		endpoint:   c.PushUrl,
		appId:      c.AppId,
		token:      "",
		authClient: authClient,
		client:     client,
	}, nil
}

func (c *HttpPushClient) refreshToken() error {
	if c.authClient == nil {
		return errors.New("can't refresh token because getting auth client fail")
	}

	token, _, err := c.authClient.GetAuthToken(context.Background())
	if err != nil {
		return errors.New("refresh token fail")
	}

	c.token = token
	return nil
}

func (c *HttpPushClient) executeApiOperation(ctx context.Context, request *clients.Request, responsePointer interface{}) error {
	err := c.sendHttpRequest(ctx, request, responsePointer)
	if err != nil {
		return err
	}

	// if need to retry for token timeout or other reasons
	retry, err := c.isNeedRetry(responsePointer)
	if err != nil {
		return err
	}

	if retry {
		err = c.sendHttpRequest(ctx, request, responsePointer)
		return err
	}
	return err
}

func (c *HttpPushClient) sendHttpRequest(ctx context.Context, request *clients.Request, responsePointer interface{}) error {
	resp, err := c.client.DoHttpRequest(ctx, request)
	if err != nil {
		return err
	}
	if resp.Status >= http.StatusInternalServerError {
		return fmt.Errorf("server error status:[%d] body:[%s]", resp.Status, string(resp.Body))
	}
	if err = json.Unmarshal(resp.Body, responsePointer); err != nil {
		log.WithError(err).Infof("json decode error response:[%s]", string(resp.Body))
		return fmt.Errorf("response json decode error:%w ", err)
	}
	return nil
}

// if token is timeout or error or other reason, need to refresh token and send again
func (c *HttpPushClient) isNeedRetry(responsePointer interface{}) (bool, error) {
	tokenError, err := isTokenError(responsePointer)
	if err != nil {
		return false, err
	}

	if !tokenError {
		return false, nil
	}

	err = c.refreshToken()
	if err != nil {
		return false, err
	}
	return true, nil
}

// if token is timeout or error, refresh token and send again
func isTokenError(responsePointer interface{}) (bool, error) {
	//the responsePointer must be point of struct
	val, _, ok := checkParamStructPtr(responsePointer)
	if !ok {
		return false, errors.New("the parameter should be pointer of the struct")
	}

	code := val.Elem().FieldByName("Code").String()
	if code == TokenTimeoutErr || code == TokenFailedErr {
		return true, nil
	}
	return false, nil
}

func checkParamStructPtr(structPtr interface{}) (reflect.Value, reflect.Type, bool) {
	val := reflect.ValueOf(structPtr)
	if val.Kind() != reflect.Ptr {
		fmt.Println("The Parameter should be Pointer of Struct!")
		return reflect.Value{}, nil, false
	}

	t := reflect.Indirect(val).Type()
	if t.Kind() != reflect.Struct {
		fmt.Println("The Parameter should be Pointer of Struct!")
		return reflect.Value{}, nil, false
	}
	return val, t, true
}
