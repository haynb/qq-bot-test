package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"tx/ai"
	"tx/conf"
	"tx/robot"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "./conf/app.yaml", "service configure file")
	flag.Parse()
	if err := conf.LoadFromFile(configFile); err != nil {
		panic(err)
	}
	robot.InitToken()
	ai.InitGPT()
	robot.Ws.InitWs()
	// 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	// 清理资源
	log.Println("Shutting down service...")
	robot.Ws.CloseWss()
}
