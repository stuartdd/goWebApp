package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/controllers"
	"github.com/stuartdd/goWebApp/logging"
	"github.com/stuartdd/goWebApp/runCommand"
)

const configRef = "../goWebAppTest.json"
const configTmp = "../goWebAppTmp.json"
const thumbnailTrimPrefix = 20
const thumbnailTrimSuffix = 4

type TLog struct {
	B       bytes.Buffer
	RSCount int
}

func (l *TLog) Close() {}
func (l *TLog) Log(s string) {
	l.B.WriteString("LOG: ")
	l.B.WriteString(s)
	l.B.WriteString("\n")
	os.Stdout.WriteString("LOG: ")
	os.Stdout.WriteString(s)
	os.Stdout.WriteString("\n")
}

func (l *TLog) LogVerbose(s string) {
	l.B.WriteString("VERBOSE: ")
	l.B.WriteString(s)
	l.B.WriteString("\n")
	os.Stdout.WriteString("VERBOSE: ")
	os.Stdout.WriteString(s)
	os.Stdout.WriteString("\n")
}

func WriteLogToFile(path string) {
	os.WriteFile(filepath.Join(path, "TLog.log"), logger.B.Bytes(), 0644)
}

func (l *TLog) VerboseFunction() func(string) {
	return l.LogVerbose
}

func (l *TLog) Get() string {
	return l.B.String()
}
func (l *TLog) IsOpen() bool {
	return true
}
func (l *TLog) LogFileName() string {
	return "DummyLogger.log"
}
func (l *TLog) Reset() {
	if l.B.Len() == 0 {
		l.RSCount = 0
	} else {
		l.RSCount = l.RSCount + 1
	}
	l.B.Truncate(0)
	l.Log(fmt.Sprintf("Log-Reset:%d", l.RSCount))
}

var serverState string = ""
var logger = &TLog{}

const postDataFile1 = "{\"Data\":\"This is data ONE for file 1\"}"
const postDataFile2 = "{\"Data\":\"This is data TWO for file 2\"}"
const testdatafile = "testdata.json"
const testConfigFile = "../goWebAppTest.json"
const testPropertyFile = "../userProperties.json"

func TestUrlRequestParamsMap(t *testing.T) {
	rootUrlList := NewRootUrlList()
	AssertMatch(t, "0", rootUrlList.AddUrlRequestMatcher("/a/b/*/c/*", "get", true), "/x/b/1/c/4", "GET", false, "")
	AssertMatch(t, "1", rootUrlList.AddUrlRequestMatcher("/a/b/*/c/*", "get", true), "/a/b/1/x/4", "GET", false, "b=1")
	AssertMatch(t, "2", rootUrlList.AddUrlRequestMatcher("/a/b/*/c/*", "get", true), "/a/b/1/c", "GET", false, "")
	AssertMatch(t, "3", rootUrlList.AddUrlRequestMatcher("/a/b/*/c/*", "get", true), "/a/b/1/c/3", "GET", true, "b=1,c=3")
	AssertMatch(t, "4", rootUrlList.AddUrlRequestMatcher("a", "get", true), "/a", "get", true, "")
	AssertMatch(t, "5", rootUrlList.AddUrlRequestMatcher("a", "get", true), "a", "get", true, "")
	AssertMatch(t, "5", rootUrlList.AddUrlRequestMatcher("/a", "get", true), "/a", "get", true, "")
	AssertMatch(t, "6", rootUrlList.AddUrlRequestMatcher("/a/b/*/*/c/*", "get", true), "/a/b/1/2/c/3", "post", false, "")
	AssertMatch(t, "7", rootUrlList.AddUrlRequestMatcher("/a/b/*/*/c/*", "get", true), "/a/b/1/2/C/3", "GET", false, "b=1")
	AssertMatch(t, "8", rootUrlList.AddUrlRequestMatcher("/a/b/*/*/c/*", "get", true), "/a/b/1/2/c/3", "get", true, "b=1,c=3")
	AssertMatch(t, "9", rootUrlList.AddUrlRequestMatcher("/a/b/*/*/c/*", "get", true), "/a/b/1/2/c/3", "GET", true, "b=1,c=3")
	AssertMatch(t, "10", rootUrlList.AddUrlRequestMatcher("/a/*/b/*/c/*", "get", true), "/a/1/b/2/c/3", "GET", true, "a=1,b=2,c=3")
	AssertMatch(t, "10", rootUrlList.AddUrlRequestMatcher("", "get", true), "/a/1/b/2/c/3", "GET", false, "")
	AssertMatch(t, "11", rootUrlList.AddUrlRequestMatcher("", "get", true), "", "GET", false, "")
	AssertMatch(t, "12", rootUrlList.AddUrlRequestMatcher("", "post", true), "", "GET", false, "")
	rootUrlList.AddUrlRequestMatcher("b", "post", true)
	if rootUrlList.String() != "a,b," {
		t.Fatal("RootUrlList is incorrect: Expected:a,b, Actual:", rootUrlList.String())
	}
}

