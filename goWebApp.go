package main

import (
	"log"
	"os"
	"time"

	"stuartdd.com/config"
	"stuartdd.com/server"
	"stuartdd.com/tools"
)

func main() {

	configFileName := ""
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}

	cfg, errorList := config.NewConfigData(configFileName)
	if errorList.Len() > 0 {
		os.Stdout.WriteString(errorList.ToString())
		os.Exit(1)
	}
	if cfg == nil {
		os.Exit(1)
	}

	actionQueue := make(chan server.ActionId, 10)
	defer close(actionQueue)

	ld := cfg.GetLogData()

	logger, err := tools.NewLogger(ld.Path, ld.FileNameMask, ld.MonitorSeconds, ld.LogLevel)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	go func() {
		for {
			a := <-actionQueue
			switch a {
			case server.Exit:
				logger.Log("Server: Exit after 1 second\n")
				time.Sleep(500 * time.Millisecond)
				logger.Close()
				time.Sleep(500 * time.Millisecond)
				os.Exit(1)
			case server.Ignore:
				logger.Log("Server: Ignore\n")
			}
		}
	}()

	webAppServer := server.NewWebAppServer(cfg, actionQueue, logger)
	webAppServer.Start()

}
