package main

import (
	"fmt"
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

	cfg, err := config.NewConfigData(configFileName)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	logger, err := tools.NewLogger(cfg.DefaultLogFileName)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	actionQueue := make(chan server.ActionId, 10)
	defer close(actionQueue)

	go func() {
		for {
			a := <-actionQueue
			switch a {
			case server.Exit:
				logger.Log("Server: Exit after 1 second\n")
				time.Sleep(1 * time.Second)
				os.Exit(1)
			case server.Ignore:
				logger.Log("Server: Ignore\n")
			}
		}
	}()

	webAppServer := server.NewWebAppServer(cfg, actionQueue, logger)
	fmt.Println(webAppServer.ToString())
	webAppServer.Start()

}