func TestGetSetPropNewFile(t *testing.T) {
	os.Remove(testPropertyFile)
	configData, _ := UpdateConfigAndLoad(t, func(cdff *config.ConfigDataFromFile) {
		cdff.UserPropertiesFile = testPropertyFile
	}, nil, true)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
		os.Remove(testPropertyFile)
	}()

	if configData.ConfigFileData.UserPropertiesFile != testPropertyFile {
		t.Fatalf("UserPropertiesFile is not %s", testPropertyFile)
	}

	url := "prop/user/bob/name/xx123/value/xxABC"
	r, respBody := RunClientGet(t, "TestGetSetPropNewFile 1", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 1", r, []string{"text/plain", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	if respBody != "xxABC" {
		t.Fatalf("Result initial set should be xxABC. It is '%s'", respBody)
	}

	url = "prop/user/bob/name/AB/value/CD"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 2", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 2", r, []string{"text/plain", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	if respBody != "CD" {
		t.Fatalf("Result first read should be CD. It is '%s'", respBody)
	}

	url = "prop/user/bob/name/xx123"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 3", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 3", r, []string{"text/plain", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	if respBody != "xxABC" {
		t.Fatalf("Result first read should be xxABC. It is '%s'", respBody)
	}

	url = "prop/user/bob/name/AB"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 4", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 4", r, []string{"text/plain", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	if respBody != "CD" {
		t.Fatalf("Result first read should be CD. It is '%s'", respBody)
	}

	url = "prop/user/bob/name/xx123/value/yyABC"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 5", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 4.1", r, []string{"text/plain", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	if respBody != "yyABC" {
		t.Fatalf("Result of change value should be yyABC. It is '%s'", respBody)
	}
	time.Sleep(300 * time.Millisecond) // Delay as write to propertiers file is async

	url = "prop/user/bob/name/xx123"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 6", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 4.2", r, []string{"text/plain", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	if respBody != "yyABC" {
		t.Fatalf("Result second read should be yyABC. It is '%s'", respBody)
	}

	url = "prop/user/bob/name/AB"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 7", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 4.3", r, []string{"text/plain", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	if respBody != "CD" {
		t.Fatalf("Result was changed by update. Should be CD. '%s'", respBody)
	}

	url = "prop/user/bob"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 8", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 5", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{"\"AB\":\"CD\"", "\"xx123\":\"yyABC\""})
	url = "prop/user/stuart"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 9", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 6", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))

	AssertEquivilent(t, "TestGetSetPropNewFile 9.5", respBody, "{\"id\":\"stuart\",\"info\":false,\"name\":\"Stuart\"}")

	url = "prop/user/frrrred/name/AB/value/XX"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 10", configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 7", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))

	AssertContains(t, respBody, []string{"User not found", "\"error\":true"})
	url = "prop/user/frrrred/name/AB"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 11", configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 8", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))

	AssertContains(t, respBody, []string{"User not found", "\"error\":true"})

	url = "prop/user/frrrred"
	r, respBody = RunClientGet(t, "TestGetSetPropNewFile 12", configData, url, 404, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 9", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))

	AssertContains(t, respBody, []string{"User not found", "\"error\":true"})
}

func TestGetSetPropNoFileDef(t *testing.T) {
	configData, _ := UpdateConfigAndLoad(t, func(cdff *config.ConfigDataFromFile) {
		cdff.UserPropertiesFile = ""
	}, nil, true)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		StopServer(t, configData)
	}()

	if configData.ConfigFileData.UserPropertiesFile != "" {
		t.Fatal("UserPropertiesFile is not empty")
	}
	url := "prop/user/bob/name/xx123/value/xxABC"
	r, respBody := RunClientGet(t, "TestGetSetPropNoFileDef 1", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 4", r, []string{"text/plain", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	if respBody != "xxABC" {
		t.Fatalf("Result should be xxABC. It is '%s'", respBody)
	}

	url = "prop/user/bob/name/xx123"
	r, respBody = RunClientGet(t, "TestGetSetPropNoFileDef 2", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 4", r, []string{"text/plain", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	if respBody != "" {
		t.Fatalf("Result should be empty. It is '%s'", respBody)
	}
	url = "prop/user/bob"
	r, respBody = RunClientGet(t, "TestGetSetPropNoFileDef 3", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 4", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertEquivilent(t, "TestGetSetPropNewFile 4", respBody, "{\"id\":\"bob\",\"info\":false,\"name\":\"Bob\"}")
}

func TestServerGetUsers(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "server/users"
	r, respBody := RunClientGet(t, "TestServerGetUsers", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestGetSetPropNewFile 4", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{
		"\"id\":\"bob\"",
		"\"name\":\"Bob\"",
	})
}

func TestServerGetTime(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "server/time"
	r, respBody := RunClientGet(t, "TestServerGetTime", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestServerGetTime", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{
		"\"time\":{\"dom\":",
		"\"millis\":",
		"\"timestamp\":",
	})
	AssertLogContains(t, logger, []string{"VERBOSE: Req:  GET:/server/time"})
}

func TestServerStatus(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "server/status"
	r, respBody := RunClientGet(t, "TestServerStatus", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestServerStatus", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{
		"\"error\":false,",
		"\"ConfigName\":\"goWebAppTest.json\"",
		"\"Processes\":[]",
		"\"Log_File\":\"DummyLogger.log\"",
	})

}
func TestServer(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)
	url := ""

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url = "files/user/stuart/loc/pics"
	r, respBody := RunClientGet(t, "TestServer 1", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestServer 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{
		"\"name\":\"pic1.jpeg\", \"encName\":\"X0XcGljMS5qcGVn\"",
		"\"error\":false,\"user\":\"stuart\",\"loc\":\"pics\",\"path\":null,",
	})
	AssertLogContains(t, logger, []string{"VERBOSE: ListFilesAsJson:"})

	logger.Reset()
	url = "files/user/stuart/loc/pics/name/t1.JSON"
	r, respBody = RunClientGet(t, "TestServer 2", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestServer 1.1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{
		"\"UserDataRoot\": \"..\"", "\"ServerName\": \"TestServer\"",
	})
	AssertLogContains(t, logger, []string{"VERBOSE: FastFile:"})

	url = fmt.Sprintf("files/user/stuart/loc/pics/path/%s/name/t5.json", encodeValue("s-testfolder"))
	r, respBody = RunClientGet(t, "TestServer 3", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestServer 1.2", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{
		"\"UserDataRoot\": \"..\"", "\"ServerName\": \"TestServer\"",
	})
	AssertLogContains(t, logger, []string{"VERBOSE: FastFile:"})

	url = "server/status"
	r, respBody = RunClientGet(t, "TestServer 4", configData, url, 200, "?", -1, 10)
	AssertHeader(t, "TestServer 2", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{
		"\"error\":false,\"status\":",
		"\"UpSince\":",
		"\"Processes\":[]",
		"goWebAppTest.json",
	})

	url = "server/users"
	r, respBody = RunClientGet(t, "TestServer 5", configData, url, 200, "?", 69, 10)
	AssertHeader(t, "TestServer 3", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{
		"\"users\"",
		"\"Bob\"",
		"\"Stuart\"",
	})

	url = "server/time"
	r, respBody = RunClientGet(t, "TestServer 6", configData, url, 200, "?", -1, 0)
	AssertHeader(t, "TestServer 4", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{
		"\"time\"",
		"\"millis\"",
		"\"timestamp\"",
	})

}

