package meizupush

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ddliu/go-httpclient"
)

//android 客户端订阅接口
func Register(appId string, appKey string, deviceId string) (string, error) {
	registerRequestMap := map[string]string{
		"appId":    appId,
		"deviceId": deviceId,
	}

	res, err := httpclient.Post(SERVER+"message/registerPush", map[string]string{
		"appId":    appId,
		"deviceId": deviceId,
		"sign":     GenerateSign(registerRequestMap, appKey),
	})
	response := &RegisterPushResponse{}
	err = ResolveResponse(res, err, response)
	if err != nil {
		return "", err
	}
	if response.Code != "200" {
		return "", fmt.Errorf(response.Message)
	}
	if response.Value.PushId == "" {
		return "", fmt.Errorf(response.Message)
	}
	return response.Value.PushId, nil
}

type RegisterPushResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Value   struct {
		ExpireTime int64  `json:"expireTime"`
		PushId     string `json:"pushId"`
	} `json:"value"`
}

//resolve push response
func ResolveResponse(res *httpclient.Response, err error, response interface{}) error {
	if err != nil {
		return err
	}

	if res == nil {
		return fmt.Errorf("registerPush empty reponse")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("registerPush empty reponse status:%v", res.Status)
	}

	err = json.Unmarshal(data, response)
	if err != nil {
		return fmt.Errorf("registerPush invalid reponse")
	}
	return nil
}
