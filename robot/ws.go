package robot

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"time"
	"tx/conf"
	"tx/logs"
)

type Payload struct {
	Op int         `json:"op"`
	D  interface{} `json:"d"`
	S  int         `json:"s,omitempty"`
	T  string      `json:"t,omitempty"`
}

type HeartbeatManager struct {
	ticker *time.Ticker
	stop   chan bool
}

type Wss struct {
	lastSequenceNumber int
	sessionId          string
	wsBaseUrl          string
	c                  *websocket.Conn
	heartbeatManager   *HeartbeatManager
}

var Ws = &Wss{}

func (ws *Wss) InitWs() {
	// 创建一个新的 GET 请求
	req, err := http.NewRequest("GET", conf.GetAppConf().QqBaseUrl+"/gateway", nil)
	if err != nil {
		logs.Logger.Errorf("Error creating request: %v", err)
		return
	}

	// 添加请求头
	req.Header.Add("Authorization", "QQBot "+GetAppToken().AccessToken)
	req.Header.Add("X-Union-Appid", conf.GetAppConf().AppId)

	// 创建一个 HTTP 客户端
	client := &http.Client{}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		logs.Logger.Errorf("Error sending request: %v", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应数据
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logs.Logger.Errorf("Error reading response body: %v", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		logs.Logger.Errorf("Error response: %v", string(body))
		return
	}
	url := struct {
		Url string `json:"url"`
	}{}
	// 解析 JSON 响应体
	err = json.Unmarshal(body, &url)
	if err != nil {
		logs.Logger.Errorf("Error parsing JSON response: %v", err)
		return
	}
	ws.wsBaseUrl = url.Url
	logs.Logger.Infof("Get wss success: %v", url.Url)
	ws.StartSvc()
}

func (ws *Wss) Identify(token string) {
	payload := Payload{
		Op: 2,
		D: map[string]interface{}{
			"token":   "QQBot " + token,
			"intents": 1 << 30,
			"shard":   []int{0, 1},
			"properties": map[string]string{
				"$os":      "linux",
				"$browser": "my_library",
				"$device":  "my_library",
			},
		},
	}
	err := ws.c.WriteJSON(payload)
	if err != nil {
		logs.Logger.Errorf("Error sending IDENTIFY payload: %v", err)
	}
	logs.Logger.Printf("Sent IDENTIFY payload: %+v", payload)
}

func (ws *Wss) NewHeartbeatManager(interval time.Duration) *HeartbeatManager {
	return &HeartbeatManager{
		ticker: time.NewTicker(interval - 5*time.Second),
		stop:   make(chan bool),
	}
}

func (ws *Wss) StartHeartbeat() {
	go func() {
		for {
			select {
			case <-ws.heartbeatManager.ticker.C:
				var heartbeatData interface{}
				if ws.lastSequenceNumber == 0 {
					heartbeatData = nil
				} else {
					heartbeatData = ws.lastSequenceNumber
				}

				ws.c.WriteJSON(Payload{Op: 1, D: heartbeatData})
				logs.Logger.Printf("Sent HEARTBEAT payload: %+v", heartbeatData)
			case <-ws.heartbeatManager.stop:
				return
			}
		}
	}()
}

func (ws *Wss) StopHeartbeat() {
	ws.heartbeatManager.stop <- true
	ws.heartbeatManager.ticker.Stop()
}

func (ws *Wss) Resume(token string) error {
	err := ws.StartWss()
	if err != nil {
		return err
	}
	payload := Payload{
		Op: 6,
		D: map[string]interface{}{
			"token":      "QQBot " + token,
			"session_id": ws.sessionId,
			"seq":        ws.lastSequenceNumber,
		},
	}
	err = ws.c.WriteJSON(payload)
	if err != nil {
		logs.Logger.Errorf("Error sending RESUME payload: %v", err)
		return err
	}
	logs.Logger.Printf("Sent RESUME payload: %+v", payload)
	return nil
}

func (ws *Wss) StartWss() error {
	logs.Logger.Printf("Connecting to %s", ws.wsBaseUrl)
	c, _, err := websocket.DefaultDialer.Dial(ws.wsBaseUrl, nil)
	if err != nil {
		logs.Logger.Errorf("dial error: %v", err)
		return err
	}
	ws.c = c
	return nil
}

func (ws *Wss) CloseWss() {
	ws.c.Close()
}

func (ws *Wss) ReadMsg() (messageType int, p []byte, err error) {
	return ws.c.ReadMessage()

}

func (ws *Wss) StartSvc() {
	err := ws.StartWss()
	if err != nil {
		log.Fatal("dial:", err)
	}
	// 接收 Hello 消息并发送鉴权
	_, message, err := ws.ReadMsg()
	if err != nil {
		log.Fatal("read:", err)
	}
	var helloPayload Payload
	json.Unmarshal(message, &helloPayload)
	ws.Identify(GetAppToken().AccessToken)
	heartbeatInterval := time.Duration(helloPayload.D.(map[string]interface{})["heartbeat_interval"].(float64)) * time.Millisecond
	ws.heartbeatManager = ws.NewHeartbeatManager(heartbeatInterval)
	ws.StartHeartbeat()
	// 处理消息
	for {
		_, message, err := ws.ReadMsg()
		if err != nil {
			logs.Logger.Errorf("read error: %v", err)
			ws.CloseWss()
			err = ws.Resume(GetAppToken().AccessToken)
			if err != nil {
				logs.Logger.Println("Failed to resume connection, exiting")
				break
			}
			logs.Logger.Infof("Resumed connection")
			continue
		}
		var payload Payload
		json.Unmarshal(message, &payload)
		switch payload.Op {
		case 0:
			switch payload.T {
			case "READY":
				logs.Logger.Printf("Received READY message: %+v", payload)
				dataMap, ok := payload.D.(map[string]interface{}) // 类型断言，确保payload.D是map[string]interface{}类型
				if !ok {
					logs.Logger.Printf("Error asserting type for payload data")
					break
				}
				ws.sessionId, ok = dataMap["session_id"].(string) // 再次使用类型断言提取session_id
				if !ok {
					logs.Logger.Printf("Error asserting type for session_id")
					break
				}
				logs.Logger.Printf("Session ID: %s", Ws.sessionId) // 打印session_id
			case "AT_MESSAGE_CREATE":
				logs.Logger.Printf("Received AT_MESSAGE_CREATE message: %+v", payload)
				dataMap, ok := payload.D.(map[string]interface{})
				if !ok {
					logs.Logger.Printf("Error asserting type for payload data")
					break
				}
				Replay(dataMap)
			}
			ws.lastSequenceNumber = payload.S // 更新序列
		case 11:
			logs.Logger.Printf("Received HEARTBEAT_ACK message: %+v", payload)
		}
	}
	ws.StopHeartbeat()
	ws.c.Close()
	logs.Logger.Println("Connection closed")
}
