# push_sdks

支持小米、华为、vivo、oppo、魅族、苹果、fcm推送

推送配置：
```
type PushServerCfg struct {
	Name                    string `yaml:"name"` //手机厂商名称 xiaomi huawei vivo oppo meizu ios google
	Redirect                string `yaml:"redirect"` //回调地址 
	AppId                   string `yaml:"appid"` //手机厂商appid
	AppSecret               string `yaml:"appsecret"` //手机厂商appsecret
	AppKey                  string `yaml:"appkey"`  //手机厂商appkey
	Package                 string `yaml:"package"`  //应用包名
	NeedAccessToken         bool   `yaml:"need_access_token"` //是否需要access token
	AuthUrl                 string `yaml:"auth_url"`  //https://login.cloud.huawei.com/oauth2/v2/token
	PushUrl                 string `yaml:"push_url"` // "https://api.push.hicloud.com"
	ExtraConfigFile         string `yaml:"extra_config_file" //ios (xxx.p12)和google(xxx.json)的配置文件
	ExtraConfigFilePassword string `yaml:"extra_config_file_password"` //ios和google 配置文件密码
	TestMod                 bool   `yaml:"test_mod"` //是否是测试配置
}
```
