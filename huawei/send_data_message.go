package huawei

import (
	"context"
	"encoding/json"
	"fmt"

	model "push_sdks/common"
)

func sendDataMessage(tokens []string, msg_data string) error {
	msgRequest, err := getDataMsgRequest(tokens, msg_data)
	if err != nil {
		err = fmt.Errorf("Failed to get message request! Error is %s\n", err.Error())
		return err
	}

	httpclient := pushClient
	resp, err := httpclient.SendMessage(context.Background(), msgRequest)
	if err != nil {
		err = fmt.Errorf("Failed to send message! Error is %s\n", err.Error())
		return err
	}

	if resp.Code != Success {
		err = fmt.Errorf("Failed to send message! Response is %+v\n", resp)
		return err
	}

	fmt.Printf("Succeed to send message! Response is %+v\n", resp)
	return nil
}

func getDataMsgRequest(tokens []string, msg_data string) (*model.MessageRequest, error) {
	msgRequest := model.NewTransparentMsgRequest()
	msgRequest.Message.Android = model.GetDefaultAndroid()
	msgRequest.Message.Token = tokens
	msgRequest.Message.Data = msg_data

	b, err := json.Marshal(msgRequest)
	if err != nil {
		fmt.Printf("Failed to marshal the default message! Error is %s\n", err.Error())
		return nil, err
	}

	fmt.Printf("Default message is %s\n", string(b))
	return msgRequest, nil
}
