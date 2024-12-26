package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/nullxjx/llm_profiler/config"

	log "github.com/sirupsen/logrus"
)

type Message struct {
	MsgType  string `json:"msgtype"`
	Markdown Text   `json:"markdown"`
}

type Text struct {
	Content string `json:"content"`
}

func SendMsg(cfg *config.Config, downloadUrl, dstDir string) {
	// 向企微推送消息
	msg := fmt.Sprintf("## 🥳🤩🥰 Performance Test Done \nDownload statistics result via 👉 [me](%s) 👈 \n"+
		//"\nThis presigned URL is available in **6 hours**. After time expired, please find result in cos:\n"+
		"> bucket: <font color=\"info\">%s</font>\n"+
		"> path: <font color=\"info\">%s</font>\n\n", downloadUrl, os.Getenv(config.Bucket), dstDir)
	if cfg.User != "" {
		msg += fmt.Sprintf("<@%s>\n", cfg.User)
	}
	SendWebHook(os.Getenv(config.WebhookUrl), msg)
}

// SendWebHook 通过webhook向企微推送消息
func SendWebHook(webhookURL, content string) {
	msg := Message{
		MsgType: "markdown",
		Markdown: Text{
			Content: content,
		},
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("Error marshalling message: %v", err)
		return
	}

	// 创建一个自定义的HTTP客户端，禁用TLS证书验证
	customClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := customClient.Post(webhookURL, "application/json", bytes.NewBuffer(msgBytes))
	if err != nil {
		log.Errorf("Error sending message: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading response body: %v", err)
		return
	}

	log.Infof("企业微信机器人响应: %v", string(body))
}