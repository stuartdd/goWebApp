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
	if r.Header["Content-Type"][0] != "application/json" {
		t.Fatal("Content-Type should be application/json. Was:" + r.Header["Content-Type"][0])
	}
	AssertLogContains(t, logger, []string{"FastFile:", "/testdata/stuart/s-pics/s-testfolder/testdata2.json"})

	logger.Reset()
	url = fmt.Sprintf("ff/user/stuart/loc/pics/path/%s/name/%s", encodeValue("s-testfolder/s-testdir1"), encodeValue("testdata.json"))
	r, respBody = RunClientGet(t, configData, url, 200, "?", -1, 10)
	AssertEquivilent(t, respBody, "{\"Data\":\"This is the data for 2\"}")
	if r.Header["Content-Type"][0] != "application/json" {
		t.Fatal("Content-Type should be application/json. Was:" + r.Header["Content-Type"][0])
	}

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
	_, respBody := RunClientGet(t, configData, url, 400, "?", -1, 10)
	AssertContains(t, respBody, []string{"Get File Error", "Bad Request"})
	AssertLogContains(t, logger, []string{"Invalid request:/ff/user/stuart/loc/pics/name"})

	logger.Reset()
	url = "ff/user/lol/loc/pics/name/fi"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertContains(t, respBody, []string{"Get File Error", "Not Found"})
	AssertLogContains(t, logger, []string{"Invalid user:/ff/user/lol/loc/pics/name/fi"})

	logger.Reset()
	url = "ff/user/stuart/loc/pac/name/fi"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertContains(t, respBody, []string{"Get File Error", "Not Found"})
	AssertLogContains(t, logger, []string{"Invalid location:/ff/user/stuart/loc/pac/name/fi"})

	logger.Reset()
	url = "ff/user/stuart/loc/pics/name/fi"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertContains(t, respBody, []string{"File not found", "Not Found"})
	AssertLogContains(t, logger, []string{"File not found", "/testdata/stuart/s-pics/fi"})

	logger.Reset()
	url = "ff/user/stuart/loc/pics/path/s-testfolder"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertContains(t, respBody, []string{"Get File Error", "Not Found"})
	AssertLogContains(t, logger, []string{"Invalid path or name:/ff/user/stuart/loc/pics/path/s-testfolder"})

	logger.Reset()
	url = "ff/user/stuart/loc/pics/path/s-testfolder/name"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertContains(t, respBody, []string{"Get File Error", "Not Found"})
	AssertLogContains(t, logger, []string{"Invalid path or name:/ff/user/stuart/loc/pics/path/s-testfolder"})

	logger.Reset()
	url = "ff/user/stuart/loc/pics/path/s-testfolder/name/fi"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertContains(t, respBody, []string{"File not found", "Not Found"})
	AssertLogContains(t, logger, []string{"File not found", "/testdata/stuart/s-pics/s-testfolder/fi"})

	logger.Reset()
	url = "ff/user/stuart/loc/pics/path/xxx/name/testdata2.json"
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertContains(t, respBody, []string{"File not found", "Not Found"})
	AssertLogContains(t, logger, []string{"File not found", "/testdata/stuart/s-pics/xxx/testdata2.json"})

	logger.Reset()
	url = fmt.Sprintf("ff/user/stuart/loc/pics/path/s-testfolder/name/%s", encodeValue("xxx.json"))
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertContains(t, respBody, []string{"File not found", "Not Found"})
	AssertLogContains(t, logger, []string{"File not found", "/testdata/stuart/s-pics/s-testfolder/xxx.json"})

	logger.Reset()
	url = fmt.Sprintf("ff/user/stuart/loc/pics/path/%s/name/%s", encodeValue("s-testfolder"), encodeValue("xxx.json"))
	_, respBody = RunClientGet(t, configData, url, 404, "?", -1, 10)
	AssertContains(t, respBody, []string{"File not found", "Not Found"})
	AssertLogContains(t, logger, []string{"File not found", "/testdata/stuart/s-pics/s-testfolder/xxx.json"})

	// url = "ff/user/stuart/loc/pics/name/benchPic.jpg"
	// _, respBody = RunClientGet(t, configData, url, 200, "?", -1, 10)
	// AssertContains(t, respBody, []string{"File not found", "Not Found"})
	// AssertLogContains(t, logger, []string{"Invalid location:/ff/user/stuart/loc/pac/name/fi"})

	// if r.Header["Content-Type"][0] != "text/plain; charset=utf-8" {
	// 	t.Fatal("Content-Type should be text/plain; charset=utf-8. Was:" + r.Header["Content-Type"][0])
	// }
	// if respBody != "xxABC" {
	// 	t.Fatalf("Result initial set should be xxABC. It is '%s'", respBody)
	// }

}

func encodeValue(unEncoded string) string {
	if unEncoded == "" {
		return ""
	}
	return controllers.EncodedValuePrefix + base64.StdEncoding.EncodeToString([]byte(unEncoded))
}
