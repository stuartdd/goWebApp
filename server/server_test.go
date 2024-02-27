package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"stuartdd.com/config"
)

var serverState string = ""

func TestClient(t *testing.T) {
	configData, err := config.NewConfigData("../goWebAppTest.json")
	if err != nil {
		t.Fatal(err)
	}
	go RunServer(configData)
	time.Sleep(100 * time.Millisecond)
	if serverState != "Running" {
		t.Fatalf("Server state is %s. Expected 'Running'", serverState)
	}

	res := RunClient(t, configData, "ABC", 404, "{\"status\":404, \"error\":\"Not Found\"}")
	AssertHeaderEquals(t, res, "Content-Length", "35")
	AssertHeaderEquals(t, res, "Content-Type", fmt.Sprintf("%s; charset=%s", config.DefaultContentType, configData.ContentTypeCharset))
	AssertHeaderEquals(t, res, "Server", configData.ServerName)
	res = RunClient(t, configData, "tap", 200, "{\"status\":200, \"error\":\"OK\"}")
	AssertHeaderEquals(t, res, "Content-Length", "28")
	res = RunClient(t, configData, "exit", 200, "{\"status\":200, \"error\":\"Accepted\"}")
	AssertHeaderEquals(t, res, "Content-Length", "34")
	if serverState != "Stopped" {
		t.Fatalf("Server state is %s. Expected 'Stopped'", serverState)
	}
}

func RunClient(t *testing.T, config *config.ConfigData, path string, expectedStatus int, expectedBody string) *http.Response {
	requestURL := fmt.Sprintf("http://localhost:%d/%s", config.Port, path)
	res, err := http.Get(requestURL)
	if err != nil {
		t.Fatalf("Client error: %s", err.Error())
	}
	if res.StatusCode != expectedStatus {
		t.Fatalf("Status for path http://localhost:%d/%s. Expected %d Actual %d", config.Port, path, expectedStatus, res.StatusCode)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("client: could not read response body: %s\n", err)
	}
	if trimString(string(resBody)) != trimString(expectedBody) {
		t.Fatalf("Status for path http://localhost:%d/%s.\nExpected '%s' \nActual   '%s'", config.Port, path, expectedBody, string(resBody))
	}
	return res
}

func trimString(res string) string {
	var buffer bytes.Buffer
	for i := 0; i < len(res); i++ {
		c := res[i]
		if c >= 32 {
			buffer.WriteByte(c)
		}
	}
	return strings.Trim(buffer.String(), " ")
}

func RunServer(config *config.ConfigData) {
	actionQueue := make(chan ActionId, 10)
	defer close(actionQueue)
	go func() {
		for {
			acId := <-actionQueue
			switch acId {
			case Exit:
				fmt.Printf("Server: Stopped\n")
				serverState = "Stopped"
			case Ignore:
				fmt.Printf("Server: Ignore\n")
			}
		}
	}()
	server := NewWebAppServer(config, actionQueue)
	serverState = "Running"
	server.Start()
}

func AssertHeaderEquals(t *testing.T, res *http.Response, headerName, expected0 string) {
	hv := res.Header[headerName]
	if len(hv) == 0 {
		t.Fatalf("Header %s.\nExpected:%s\nActual:  Header is empty", headerName, expected0)
	}
	if hv[0] != expected0 {
		t.Fatalf("Header[0] %s.\nExpected:%s\nActual:  %s", headerName, expected0, hv[0])
	}
}