func TestStaticTemplate(t *testing.T) {
	logger.Reset()
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	var url string
	var resp *http.Response
	var content string

	logger.Reset()
	url = "static/html/simple.html?arg1=ARG1"
	resp, content = RunClientGet(t, "TestStaticTemplate 2", configData, url, 200, "?", 98, 5)
	AssertHeader(t, "TestStaticTemplate 1", resp, []string{"text/html", "charset=utf-8"}, "")
	AssertContains(t, content, []string{
		"<h1>ARG1 %{arg1}</h1>",
	})
	AssertLogContains(t, logger, []string{"FastFile: ", "/html/simple.html"})

	logger.Reset()
	url = "static/tgo.html?arg1=ARG2"
	resp, content = RunClientGet(t, "TestStaticTemplate 2", configData, url, 200, "?", 217, 50)
	AssertHeader(t, "TestStaticTemplate 2", resp, []string{"text/html", "charset=utf-8"}, "")
	AssertContains(t, content, []string{
		"Application name GoWebApp",
		"<p>Config Env:This is a test</p>",
		"<p>Query arg:ARG2</p>",
		"<p>Not found:%{xxxxx}</p>",
		"<p>PWD:/",
		"goWebApp/server</p>",
	})
	AssertLogContains(t, logger, []string{"Resp: Status:200"})

	logger.Reset()
	url = ""
	resp, content = RunClientGet(t, "TestStaticTemplate 0", configData, url, 200, "?", 220, 50)
	AssertHeader(t, "TestStaticTemplate 0", resp, []string{"text/html", "charset=utf-8"}, "")
	AssertContains(t, content, []string{
		"<!DOCTYPE html>",
		"Application name GoWebApp",
		"<p>Config Env:This is a test</p>",
		"Query arg:%{arg1}",
		"found:%{xxxxx}",
		"<p>PWD:/",
		"goWebApp/server</p>",
	})
	AssertLogContains(t, logger, []string{"Resp: Status:200", "/static/tgo.html"})

	logger.Reset()
	url = "tgo.html?arg1=ARG1"
	resp, content = RunClientGet(t, "TestStaticTemplate 1", configData, url, 200, "?", 217, 50)
	AssertHeader(t, "TestStaticTemplate 1", resp, []string{"text/html", "charset=utf-8"}, "")
	AssertContains(t, content, []string{
		"Application name GoWebApp",
		"<p>Config Env:This is a test</p>",
		"<p>Query arg:ARG1</p>",
		"<p>Not found:%{xxxxx}</p>",
		"<p>PWD:/",
		"goWebApp/server</p>",
	})
	AssertLogContains(t, logger, []string{"Resp: Status:200"})

}

func TestStatic(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	var url string
	var resp *http.Response

	logger.Reset()
	url = "favicon.ico"
	resp, _ = RunClientGet(t, "TestStatic 1", configData, url, 200, "?", 177174, 10)
	AssertHeader(t, "TestStatic 1", resp, []string{"image/vnd.microsoft.icon"}, "")
	AssertLogContains(t, logger, []string{"FastFile:", "testdata/static/favicon.ico"})

	logger.Reset()
	url = "images/favicon2.ico"
	resp, _ = RunClientGet(t, "TestStatic 6", configData, url, 200, "?", 177174, 10)
	AssertHeader(t, "TestStatic 6", resp, []string{"image/vnd.microsoft.icon"}, "")
	AssertLogContains(t, logger, []string{"FastFile", "testdata/static/images/favicon2.ico"})

	logger.Reset()
	url = "static/simple.html"
	resp, _ = RunClientGet(t, "TestStatic 2", configData, url, 200, "?", 103, 10)
	AssertHeader(t, "TestStatic 2", resp, []string{"text/html", "charset=utf-8"}, "103")
	AssertLogContains(t, logger, []string{"Read Template File", "testdata/static/simple.htm"})

	logger.Reset()
	url = "static/images/pic.jpeg"
	resp, _ = RunClientGet(t, "TestStatic 3", configData, url, 200, "?", 4821, 10)
	AssertHeader(t, "TestStatic 3", resp, []string{"image/jpeg"}, "")
	AssertLogContains(t, logger, []string{"FastFile", "images/pic.jpeg"})

	logger.Reset()
	url = "images/pic.jpeg"
	resp, _ = RunClientGet(t, "TestStatic 4", configData, url, 200, "?", 4821, 10)
	AssertHeader(t, "TestStatic 4", resp, []string{"image/jpeg"}, "")
	AssertLogContains(t, logger, []string{"FastFile", "images/pic.jpeg"})

	logger.Reset()
	url = "static/images/favicon2.ico"
	resp, _ = RunClientGet(t, "TestStatic 5", configData, url, 200, "?", 177174, 10)
	AssertHeader(t, "TestStatic 5", resp, []string{"image/vnd.microsoft.icon"}, "")
	AssertLogContains(t, logger, []string{"FastFile", "static/images/favicon2.ico"})

	url = "static/notfound.pic"
	RunClientGet(t, "TestStatic 7", configData, url, 404, "?", -1, 0)
	url = "static"
	RunClientGet(t, "TestStatic 8", configData, url, 404, "?", -1, 0)

}

func TestFilePath(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "paths/user/stuart/loc/home"
	r, respBody := RunClientGet(t, "TestFilePath 1", configData, url, 200, "?", -1, 0)
	AssertHeader(t, "TestFilePath 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{"\"error\":false", "\"name\":\"s-pics\""})
	url = "files/user/stuart/loc/home"
	r, respBody = RunClientGet(t, "TestFilePath 3", configData, url, 200, "\"path\":null|\"error\":false", -1, 0)
	AssertHeader(t, "TestFilePath 2", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(respBody)))
	AssertContains(t, respBody, []string{"\"error\":false", "\"name\":\"t2.Data\"", "\"encName\":\"X0XdDIuRGF0YQ==\""})

}
func TestTree(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "files/user/stuart/loc/testtree/tree"

	r, dirList := RunClientGet(t, "TestTree 1", configData, url, 200, "?", -1, 0)
	AssertHeader(t, "TestFilePath 2", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(dirList)))

	tn := make(map[string]interface{})
	err := json.Unmarshal([]byte(dirList), &tn)
	if err != nil {
		t.Fatalf("failed to understand the JSON. Error:%s", err.Error())
	}
	if tn["error"] != false {
		t.Fatalf("response 'error' is true")
	}
	if tn["tree"] == nil {
		t.Fatalf("response 'tree' is nil")
	}

	RunClientGet(t, "TestTree 2", configData, "favicon.ico", 200, "?", -1, 0)

}

