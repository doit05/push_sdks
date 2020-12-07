package oppopush

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func doPost(url string, form url.Values) ([]byte, error) {
	var result []byte
	var req *http.Request
	var resp *http.Response
	var err error
	requestBodyString := form.Encode()
	req, err = http.NewRequest("POST", url, strings.NewReader(requestBodyString))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	tryTime := 0
tryAgain:
	resp, err = client.Do(req)
	if err != nil {
		tryTime++
		if tryTime < 3 {
			// reset Request.Body
			//http: ContentLength=222 with Body length 0
			req.Body = ioutil.NopCloser(strings.NewReader(requestBodyString))
			goto tryAgain
		}
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("network error")
	}
	result, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	str := string(result)
	str, err = strconv.Unquote(str)
	if err != nil {
		str = string(result)
	}
	return []byte(str), nil
}

func doGet(url string, params string) ([]byte, error) {
	var result []byte
	var req *http.Request
	var resp *http.Response
	var err error
	req, err = http.NewRequest("GET", url+params, nil)
	if err != nil {
		panic(err)
	}
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("network error")
	}
	result, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func doUpload(url, filePath, fileName string, params map[string]string) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	formFile, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}

	// 从文件读取数据，写入表单
	srcFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()
	_, err = io.Copy(formFile, srcFile)
	if err != nil {
		return nil, err
	}

	// 发送表单
	contentType := writer.FormDataContentType()
	writer.Close() // 发送之前必须调用Close()以写入结尾行
	res, err := http.Post(url, contentType, body)
	if err != nil {
		return nil, err
	}
	if res != nil {
		data, err := ioutil.ReadAll(res.Body)
		return data, err
	}
	return nil, fmt.Errorf("upload icon fail")
}
