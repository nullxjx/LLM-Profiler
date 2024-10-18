package main

import (
	"github.com/nullxjx/LLM-Profiler/cmd"
	"github.com/nullxjx/LLM-Profiler/common"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(&common.MyFormatter{})
	log.SetLevel(log.DebugLevel)
	cmd.Execute()
}