func TestPostFileAndDelete(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	file := fmt.Sprintf("%s/%s", configData.GetUserData("stuart").Locations["picsPlus"], testdatafile)
	_, err := os.Stat(file)
	if err == nil {
		os.Remove(file)
		time.Sleep(100 * time.Millisecond)
	}
	url := fmt.Sprintf("files/user/stuart/loc/picsPlus/name/%s", testdatafile)

	RunClientPost(t, configData, url, 202, "File:Action:save", postDataFile1)
	r, resBody := RunClientGet(t, "TestPostFileAndDelete 1", configData, url, 200, "?", -1, 0)
	AssertHeader(t, "TestPostFileAndDelete 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if resBody != postDataFile1 {
		t.Fatalf("Response body does not equal postDataFile1")
	}
	// Try to save again with different content but should not overwrite so content remains the same!
	RunClientPost(t, configData, url, 412, "File exists", postDataFile2)
	r, resBody = RunClientGet(t, "TestPostFileAndDelete 2", configData, url, 200, "?", -1, 0)
	AssertHeader(t, "TestPostFileAndDelete 2", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if resBody != postDataFile1 {
		t.Fatalf("Response body does not equal postDataFile1")
	}
	_, err = os.Stat(file)
	if err != nil {
		t.Fatalf("File was not created")
	}
	// TODO
	RunClientDelete(t, configData, url, 202, "\"cause\":\"File deleted OK\"")
	_, err = os.Stat(file)
	if err == nil {
		t.Fatalf("File was not deleted")
	}
	AssertLogContains(t, logger, []string{
		fmt.Sprintf("Req:  POST:/%s", url),
		fmt.Sprintf("Req:  DELETE:/%s", url),
		fmt.Sprintf("Req:  GET:/%s", url),
		"Error: Status:412",
		"\"msg\":\"Precondition Failed\"",
		"testdata.json already exists",
	})

}

func TestPostFileOverwriteAndDelete(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	file := fmt.Sprintf("%s/%s", configData.GetUserData("stuart").Locations["picsPlus"], testdatafile)
	_, err := os.Stat(file)
	if err == nil {
		os.Remove(file)
		time.Sleep(100 * time.Millisecond)
	}
	urlA := fmt.Sprintf("files/user/stuart/loc/picsPlus/name/%s?action=replace", testdatafile)
	urlB := fmt.Sprintf("files/user/stuart/loc/picsPlus/name/%s", testdatafile)

	RunClientPost(t, configData, urlB, 202, "File:Action:save", postDataFile1)
	r, resBody := RunClientGet(t, "TestPostFileOverwriteAndDelete 1", configData, urlB, 200, "?", -1, 0)
	AssertHeader(t, "TestPostFileAndDelete 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if resBody != postDataFile1 {
		t.Fatalf("Response body does not equal postDataFile1")
	}
	// Try to save again with different content this should not overwrite so content will change!
	RunClientPost(t, configData, urlA, 202, "File:Action:replace", postDataFile2)
	r, resBody = RunClientGet(t, "TestPostFileOverwriteAndDelete 2", configData, urlB, 200, "?", -1, 0)
	AssertHeader(t, "TestPostFileAndDelete 2", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if resBody != postDataFile2 {
		t.Fatalf("Response body does not equal postDataFile2")
	}
	_, err = os.Stat(file)
	if err != nil {
		t.Fatalf("File was not created")
	}
	// TODO
	RunClientDelete(t, configData, urlB, 202, "\"cause\":\"File deleted OK\"")
	_, err = os.Stat(file)
	if err == nil {
		t.Fatalf("File was not deleted")
	}
	AssertLogContains(t, logger, []string{
		fmt.Sprintf("Req:  POST:/%s", urlA),
		fmt.Sprintf("Req:  DELETE:/%s", urlB),
		fmt.Sprintf("Req:  GET:/%s", urlB)})
}

func TestPostFileAppendAndDelete(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	file := fmt.Sprintf("%s/%s", configData.GetUserData("stuart").Locations["picsPlus"], testdatafile)
	_, err := os.Stat(file)
	if err == nil {
		os.Remove(file)
		time.Sleep(100 * time.Millisecond)
	}
	urlA := fmt.Sprintf("files/user/stuart/loc/picsPlus/name/%s?action=append", testdatafile)
	urlB := fmt.Sprintf("files/user/stuart/loc/picsPlus/name/%s", testdatafile)

	RunClientPost(t, configData, urlB, 202, "File:Action:save", postDataFile1)
	r, resBody := RunClientGet(t, "TestPostFileAppendAndDelete 1", configData, urlB, 200, "?", -1, 0)
	AssertHeader(t, "TestPostFileAppendAndDelete 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if resBody != postDataFile1 {
		t.Fatalf("Response body does not equal postDataFile1")
	}
	// Try to save again with different content this should append so content will change!
	RunClientPost(t, configData, urlA, 202, "File:Action:append", postDataFile2)
	r, resBody = RunClientGet(t, "TestPostFileAppendAndDelete 2", configData, urlB, 200, "?", -1, 0)
	AssertHeader(t, "TestPostFileAppendAndDelete 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if resBody != postDataFile1+postDataFile2 {
		t.Fatalf("Response body does not equal postDataFile2 + postDataFile2")
	}
	_, err = os.Stat(file)
	if err != nil {
		t.Fatalf("File was not created")
	}
	// TODO
	RunClientDelete(t, configData, urlB, 202, "\"cause\":\"File deleted OK\"")
	_, err = os.Stat(file)
	if err == nil {
		t.Fatalf("File was not deleted")
	}
	AssertLogContains(t, logger, []string{
		fmt.Sprintf("Req:  POST:/%s", urlA),
		fmt.Sprintf("Req:  DELETE:/%s", urlB),
		fmt.Sprintf("Req:  GET:/%s", urlB)})
}

func TestReadDir(t *testing.T) {

	logger.Reset()
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	r, resBody := RunClientGet(t, "TestReadDir 1", configData, "files/user/stuart/loc/pics", 200, "?", -1, 0)
	AssertHeader(t, "TestReadDir 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))

	AssertContains(t, resBody, []string{
		"\"error\":false,\"user\":\"stuart\",\"loc\":\"pics\",\"path\":null,\"files\"",
		"{\"name\":\"t2.Data\", \"encName\":\"X0XdDIuRGF0YQ==\"}",
		"{\"name\":\"t1.JSON\", \"encName\":\"X0XdDEuSlNPTg==\"}",
		"{\"name\":\"pic1.jpeg\", \"encName\":\"X0XcGljMS5qcGVn\"}",
	})

	r, resBody = RunClientGet(t, "TestReadDir 2", configData, "files/user/stuart/loc/picsPlus", 200, "?", -1, 0)
	AssertHeader(t, "TestReadDir 2", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))

	AssertContains(t, resBody, []string{
		"\"error\":false,\"user\":\"stuart\",\"loc\":\"picsPlus\",\"path\":null,\"files\"",
		"{\"name\":\"t5.json\", \"encName\":\"X0XdDUuanNvbg==\"}",
		"{\"name\":\"testdata2.json\", \"encName\":\"X0XdGVzdGRhdGEyLmpzb24=\"}",
	})

	AssertLogContains(t, logger, []string{"Req:  GET:/files/", "Resp: Status:200"})
	AssertLogContains(t, logger, []string{"GET:/files/user/stuart/loc/pics", "GET:/files/user/stuart/loc/picsPlus"})
}

func TestReadDirNotFound(t *testing.T) {

	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	r, resBody := RunClientGet(t, "TestReadDirNotFound 1", configData, "files/user/stuart/loc/picsMissing", http.StatusNotFound, "?", -1, 0)
	AssertHeader(t, "TestReadDirNotFound 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	AssertContains(t, resBody, []string{
		"\"error\":true",
		"\"cause\":\"Dir not found\"",
	})
	AssertLogContains(t, logger, []string{"Dir not found", "\"status\":404"})
}

func TestReadFile(t *testing.T) {
	logger.Reset()
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	r, resBody := RunClientGet(t, "TestReadFile 1", configData, "files/user/stuart/loc/pics/name/t1.JSON", http.StatusOK, "?", 251, 0)
	AssertHeader(t, "TestReadDirNotFound 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if !strings.HasPrefix(trimString(resBody), "{ \"ServerName\": \"TestServer\", \"Users\":") {
		t.Fatalf("Respons body does not start with...")
	}
	AssertLogContains(t, logger, []string{"FastFile:", "testdata/stuart/s-pics/t1.JSON"})
}

func TestReadFileWithPath(t *testing.T) {
	logger.Reset()
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	r, resBody := RunClientGet(t, "TestReadFileWithPath 1", configData, "files/user/stuart/loc/pics/path/s-testfolder/name/t5.json", http.StatusOK, "?", 251, 0)
	AssertHeader(t, "TestReadFileWithPath 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if !strings.HasPrefix(trimString(resBody), "{ \"ServerName\": \"TestServer\", \"Users\":") {
		t.Fatalf("Respons body does not start with...")
	}

	AssertLogContains(t, logger, []string{"FastFile:", "testdata/stuart/s-pics/s-testfolder/t5.json"})
}

func TestReadFileWithPath64enc(t *testing.T) {
	logger.Reset()
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	url := fmt.Sprintf("files/user/stuart/loc/pics/path/%s/name/%s", encodeValue("s-testfolder/s-testdir1"), encodeValue("testdata.json"))

	r, resBody := RunClientGet(t, "TestReadFileWithPath64enc 1", configData, url, http.StatusOK, "?", 33, 0)
	AssertHeader(t, "TestReadFileWithPath64enc 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if !strings.HasPrefix(trimString(resBody), "{\"Data\":\"This is the data for 2\"}") {
		t.Fatalf("Respons body does not start with...")
	}

	AssertLogContains(t, logger, []string{"FastFile:", "testdata/stuart/s-pics/s-testfolder/s-testdir1/testdata.json"})
}

func TestReadFileWTestOption(t *testing.T) {
	logger.Reset()
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	url := fmt.Sprintf("test/user/stuart/loc/pics/name/%s", encodeValue("t1.JSON"))
	r, resBody := RunClientGet(t, "TestReadFileWTestOption 1", configData, url, http.StatusOK, "?", 251, 0)
	AssertHeader(t, "TestReadFileWTestOption 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	AssertContains(t, resBody, []string{"\"pics\": \"testdata\"", "\"ServerName\": \"TestServer\""})
	AssertLogContains(t, logger, []string{"VERBOSE: Req:", "name/X0XdDEuSlNPTg=="})
	AssertLogNotContains(t, logger, []string{"LOG: FastFile:"})
}

func TestReadFileNotUser(t *testing.T) {

	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	logger.Reset()
	r, resBody := RunClientGet(t, "TestReadFileNotUser 1", configData, "files/user/nouser/loc/pics/name/t1.JSON", http.StatusNotFound, "?", 70, 0)
	AssertHeader(t, "TestReadFileNotUser 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	AssertContains(t, string(resBody), []string{"\"error\":true", "\"cause\":\"Get File Error\""})
	AssertLogContains(t, logger, []string{"Invalid user:", "Get File Error", "/files/user/nouser/loc/pics/name/t1.JSON"})
}

func TestReadFileNotLoc(t *testing.T) {

	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	r, resBody := RunClientGet(t, "TestReadFileNotLoc 1", configData, "files/user/stuart/loc/noloc/name/t1.JSON", http.StatusNotFound, "?", 70, 0)
	AssertHeader(t, "TestReadFileNotLoc 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	AssertContains(t, string(resBody), []string{"\"error\":true", "\"cause\":\"Get File Error\""})
	AssertLogContains(t, logger, []string{"Invalid location:", "Get File Error", "/files/user/stuart/loc/noloc/name/t1.JSON"})
}

func TestReadFileNotName(t *testing.T) {

	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	r, resBody := RunClientGet(t, "TestReadFileNotName 1", configData, "files/user/stuart/loc/pics/name/notExist", http.StatusNotFound, "?", 70, 0)
	AssertHeader(t, "TestReadFileNotName 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	AssertContains(t, string(resBody), []string{"\"error\":true", "\"cause\":\"File not found\""})
	AssertLogContains(t, logger, []string{"\"error\":true", "\"status\":404", "\"cause\":\"File not found\""})
}

