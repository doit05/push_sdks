package common

const (
	CALLBACK_STATUS_OK                    = 0
	CALLBACK_STATUS_NEED_RETRY            = 1  //需要重推
	CALLBACK_STATUS_INVALID_DEVICE_TOKEN  = 2  //无效的设备id
	CALLBACK_STATUS_INACTIVE_DEVICE_TOKEN = 3  //超过30天未接入的设备id
	CALLBACK_STATUS_PUSH_RATE_LIMIT       = 4  //消息频控丢弃
	CALLBACK_STATUS_PUSH_TOTAL_LIMIT      = 5  //消息单日发送总量限制
	CALLBACK_STATUS_DISABLE_PUSH          = 6  //客户端调用了disablePush接口禁用Push。
	CALLBACK_STATUS_UNINSTALL_APP         = 7  //应用未安装
	CALLBACK_STATUS_PUSH_TOTAL_RATE_LIMIT = 8  //应用推送速率过快
	CALLBACK_STATUS_RATE_LIMIT            = 9  //vivo发送频率控制，每15分钟发一条
	CALLBACK_STATUS_RATE_TOTAL_LIMIT      = 10 //vivo发送频率控制，每天8条
	CALLBACK_STATUS_FORBID_TOTAL_LIMIT    = 11 //vivo发送禁止发送typeid
	UNKONW                                = 10000
)

func GetCallBackMsg(status int64) string {
	msg := ""
	switch status {
	case CALLBACK_STATUS_OK:
		msg = "success"
	case CALLBACK_STATUS_NEED_RETRY:
		msg = "needretry"
	case CALLBACK_STATUS_INVALID_DEVICE_TOKEN:
		msg = "无效的设备id"
	case CALLBACK_STATUS_INACTIVE_DEVICE_TOKEN:
		msg = "超过30天未接入的设备id"
	case CALLBACK_STATUS_PUSH_RATE_LIMIT:
		msg = "Cache消息频控丢弃"
	case CALLBACK_STATUS_PUSH_TOTAL_LIMIT:
		msg = "消息单日发送总量限制"
	case CALLBACK_STATUS_DISABLE_PUSH:
		msg = "客户端调用了disablePush接口禁用Push"
	case CALLBACK_STATUS_UNINSTALL_APP:
		msg = "应用未安装"
	case CALLBACK_STATUS_PUSH_TOTAL_RATE_LIMIT:
		msg = "应用推送速率过快"
	case CALLBACK_STATUS_RATE_LIMIT:
		msg = "vivo发送超出频率限制"
	case CALLBACK_STATUS_RATE_TOTAL_LIMIT:
		msg = "vivo发送超出单日限制"
	case CALLBACK_STATUS_FORBID_TOTAL_LIMIT:
		msg = "vivo禁止发送"
	default:
		msg = "UNKONW"

	}
	return msg

}

type CallbackResponse struct {
	Data []*CallbackResponseItem `json:"data"`
}

type CallbackResponseItem struct {
	MsgId        int64  `json:"msg_id"`
	BiTag        string `json:"biTag"`
	Appid        string `json:"appid"`
	Token        string `json:"token"`
	Status       int64  `json:"status"`
	Timestamp    int64  `json:"timestamp"`
	RequestId    string `json:"requestId"`
	Description  string `json:"description"`
	DeviceVendor string `json:"device_vendor"`
	PackageName  string `json:"package"`
}

type CallbackParam struct {
	MsgId   int64  `json:"msgId"`
	Package string `json:"package"`
}
