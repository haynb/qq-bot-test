package robot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	"tx/conf"
	"tx/logs"
)

type Token struct {
	AccessToken string `json:"access_token"`
	ExpireTime  string `json:"expires_in"`
}

var appToken = &Token{}

// InitToken 初始化 token
func InitToken() {
	requestData := map[string]string{
		"appId":        conf.GetAppConf().AppId,
		"clientSecret": conf.GetAppConf().ClientSecret,
	}
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		logs.Logger.Errorf("Error marshalling request data: %v", err)
		return
	}

	// 发送 POST 请求
	resp, err := http.Post("https://bots.qq.com/app/getAppAccessToken", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logs.Logger.Errorf("Error sending request: %v", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应数据
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		logs.Logger.Errorf("Error response: %v", string(body))
		return
	}
	if err := json.Unmarshal(body, &appToken); err != nil {
		fmt.Println("Error unmarshalling success response:", err)
		return
	}
	logs.Logger.Infof("Get token success: %v . ExpireTime: %v", appToken, appToken.ExpireTime)
	// 设置定时任务，提前一分钟刷新 token
	expireSeconds, err := strconv.Atoi(appToken.ExpireTime)
	if err != nil {
		fmt.Println("Error parsing ExpireTime:", err)
		return
	}
	// 将秒数转换为 time.Duration
	duration := time.Duration(expireSeconds) * time.Second
	fmt.Println(duration)
	time.AfterFunc(duration-30*time.Second, RefreshToken)
}

func GetAppToken() *Token {
	return appToken
}

func RefreshToken() {
	logs.Logger.Info("Refresh token")
	InitToken()
}