func TestReadFileIsDir(t *testing.T) {

	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	r, resBody := RunClientGet(t, "TestReadFileIsDir 1", configData, "files/user/stuart/loc/pics/name/s-testfolder", http.StatusBadRequest, "?", 72, 1)
	AssertHeader(t, "TestReadFileIsDir 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	AssertContains(t, string(resBody), []string{"\"error\":true", "\"cause\":\"Is a Directory\""})
	AssertLogContains(t, logger, []string{
		"/stuart/s-pics/s-testfolder is a Directory",
		"Resp: Error: Status:400",
		"\"error\":true", "\"status\":400",
		"\"cause\":\"Is a Directory\""})
}

func TestServerTime(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	r, resBody := RunClientGet(t, "TestServerTime 1", configData, "server/time", 200, "?", 173, 20)
	AssertHeader(t, "TestServerTime 1", r, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if !strings.HasPrefix(trimString(resBody), "{\"time\":{") {
		t.Fatalf("Respons body does not start with...")
	}

	if strings.Contains(logger.Get(), "LOG: Req:  GET:/server/time") {
		os.Stderr.WriteString(logger.Get())
		t.Fatal("Log must NOT contain the time request response")
	}
}

func TestServerLog(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	// Make sure there is a log file to get!
	resp, resBody := RunClientGet(t, "TestServerLog 0", configData, "server/status", 200, "?", -1, 0)
	AssertHeader(t, "TestServerLog 0", resp, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	WriteLogToFile(configData.GetLogDataPath())
	time.Sleep(100 * time.Millisecond)

	resp, resBody = RunClientGet(t, "TestServerLog 1", configData, "server/log", 200, "?", -1, 0)
	AssertHeader(t, "TestServerLog 1", resp, []string{"text/plain", "charset=utf-8"}, "")

	resp, resBody0 := RunClientGet(t, "TestServerLog 2", configData, "server/log?offset=0", 200, "?", -1, 0)
	AssertHeader(t, "TestServerLog 2", resp, []string{"text/plain", "charset=utf-8"}, "")
	if resBody != resBody0 {
		t.Fatalf("Respons body default offset !=  Respons body offset=0")
	}
	resp, s := RunClientGet(t, "TestServerLog 3", configData, "server/log?offset=A", 200, "?", -1, 0)
	AssertHeader(t, "TestServerLog 3", resp, []string{"text/plain", "charset=utf-8"}, "")
	AssertContains(t, s, []string{"##I Index:A is not an integer. Index set to 0"})

	RunClientDelete(t, configData, "server/log/fred", http.StatusNotFound, "File not found")
	RunClientDelete(t, configData, "server/log/..", http.StatusForbidden, "Is a directory")
	RunClientDelete(t, configData, "server/log/DummyLogger.log", http.StatusForbidden, "Cannot remove current log")
	RunClientDelete(t, configData, "server/log/TLog.log", http.StatusAccepted, "Log file 'TLog.log' deleted OK")
	RunClientDelete(t, configData, "server/log/TLog.log", http.StatusNotFound, "File not found")

}

func TestServerIsUp(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	resp, resBody := RunClientGet(t, "TestServerIsUp 1", configData, "isup", 200, "{\"error\":false,\"msg\":\"OK\",\"cause\":\"ServerIsUp\",\"status\":200}", 60, 2)
	AssertHeader(t, "TestServerIsUp 1", resp, []string{"application/json", "charset=utf-8"}, strconv.Itoa(len(resBody)))
	if !strings.Contains(logger.Get(), "VERBOSE: Req:  GET:/isup") {
		os.Stderr.WriteString(logger.Get())
		t.Fatal("Log should contain 'VERBOSE: Req:  GET:/isup'")
	}
	if strings.Contains(logger.Get(), "LOG: Req:  GET:/isup") {
		os.Stderr.WriteString(logger.Get())
		t.Fatal("Log should NOT contain 'LOG: Req:  GET:/isup'")
	}
}

func TestClient(t *testing.T) {
	configData := loadConfigData(t, testConfigFile)
	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	res, _ := RunClientGet(t, "TestClient 1", configData, "ABC/123", 404, "{\"error\":true, \"status\":404, \"msg\":\"Not Found\", \"cause\":\"Resource not found\"}", 74, 0)
	AssertHeader(t, "TestClient 1", res, []string{config.DefaultContentType, "charset=utf-8"}, "")
	res, _ = RunClientGet(t, "TestClient 1", configData, "ABC", 404, "{\"error\":true, \"status\":404, \"msg\":\"Not Found\", \"cause\":\"File not found\"}", 70, 0)
	AssertHeader(t, "TestClient 1", res, []string{config.DefaultContentType, "charset=utf-8"}, "")
	res, _ = RunClientGet(t, "TestClient 2", configData, "ping", 200, "{\"error\":false, \"status\":200, \"msg\":\"OK\", \"cause\":\"Ping\"}", 54, 2)
	AssertHeader(t, "TestClient 2", res, []string{config.DefaultContentType, "charset=utf-8"}, "")
	res, _ = RunClientGet(t, "TestClient 3", configData, "server/exit", http.StatusAccepted, "{\"error\":false, \"status\":202, \"msg\":\"Accepted\", \"cause\":\"[11] Exit Requested\"}", 75, 2)
	AssertHeader(t, "TestClient 3", res, []string{config.DefaultContentType, "charset=utf-8"}, "")
	AssertLogContains(t, logger, []string{"Server Error: Status:404. Msg:File not found", "/testdata/static/ABC", "GET:/ping", "GET:/server/exit", "Server Shutdown Clean"})
}

// /////////////////////////////////////////////////////////////////////////////
func RunClientPost(t *testing.T, config *config.ConfigData, path string, expectedStatus int, expectedBody string, data string) (*http.Response, string) {
	requestURL := fmt.Sprintf("http://localhost%s/%s", config.GetPortString(), path)
	myReader := strings.NewReader(data)
	res, err := http.Post(requestURL, "application/json", myReader)
	if err != nil {
		t.Fatalf("Client Post error: %s", err.Error())
	}
	if res.StatusCode != expectedStatus {
		t.Fatalf("Status for path http://localhost%s/%s. Expected %d Actual %d", config.GetPortString(), path, expectedStatus, res.StatusCode)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Client Post: could not read response body: %s\n", err)
	}
	if !strings.Contains(string(resBody), expectedBody) {
		t.Fatalf("Client Post: Dose not contain expected\nBody    %s:\nExpected: %s", string(resBody), expectedBody)
	}
	return res, ""
}

func RunClientDelete(t *testing.T, config *config.ConfigData, path string, expectedStatus int, data string) (*http.Response, string) {
	myReader := strings.NewReader(data)
	requestURL := fmt.Sprintf("http://localhost%s/%s", config.GetPortString(), path)
	req, err := http.NewRequest("DELETE", requestURL, myReader)
	if err != nil {
		t.Fatalf("Client DELETE request: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Client Post error: %s", err.Error())
	}
	if res.StatusCode != expectedStatus {
		t.Fatalf("Status for path http://localhost%s/%s. Expected %d Actual %d", config.GetPortString(), path, expectedStatus, res.StatusCode)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("client: could not read response body: %s\n", err)
	}
	if !strings.Contains(string(resBody), data) {
		t.Fatalf("\nResponse:      %s\ndoesNotContain:%s\n", resBody, data)
	}
	return res, string(resBody)
}

func RunClientGet(t *testing.T, id string, config *config.ConfigData, path string, expectedStatus int, expectedBody string, expectedLen int, plusMinus int) (*http.Response, string) {
	requestURL := fmt.Sprintf("http://localhost%s/%s", config.GetPortString(), path)
	res, err := http.Get(requestURL)
	if err != nil {
		t.Fatalf("RunClientGet:id:%s. Client error: %s", id, err.Error())
	}
	if res.StatusCode != expectedStatus {
		t.Fatalf("RunClientGet:id:%s. Status for path http://localhost%s/%s. Expected %d Actual %d", id, config.GetPortString(), path, expectedStatus, res.StatusCode)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("RunClientGet:id:%s. client: could not read response body: %s\n", id, err)
	}
	expectedList := strings.Split(expectedBody, "|")
	if len(expectedList) > 1 {
		for _, ex := range expectedList {
			if !strings.Contains(string(resBody), ex) {
				t.Fatalf("RunClientGet:id:%s. Body \n%s\ndoes not contain '%s'", id, string(resBody), ex)
			}
		}
	} else {
		if expectedBody != "?" {
			AssertEquivilent(t, fmt.Sprintf("RunClientGet:id[%s]", id), string(resBody), expectedBody)
		}
	}

	if expectedLen >= 0 {
		minLen := expectedLen - plusMinus
		maxLen := expectedLen + plusMinus
		len, err := strconv.Atoi(res.Header["Content-Length"][0])
		if err != nil {
			t.Fatalf("%s: Status for path http://localhost%s/%s.\nExpectedMin '%d'.\nExpectedMax '%d' Content-Length conversion error:'%s'", id, config.GetPortString(), path, minLen, maxLen, err)
		}
		if len < minLen || len > maxLen {
			t.Fatalf("%s: Status for path http://localhost%s/%s.\nExpectedMin '%d'.\nExpectedMax '%d' \nActual   '%d'\n%s", id, config.GetPortString(), path, minLen, maxLen, len, resBody)
		}
	}
	return res, string(resBody)
}

func StopServer(t *testing.T, config *config.ConfigData) {
	if serverState == "Running" {
		path := "server/exit"
		requestURL := fmt.Sprintf("http://localhost%s/%s", config.GetPortString(), path)
		res, err := http.Get(requestURL)
		if err != nil {
			t.Fatalf("StopServer: Client error: %s", err.Error())
		}
		if res.StatusCode != 202 {
			t.Fatalf("StopServer: Status for path http://localhost%s/%s. Expected %d Actual %d", config.GetPortString(), path, 202, res.StatusCode)
		}
		s, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("StopServer: client: could not read response body: %s\n", err)
		}
		AssertContains(t, string(s), []string{"Exit Requested", "error\":false", "msg\":\"Accepted", "status\":20"})
		time.Sleep(500 * time.Millisecond)
	}
	if serverState == "Running" {
		t.Fatal("StopServer: Server was not stopped")
	}
}

func RunServer(config *config.ConfigData, logger logging.Logger) {
	actionQueue := make(chan *ActionEvent, 10)
	defer close(actionQueue)

	var lrm *runCommand.LongRunningManager
	var err error
	if config.GetExecPath() != "" {
		lrm, err = runCommand.NewLongRunningManager(config.GetExecPath(), logger.Log)
		if err != nil {
			panic(fmt.Sprintf("LongRunningManager: failed to initialise. '%s'. ABORTED", err.Error()))
		}
	}
	webAppServer, err := NewWebAppServer(config, actionQueue, lrm, logger)
	if err != nil {
		panic(fmt.Sprintf("NewWebAppServer: failed to initialise. '%s'. ABORTED", err.Error()))
	}
	go func() {
		for {
			ae := <-actionQueue
			if ae != nil {
				switch ae.Id {
				case Exit:
					serverState = "Stopped"
					webAppServer.Close(ae.Rc)
				case Ignore:
					fmt.Printf("Server: Ignore\n")
				}
			}
		}
	}()

	serverState = "Running"
	rc := webAppServer.Start()
	fmt.Printf("Returned from Server. RC:%d. State:%s\n", rc, serverState)

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

func AssertLogContains(t *testing.T, log *TLog, list []string) {
	l := log.Get()
	for _, x := range list {
		if !strings.Contains(l, x) {
			t.Fatalf("Log does NOT contain '%s'.\n%s", x, log.Get())
		}
	}
}

func AssertLogNotContains(t *testing.T, log *TLog, list []string) {
	l := log.Get()
	for _, x := range list {
		if strings.Contains(l, x) {
			t.Fatalf("Log MUST NOT contain '%s'.\n%s", x, log.Get())
		}
	}
}

func AssertContains(t *testing.T, actual string, expectedList []string) {
	for i := 0; i < len(expectedList); i++ {
		expected := expectedList[i]
		if !strings.Contains(actual, expected) {
			t.Fatalf("Value \n%s\nDoes NOT contain '%s'", actual, expected)
		}
	}
}

func AssertHeader(t *testing.T, id string, resp *http.Response, ct []string, cl string) {
	for _, v := range ct {
		AssertHeaderContains(t, id, resp, "Content-Type", v)
	}
	if cl != "" {
		AssertHeaderEqual(t, id, resp, "Content-Length", cl)
	}
	AssertHeaderContains(t, id, resp, "Date", "GMT")
	AssertHeaderEqual(t, id, resp, "Server", "TestServer")
}

func AssertHeaderEqual(t *testing.T, id string, r *http.Response, name, value string) {
	v := r.Header[name]
	if v == nil {
		t.Fatalf("ID[%s] Header '%s' was not found ", id, name)
	}
	if len(v) == 0 {
		t.Fatalf("ID[%s] Header '%s' is empty ", id, name)
	}
	if r.Header[name][0] != value {
		t.Fatalf("ID[%s] Header %s should equal '%s'. Header Was: '%s'", id, name, value, r.Header[name][0])
	}
}

func AssertHeaderContains(t *testing.T, id string, r *http.Response, name, value string) {
	if !strings.Contains(r.Header[name][0], value) {
		t.Fatalf("ID[%s] Header %s should contain '%s'. Header Was: '%s'", id, name, value, r.Header[name][0])
	}
}

func AssertEquivilent(t *testing.T, id string, actualJson string, expectedJson string) {
	act := make(map[string]interface{}, 0)
	err := json.Unmarshal([]byte(actualJson), &act)
	if err != nil {
		t.Fatalf("AssertEquivilent:%s. Actual Is not valid JSON\n%s\nError:'%s'", id, actualJson, err.Error())
	}
	exp := make(map[string]interface{}, 0)
	err = json.Unmarshal([]byte(expectedJson), &exp)
	if err != nil {
		t.Fatalf("AssertEquivilent:%s. Expected Is not valid JSON\n%s\nError:'%s'", id, expectedJson, err.Error())
	}

	for n, v1 := range exp {
		v2 := act[n]
		if fmt.Sprintf("%v", v1) != fmt.Sprintf("%v", v2) {
			t.Fatalf("AssertEquivilent:%s. Value \n%s\nIs not a subset of\n%s", id, actualJson, expectedJson)
		}
	}
}

func trimString(res string) string {
	var buffer bytes.Buffer
	spaceCount := 0
	for i := 0; i < len(res); i++ {
		c := res[i]
		if c >= 32 {
			if c == 32 {
				spaceCount++
			} else {
				spaceCount = 0
			}
			if spaceCount <= 1 {
				buffer.WriteByte(c)
			}
		}
	}
	return strings.Trim(buffer.String(), " ")
}

func AssertMatch(t *testing.T, message string, matcher *urlRequestMatcher, url string, reqType string, match bool, params string) {
	requestUriparts := strings.Split(strings.TrimSpace(url), "/")
	if requestUriparts[0] == "" {
		requestUriparts = requestUriparts[1:]
	}
	p, ok, _ := matcher.Match(requestUriparts, reqType, nil)
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	// Sort keys
	sort.Strings(keys)
	// Print sorted map
	var buffer bytes.Buffer
	for i, k := range keys {
		buffer.WriteString(k)
		buffer.WriteRune('=')
		buffer.WriteString(p[k])
		if i < len(keys)-1 {
			buffer.WriteRune(',')
		}
	}

	if ok != match {
		if match {
			t.Fatalf("%s.\nExpected to match %s:%s. Actual %s, Params %s", message, reqType, url, matcher, buffer.String())
		} else {
			t.Fatalf("%s.\nExpected to NOT match %s:%s. Actual %s, Params %s", message, reqType, url, matcher, buffer.String())
		}
	}
	if buffer.String() != params {
		t.Fatalf("%s.\nExpected Params %s. Actual Params %s", message, params, buffer.String())
	}
}

func loadConfigData(t *testing.T, file string) *config.ConfigData {
	errList := config.NewConfigErrorData()
	configData := config.NewConfigData(file, "goWebApp", false, false, false, errList)
	if errList.ErrorCount() > 1 {
		t.Fatal(errList.String())
	}
	if configData == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errList.String())
	}
	return configData
}

