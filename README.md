# push_sdks

支持小米、华为、vivo、oppo、魅族、苹果、fcm推送

推送配置：
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
