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
	resp, _ := RunClientGet(t, "TestHttpContentLocPath 1", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestHttpContentLocPath 1", resp, []string{"image/jpeg"}, "6038855")
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
	resp, content := RunClientGet(t, "TestHttpContentJson 1", configData, "ping", 200, "?", -1, 10)
	AssertHeader(t, "TestHttpContentJson 1", resp, []string{"application/json", "charset=utf-8"}, "54")
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
	resp, _ := RunClientGet(t, "TestHttpContentJpeg 1", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestHttpContentJpeg 1", resp, []string{"image/jpeg"}, "6038855")
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
	resp, _ := RunClientGet(t, "TestHttpContentSatic 1", configData, "static/images/pic.jpeg", 200, "?", -1, 10)
	AssertHeader(t, "TestHttpContentSatic 1", resp, []string{"image/jpeg"}, "4821")
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
	resp, _ := RunClientGet(t, "TestHttpContentIcon 1", configData, "static/favicon.ico", 200, "?", -1, 10)
	AssertHeader(t, "TestHttpContentIcon 1", resp, []string{"image/vnd.microsoft.icon"}, "177174")
}
