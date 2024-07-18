package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"tx/conf"
	"tx/logs"
)

type OpenaiGpt struct {
	BaseUrl    string
	ApiKey     string
	Model      string
	Message    []map[string]string
	MaxHistory int // 增加字段以存储最大历史记录限制
}

type Response struct {
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

var GptClient *OpenaiGpt

func InitGPT() {
	// 初始化Gpt
	GptClient = &OpenaiGpt{
		BaseUrl: conf.GetAppConf().OpenaiBaseUrl,
		ApiKey:  conf.GetAppConf().OpenaiKey,
		Model:   conf.GetAppConf().OpenaiDefaultModel,
		Message: []map[string]string{{"role": "system", "content": "你的名字叫做无敌小可爱，你是一个聪明可爱的小助手，致力于解答" +
			"别人的问题，尽量使用俏皮可爱的语气回答问题，尽可能使用表情包或者颜文字来回答问题，让别人感觉到你是一个有趣的ai助手。"}},
		MaxHistory: conf.GetAppConf().OpenaiMaxHistory,
	}
}

func (gpt *OpenaiGpt) AddHistory(ai string) {
	// 添加AI回复
	gpt.Message = append(gpt.Message, map[string]string{"role": "assistant", "content": ai})
	// 检查历史记录是否超过限制
	if len(gpt.Message) > gpt.MaxHistory*2 {
		// 删除索引1和2的内容，但保留索引0
		gpt.Message = append(gpt.Message[:1], gpt.Message[3:]...)
	}
}

func (gpt *OpenaiGpt) Ask(query string) (Response, error) {
	var response Response
	// 添加用户输入
	gpt.Message = append(gpt.Message, map[string]string{"role": "user", "content": query})
	// 构造请求体
	payload := map[string]interface{}{
		"model":       gpt.Model,
		"messages":    gpt.Message,
		"temperature": 1,
		"stream":      false,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return response, err
	}
	// 创建请求
	req, err := http.NewRequest("POST", gpt.BaseUrl+"/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return response, err
	}
	// 添加必要的头部信息
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gpt.ApiKey)
	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}
	if resp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("server returned non-200 status: %d", resp.StatusCode)
	}
	logs.Logger.Infof("Response: %s", string(body))
	// 解析响应
	err = json.Unmarshal(body, &response)
	if err != nil {
		return response, err
	}
	// 检查是否有有效的 choices
	if len(response.Choices) > 0 {
		logs.Logger.Infof("Response: %s", response.Choices[0].Message.Content)
		logs.Logger.Infof("Total tokens: %d", response.Usage.TotalTokens)
	} else {
		logs.Logger.Errorf("No valid choices in response")
	}
	gpt.AddHistory(response.Choices[0].Message.Content)
	return response, nil
}
