package push_sdks

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"push_sdks/applepush"
	"push_sdks/common"
	"push_sdks/config"
	"push_sdks/googlepush"
	"push_sdks/huawei"
	"push_sdks/meizupush"
	"push_sdks/oppopush"
	"push_sdks/vivopush"
	"push_sdks/xiaomipush"

	log "github.com/sirupsen/logrus"
)

type SdkApi interface {
	NeedAccessToken() bool
	Name() string
	GetToken() (token string, expire_in int, err error)
	PushMsg(msg *common.Msg, tokens []string) (failsInfoMap map[string]*common.CallbackResponseItem, err error)
	PushReciver(r *http.Request) (*common.CallbackResponse, map[string]interface{}, error)
}

var pushServers = sync.Map{}
var defaultName string
var pushConfigServers map[string]*config.PushServerCfg

const (
	MAX_MSG_TITLE_LENGTH = 40
	MAX_BATCH_MSG_NUM    = 800
)


func InitPushServers(cfg []config.PushServerCfg) (err error) {
	pushConfigServers = make(map[string]*config.PushServerCfg, len(cfg))
	for _, item := range cfg {
		val := item
		pushConfigServers[val.GetPushServerKey()] = &val
	}

	for _, val := range pushConfigServers {
		err = initSigleServerConfig(val)
		if err != nil {
			return err
		}
	}
	defaultName = cfg[0].GetPushServerKey()
	log.Debugf("pushServers :%v", pushServers)
	return nil
}

func initSigleServerConfig(serverCofig *config.PushServerCfg) error {
	md5Key := "md5_" + serverCofig.GetPushServerKey()
	md5Data, err := serverCofig.GetMd5()
	if err != nil {
		return err
	}

	if md5Str, ok := pushServers.Load(md5Key); ok {
		if md5Str == md5Data {
			return nil
		}
	}

	item := serverCofig
	var sdk SdkApi
	switch item.Name {
	case "huawei":
		sdk, err = huawei.NewClient(*item)
		if err != nil {
			return err
		}
	case "xiaomi":
		sdk, err = xiaomipush.NewClient(*item)
		if err != nil {
			return err
		}
	case "oppo":
		sdk, err = oppopush.NewClient(*item)
		if err != nil {
			return err
		}
	case "vivo":
		sdk, err = vivopush.NewClient(*item)
		if err != nil {
			return err
		}
	case "ios":
		sdk, err = applepush.NewClient(*item)
		if err != nil {
			return err
		}

	case "google":
		sdk, err = googlepush.NewClient(*item)
		if err != nil {
			return err
		}
	case "meizu":
		sdk, err = meizupush.NewClient(*item)
		if err != nil {
			return err
		}
	default:
		err := fmt.Errorf("invalid push servers name:[%s]", item.Name)
		log.Error(err)
		return err
	}
	serverKey := serverCofig.GetPushServerKey()
	_, exist := pushServers.LoadOrStore(serverKey, sdk)
	if exist {
		pushServers.Delete(serverKey)
		pushServers.Store(serverKey, sdk)
		pushServers.Delete(md5Key)
	}
	pushServers.Store(md5Key, md5Data)
	log.WithField("server", item).WithField(md5Key, md5Data).Debugf("push server config init")
	return nil
}

func GetDefultName() string {
	return defaultName
}

func GetPushSdkByName(packageName, name string) SdkApi {
	if packageName == "" {
		packageName = defaultName
	}

	serverKey := name + "_" + packageName
	serverKey = strings.ToLower(serverKey)
	return GetPushSdkByKey(serverKey)
}

func GetPushSdkByKey(name string) SdkApi {
	name = strings.ToLower(name)
	sdk, ok := pushServers.Load(name)
	if !ok || sdk == nil {
		sdk, _ = pushServers.Load(GetDefultName())
	}
	return sdk.(SdkApi)
}

func GetPushServers() sync.Map {
	return pushServers
}

func GetPushServerByName(name string) SdkApi {
	serverKey := defaultName
	for _, pushConfig := range pushConfigServers {
		if pushConfig.Name == name {
			serverKey = pushConfig.GetPushServerKey()
		}
	}
	ret, ok := pushServers.Load(serverKey)
	if ok {
		return ret.(SdkApi)
	}
	return nil
}

func PushBatchMsg(ctx context.Context, msg *common.Msg, name string, tokens []string) (map[string]*common.CallbackResponseItem, error) {
	log.WithFields(log.Fields{
		"name":         name,
		"deviceTokens": tokens,
	}).Debug("begin to push msg")
	var err error
	sdk := GetPushSdkByName(msg.PackageName, name)
	resultList, err := sdk.PushMsg(formatMsg(msg), tokens)
	if err != nil {
		log.WithError(err).Errorf("push msg error client name :[%s]", name)
		return resultList, err
	}
	log.WithFields(log.Fields{
		"name":         name,
		"deviceTokens": tokens,
		"err":          err,
		"resultList":   resultList,
	}).Debug("end to push msg")
	return resultList, err
}

func PushMsg(ctx context.Context, msg *common.Msg, name, token string) (*common.CallbackResponseItem, error) {
	log.WithFields(log.Fields{
		"name":  name,
		"token": token,
		"msg":   msg,
	}).Debug("begin to push msg")
	sdk := GetPushSdkByName(msg.PackageName, name)
	resultList, err := sdk.PushMsg(formatMsg(msg), []string{token})
	log.WithFields(log.Fields{
		"name":   name,
		"token":  token,
		"msg":    msg,
		"err":    err,
		"result": resultList,
	}).Debug("end to push msg")
	if len(resultList) > 0 {
		return resultList[token], err
	}
	return nil, err
}

func formatMsg(msg *common.Msg) *common.Msg {
	if len(msg.MsgTitle) > MAX_MSG_TITLE_LENGTH {
		msg.MsgTitle = msg.MsgTitle[:MAX_MSG_TITLE_LENGTH]
	}
	return msg
}
