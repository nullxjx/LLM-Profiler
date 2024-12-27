package main

import (
	"os"
	"strconv"

	"github.com/nullxjx/llm_profiler/cmd"
	format "github.com/nullxjx/llm_profiler/pkg/log"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(&format.MyFormatter{})
	logLevel := format.DefaultLogLevel
	level, err := strconv.Atoi(os.Getenv(format.EnvLog))
	if err == nil {
		logLevel = level
	}
	log.SetLevel(log.Level(logLevel))
	cmd.Execute()
}
