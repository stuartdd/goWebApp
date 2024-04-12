package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"stuartdd.com/config"
	"stuartdd.com/logging"
	"stuartdd.com/server"
)

func main() {
	var addUserName string
	configFileName, _ := getArg("config=")
	s, _ := getArg("create")
	createFlag := s != ""
	s, pos := getArg("add")
	addFlag := s != ""
	if addFlag && pos == 0 {
		osExitWithMessage(1, "Add user. Name not found")
	}
	if addFlag {
		if createFlag {
			osExitWithMessage(1, "Cannot use Add and Create at the same time")
		}
		addUserName = os.Args[pos]
		osExitWithMessage(-1, fmt.Sprintf("Add User: %s", addUserName))
	} else {
		if createFlag {
			osExitWithMessage(-1, "Will Create USER locations that are missing")
		}
	}
	cfg, errorList := config.NewConfigData(configFileName, createFlag)
	if errorList.ErrorCount() > 0 {
		os.Stdout.WriteString(errorList.String())
		osExitWithMessage(1, "Config Errors. Cannot continue")
	}
	if cfg == nil {
		osExitWithMessage(1, "Config not loaded. Cannot continue")
	}

	if addFlag {
		osExitWithMessage(0, "Adding user")
	}

	if createFlag {
		osExitWithMessage(0, "Create locations complete")
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

func getArg(name string) (string, int) {
	nl := strings.ToLower(name)
	for i := 1; i < len(os.Args); i++ {
		al := strings.ToLower(os.Args[i])
		if al == nl {
			if i < (len(os.Args) - 1) {
				return nl, i + 1
			}
			return nl, 0
		}
		if strings.HasPrefix(al, nl) {
			if i < (len(os.Args) - 1) {
				return al[len(nl):], i + 1
			}
			return al[len(nl):], 0
		}
	}
	return "", 0
}

func osExitWithMessage(rc int, message string) {
	os.Stdout.WriteString(message)
	os.Stdout.WriteString("\n")
	if rc >= 0 {
		os.Exit(rc)
	}
}
