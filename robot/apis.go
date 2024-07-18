package robot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"tx/ai"
	"tx/conf"
	"tx/logs"
)

type Message struct {
	Content   string `json:"content"`
	MessageID string `json:"msg_id"`
}

func CommonRequest(path string, jsonData []byte) ([]byte, error) {
	// 创建请求URL
	url := fmt.Sprintf(conf.GetAppConf().QqBaseUrl+"%s", path)
	logs.Logger.Infof("Request URL: %s", url)
	// 创建请求体
	logs.Logger.Infof("CommonRequest body: %s", string(jsonData))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	// 添加请求头
	req.Header.Add("Authorization", "QQBot "+GetAppToken().AccessToken)
	req.Header.Add("X-Union-Appid", conf.GetAppConf().AppId)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应数据
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logs.Logger.Errorf("Error reading response body: %v", err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error response: %v", string(body))
	}
	return body, nil
}

func removeTags(input string) string {
	// 创建正则表达式，匹配<>及其内部的内容
	re := regexp.MustCompile(`<.*?>`)
	// 使用正则表达式替换掉匹配的部分
	result := re.ReplaceAllString(input, "")
	return result
}

func Replay(data map[string]interface{}) {
	// 访问嵌套的Map来获取特定字段
	content := data["content"]
	logs.Logger.Infof("Received message: %s", content)
	// 使用ai
	replayContent := "未响应,请重试~"
	result, err := ai.GptClient.Ask(removeTags(content.(string)))
	if err != nil || len(result.Choices) == 0 {
		logs.Logger.Errorf("Error sending request: %v", err)
	} else {
		replayContent = result.Choices[0].Message.Content + "\n\n\n 使用tokens： " + strconv.Itoa(result.Usage.TotalTokens)
	}
	channelID := data["channel_id"]
	messageID := data["id"]
	// 构建消息回复的JSON内容
	reply := map[string]interface{}{
		"content": replayContent,
		"msg_id":  messageID,
	}
	replyJSON, _ := json.Marshal(reply)
	body, err := CommonRequest(fmt.Sprintf("/channels/%s/messages", channelID), replyJSON)
	if err != nil {
		logs.Logger.Errorf("Error sending response: %v", err)
		return
	}
	logs.Logger.Infof("Response: %s", string(body))
}
