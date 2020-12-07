package huawei

import (
	"context"
	"encoding/json"
	"fmt"

	model "push_go/push_sdks/common"
)

func sendConditionMessage() error {
	msgRequest, err := getConditionMsgRequest()
	if err != nil {
		return fmt.Errorf("Failed to get message request! Error is %s\n", err.Error())

	}

	resp, err := client.SendMessage(context.Background(), msgRequest)
	if err != nil {
		return fmt.Errorf("Failed to send message! Error is %s\n", err.Error())
	}

	if resp.Code != Success {
		return fmt.Errorf("Failed to send message! Response is %+v\n", resp)
	}

	fmt.Printf("Succeed to send message! Response is %+v\n", resp)
	return nil
}

func getConditionMsgRequest() (*model.MessageRequest, error) {
	msgRequest := model.NewNotificationMsgRequest()
	msgRequest.Message.Android = model.GetDefaultAndroid()
	msgRequest.Message.Condition = TargetCondition
	msgRequest.Message.Android.Notification = model.GetDefaultAndroidNotification()

	b, err := json.Marshal(msgRequest)
	if err != nil {
		fmt.Printf("Failed to marshal the default message! Error is %s\n", err.Error())
		return nil, err
	}

	fmt.Printf("Default message is %s\n", string(b))
	return msgRequest, nil
}
