package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

type MyFormatter struct{}

func (f *MyFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := time.Now().Format(time.RFC3339)

	// 根据日志级别设置颜色
	color := getColorByLevel(entry.Level)

	// 将颜色代码添加到时间戳和日志级别字段，日志级别字段后添加\x1b[0m以重置颜色
	msg := fmt.Sprintf("%s%s [%s]\x1b[0m %s\n", color, timestamp, entry.Level, entry.Message)

	return []byte(msg), nil
}

func getColorByLevel(level log.Level) string {
	switch level {
	case log.DebugLevel:
		return "\x1b[34m" // 蓝色
	case log.WarnLevel:
		return "\x1b[33m" // 黄色
	case log.ErrorLevel, log.FatalLevel, log.PanicLevel:
		return "\x1b[31m" // 红色
	default:
		return "\x1b[32m" // 绿色
	}
}

func SetLogFile(logFile string) error {
	// 获取文件所在的目录
	dir := filepath.Dir(logFile)

	// 创建目录（如果不存在）
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Errorf("failed to create directories: %v", err)
		return err
	}

	// 创建日志文件
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// 设置 logrus 输出到文件和控制台
	log.SetOutput(io.MultiWriter(os.Stdout, file))

	return nil
}
