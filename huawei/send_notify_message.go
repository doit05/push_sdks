package huawei

import (
	"context"
	"encoding/json"
	"fmt"

	model "push_sdks/common"
)

func sendNotifyMessage(tokens []string) {
	msgRequest, err := getNotifyMsgRequest(tokens)
	if err != nil {
		fmt.Printf("Failed to get message request! Error is %s\n", err.Error())
		return
	}

	resp, err := client.SendMessage(context.Background(), msgRequest)
	if err != nil {
		fmt.Printf("Failed to send message! Error is %s\n", err.Error())
		return
	}

	if resp.Code != Success {
		fmt.Printf("Failed to send message! Response is %+v\n", resp)
		return
	}

	fmt.Printf("Succeed to send message! Response is %+v\n", resp)
}

func getNotifyMsgRequest(tokens []string) (*model.MessageRequest, error) {
	msgRequest := model.NewNotificationMsgRequest()
	msgRequest.Message.Token = tokens
	msgRequest.Message.Android = model.GetDefaultAndroid()
	msgRequest.Message.Android.Notification = model.GetDefaultAndroidNotification()

	b, err := json.Marshal(msgRequest)
	if err != nil {
		fmt.Printf("Failed to marshal the default message! Error is %s\n", err.Error())
		return nil, err
	}

	fmt.Printf("Default message is %s\n", string(b))
	return msgRequest, nil
}
