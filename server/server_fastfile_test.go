package server

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stuartdd/goWebApp/controllers"
)

func TestFastFile(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()
	logger.Reset()
	url := fmt.Sprintf("ff/user/stuart/loc/pics/path/%s/name/%s", encodeValue("s-testfolder"), encodeValue("testdata2.json"))
	r, respBody := RunClientGet(t, configData, url, 200, "?", -1, 10)
	AssertEquivilent(t, respBody, "{\"Data\":\"This is the data for 2\"}")
	AssertHeader(t, "TestFastFile 1", r, []string{"application/json", "charset=utf-8"}, "33")
	AssertLogContains(t, logger, []string{"FastFile:", "/testdata/stuart/s-pics/s-testfolder/testdata2.json"})

	logger.Reset()
	url = fmt.Sprintf("ff/user/stuart/loc/pics/path/%s/name/%s", encodeValue("s-testfolder/s-testdir1"), encodeValue("testdata.json"))
	r, respBody = RunClientGet(t, configData, url, 200, "?", -1, 10)
	AssertEquivilent(t, respBody, "{\"Data\":\"This is the data for 2\"}")
	AssertHeader(t, "TestFastFile 2", r, []string{"application/json", "charset=utf-8"}, "33")
}

func TestFastFileErrors(t *testing.T) {

	configData := loadConfigData(t, testConfigFile)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()
	url := "ff/user/stuart/loc/pics/name"
	r, respBody := RunClientGet(t, configData, url, 400, "?", -1, 10)
	AssertHeader(t, "TestFastFileErrors 1", r, []string{"application/json", "charset=utf-8"}, "72")
	AssertContains(t, respBody, []string{"Get File Error", "Bad Request"})
	AssertLogContains(t, logger, []string{"Invalid request:/ff/user/stuart/loc/pics/name"})

	logger.Reset()
	url = "ff/user/lol/loc/pics/name/fi"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestFastFileErrors 2", r, []string{"application/json", "charset=utf-8"}, "72")
	AssertContains(t, respBody, []string{"Get File Error", "Not Found"})
	AssertLogContains(t, logger, []string{"Invalid user:/ff/user/lol/loc/pics/name/fi"})

	logger.Reset()
	url = "ff/user/stuart/loc/pac/name/fi"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestFastFileErrors 3", r, []string{"application/json", "charset=utf-8"}, "72")
	AssertContains(t, respBody, []string{"Get File Error", "Not Found"})
	AssertLogContains(t, logger, []string{"Invalid location:/ff/user/stuart/loc/pac/name/fi"})

	logger.Reset()
	url = "ff/user/stuart/loc/pics/name/fi"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestFastFileErrors 4", r, []string{"application/json", "charset=utf-8"}, "72")
	AssertContains(t, respBody, []string{"File not found", "Not Found"})
	AssertLogContains(t, logger, []string{"File not found", "/testdata/stuart/s-pics/fi"})

	logger.Reset()
	url = "ff/user/stuart/loc/pics/path/s-testfolder"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestFastFileErrors 5", r, []string{"application/json", "charset=utf-8"}, "72")
	AssertContains(t, respBody, []string{"Get File Error", "Not Found"})
	AssertLogContains(t, logger, []string{"Invalid path or name:/ff/user/stuart/loc/pics/path/s-testfolder"})

	logger.Reset()
	url = "ff/user/stuart/loc/pics/path/s-testfolder/name"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestFastFileErrors 6", r, []string{"application/json", "charset=utf-8"}, "72")
	AssertContains(t, respBody, []string{"Get File Error", "Not Found"})
	AssertLogContains(t, logger, []string{"Invalid path or name:/ff/user/stuart/loc/pics/path/s-testfolder"})

	logger.Reset()
	url = "ff/user/stuart/loc/pics/path/s-testfolder/name/fi"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestFastFileErrors 7", r, []string{"application/json", "charset=utf-8"}, "72")
	AssertContains(t, respBody, []string{"File not found", "Not Found"})
	AssertLogContains(t, logger, []string{"File not found", "/testdata/stuart/s-pics/s-testfolder/fi"})

	logger.Reset()
	url = "ff/user/stuart/loc/pics/path/xxx/name/testdata2.json"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestFastFileErrors 8", r, []string{"application/json", "charset=utf-8"}, "72")
	AssertContains(t, respBody, []string{"File not found", "Not Found"})
	AssertLogContains(t, logger, []string{"File not found", "/testdata/stuart/s-pics/xxx/testdata2.json"})

	logger.Reset()
	url = fmt.Sprintf("ff/user/stuart/loc/pics/path/s-testfolder/name/%s", encodeValue("xxx.json"))
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestFastFileErrors 9", r, []string{"application/json", "charset=utf-8"}, "72")
	AssertContains(t, respBody, []string{"File not found", "Not Found"})
	AssertLogContains(t, logger, []string{"File not found", "/testdata/stuart/s-pics/s-testfolder/xxx.json"})

	logger.Reset()
	url = fmt.Sprintf("ff/user/stuart/loc/pics/path/%s/name/%s", encodeValue("s-testfolder"), encodeValue("xxx.json"))
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestFastFileErrors 10", r, []string{"application/json", "charset=utf-8"}, "72")
	AssertContains(t, respBody, []string{"File not found", "Not Found"})
	AssertLogContains(t, logger, []string{"File not found", "/testdata/stuart/s-pics/s-testfolder/xxx.json"})

}

func encodeValue(unEncoded string) string {
	if unEncoded == "" {
		return ""
	}
	return controllers.EncodedValuePrefix + base64.StdEncoding.EncodeToString([]byte(unEncoded))
}
