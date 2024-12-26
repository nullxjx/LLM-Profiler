package http

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nullxjx/llm_profiler/internal/infer/type/stream"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Post(ctx context.Context, url string, rawBody interface{}) ([]byte, error) {
	body, jsonErr := json.Marshal(rawBody)
	if jsonErr != nil {
		return nil, jsonErr
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("infer error, status code: %d, body: %v", resp.StatusCode, string(respBody)))
	}

	return respBody, nil
}

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
				out <- []byte(fmt.Sprintf(string(stream.ErrorEvent), err.Error()))
				return
			}
			if err == context.Canceled {
				log.Infof("HttpClient Stream canceled: %v", err)
				out <- []byte(fmt.Sprintf(string(stream.ErrorEvent), err.Error()))
				return
			}
			if err != nil {
				log.Errorf("HttpClient error reading response data: %v", err)
				out <- []byte(fmt.Sprintf(string(stream.ErrorEvent), err.Error()))
				return
			}
			out <- line
		}
	}()
	return out, nil
}
