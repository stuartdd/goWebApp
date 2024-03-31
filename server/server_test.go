package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"stuartdd.com/config"
	"stuartdd.com/logging"
)

type TLog struct {
	B bytes.Buffer
}

func (l *TLog) Close() {}
func (l *TLog) Log(s string) {
	l.B.WriteString(s)
	l.B.WriteString("\n")
}
func (l *TLog) Get() string {
	return l.B.String()
}
func (l *TLog) IsOpen() bool {
	return true
}

var serverState string = ""
var logger = &TLog{}

const postDataFile1 = "{\"Data\":\"This is the data for 1\"}"
const postDataFile2 = "{\"Data\":\"This is the data for 2\"}"
const testdatapath = "../testdata/testfolder"
const testdatafile = "testdata.json"

func TestUrlRequestParamsMap(t *testing.T) {
	AssertMatch(t, "0", NewUrlRequestMatcher("/a/b/*/c/*", "get"), "/x/b/1/c/4", "GET", false, "")
	AssertMatch(t, "1", NewUrlRequestMatcher("/a/b/*/c/*", "get"), "/a/b/1/x/4", "GET", false, "b=1")
	AssertMatch(t, "2", NewUrlRequestMatcher("/a/b/*/c/*", "get"), "/a/b/1/c", "GET", false, "")
	AssertMatch(t, "3", NewUrlRequestMatcher("/a/b/*/c/*", "get"), "/a/b/1/c/3", "GET", true, "b=1,c=3")
	AssertMatch(t, "4", NewUrlRequestMatcher("a", "get"), "/a", "get", false, "")
	AssertMatch(t, "5", NewUrlRequestMatcher("a", "get"), "a", "get", true, "")
	AssertMatch(t, "5", NewUrlRequestMatcher("/a", "get"), "/a", "get", true, "")
	AssertMatch(t, "6", NewUrlRequestMatcher("/a/b/*/*/c/*", "get"), "/a/b/1/2/c/3", "post", false, "")
	AssertMatch(t, "7", NewUrlRequestMatcher("/a/b/*/*/c/*", "get"), "/a/b/1/2/C/3", "GET", false, "b=1")
	AssertMatch(t, "8", NewUrlRequestMatcher("/a/b/*/*/c/*", "get"), "/a/b/1/2/c/3", "get", true, "b=1,c=3")
	AssertMatch(t, "9", NewUrlRequestMatcher("/a/b/*/*/c/*", "get"), "/a/b/1/2/c/3", "GET", true, "b=1,c=3")
	AssertMatch(t, "10", NewUrlRequestMatcher("/a/*/b/*/c/*", "get"), "/a/1/b/2/c/3", "GET", true, "a=1,b=2,c=3")
	AssertMatch(t, "10", NewUrlRequestMatcher("", "get"), "/a/1/b/2/c/3", "GET", false, "")
	AssertMatch(t, "11", NewUrlRequestMatcher("", "get"), "", "GET", true, "")
	AssertMatch(t, "12", NewUrlRequestMatcher("", "post"), "", "GET", false, "")
}
func TestServer(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 || configData == nil {
		t.Fatal(errList.String())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "files/user/stuart/loc/data/name/state.json"
	_, respBody := RunClientGet(t, configData, url, 200, "?", 111, 10)
	AssertContains(t, respBody, []string{
		"\"imagesPerRow\"",
		"\"displayOptions\"",
		"\"optionSuppressTime\"",
	})

	url = "server/users"
	_, respBody = RunClientGet(t, configData, url, 200, "?", 69, 10)
	AssertContains(t, respBody, []string{
		"\"users\"",
		"\"Bob\"",
		"\"Stuart\"",
	})

	url = "server/time"
	_, respBody = RunClientGet(t, configData, url, 200, "?", -1, 0)
	AssertContains(t, respBody, []string{
		"\"time\"",
		"\"millis\"",
		"\"timestamp\"",
	})

}
func TestStatic(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 || configData == nil {
		t.Fatal(errList.String())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "static/images/pic.jpg"
	resp, _ := RunClientGet(t, configData, url, 200, "?", 2221, 10)
	if resp.Header["Content-Type"][0] != "image/jpeg" {
		t.Fatalf("incorrect content type :%s", resp.Header["Content-Type"][0])
	}

	url = "static/images/favicon.ico"
	resp, _ = RunClientGet(t, configData, url, 200, "?", 177174, 10)
	if resp.Header["Content-Type"][0] != "image/vnd.microsoft.icon" {
		t.Fatalf("incorrect content type :%s", resp.Header["Content-Type"][0])
	}

	url = "static/simple.html"
	resp, _ = RunClientGet(t, configData, url, 200, "?", 103, 10)
	if resp.Header["Content-Type"][0] != "text/html; charset=utf-8" {
		t.Fatalf("incorrect content type :%s", resp.Header["Content-Type"][0])
	}

	url = "static/notfound.pic"
	RunClientGet(t, configData, url, 404, "?", -1, 0)
	url = "static"
	RunClientGet(t, configData, url, 404, "?", -1, 0)

}
func TestFilePath(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 || configData == nil {
		t.Fatal(errList.String())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "paths/user/stuart/loc/home"
	RunClientGet(t, configData, url, 200, "?", -1, 0)

	// url = "files/user/stuart/loc/home/path/" + controllers.encodePath("s-pics")
	// RunClientGet(t, configData, url, 200, "?", -1, 0)

	url = "files/user/stuart/loc/home"
	RunClientGet(t, configData, url, 200, "\"path\":null|\"error\":false", -1, 0)

}

func TestTree(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 || configData == nil {
		t.Fatal(errList.String())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "files/user/stuart/loc/testtree/tree"

	_, dirList := RunClientGet(t, configData, url, 200, "?", -1, 0)

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

	RunClientGet(t, configData, "favicon.ico", 200, "?", -1, 0)

}
func TestGetFavicon(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 || configData == nil {
		t.Fatal(errList.String())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	resp, _ := RunClientGet(t, configData, "favicon.ico", 200, "?", -1, 0)
	if resp.StatusCode != 200 {
		t.Fatalf("did not get the icon!")
	}
	if resp.Header["Content-Type"][0] != "image/vnd.microsoft.icon" {
		t.Fatalf("incorrect content type :%s", resp.Header["Content-Type"][0])
	}

}

func TestPostFile(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 || configData == nil {
		t.Fatal(errList.String())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	file := fmt.Sprintf("%s/%s", testdatapath, testdatafile)
	_, err := os.Stat(file)
	if err == nil {
		os.Remove(file)
		time.Sleep(100 * time.Millisecond)
	}

	url := fmt.Sprintf("files/user/stuart/loc/picsPlus/name/%s", testdatafile)

	RunClientPost(t, configData, url, 202, postDataFile1)
	_, resBody := RunClientGet(t, configData, url, 200, "?", -1, 0)
	if resBody != postDataFile1 {
		t.Fatalf("Respons body does not equal postDataFile1")
	}
	RunClientPost(t, configData, url, 202, postDataFile2)
	_, resBody = RunClientGet(t, configData, url, 200, "?", -1, 0)
	if resBody != postDataFile2 {
		t.Fatalf("Respons body does not equal postDataFile2")
	}

	os.Remove(file)

}
func TestReadDir(t *testing.T) {

	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 || configData == nil {
		t.Fatal(errList.String())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	_, resBody := RunClientGet(t, configData, "files/user/stuart/loc/pics", 200, "?", -1, 0)
	//
	//"{\"error\":false,\"user\":\"stuart\",\"loc\":\"pics\",\"path\":null,\"files\":[
	//	{\"size\": 0,\"name\":{\"name\":\"pic1.jpeg\", \"encName\":\"X0XcGljMS5qcGVn\"}},
	//  {\"size\": 0,\"name\":{\"name\":\"t1.JSON\", \"encName\":\"X0XdDEuSlNPTg==\"}},
	//  {\"size\": 0,\"name\":{\"name\":\"t2.Data\", \"encName\":\"X0XdDIuRGF0YQ==\"}}]}"
	//
	AssertContains(t, resBody, []string{
		"\"error\":false,\"user\":\"stuart\",\"loc\":\"pics\",\"path\":null,\"files\"",
		"{\"name\":\"t2.Data\", \"encName\":\"X0XdDIuRGF0YQ==\"}",
		"{\"name\":\"t1.JSON\", \"encName\":\"X0XdDEuSlNPTg==\"}",
		"{\"name\":\"pic1.jpeg\", \"encName\":\"X0XcGljMS5qcGVn\"}",
	})

	_, resBody = RunClientGet(t, configData, "files/user/stuart/loc/picsPlus", 200, "?", -1, 0)
	//
	//"{\"error\":false,\"user\":\"stuart\",\"loc\":\"picsPlus\",\"path\":null,\"files\":[{\"size\": 0,\"name\":{\"name\":\"t5.json\", \"encName\":\"X0XdDUuanNvbg==\"}},{\"size\": 0,\"name\":{\"name\":\"testdata.json\", \"encName\":\"X0XdGVzdGRhdGEuanNvbg==\"}}]}"
	//
	AssertContains(t, resBody, []string{
		"\"error\":false,\"user\":\"stuart\",\"loc\":\"picsPlus\",\"path\":null,\"files\"",
		"{\"name\":\"t5.json\", \"encName\":\"X0XdDUuanNvbg==\"}",
		"{\"name\":\"testdata.json\", \"encName\":\"X0XdGVzdGRhdGEuanNvbg==\"}",
	})

	_, resBody = RunClientGet(t, configData, "files/user/stuart/loc/picsMissing", 404, "?", -1, 0)
	if resBody != "{\"error\":true, \"status\":404, \"msg\":\"Not Found\", \"reason\":\"Dir not found\"}" {
		t.Fatalf("Respons body does not equal..3")
	}
	AssertLogContains(t, logger, []string{"Server running", ":8083.", "Req:  /files/", "Resp: Status:200"})
	os.Stderr.WriteString(logger.Get())
}

func TestReadFile(t *testing.T) {

	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 || configData == nil {
		t.Fatal(errList.String())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "files/user/stuart/loc/pics/name/t1.JSON", 200, "?", 251, 0)
	if !strings.HasPrefix(trimString(resBody), "{ \"ServerName\": \"TestServer\", \"Users\":") {
		t.Fatalf("Respons body does not start with...")
	}

	_, resBody = RunClientGet(t, configData, "files/user/stuart/loc/pics/name/s-testfolder", 404, "?", 73, 0)
	if !strings.Contains(trimString(resBody), "Is not a file") {
		t.Fatalf("Respons body does not contain 'Is not a file'")
	}

	AssertLogContains(t, logger, []string{"Server running", ":8083.", "Req:  /files/", "Resp: Status:200"})
	os.Stderr.WriteString(logger.Get())
}

func TestClient(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 || configData == nil {
		t.Fatal(errList.String())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	res, _ := RunClientGet(t, configData, "ABC", 404, "{\"error\":true, \"status\":404, \"msg\":\"Not Found\", \"reason\":\"Resource not found\"}", 78, 0)
	AssertHeaderEquals(t, res, "Content-Type", fmt.Sprintf("%s; charset=%s", config.DefaultContentType, configData.GetContentTypeCharset()))
	AssertHeaderEquals(t, res, "Server", configData.GetServerName())
	RunClientGet(t, configData, "ping", 200, "{\"error\":false, \"status\":200, \"msg\":\"OK\", \"reason\":\"Ping\"}", 58, 0)
	RunClientGet(t, configData, "exit", http.StatusAccepted, "{\"error\":false, \"status\":202, \"msg\":\"Accepted\", \"reason\":\"Server Stopped\"}", 74, 0)
	AssertLogContains(t, logger, []string{"Req:  /ABC", "Error: Status:404"})
	os.Stderr.WriteString(logger.Get())
}

// /////////////////////////////////////////////////////////////////////////////
func RunClientPost(t *testing.T, config *config.ConfigData, path string, expectedStatus int, data string) (*http.Response, string) {
	requestURL := fmt.Sprintf("http://localhost%s/%s", config.GetPortString(), path)
	myReader := strings.NewReader(data)
	res, err := http.Post(requestURL, "application/json", myReader)
	if err != nil {
		t.Fatalf("Client Post error: %s", err.Error())
	}
	if res.StatusCode != expectedStatus {
		t.Fatalf("Status for path http://localhost%s/%s. Expected %d Actual %d", config.GetPortString(), path, expectedStatus, res.StatusCode)
	}
	return res, ""
}

func RunClientGet(t *testing.T, config *config.ConfigData, path string, expectedStatus int, expectedBody string, expectedLen int, plusMinus int) (*http.Response, string) {
	requestURL := fmt.Sprintf("http://localhost%s/%s", config.GetPortString(), path)
	res, err := http.Get(requestURL)
	if err != nil {
		t.Fatalf("Client error: %s", err.Error())
	}
	if res.StatusCode != expectedStatus {
		t.Fatalf("Status for path http://localhost%s/%s. Expected %d Actual %d", config.GetPortString(), path, expectedStatus, res.StatusCode)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("client: could not read response body: %s\n", err)
	}
	expectedList := strings.Split(expectedBody, "|")
	if len(expectedList) > 1 {
		for _, ex := range expectedList {
			if !strings.Contains(string(resBody), ex) {
				t.Fatalf("Body \n%s\ndoes not contain '%s'", string(resBody), ex)
			}
		}
	} else {
		if expectedBody != "?" {
			if trimString(string(resBody)) != trimString(expectedBody) {
				t.Fatalf("Status for path http://localhost%s/%s.\nExpected '%s' \nActual   '%s'", config.GetPortString(), path, expectedBody, string(resBody))
			}
		}
	}

	if expectedLen >= 0 {
		minLen := expectedLen - plusMinus
		maxLen := expectedLen + plusMinus
		len, err := strconv.Atoi(res.Header["Content-Length"][0])
		if err != nil {
			t.Fatalf("Status for path http://localhost%s/%s.\nExpectedMin '%d'.\nExpectedMax '%d' Content-Length conversion error:'%s'", config.GetPortString(), path, minLen, maxLen, err)

		}
		if len < minLen || len > maxLen {
			t.Fatalf("Status for path http://localhost%s/%s.\nExpectedMin '%d'.\nExpectedMax '%d' \nActual   '%d'", config.GetPortString(), path, minLen, maxLen, len)
		}
	}
	return res, string(resBody)
}

func RunServer(config *config.ConfigData, logger logging.Logger) {
	actionQueue := make(chan ActionId, 10)
	defer close(actionQueue)
	go func() {
		for {
			acId := <-actionQueue
			switch acId {
			case Exit:
				fmt.Printf("Server: Stopped\n")
			case Ignore:
				fmt.Printf("Server: Ignore\n")
			}
		}
	}()

	server := NewWebAppServer(config, actionQueue, logger)
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

func AssertLogContains(t *testing.T, log *TLog, list []string) {
	l := log.Get()
	for i := 0; i < len(list); i++ {
		x := list[i]
		if !strings.Contains(l, x) {
			t.Fatalf("Log does NOT contain '%s'.\n%s", x, log.Get())
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

func AssertMatch(t *testing.T, message string, matcher *UrlRequestMatcher, url string, reqType string, match bool, params string) {
	requestUriparts := strings.Split(strings.TrimSpace(url), "/")
	if requestUriparts[0] == "" {
		requestUriparts = requestUriparts[1:]
	}
	isAbsolutePath := strings.HasPrefix(url, "/")

	p, ok := matcher.Match(requestUriparts, isAbsolutePath, reqType)
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
