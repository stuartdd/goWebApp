package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"stuartdd.com/config"
	"stuartdd.com/tools"
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

func TestFilePath(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 {
		t.Fatal(errList.ToString())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "paths/user/stuart/loc/home"
	RunClientGet(t, configData, url, 200, "?", -1)

	url = "files/user/stuart/loc/home/path/cy1waWNz"
	RunClientGet(t, configData, url, 200, "?", -1)

	url = "files/user/stuart/loc/home"
	RunClientGet(t, configData, url, 200, "\"path\":null|\"error\":false", -1)

}

func TestTree(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 {
		t.Fatal(errList.ToString())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "files/user/stuart/loc/testtree/tree"

	_, dirList := RunClientGet(t, configData, url, 200, "?", -1)

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

	RunClientGet(t, configData, "favicon.ico", 200, "?", -1)

}
func TestGetFavicon(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 {
		t.Fatal(errList.ToString())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	resp, _ := RunClientGet(t, configData, "favicon.ico", 200, "?", -1)
	if resp.StatusCode != 200 {
		t.Fatalf("did not get the icon!")
	}
	if resp.Header["Content-Type"][0] != "image/vnd.microsoft.icon" {
		t.Fatalf("incorrect content type :%s", resp.Header["Content-Type"][0])
	}

}

func TestPostFile(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 {
		t.Fatal(errList.ToString())
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
	_, resBody := RunClientGet(t, configData, url, 200, "?", -1)
	if resBody != postDataFile1 {
		t.Fatalf("Respons body does not equal postDataFile1")
	}
	RunClientPost(t, configData, url, 202, postDataFile2)
	_, resBody = RunClientGet(t, configData, url, 200, "?", -1)
	if resBody != postDataFile2 {
		t.Fatalf("Respons body does not equal postDataFile2")
	}

	os.Remove(file)

}
func TestReadDir(t *testing.T) {

	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 {
		t.Fatal(errList.ToString())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	_, resBody := RunClientGet(t, configData, "files/user/stuart/loc/pics", 200, "?", -1)
	if resBody != "{\"error\":false, \"dDEuSlNPTg==\":\"t1.JSON\",\"dDIuRGF0YQ==\":\"t2.Data\"}" {
		t.Fatalf("Respons body does not equal..1")
	}

	_, resBody = RunClientGet(t, configData, "files/user/stuart/loc/picsPlus", 200, "?", -1)
	if !strings.Contains(resBody, "\"dDUuanNvbg==\":\"t5.json\"") {
		t.Fatalf("Respons body does contain \"fidDUuanNvbg==le\":\"t5.json\"")
	}
	if !strings.Contains(resBody, ":\"testdata.json\"") {
		t.Fatalf("Respons body does contain :\"testdata.json\"")
	}
	if !strings.Contains(resBody, "\"error\":false") {
		t.Fatalf("Respons body does contain \"error\":false")
	}

	_, resBody = RunClientGet(t, configData, "files/user/stuart/loc/picsMissing", 404, "?", -1)
	if resBody != "{\"error\":true, \"status\":404, \"msg\":\"Not Found\", \"reason\":\"Dir not found\"}" {
		t.Fatalf("Respons body does not equal..3")
	}
	AssertLogContains(t, logger, []string{"Server running", ":8083.", "Req:  /files/", "Resp: Status:200"})
	os.Stderr.WriteString(logger.Get())
}

func TestReadFile(t *testing.T) {

	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 {
		t.Fatal(errList.ToString())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "files/user/stuart/loc/pics/name/t1.JSON", 200, "?", 251)
	if !strings.HasPrefix(trimString(resBody), "{ \"ServerName\": \"TestServer\", \"Users\":") {
		t.Fatalf("Respons body does not start with...")
	}

	_, resBody = RunClientGet(t, configData, "files/user/stuart/loc/pics/name/s-testfolder", 404, "?", 73)
	if !strings.Contains(trimString(resBody), "Is not a file") {
		t.Fatalf("Respons body does not contain 'Is not a file'")
	}

	AssertLogContains(t, logger, []string{"Server running", ":8083.", "Req:  /files/", "Resp: Status:200"})
	os.Stderr.WriteString(logger.Get())
}

func TestClient(t *testing.T) {
	configData, errList := config.NewConfigData("../goWebAppTest.json")
	if errList.Len() > 1 {
		t.Fatal(errList.ToString())
	}

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	res, _ := RunClientGet(t, configData, "ABC", 404, "{\"error\":true, \"status\":404, \"msg\":\"Not Found\", \"reason\":\"Resource not found\"}", 78)
	AssertHeaderEquals(t, res, "Content-Type", fmt.Sprintf("%s; charset=%s", config.DefaultContentType, configData.GetContentTypeCharset()))
	AssertHeaderEquals(t, res, "Server", configData.GetServerName())
	RunClientGet(t, configData, "ping", 200, "{\"error\":false, \"status\":200, \"msg\":\"OK\", \"reason\":\"Ping\"}", 58)
	RunClientGet(t, configData, "exit", http.StatusAccepted, "{\"error\":false, \"status\":202, \"msg\":\"Accepted\", \"reason\":\"Server Stopped\"}", 74)
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

func RunClientGet(t *testing.T, config *config.ConfigData, path string, expectedStatus int, expectedBody string, expectedLen int) (*http.Response, string) {
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
		len := res.Header["Content-Length"]
		if len[0] != strconv.Itoa(expectedLen) {
			t.Fatalf("Status for path http://localhost%s/%s.\nExpected '%d' \nActual   '%s'", config.GetPortString(), path, expectedLen, len[0])
		}
	}
	return res, string(resBody)
}

func RunServer(config *config.ConfigData, logger tools.Logger) {
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
