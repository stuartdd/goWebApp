package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"stuartdd.com/config"
	"stuartdd.com/server"
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

	actionQueue := make(chan server.ActionId, 10)
	defer close(actionQueue)

	go func() {
		for {
			a := <-actionQueue
			switch a {
			case server.Exit:
				fmt.Printf("Server: Exit after 1 second\n")
				time.Sleep(1 * time.Second)
				os.Exit(1)
			case server.Ignore:
				fmt.Printf("Server: Ignore\n")
			}
		}
	}()

	webAppServer := server.NewWebAppServer(cfg, actionQueue)
	fmt.Println(webAppServer.ToString())
	webAppServer.Start()

}
