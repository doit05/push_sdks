package meizupush

import (
	"push_sdks/clients"
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ddliu/go-httpclient"
)

const (
	pushThroughMessageByPushId      = PUSH_API_SERVER + "/garcia/api/server/push/unvarnished/pushByPushId"
	pushNotificationMessageByPushId = PUSH_API_SERVER + "/garcia/api/server/push/varnished/pushByPushId"
	pushThroughMessageByAlias       = PUSH_API_SERVER + "/garcia/api/server/push/unvarnished/pushByAlias"
	pushNotificationMessageByAlias  = PUSH_API_SERVER + "/garcia/api/server/push/varnished/pushByAlias"
)

/**
 * 通过PushId推送透传消息
 */
func PushThroughMessageByPushId(appId string, pushIds string, messageJson string, appKey string) PushResponse {
	pushThroughMessageMap := map[string]string{
		"appId":       appId,
		"pushIds":     pushIds,
		"messageJson": messageJson,
	}

	sign := GenerateSign(pushThroughMessageMap, appKey)
	pushThroughMessageMap["sign"] = sign

	res, err := httpclient.Post(pushThroughMessageByPushId, pushThroughMessageMap)

	return ResolvePushResponse(res, err)
}

//pushId推送接口（通知栏消息）
func (c *Client) PushNotificationMessageByPushId(appId string, pushIds string, messageJson string, appKey string) PushResponse {
	pushNotificationMessageMap := map[string]string{
		"appId":       appId,
		"pushIds":     pushIds,
		"messageJson": messageJson,
	}

	sign := GenerateSign(pushNotificationMessageMap, appKey)
	pushNotificationMessageMap["sign"] = sign

	result, err := Post(c.client, pushNotificationMessageByPushId, pushNotificationMessageMap)
	response := PushResponse{}
	if err != nil {
		response = PushResponse{
			Message: err.Error(),
		}
	} else {
		err = json.Unmarshal(result.Body, &response)
		if err != nil {
			response.Message = err.Error()
		}
		if response.RetCode != "" {
			response.Code, err = strconv.Atoi(response.RetCode)
			if err != nil {
				response.Message = err.Error()
			}
		}
	}
	return response
}

//别名推送接口（透传消息
func PushThroughMessageByAlias(appId string, alias string, messageJson string, appKey string) PushResponse {
	pushThroughMessageMap := map[string]string{
		"appId":       appId,
		"alias":       alias,
		"messageJson": messageJson,
	}

	sign := GenerateSign(pushThroughMessageMap, appKey)
	pushThroughMessageMap["sign"] = sign

	res, err := httpclient.Post(pushThroughMessageByAlias, pushThroughMessageMap)

	return ResolvePushResponse(res, err)
}

//别名推送接口（通知栏消息）
func PushNotificationMessageByAlias(appId string, alias string, messageJson string, appKey string) PushResponse {
	pushNotificationMessageMap := map[string]string{
		"appId":       appId,
		"alias":       alias,
		"messageJson": messageJson,
	}

	sign := GenerateSign(pushNotificationMessageMap, appKey)
	pushNotificationMessageMap["sign"] = sign

	res, err := httpclient.Post(pushNotificationMessageByAlias, pushNotificationMessageMap)

	return ResolvePushResponse(res, err)
}

func Post(client *clients.HTTPClient, url string, params interface{}) (*clients.Response, error) {
	paramsValues := toUrlValues(params)
	body := paramsValues.Encode()

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Close = true
	result, err := client.DoRequest(req)
	return result, err
}

func toUrlValues(v interface{}) url.Values {
	switch t := v.(type) {
	case url.Values:
		return t
	case map[string][]string:
		return url.Values(t)
	case map[string]string:
		rst := make(url.Values)
		for k, v := range t {
			rst.Add(k, v)
		}
		return rst
	case nil:
		return make(url.Values)
	default:
		panic("Invalid value")
	}
}
