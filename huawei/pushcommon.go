package huawei

import (
	"push_go/config"
	"fmt"
	"sync"
)

var (
	pushClient *HttpPushClient
	once       sync.Once
	client     *HttpPushClient
)

var (
	//TargetToken the topic to be subscribed/unsubscribed
	TargetTopic = "topic"

	//TargetCondition the condition of the devices operated
	TargetCondition = "'topic' in topics && ('topic' in topics || 'TopicC' in topics)"
)

func GetPushClient(conf *config.PushServerCfg) *HttpPushClient {
	once.Do(func() {
		client, err := NewHttpClient(conf)
		if err != nil {
			fmt.Printf("Failed to new common client! Error is %s\n", err.Error())
			panic(err)
		}
		pushClient = client
	})

	client = pushClient
	return pushClient
}
