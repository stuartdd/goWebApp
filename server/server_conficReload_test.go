package server

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestConfigReloadConfigChanged(t *testing.T) {
	os.Remove(testConfigFileTmp)
	updateConfigData(t, testConfigFile, testConfigFileTmp, "\"ReloadConfigSeconds\":", "\"ReloadConfigSeconds\":1,")
	configData := loadConfigData(t, testConfigFileTmp)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()

	_, resp := RunClientGet(t, configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Reload Config in\":\"1 seconds\""})
	AssertLogContains(t, logger, []string{"GET:/server/status", "Status:200"})
	logger.Reset()

	os.Remove(testConfigFileTmp)

	updateConfigData(t, testConfigFile, testConfigFileTmp, "\"ReloadConfigSeconds\":", "\"ReloadConfigSeconds\":999,")
	time.Sleep(1000 * time.Millisecond)
	_, _ = RunClientGet(t, configData, "ping", 200, "?", -1, 10)
	AssertLogContains(t, logger, []string{fmt.Sprintf("%s Failed to load", testConfigFileTmp), "missingfolder] Not found"})
	_, resp = RunClientGet(t, configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Reload Config in\":\"1 seconds\""}) // Ensure data not changed! ReloadConfigSeconds still == 1
}

func TestConfigReloadFailedMissingFolder(t *testing.T) {
	os.Remove(testConfigFileTmp)
	updateConfigData(t, testConfigFile, testConfigFileTmp, "\"ReloadConfigSeconds\":", "\"ReloadConfigSeconds\":1,")
	configData := loadConfigData(t, testConfigFileTmp)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()

	_, resp := RunClientGet(t, configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Reload Config in\":\"1 seconds\""})
	AssertLogContains(t, logger, []string{"GET:/server/status", "Status:200"})
	logger.Reset()

	os.Remove(testConfigFileTmp)
	updateConfigData(t, testConfigFile, testConfigFileTmp, "\"ReloadConfigSeconds\":", "\"ReloadConfigSeconds\":999,")
	time.Sleep(1000 * time.Millisecond)
	_, _ = RunClientGet(t, configData, "ping", 200, "?", -1, 10)
	AssertLogContains(t, logger, []string{fmt.Sprintf("%s Failed to load", testConfigFileTmp), "missingfolder] Not found"})
	_, resp = RunClientGet(t, configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Reload Config in\":\"1 seconds\""}) // Ensure data not changed! ReloadConfigSeconds still == 1
}
