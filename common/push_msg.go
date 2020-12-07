package common

type Msg struct {
	Priority    uint32
	ExpireTime  int64
	MsgType     int32
	MsgTitle    string
	SubMsgTile  string
	MsgBody     string
	PushData    string
	MsgAction   string
	Id          int64
	ImgUrl      string
	TypeId      string
	PackageName string
	ChannelID   string
}
