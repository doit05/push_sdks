package meizupush

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/ddliu/go-httpclient"
)

const (
	SERVER      = "https://api-push.meizu.com/garcia/api/client/"
	STATUS_CODE = 200
	//服务端SDK调用API的应用的私钥Secret Key为 appSecret
	PUSH_API_SERVER = "https://api-push.meizu.com"
)

type PushResponse struct {
	Code     int
	RetCode  string      `json:"code"`
	Message  string      `json:"message"`
	Value    interface{} `json:"value"`
	Redirect string      `json:"redirect"`
	MsgId    string      `json:"msgId"`
}

// md5 sign
func GenerateSign(params map[string]string, appKey string) string {
	var signStr string
	if params != nil {
		keys := make([]string, len(params))
		i := 0
		for key, _ := range params {
			keys[i] = key
			i++
		}
		sort.Strings(keys)
		for _, k := range keys {
			signStr += k + "=" + params[k]
		}
		signStr += appKey
		fmt.Println("signStr ", signStr)
	}
	return PushParamMD5(signStr)
}

func PushParamMD5(paramstr string) string {
	hasher := md5.New()
	hasher.Write([]byte(paramstr))
	return hex.EncodeToString(hasher.Sum(nil))
}

//resolve push response
func ResolvePushResponse(res *httpclient.Response, err error) PushResponse {
	var response PushResponse
	if err != nil {
		response = PushResponse{
			Code:    0,
			Message: err.Error(),
		}
	} else {
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		response = PushResponse{
		}
		err = json.Unmarshal(buf.Bytes(), &response)
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
