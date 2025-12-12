package server

import (
	"fmt"
	"testing"
	"time"
)

func TestHttpContentLocPath(t *testing.T) {
	// change ReloadConfigSeconds to 1 to test reload
	configData := loadConfigData(t, testConfigFile)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()
	url := fmt.Sprintf("ff/user/stuart/loc/pics/name/%s", encodeValue("benchPic.jpg"))
	resp, _ := RunClientGet(t, configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestHttpContentLocPath", resp, []string{"image/jpeg"}, "6038855")
}
func TestHttpContentJson(t *testing.T) {
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
	AssertHeader(t, "TestHttpContentJson", resp, []string{"application/json", "charset=utf-8"}, "54")
	AssertContains(t, content, []string{"\"cause\":\"Ping\""})
}

func TestHttpContentJpeg(t *testing.T) {
	// change ReloadConfigSeconds to 1 to test reload
	configData := loadConfigData(t, testConfigFile)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()
	url := fmt.Sprintf("ff/user/stuart/loc/pics/name/%s", encodeValue("benchPic.jpg"))
	resp, _ := RunClientGet(t, configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestHttpContentJpeg", resp, []string{"image/jpeg"}, "6038855")
}

func TestHttpContentSatic(t *testing.T) {
	// change ReloadConfigSeconds to 1 to test reload
	configData := loadConfigData(t, testConfigFile)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()
	resp, _ := RunClientGet(t, configData, "static/images/pic.jpeg", 200, "?", -1, 10)
	AssertHeader(t, "TestHttpContentSatic", resp, []string{"image/jpeg"}, "4821")
}

func TestHttpContentIcon(t *testing.T) {
	// change ReloadConfigSeconds to 1 to test reload
	configData := loadConfigData(t, testConfigFile)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()
	resp, _ := RunClientGet(t, configData, "static/favicon1.ico", 200, "?", -1, 10)
	AssertHeader(t, "TestHttpContentIcon", resp, []string{"image/vnd.microsoft.icon"}, "190985")
}
