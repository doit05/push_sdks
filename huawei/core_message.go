package huawei

import (
	"push_go/clients"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	model "push_go/push_sdks/common"
)

// SendMessage sends a message to huawei cloud common
// One of Token, Topic and Condition fields must be invoked in message
// If validationOnly is set to true, the message can be verified by not sent to users
func (c *HttpPushClient) SendMessage(ctx context.Context, msgRequest *model.MessageRequest) (*model.MessageResponse, error) {
	result := &model.MessageResponse{}

	err := ValidateMessage(msgRequest.Message)
	if err != nil {
		return nil, err
	}

	request, err := c.getSendMsgRequest(msgRequest)
	if err != nil {
		return nil, err
	}

	err = c.executeApiOperation(ctx, request, result)
	if err != nil {
		return result, err
	}
	return result, err
}

func (c *HttpPushClient) getSendMsgRequest(msgRequest *model.MessageRequest) (*clients.Request, error) {
	body, err := json.Marshal(msgRequest)
	if err != nil {
		return nil, err
	}

	request := &clients.Request{
		Method: http.MethodPost,
		URL:    fmt.Sprintf(SendMessageFmt, c.endpoint, c.appId),
		Body:   body,
		Header: []clients.HTTPOption{
			clients.SetHeader("Content-Type", "application/json;charset=utf-8"),
			clients.SetHeader("Authorization", "Bearer "+c.token),
		},
	}
	return request, nil
}
