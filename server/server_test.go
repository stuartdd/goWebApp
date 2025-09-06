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

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/logging"
	"github.com/stuartdd/goWebApp/runCommand"
)

type TLog struct {
	B bytes.Buffer
}

func (l *TLog) Close() {}
func (l *TLog) Log(s string) {
	l.B.WriteString(s)
	l.B.WriteString("\n")
}

func (l *TLog) VerboseFunction() func(string) {
	return nil
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

var serverState string = ""
var logger = &TLog{}

const postDataFile1 = "{\"Data\":\"This is data ONE for file 1\"}"
const postDataFile2 = "{\"Data\":\"This is data TWO for file 2\"}"
const testdatafile = "testdata.json"

func TestUrlRequestParamsMap(t *testing.T) {
	AssertMatch(t, "0", NewUrlRequestMatcher("/a/b/*/c/*", "get", true), "/x/b/1/c/4", "GET", false, "")
	AssertMatch(t, "1", NewUrlRequestMatcher("/a/b/*/c/*", "get", true), "/a/b/1/x/4", "GET", false, "b=1")
	AssertMatch(t, "2", NewUrlRequestMatcher("/a/b/*/c/*", "get", true), "/a/b/1/c", "GET", false, "")
	AssertMatch(t, "3", NewUrlRequestMatcher("/a/b/*/c/*", "get", true), "/a/b/1/c/3", "GET", true, "b=1,c=3")
	AssertMatch(t, "4", NewUrlRequestMatcher("a", "get", true), "/a", "get", false, "")
	AssertMatch(t, "5", NewUrlRequestMatcher("a", "get", true), "a", "get", true, "")
	AssertMatch(t, "5", NewUrlRequestMatcher("/a", "get", true), "/a", "get", true, "")
	AssertMatch(t, "6", NewUrlRequestMatcher("/a/b/*/*/c/*", "get", true), "/a/b/1/2/c/3", "post", false, "")
	AssertMatch(t, "7", NewUrlRequestMatcher("/a/b/*/*/c/*", "get", true), "/a/b/1/2/C/3", "GET", false, "b=1")
	AssertMatch(t, "8", NewUrlRequestMatcher("/a/b/*/*/c/*", "get", true), "/a/b/1/2/c/3", "get", true, "b=1,c=3")
	AssertMatch(t, "9", NewUrlRequestMatcher("/a/b/*/*/c/*", "get", true), "/a/b/1/2/c/3", "GET", true, "b=1,c=3")
	AssertMatch(t, "10", NewUrlRequestMatcher("/a/*/b/*/c/*", "get", true), "/a/1/b/2/c/3", "GET", true, "a=1,b=2,c=3")
	AssertMatch(t, "10", NewUrlRequestMatcher("", "get", true), "/a/1/b/2/c/3", "GET", false, "")
	AssertMatch(t, "11", NewUrlRequestMatcher("", "get", true), "", "GET", false, "")
	AssertMatch(t, "12", NewUrlRequestMatcher("", "post", true), "", "GET", false, "")
}
func TestServerGetUsers(t *testing.T) {
	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "server/users"
	_, respBody := RunClientGet(t, configData, url, 200, "?", -1, 10)
	AssertContains(t, respBody, []string{
		"\"id\":\"bob\"",
		"\"name\":\"Bob\"",
	})
}

func TestServerGetTime(t *testing.T) {
	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "server/time"
	_, respBody := RunClientGet(t, configData, url, 200, "?", -1, 10)
	AssertContains(t, respBody, []string{
		"\"time\":{\"dom\":",
		"\"millis\":",
		"\"timestamp\":",
	})
}
func TestServerStatus(t *testing.T) {
	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "server/status"
	_, respBody := RunClientGet(t, configData, url, 200, "?", -1, 10)
	AssertContains(t, respBody, []string{
		"\"error\":false,",
		"\"ConfigName\":\"goWebAppTest.json\"",
		"\"Processes\":[]",
		"\"Log_File\":\"DummyLogger.log\"",
	})

}
func TestServer(t *testing.T) {
	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "files/user/stuart/loc/pics"
	_, respBody := RunClientGet(t, configData, url, 200, "?", -1, 10)
	AssertContains(t, respBody, []string{
		"\"name\":\"pic1.jpeg\", \"encName\":\"X0XcGljMS5qcGVn\"",
		"\"error\":false,\"user\":\"stuart\",\"loc\":\"pics\",\"path\":null,",
	})

	url = "server/status"
	_, respBody = RunClientGet(t, configData, url, 200, "?", -1, 10)
	AssertContains(t, respBody, []string{
		"\"error\":false,\"status\":",
		"\"UpSince\":",
		"\"Processes\":[]",
		"goWebAppTest.json",
	})

	url = "files/user/stuart/loc/data/name/state.json"
	_, respBody = RunClientGet(t, configData, url, 200, "?", 112, 10)
	AssertContains(t, respBody, []string{
		"\"displayOptions\"",
		"\"optionShowResponse\"",
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
	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	url := "static/images/pic.jpeg"
	resp, _ := RunClientGet(t, configData, url, 200, "?", 4821, 10)
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
	RunClientGet(t, configData, url, 403, "?", -1, 0)

}
func TestFilePath(t *testing.T) {
	configData := loadConfigData(t)

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
	configData := loadConfigData(t)

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
	configData := loadConfigData(t)

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

func TestPostFileAndDelete(t *testing.T) {
	configData := loadConfigData(t)

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
	_, resBody := RunClientGet(t, configData, url, 200, "?", -1, 0)
	if resBody != postDataFile1 {
		t.Fatalf("Response body does not equal postDataFile1")
	}
	// Try to save again with different content but should not overwrite so content remains the same!
	RunClientPost(t, configData, url, 412, "File exists", postDataFile2)
	_, resBody = RunClientGet(t, configData, url, 200, "?", -1, 0)
	if resBody != postDataFile1 {
		t.Fatalf("Response body does not equal postDataFile1")
	}
	_, err = os.Stat(file)
	if err != nil {
		t.Fatalf("File was not created")
	}
	// TODO
	RunClientDelete(t, configData, url, 202, "\"reason\":\"File deleted OK\"")
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
	})

}

func TestPostFileOverwriteAndDelete(t *testing.T) {
	configData := loadConfigData(t)

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
	_, resBody := RunClientGet(t, configData, urlB, 200, "?", -1, 0)
	if resBody != postDataFile1 {
		t.Fatalf("Response body does not equal postDataFile1")
	}
	// Try to save again with different content this should not overwrite so content will change!
	RunClientPost(t, configData, urlA, 202, "File:Action:replace", postDataFile2)
	_, resBody = RunClientGet(t, configData, urlB, 200, "?", -1, 0)
	if resBody != postDataFile2 {
		t.Fatalf("Response body does not equal postDataFile2")
	}
	_, err = os.Stat(file)
	if err != nil {
		t.Fatalf("File was not created")
	}
	// TODO
	RunClientDelete(t, configData, urlB, 202, "\"reason\":\"File deleted OK\"")
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
	configData := loadConfigData(t)

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
	_, resBody := RunClientGet(t, configData, urlB, 200, "?", -1, 0)
	if resBody != postDataFile1 {
		t.Fatalf("Response body does not equal postDataFile1")
	}
	// Try to save again with different content this should append so content will change!
	RunClientPost(t, configData, urlA, 202, "File:Action:append", postDataFile2)
	_, resBody = RunClientGet(t, configData, urlB, 200, "?", -1, 0)
	if resBody != postDataFile1+postDataFile2 {
		t.Fatalf("Response body does not equal postDataFile2 + postDataFile2")
	}
	_, err = os.Stat(file)
	if err != nil {
		t.Fatalf("File was not created")
	}
	// TODO
	RunClientDelete(t, configData, urlB, 202, "\"reason\":\"File deleted OK\"")
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

	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}
	_, resBody := RunClientGet(t, configData, "files/user/stuart/loc/pics", 200, "?", -1, 0)

	AssertContains(t, resBody, []string{
		"\"error\":false,\"user\":\"stuart\",\"loc\":\"pics\",\"path\":null,\"files\"",
		"{\"name\":\"t2.Data\", \"encName\":\"X0XdDIuRGF0YQ==\"}",
		"{\"name\":\"t1.JSON\", \"encName\":\"X0XdDEuSlNPTg==\"}",
		"{\"name\":\"pic1.jpeg\", \"encName\":\"X0XcGljMS5qcGVn\"}",
	})

	_, resBody = RunClientGet(t, configData, "files/user/stuart/loc/picsPlus", 200, "?", -1, 0)

	AssertContains(t, resBody, []string{
		"\"error\":false,\"user\":\"stuart\",\"loc\":\"picsPlus\",\"path\":null,\"files\"",
		"{\"name\":\"t5.json\", \"encName\":\"X0XdDUuanNvbg==\"}",
		"{\"name\":\"testdata2.json\", \"encName\":\"X0XdGVzdGRhdGEyLmpzb24=\"}",
	})

	AssertLogContains(t, logger, []string{"Server Started", ":8083.", "Req:  GET:/files/", "Resp: Status:200"})
	AssertLogContains(t, logger, []string{"GET:/files/user/stuart/loc/pics", "GET:/files/user/stuart/loc/picsPlus"})
}

func TestReadDirNotFound(t *testing.T) {

	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "files/user/stuart/loc/picsMissing", http.StatusNotFound, "?", -1, 0)
	AssertContains(t, resBody, []string{
		"\"error\":true",
		"\"reason\":\"Dir not found\"",
	})
	AssertLogContains(t, logger, []string{"Dir not found", "\"status\":404"})
}

