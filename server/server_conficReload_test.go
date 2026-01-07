package server

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stuartdd/goWebApp/config"
)

func TestConfigReloadConfigChanged(t *testing.T) {
	configData, tempConfigFile := UpdateConfigAndLoad(t, func(cdff *config.ConfigDataFromFile) {
		cdff.ReloadConfigSeconds = 1
	}, nil, true)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
		os.Remove(tempConfigFile)
	}()

	_, resp := RunClientGet(t, "TestConfigReloadConfigChanged 1", configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"1 seconds"})
	AssertLogContains(t, logger, []string{"GET:/server/status", "Status:200"})
	logger.Reset()

	// change ReloadConfigSeconds to 2 and remove error 'picsMissing'
	_, tempConfigFile = UpdateConfigAndLoad(t, func(cdff *config.ConfigDataFromFile) {
		cdff.ReloadConfigSeconds = 2
		cdff.Users["stuart"].Locations["picsMissing"] = ""
	}, nil, false)

	time.Sleep(1000 * time.Millisecond)
	_, _ = RunClientGet(t, "TestConfigReloadConfigChanged 2", configData, "ping", 200, "?", -1, 10)
	AssertLogNotContains(t, logger, []string{fmt.Sprintf("%s Failed to load", tempConfigFile), "missingfolder] Not found"})
	AssertLogContains(t, logger, []string{fmt.Sprintf("%s file reload OK", tempConfigFile)})
	logger.Reset()
	// Timer is now 2 seconds. Wait 1 and check that no reload occurs
	time.Sleep(1000 * time.Millisecond)
	_, resp = RunClientGet(t, "TestConfigReloadConfigChanged 3", configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"2 seconds"})
	AssertLogNotContains(t, logger, []string{"file reload OK"})

	// Wait another second. Check that a reload does occur
	logger.Reset()
	time.Sleep(1000 * time.Millisecond)
	_, resp = RunClientGet(t, "TestConfigReloadConfigChanged 4", configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"2 seconds"})
	AssertLogContains(t, logger, []string{fmt.Sprintf("%s file reload OK", tempConfigFile)})

	// t.Fatal("\n ************\n", logger.Get(), "************\n")
}

func TestConfigReloadFailedMissingFolder(t *testing.T) {
	configData, tempConfigFile := UpdateConfigAndLoad(t, func(cdff *config.ConfigDataFromFile) {
		cdff.ReloadConfigSeconds = 1
	}, nil, true)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
		os.Remove(tempConfigFile)
	}()

	// Get server status and test response to confirm ReloadConfigSeconds
	_, resp := RunClientGet(t, "TestConfigReloadFailedMissingFolder 1", configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"1 seconds"})
	AssertLogContains(t, logger, []string{"GET:/server/status", "Status:200"})
	logger.Reset()

	_, tempConfigFile = UpdateConfigAndLoad(t, func(cdff *config.ConfigDataFromFile) {
		cdff.ReloadConfigSeconds = 999
	}, nil, false)

	// wait for next request to force reload
	time.Sleep(1000 * time.Millisecond)
	logger.Reset()

	// Next request should run ok but the reload should fail as there is missingfolder error
	_, resp = RunClientGet(t, "TestConfigReloadFailedMissingFolder 2", configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"1 seconds"})
	AssertLogContains(t, logger, []string{fmt.Sprintf("%s Failed to load", tempConfigFile), "missingfolder] Not found"})
	logger.Reset()
	// Ensure changes NOT applied
	_, resp = RunClientGet(t, "TestConfigReloadFailedMissingFolder 3", configData, "server/status", 200, "?", -1, 10)
	AssertContains(t, resp, []string{",\"Config Reload\":\"1 seconds"})
}
