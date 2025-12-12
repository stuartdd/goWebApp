package server

import (
	"testing"
	"time"
)

func TestHttpContent(t *testing.T) {
	// change ReloadConfigSeconds to 1 to test reload
	configData := loadConfigData(t, testConfigFile)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()
	resp, content := RunClientGet(t, configData, "ping", 200, "?", -1, 10)
	AssertContains(t, content, []string{"\"cause\":\"Ping\""})
	AssertContains(t, resp.Header["Content-Type"][0], []string{"application/json", "charset=utf-8"})
	AssertContains(t, resp.Header["Content-Length"][0], []string{"54"})
	AssertContains(t, resp.Header["Date"][0], []string{"GMT"})
	AssertContains(t, resp.Header["Server"][0], []string{"TestServer"})

}
