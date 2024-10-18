package common

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/nullxjx/LLM-Profiler/config"
	log "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

const (
	CosSecretId  = "CosSecretId"
	CosSecretKey = "CosSecretKey"
)

// SaveFilesToCos 把 saveDir 目录中的文件保存到腾讯云cos
func SaveFilesToCos(cfg *config.Config) (string, string, error) {
	saveDir := cfg.SaveDir
	// 创建 COS 客户端
	u, _ := url.Parse(fmt.Sprintf("http://%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region))
	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv(CosSecretId),
			SecretKey: os.Getenv(CosSecretKey),
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	})

	downloadUrl := ""
	dstDir := fmt.Sprintf("%s/%s", cfg.SubFolder, cfg.SaveDir)
	if !cfg.Auto {
		now := time.Now().Format("2006-01-02-15-04-05")
		// 为了避免cos上面同名目录冲突，在这里加上一个时间作为子目录
		dstDir = fmt.Sprintf("%s/%s/%s", cfg.SubFolder, cfg.SaveDir, now)
	}

	// 遍历目录中的所有文件并上传
	err := filepath.Walk(saveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 如果是文件，则上传到 COS
		if !info.IsDir() {
			err = uploadFileToCOS(client, path, saveDir, dstDir)
			if err != nil {
				return err
			}
			if isStatisticsFile(info.Name()) {
				downloadUrl = generatePresignedURL(client, path, saveDir, dstDir)
			}
		}

		return nil
	})

	if err != nil {
		log.Errorf("Error uploading files: %v", err)
		return downloadUrl, dstDir, err
	} else {
		log.Infof("✅ All files uploaded successfully")
		return downloadUrl, dstDir, nil
	}
}

func uploadFileToCOS(client *cos.Client, filePath, srcDir, dstDir string) error {
	// 读取文件内容
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.New(fmt.Sprintf("Error reading file %s: %v", filePath, err))
	}

	// 设置 COS 中的对象键（文件路径）
	objectKey := filepath.Join(dstDir, filePath[len(srcDir):])

	// 上传文件到 COS
	_, err = client.Object.Put(context.Background(), objectKey, ioutil.NopCloser(bytes.NewReader(fileContent)), nil)
	if err != nil {
		return errors.New(fmt.Sprintf("Error uploading file %s: %v", filePath, err))
	}
	return nil
}

func generatePresignedURL(client *cos.Client, filePath, srcDir, dstDir string) string {
	// 设置 COS 中的对象键（文件路径）
	objectKey := filepath.Join(dstDir, filePath[len(srcDir):])

	// 生成预签名 URL
	secretID := os.Getenv(CosSecretId)
	secretKey := os.Getenv(CosSecretKey)
	presignedURL, err := client.Object.GetPresignedURL(context.Background(), http.MethodGet, objectKey, secretID, secretKey, 24*time.Hour, nil)
	if err != nil {
		log.Errorf("Error generating presigned URL for file %s: %v", filePath, err)
	} else {
		//log.Infof("Presigned URL for file %s: %s", filePath, presignedURL.String())
	}
	return presignedURL.String()
}

func isStatisticsFile(filename string) bool {
	pattern := regexp.MustCompile(`^statistics_\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}\.json$`)
	return pattern.MatchString(filename)
}

// DownloadCosFile 从cos上下载文件到本地
func DownloadCosFile(secretID, secretKey, region, bucket, cosPath, localPath string) error {
	u, _ := url.Parse(fmt.Sprintf("http://%s.cos.%s.myqcloud.com", bucket, region))
	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	})

	response, err := client.Object.Get(context.Background(), cosPath, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(localPath, content, 0644)
	if err != nil {
		return err
	}

	log.Infof("⬇⬇⬇ download cos file %s to local path: %s", cosPath, localPath)
	return nil
}
