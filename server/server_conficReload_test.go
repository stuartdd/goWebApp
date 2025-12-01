package server

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestConfigReloadConfigChanged(t *testing.T) {
	// change ReloadConfigSeconds to 1 to test reload
	os.Remove(testConfigFileTmp1)
	updateConfigData(t, testConfigFile, testConfigFileTmp1, "\"ReloadConfigSeconds\":", "\"ReloadConfigSeconds\":1,")
	configData := loadConfigData(t, testConfigFileTmp1)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
		os.Remove(testConfigFileTmp1)
		os.Remove(testConfigFileTmp2)

	}()

	_, resp := RunClientGet(t, configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"1 seconds"})
	AssertLogContains(t, logger, []string{"GET:/server/status", "Status:200"})
	logger.Reset()

	// change ReloadConfigSeconds to 2 and remove error 'picsMissing'
	os.Remove(testConfigFileTmp1)
	updateConfigData(t, testConfigFile, testConfigFileTmp2, "\"ReloadConfigSeconds\":", "\"ReloadConfigSeconds\":2,")
	updateConfigData(t, testConfigFileTmp2, testConfigFileTmp1, "\"picsMissing\":", "")
	// current timer should be 1 so next request should reload after 1 second wait
	time.Sleep(1000 * time.Millisecond)
	_, _ = RunClientGet(t, configData, "ping", 200, "?", -1, 10)
	AssertLogNotContains(t, logger, []string{fmt.Sprintf("%s Failed to load", testConfigFileTmp1), "missingfolder] Not found"})
	AssertLogContains(t, logger, []string{fmt.Sprintf("%s file reload OK", testConfigFileTmp1)})
	logger.Reset()
	// Timer is now 2 seconds. Wait 1 and check that no reload occurs
	time.Sleep(1000 * time.Millisecond)
	_, resp = RunClientGet(t, configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"2 seconds"})
	AssertLogNotContains(t, logger, []string{"file reload OK"})

	// Wait another second. Check that a reload does occur
	logger.Reset()
	time.Sleep(1000 * time.Millisecond)
	_, resp = RunClientGet(t, configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"2 seconds"})
	AssertLogContains(t, logger, []string{fmt.Sprintf("%s file reload OK", testConfigFileTmp1)})

	// t.Fatal("\n ************\n", logger.Get(), "************\n")
}

func TestConfigReloadFailedMissingFolder(t *testing.T) {
	// change ReloadConfigSeconds to 1 to test reload
	os.Remove(testConfigFileTmp1)
	updateConfigData(t, testConfigFile, testConfigFileTmp1, "\"ReloadConfigSeconds\":", "\"ReloadConfigSeconds\":1,")
	// load config data and run the server
	configData := loadConfigData(t, testConfigFileTmp1)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()

	// Get server status and test response to confirm ReloadConfigSeconds
	_, resp := RunClientGet(t, configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"1 seconds"})
	AssertLogContains(t, logger, []string{"GET:/server/status", "Status:200"})
	logger.Reset()

	// change ReloadConfigSeconds to 999 to test reload
	os.Remove(testConfigFileTmp1)
	updateConfigData(t, testConfigFile, testConfigFileTmp1, "\"ReloadConfigSeconds\":", "\"ReloadConfigSeconds\":999,")
	// wait for next request to force reload
	time.Sleep(1000 * time.Millisecond)
	logger.Reset()

	// Next request should run ok but the reload should fail as there is missingfolder error
	_, resp = RunClientGet(t, configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"1 seconds"})
	AssertLogContains(t, logger, []string{"goWebAppTestTmp1.json Failed to load", "missingfolder] Not found"})
	logger.Reset()
	// Ensure changes NOT applied
	_, resp = RunClientGet(t, configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"1 seconds"})
}
