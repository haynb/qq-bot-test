package logs

import (
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
)

var Logger = logrus.New()

func init() {
	// 实例化一个 logrus 的 Logger
	Logger = logrus.New()

	// 设置 Formatter 为 TextFormatter，可以选择 JSONFormatter
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,                  // 完整时间戳
		TimestampFormat: "2006-01-02 15:04:05", // 时间戳格式
	})

	// 设置日志级别
	Logger.SetLevel(logrus.DebugLevel)

	// 启用报告调用者信息
	Logger.SetReportCaller(true)

	// 确保 logs 目录存在
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		Logger.Fatal("创建 logs 目录失败: ", err)
	}

	// 创建日志文件
	file, err := os.OpenFile(filepath.Join("logs", "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		Logger.Fatal("创建日志文件失败: ", err)
	}

	// 设置输出为标准输出和文件
	Logger.SetOutput(io.MultiWriter(os.Stdout, file))
}
