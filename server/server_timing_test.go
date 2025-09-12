package server

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/runCommand"
)

var configData = LoadConfigDataBenchmark()
var requestURL = ""

// GOMAXPROCS=1 go test -bench=BenchmarkServer -benchmem -benchtime=10s
func BenchmarkServer(b *testing.B) {
	res, err := http.Get(requestURL + "/favicon.ico")
	if err != nil {
		b.Fatalf("Client error: %s", err.Error())
	}
	if res.StatusCode != 200 {
		b.Fatalf("Status Expected 200 Actual %d", res.StatusCode)
	}
	logger.Log(fmt.Sprintf("Run:%d",b.N))
}

func TestLoad(t *testing.T) {
	logger.Log("------------->" + configData.ConfigName)
}

// ------------------------------------------------------------------------------------------------------------

func LoadConfigDataBenchmark() *config.ConfigData {
	errList := config.NewConfigErrorData()
	cd := config.NewConfigData("../goWebAppTest.json", "goWebApp", false, false, false, errList)
	if errList.ErrorCount() > 1 || cd == nil {
		panic(errList.String())
	}
	if cd == nil {
		panic(fmt.Sprintf("Config is nil. Load failed\n%s", errList.String()))
	}
	go RunServer(cd, logger)
	requestURL = fmt.Sprintf("http://localhost%s", cd.GetPortString())
	time.Sleep(100 * time.Millisecond)
	logger.Log("LoadConfigDataBenchmark ------------->" + requestURL)
	return cd
}

func RunServerBenchmark(config *config.ConfigData) {
	actionQueue := make(chan *ActionEvent, 10)
	defer close(actionQueue)
	go func() {
		for {
			ae := <-actionQueue
			switch ae.Id {
			case Exit:
				fmt.Printf("Server: Stopped\n")
			case Ignore:
				fmt.Printf("Server: Ignore\n")
			}
		}
	}()
	var lrm *runCommand.LongRunningManager
	var err error
	if config.GetExecPath() != "" {
		lrm, err = runCommand.NewLongRunningManager(config.GetExecPath(), logger.Log)
		if err != nil {
			panic(fmt.Sprintf("LongRunningManager: failed to initialise. '%s'. ABORTED", err.Error()))
		}
	}

	server, err := NewWebAppServer(config, actionQueue, lrm, logger)
	if err != nil {
		panic(fmt.Sprintf("NewWebAppServer: failed to initialise. '%s'. ABORTED", err.Error()))
	}
	server.Start()
}
