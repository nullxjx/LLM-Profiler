package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

// Stream 发起流式请求，需调用unwrapStreamError来获取流式过程中的报错信息
func Stream(ctx context.Context, url string, header, queryParam map[string]string, body interface{}) (
	<-chan []byte, error) {
	client := resty.New()
	client.SetDoNotParseResponse(true) // 不解析响应
	for k, v := range header {
		client.Header.Add(k, v)
	}
	request := client.R()
	if queryParam != nil {
		request = request.SetQueryParams(queryParam)
	}
	body, jsonErr := json.Marshal(body)
	if jsonErr != nil {
		return nil, jsonErr
	}
	request.Body = body
	resp, jsonErr := request.SetContext(ctx).Post(url)
	if jsonErr != nil {
		log.Errorf("HttpClient Stream error: %v", jsonErr)
		return nil, jsonErr
	}
	out := make(chan []byte, 4096)
	reader := bufio.NewReader(resp.RawBody())
	go func() {
		defer close(out)
		defer resp.RawBody().Close()
		for {
			line, err := reader.ReadBytes('\n')
			if err == io.EOF {
				//log.Infof("HttpClient reach the end of response data: %v", err)
				if string(line) != "" {
					log.Warnf("Received incomplete line: %v", string(line))
					out <- line
				}
				out <- []byte(fmt.Sprintf(string(ErrorEvent), err.Error()))
				return
			}
			if err == context.Canceled {
				log.Infof("HttpClient Stream canceled: %v", err)
				out <- []byte(fmt.Sprintf(string(ErrorEvent), err.Error()))
				return
			}
			if err != nil {
				log.Errorf("HttpClient error reading response data: %v", err)
				out <- []byte(fmt.Sprintf(string(ErrorEvent), err.Error()))
				return
			}
			out <- line
		}
	}()
	return out, nil
}

// UnwrapStreamError 从流式请求中解析出原始的错误信息，并返回给调用方
func UnwrapStreamError(data []byte) error {
	dataStr := string(data)
	re := regexp.MustCompile(fmt.Sprintf(string(ErrorEvent), "(.*)"))
	matches := re.FindStringSubmatch(dataStr)
	if len(matches) < 1 {
		return nil
	}
	errMsg := matches[1]
	switch errMsg {
	case io.EOF.Error():
		return io.EOF
	case context.Canceled.Error():
		return context.Canceled
	default:
		return fmt.Errorf(errMsg)
	}
}