func TestReadFile(t *testing.T) {

	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "files/user/stuart/loc/pics/name/t1.JSON", http.StatusOK, "?", 251, 0)
	if !strings.HasPrefix(trimString(resBody), "{ \"ServerName\": \"TestServer\", \"Users\":") {
		t.Fatalf("Respons body does not start with...")
	}

	AssertLogContains(t, logger, []string{"Server Started", ":8083.", "Req:  GET:/files/", "Resp: Status:200"})
	os.Stderr.WriteString(logger.Get())
}

func TestReadFileNotUser(t *testing.T) {

	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "files/user/nouser/loc/pics/name/t1.JSON", http.StatusNotFound, "?", 74, 0)
	AssertContains(t, string(resBody), []string{"\"error\":true", "\"reason\":\"user not found\""})
	AssertLogContains(t, logger, []string{"\"error\":true", "\"status\":404", "\"reason\":\"user not found\""})
}

func TestReadFileNotLoc(t *testing.T) {

	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "files/user/stuart/loc/noloc/name/t1.JSON", http.StatusNotFound, "?", 78, 0)
	AssertContains(t, string(resBody), []string{"\"error\":true", " \"reason\":\"location not found\""})
	AssertLogContains(t, logger, []string{"\"error\":true", "\"status\":404", "\"reason\":\"location not found\""})
}

