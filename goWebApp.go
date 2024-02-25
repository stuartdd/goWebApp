package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"time"

	"stuartdd.com/config"
)

type ActionId int

const (
	ignore ActionId = iota
	exit1
	exit2
)

func main() {
	configFileName := ""
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}
	err := config.LoadConfigData(configFileName)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	configData := config.GetConfigDataInstance()

	cAsString, err := configData.ToString()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	fmt.Printf("Config :%s\n", cAsString)

	actionQueue := make(chan ActionId, 10)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/exit1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Exit 1")
		actionQueue <- exit1
	})

	http.HandleFunc("/exit2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Exit 2")
		actionQueue <- exit2
	})

	http.HandleFunc("/ignore", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Ignore")
		actionQueue <- ignore
	})

	fmt.Printf("Server '%s' running on port %s\n", configData.ConfigName, configData.PortString())

	go func() {
		for {
			a := <-actionQueue
			switch a {
			case exit1:
				close(actionQueue)
				fmt.Printf("Server Exit 100 ms\n")
				time.Sleep(100 * time.Millisecond)
				os.Exit(1)
			case exit2:
				close(actionQueue)
				fmt.Printf("Server Exit 5 Sec\n")
				time.Sleep(5 * time.Second)
				os.Exit(2)
			case ignore:
				fmt.Printf("Server Ignore\n")
			}
		}
	}()

	log.Fatal(http.ListenAndServe(configData.PortString(), nil))

}
