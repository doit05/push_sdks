package config

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"strings"
)

type PushServerCfg struct {
	Name                    string `yaml:"name"`
	Redirect                string `yaml:"redirect"`
	AppId                   string `yaml:"appid"`
	AppSecret               string `yaml:"appsecret"`
	AppKey                  string `yaml:"appkey"`
	Package                 string `yaml:"package"`
	NeedAccessToken         bool   `yaml:"need_access_token"`
	AuthUrl                 string `yaml:"auth_url"`
	PushUrl                 string `yaml:"push_url"` // "https://api.push.hicloud.com"
	ExtraConfigFile         string `yaml:"extra_config_file"`
	ExtraConfigFilePassword string `yaml:"extra_config_file_password"`
	TestMod                 bool   `yaml:"test_mod"`
}

func (p *PushServerCfg) GetPushServerKey() string {
	if p.Package == "" {
		return strings.ToLower(p.Name)
	}
	return strings.ToLower(p.Name + "_" + p.Package)
}

func (info *PushServerCfg) GetMd5() (string, error) {
	var md5Map = map[string]interface{}{
		"Package":            info.Package,
		"DeviceVendor":       info.Name,
		"AppId":              info.AppId,
		"AppKey":             info.AppKey,
		"AppSecret":          info.AppSecret,
		"ConfigFileName":     info.ExtraConfigFile,
		"ConfigFilePassword": info.ExtraConfigFilePassword,
		"TestMod":            info.TestMod,
	}
	md5Dta, err := json.Marshal(md5Map)
	if err != nil {
		return "", err
	}
	h := md5.New()
	_, err = h.Write(md5Dta)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