func TestReadFileNotName(t *testing.T) {

	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "files/user/stuart/loc/pics/name/notExist", http.StatusNotFound, "?", 74, 0)
	AssertContains(t, string(resBody), []string{"\"error\":true", " \"reason\":\"File not found\""})
	AssertLogContains(t, logger, []string{"\"error\":true", "\"status\":404", "\"reason\":\"File not found\""})
}

func TestReadFileIsDir(t *testing.T) {

	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "files/user/stuart/loc/pics/name/s-testfolder", http.StatusForbidden, "?", 74, 0)
	AssertContains(t, string(resBody), []string{"\"error\":true", " \"reason\":\"Is a directory\""})
	AssertLogContains(t, logger, []string{
		"/stuart/s-pics/s-testfolder is a Directory",
		"Resp: Error: Status:403",
		"\"error\":true", "\"status\":403",
		"\"reason\":\"Is a directory\""})
}

func TestServerTime(t *testing.T) {
	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "server/time", 200, "?", 173, 20)
	if !strings.HasPrefix(trimString(resBody), "{\"time\":{") {
		t.Fatalf("Respons body does not start with...")
	}

	if strings.Contains(logger.Get(), "/time") {
		os.Stderr.WriteString(logger.Get())
		t.Fatal("Log must NOT contain the time request response")
	}
}