func UpdateConfigAndLoad(t *testing.T, callBack func(*config.ConfigDataFromFile), errList *config.ConfigErrorData, load bool) (*config.ConfigData, string) {
	content, err := os.ReadFile(configRef)
	if err != nil {
		t.Fatalf("Failed to read config data file:%s. Error:%s", configRef, err.Error())
	}
	configDataFromFile := &config.ConfigDataFromFile{
		ThumbnailTrim: []int{thumbnailTrimPrefix, thumbnailTrimSuffix},
	}
	err = json.Unmarshal(content, &configDataFromFile)
	if err != nil {
		t.Fatalf("Failed to understand the config data in the file:%s. Error:%s", configRef, err.Error())
	}
	callBack(configDataFromFile)
	cf2, err := json.MarshalIndent(configDataFromFile, "", "  ")
	if err != nil {
		t.Fatalf("Failed to martial new json. Error:%s", err.Error())
	}
	err = os.WriteFile(configTmp, cf2, os.ModePerm)
	if err != nil {
		t.Fatalf("Failed to write new json file %s. Error:%s", configTmp, err.Error())
	}
	if load {
		defer os.Remove(configTmp)

		maxErr := 9
		if errList == nil {
			maxErr = 1
			errList = config.NewConfigErrorData()
		}
		configData := config.NewConfigData(configTmp, "goWebApp", false, false, false, errList)
		if errList.ErrorCount() > maxErr || configData == nil {
			t.Fatal(errList.String())
		}
		return configData, configTmp
	}
	return nil, configTmp
}

func encodeValue(unEncoded string) string {
	if unEncoded == "" {
		return ""
	}
	return controllers.EncodedValuePrefix + base64.StdEncoding.EncodeToString([]byte(unEncoded))
}
