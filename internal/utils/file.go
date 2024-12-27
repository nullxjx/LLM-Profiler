package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	log "github.com/sirupsen/logrus"
)

const (
	TimeFormat = "2006-01-02-15-04-05"
)

func IsDirEmpty(dir string) bool {
	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return true
	}

	// 读取目录中的文件信息列表
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Errorf("Error reading directory: %v\n", err)
		return true
	}

	// 判断目录是否为空
	if len(files) == 0 {
		return true
	}
	return false
}

// Save2Json 保存数据到json文件中，saveDir为保存文件的路径
func Save2Json(v any, saveDir string) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		log.Errorf("Error encoding JSON: %v", err)
		return
	}

	// 获取文件所在的目录
	dir := filepath.Dir(saveDir)

	// 创建目录（如果不存在）
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Errorf("failed to create directories: %v", err)
		return
	}

	file, err := os.Create(saveDir)
	if err != nil {
		log.Errorf("Error creating file: %v", err)
		return
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		log.Errorf("Error writing JSON data to file: %v", err)
		return
	}

	//log.Infof("JSON data saved to file: %v", saveDir)
}

func KeepFinalResult(saveDir string) {
	log.Debugf("🤖🤖🤖 clearing unused files...")
	// 读取目录中的所有文件
	files, err := ioutil.ReadDir(saveDir)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return
	}

	// 定义正则表达式来匹配文件名
	pattern := regexp.MustCompile(`^statistics_\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}\.json$`)

	// 筛选出符合命名规则的文件
	var matchedFiles []os.FileInfo
	for _, file := range files {
		if pattern.MatchString(file.Name()) {
			matchedFiles = append(matchedFiles, file)
		}
	}

	// 按时间戳排序文件
	sort.Slice(matchedFiles, func(i, j int) bool {
		return matchedFiles[i].ModTime().After(matchedFiles[j].ModTime())
	})

	// 删除除最新文件之外的其他文件
	for i := 1; i < len(matchedFiles); i++ {
		filePath := filepath.Join(saveDir, matchedFiles[i].Name())
		err := os.Remove(filePath)
		if err != nil {
			log.Infof("Error deleting file %s: %v", filePath, err)
		} else {
			log.Debugf("Deleted file %s", filePath)
		}
	}
}