func TestServerIsUp(t *testing.T) {
	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "isup", 200, "?", 64, 0)
	if !strings.HasPrefix(trimString(resBody), "{\"error\":false, ") {
		t.Fatalf("Respons body does not start with...")
	}
	AssertContains(t, string(resBody), []string{"ServerIsUp"})
	if strings.Contains(logger.Get(), "/isup") {
		os.Stderr.WriteString(logger.Get())
		t.Fatal("Log must NOT contain the isup request response")
	}
}

func TestQuietGetFile(t *testing.T) {
	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	_, resBody := RunClientGet(t, configData, "test/user/stuart/loc/pics/name/t1.JSON", 200, "?", 251, 0)
	if !strings.HasPrefix(trimString(resBody), "{ \"ServerName\": \"TestServer\", \"Users\":") {
		t.Fatalf("Respons body does not start with...")
	}

	AssertContains(t, string(resBody), []string{"TestServer"})
	if strings.Contains(logger.Get(), "test/user") {
		os.Stderr.WriteString(logger.Get())
		t.Fatal("Log must NOT contain the 'test/user' request response")
	}
}

func TestClient(t *testing.T) {
	configData := loadConfigData(t)

	if serverState != "Running" {
		go RunServer(configData, logger)
		time.Sleep(100 * time.Millisecond)
	}

	res, _ := RunClientGet(t, configData, "ABC", 404, "{\"error\":true, \"status\":404, \"msg\":\"Not Found\", \"reason\":\"Resource not found\"}", 78, 0)
	AssertHeaderEquals(t, res, "Content-Type", fmt.Sprintf("%s; charset=%s", config.DefaultContentType, configData.GetContentTypeCharset()))
	AssertHeaderEquals(t, res, "Server", configData.GetServerName())
	RunClientGet(t, configData, "ping", 200, "{\"error\":false, \"status\":200, \"msg\":\"OK\", \"reason\":\"Ping\"}", 58, 0)
	RunClientGet(t, configData, "server/exit", http.StatusAccepted, "{\"error\":false, \"status\":202, \"msg\":\"Accepted\", \"reason\":\"[11] Exit Requested\"}", 79, 0)
	AssertLogContains(t, logger, []string{"Req:  GET:/ABC", "Error: Status:404"})
	os.Stderr.WriteString(logger.Get())
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
	actionQueue := make(chan *ActionEvent, 10)
	defer close(actionQueue)
	go func() {
		for {
			ae := <-actionQueue
			switch ae.Id {
			case Exit:
				fmt.Printf("Server: Stopped\n")
			case Ignore:
				fmt.Printf("Server: Ignore\n")
			}
		}
	}()
	var lrm *runCommand.LongRunningManager
	var err error
	if config.GetExecPath() != "" {
		lrm, err = runCommand.NewLongRunningManager(config.GetExecPath(), logger.Log)
		if err != nil {
			panic(fmt.Sprintf("LongRunningManager: failed to initialise. '%s'. ABORTED", err.Error()))
		}
	}

	server, err := NewWebAppServer(config, actionQueue, lrm, logger)
	if err != nil {
		panic(fmt.Sprintf("NewWebAppServer: failed to initialise. '%s'. ABORTED", err.Error()))
	}
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
	for _, x := range list {
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

	p, ok, _ := matcher.Match(requestUriparts, isAbsolutePath, reqType, nil)
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

func loadConfigData(t *testing.T) *config.ConfigData {
	errList := config.NewConfigErrorData()
	configData := config.NewConfigData("../goWebAppTest.json", "goWebApp", false, false, false, errList)
	if errList.ErrorCount() > 1 || configData == nil {
		t.Fatal(errList.String())
	}
	if configData == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errList.String())
	}
	return configData
}
