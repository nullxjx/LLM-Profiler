package main

import (
	"github.com/nullxjx/llm_profiler/cmd"
	logformat "github.com/nullxjx/llm_profiler/pkg/log"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(&logformat.MyFormatter{})
	log.SetLevel(log.DebugLevel)
	cmd.Execute()
}
