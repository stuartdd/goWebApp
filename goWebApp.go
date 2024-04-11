package main

import (
	"log"
	"os"
	"strings"
	"time"

	"stuartdd.com/config"
	"stuartdd.com/logging"
	"stuartdd.com/server"
)

func main() {

	configFileName := getArg("config=")
	createFlag := getArg("create") != ""
	if createFlag {
		os.Stdout.WriteString("Will Create locations that are missing:\n")
	}

	cfg, errorList := config.NewConfigData(configFileName, createFlag)
	if errorList.ErrorCount() > 0 {
		os.Stdout.WriteString(errorList.String())
		os.Exit(1)
	}
	if cfg == nil {
		os.Exit(1)
	}

	if createFlag {
		os.Stdout.WriteString("Create locations complete:\n")
		os.Exit(0)
	}

	actionQueue := make(chan server.ActionId, 10)
	defer close(actionQueue)

	ld := cfg.GetLogData()

	logger, err := logging.NewLogger(ld.Path, ld.FileNameMask, ld.MonitorSeconds, ld.LogLevel)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	logger.Log("Application Start :-----------------------------------------------------------------")
	for _, l := range errorList.Logs() {
		logger.Log(l)
	}
	go func() {
		for {
			a := <-actionQueue
			switch a {
			case server.Exit:
				logger.Log("Server Terminated :-----------------------------------------------------------------")
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

func getArg(name string) string {
	nl := strings.ToLower(name)
	for i := 1; i < len(os.Args); i++ {
		al := strings.ToLower(os.Args[i])
		if al == nl {
			return nl
		}
		if strings.HasPrefix(al, nl) {
			return al[len(nl):]
		}
	}
	return ""
}
