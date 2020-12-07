package common

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var suportImgExt = map[string]bool{
	"PNG":  true,
	"JPG":  true,
	"JPEG": true,
}

func DowlodPic(url string, path, prefix, fileName string) (string, error) {
	if fileName == "" {
		return "", fmt.Errorf("invalid img url:[%s]", url)
	}
	fileName = path + "/" + prefix + fileName
	if Exists(fileName) {
		return fileName, nil
	}
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return "", fmt.Errorf("download url error res:[%v]", string(data))
	}

	err = ioutil.WriteFile(fileName, data, 0666)
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func GetImgNameFromUrl(url string) string {
	if url == "" {
		return ""
	}
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		arr := strings.Split(url, "/")
		fileName := arr[len(arr)-1]
		ext := filepath.Ext(fileName)
		if len(ext) < 4 { //".png"
			return ""
		}
		if !suportImgExt[strings.ToUpper(ext[1:])] {
			return ""
		}
		return fileName
	}
	return ""
}

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
