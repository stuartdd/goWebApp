package controllers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"stuartdd.com/config"
	"stuartdd.com/runCommand"
)

const encodedPathPrefix = "X0X"

type Handler interface {
	Submit() *ResponseData
}

type StaticFileHandler struct {
	filePath   []string
	configData *config.ConfigData
}

func NewStaticFileHandler(file []string, configData *config.ConfigData) *StaticFileHandler {
	return &StaticFileHandler{
		filePath:   file,
		configData: configData,
	}
}

func (p *StaticFileHandler) Submit() *ResponseData {
	list := []string{p.configData.GetServerStaticRoot()}
	list = append(list, p.filePath...)
	fullFile := filepath.Join(list...)

	stats, err := os.Stat(fullFile)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("File not found", true)
	}

	if stats.IsDir() {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Is not a file", true)
	}

	fileContent, err := os.ReadFile(fullFile)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("File could not be read", true)
	}
	return NewResponseData(http.StatusOK).WithContentBytesJson(fileContent).WithMimeType(p.filePath[len(p.filePath)-1])
}

type ReadFileHandler struct {
	parameters *UrlRequestParts
	configData *config.ConfigData
}

func NewReadFileHandler(urlParts *UrlRequestParts, configData *config.ConfigData) Handler {
	return &ReadFileHandler{
		parameters: urlParts,
		configData: configData,
	}
}

func (p *ReadFileHandler) Submit() *ResponseData {
	file, err := p.parameters.GetUserLocNamePath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("File not found", true)
	}
	stats, err := os.Stat(file)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("File not found", true)
	}

	if stats.IsDir() {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Is not a file", true)
	}

	fileContent, err := os.ReadFile(file)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("File could not be read", true)
	}
	return NewResponseData(http.StatusOK).WithContentBytesJson(fileContent).WithMimeType(p.parameters.GetName())
}

type DirHandler struct {
	requestData *UrlRequestParts
	listFiles   bool
}

func NewDirHandler(urlRequestData *UrlRequestParts, configData *config.ConfigData, listFiles bool) Handler {
	return &DirHandler{
		requestData: urlRequestData,
		listFiles:   listFiles,
	}
}

func (p *DirHandler) Submit() *ResponseData {
	var err error

	file, err := p.requestData.GetUserLocPath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Dir not found", true)
	}

	if p.requestData.HasParam(PathParam) {
		path := DecodePath(p.requestData.GetParam(PathParam))
		p.requestData.SetParam(PathParam, path)
		file = filepath.Join(file, path)
	}

	stats, err := os.Stat(file)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Dir not found", true)
	}
	if !stats.IsDir() {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Is not a dir", true)
	}
	if p.listFiles {
		entries, err := os.ReadDir(file)
		if err != nil {
			return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Dir cannot be read", true)
		}
		return NewResponseData(http.StatusOK).WithContentBytesJson(filesAsJson(entries, p.requestData))
	} else {
		return NewResponseData(http.StatusOK).WithContentBytesJson(listDirectoriesAsJson(file, p.requestData))
	}
}

type TreeHandler struct {
	parameters *UrlRequestParts
}

func NewTreeHandler(urlParts *UrlRequestParts, configData *config.ConfigData) Handler {
	return &TreeHandler{
		parameters: urlParts,
	}
}

func (p *TreeHandler) Submit() *ResponseData {
	file, err := p.parameters.GetUserLocPath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Dir not found", true)
	}
	stats, err := os.Stat(file)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Dir not found", true)
	}
	if !stats.IsDir() {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Is not a dir", true)
	}

	root := NewTreeNode("fs")
	err = filepath.Walk(file,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && !strings.HasPrefix(path, ".") && !strings.HasPrefix(path, "_") {
				root.AddPath(path)
			}
			return nil
		})
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Dir cannot be read", true)
	}
	return NewResponseData(http.StatusOK).WithContentBytesJson(treeAsJson(root, p.parameters)).WithMimeType("json")
}

type PostFileHandler struct {
	parameters *UrlRequestParts
	request    *http.Request
}

func NewPostFileHandler(urlParts *UrlRequestParts, configData *config.ConfigData, r *http.Request) Handler {
	return &PostFileHandler{
		parameters: urlParts,
		request:    r,
	}
}

func (p *PostFileHandler) Submit() *ResponseData {
	dir, err := p.parameters.GetUserLocPath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Dir not found", true)
	}
	stats, err := os.Stat(dir)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Dir not found", true)
	}
	if !stats.IsDir() {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Is not a dir", true)
	}
	body, err := io.ReadAll(p.request.Body)
	if err != nil {
		return NewResponseData(http.StatusUnprocessableEntity).WithContentReasonAsJson("Failed to read input", true)
	}
	file, err := p.parameters.GetUserLocNamePath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("File name not found", true)
	}
	err = os.WriteFile(file, body, 0644)
	if err != nil {
		return NewResponseData(http.StatusUnprocessableEntity).WithContentReasonAsJson("Failed to save data", true)
	}

	return NewResponseData(http.StatusAccepted).WithContentReasonAsJson("File saved", false)
}

type ExecHandler struct {
	parameters *UrlRequestParts
	createMap  func([]byte, []byte, int) map[string]interface{}
}

func NewExecHandler(urlParts *UrlRequestParts, configData *config.ConfigData, createMapFunc func([]byte, []byte, int) map[string]interface{}) Handler {
	return &ExecHandler{
		parameters: urlParts,
		createMap:  createMapFunc,
	}
}

func (p *ExecHandler) Submit() *ResponseData {
	execInfo, err := p.parameters.GetUserExecInfo()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Exec not found", true)
	}
	execData := runCommand.NewExecData(execInfo.Cmd, execInfo.Dir, execInfo.GetOutLogFile(), execInfo.GetErrLogFile(), func(r []rune) string {
		return p.parameters.SubstituteFromMap(r, false)
	})
	stdOut, stdErr, code, err := execData.Run()
	if err != nil {
		return NewResponseData(http.StatusFailedDependency).WithContentReasonAsJson(err.Error(), true)
	}
	var dataMap map[string]interface{}
	if p.createMap != nil {
		dataMap = p.createMap(stdOut, stdErr, code)
	} else {
		if code == 0 {
			dataMap = map[string]interface{}{"error": false, "exitCode": code, "stdOut": string(stdOut), "stdErr": string(stdErr)}
		} else {
			dataMap = map[string]interface{}{"error": true, "exitCode": code, "stdOut": string(stdOut), "stdErr": string(stdErr)}
		}
	}
	if code != 0 {
		return NewResponseData(http.StatusOK).WithContentMapJson(dataMap).SetShouldLog()
	}
	return NewResponseData(http.StatusOK).WithContentMapJson(dataMap)
}

//-------------------------------------------------------------------
/*
return "{\"time\":{"
                + "\"millis\":" + System.currentTimeMillis() + ", "
                + "\"time2\":" + "\"" + time2(dt) + "\", "
 *               + "\"time3\":" + "\"" + time3(dt) + "\", "
 *               + "\"monthDay\":" + "\"" + monthDay(dt) + "\", "
                + "\"year\":" + dt.year().get() + ", "
                + "\"month\":" + dt.monthOfYear().get() + ", "
                + "\"dom\":" + dt.dayOfMonth().get() + ", "
                + "\"mon\":\"" + dt.monthOfYear().getAsText() + "\", "
                + "\"timestamp\":\"" + ConfigDataManager.formattedTimeStamp() + "\"}}";

 * {"time":{"millis":1554504586062, "time2":"23:49", "time3":"23:49:46", "monthDay":"April:05", "year":2019, "month":4, "dom":5, "mon":"April", "timestamp":"05-04-2019T23:49:46.0+0100"}}
 * {"time":{"dom":29,"millis":1711671100876,"mon":"March","month":3,"monthDay":"March:29","time2":"00:11","time3":"00:11:40","timestamp":"2024-03-29T00:11:40Z","year":2024}}
*/
func GetTimeAsMap() map[string]interface{} {
	t := time.Now()
	day := t.Day()
	mon := t.Month()
	monName := mon.String()
	t2 := fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
	m1 := make(map[string]interface{}, 0)
	m2 := make(map[string]interface{}, 0)
	m2["millis"] = t.UnixMilli()
	m2["year"] = t.Year()
	m2["monthDay"] = fmt.Sprintf("%s:%02d", monName, day)
	m2["month"] = mon
	m2["dom"] = day
	m2["mon"] = monName
	m2["time3"] = fmt.Sprintf("%s:%02d", t2, t.Second())
	m2["time2"] = t2
	m2["timestamp"] = t.Format(time.RFC3339)
	m1["time"] = m2
	return m1
}

/*
{"users": [{"id":"stuart","name":"Stuart"},{"id":"shared"},{"id":"nonuser"},{"id":"test","src":"src"}]}
*/
func GetUsersAsMap(users *map[string]config.UserData) map[string]interface{} {
	m1 := make(map[string]interface{}, 0)
	l1 := make([]map[string]string, 0)
	for id, ud := range *users {
		l1 = append(l1, map[string]string{"id": id, "name": ud.Name})
	}
	m1["users"] = l1
	return m1
}

func GetFaveIcon(configData *config.ConfigData) *ResponseData {
	if configData.GetFaviconIcoPath() == "" {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("favicon.ico not defined", true)
	}
	fileContent, err := os.ReadFile(configData.GetFaviconIcoPath())
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("favicon.ico not found", true)
	}
	return NewResponseData(http.StatusOK).WithContentBytesJson(fileContent).WithMimeType("ico")
}

func GetServerStatusAsMap(configData *config.ConfigData, upSince time.Time) map[string]interface{} {
	m := make(map[string]interface{}, 0)
	m["error"] = false
	m["reloadConfig"] = configData.GetTimeToReloadConfig()
	m["configName"] = configData.ConfigName
	m["upSince"] = upSince.Format(time.ANSIC)
	m["upTime"] = fmtDuration(time.Since(upSince))
	var st runtime.MemStats
	runtime.ReadMemStats(&st)
	m["Alloc"] = fmtAlloc(st.Alloc)
	m["TotalAlloc"] = fmtAlloc(st.TotalAlloc)
	m["Sys"] = fmtAlloc(st.Sys)
	return m
}

func fmtAlloc(al uint64) string {
	return fmt.Sprintf("%d MiB (%d B)", al/1024/1024, al)
}
func fmtDuration(d time.Duration) string {
	secs := int64(d.Seconds())
	h := secs / 3600
	secs -= h * 3600
	m := secs / 60
	secs -= m * 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, secs)
}

func statusAsJson(status int, reason string, error bool) []byte {
	var b bytes.Buffer
	b.WriteString("{\"error\":")
	b.WriteString(fmt.Sprintf("%t", error))
	b.WriteString(", \"status\":")
	b.WriteString(strconv.Itoa(status))
	b.WriteString(", \"msg\":\"")
	b.WriteString(http.StatusText(status))
	b.WriteString("\", \"reason\":\"")
	b.WriteString(reason)
	b.WriteString("\"}")
	return b.Bytes()
}

func treeAsJson(root *TreeDirNode, params *UrlRequestParts) []byte {
	var buffer bytes.Buffer
	buffer.WriteRune('{')
	writeJsonHeader(params, &buffer)
	buffer.WriteString(",\"tree\":")
	buffer.WriteString(string(root.ToJson(false)))
	buffer.WriteString("}")
	return buffer.Bytes()
}

func filesAsJson(ents []fs.DirEntry, params *UrlRequestParts) []byte {
	var buffer bytes.Buffer
	entLen := len(ents)
	buffer.WriteRune('{')
	writeJsonHeader(params, &buffer)
	buffer.WriteRune(',')
	writePathToJson(params.GetOptionalParam(PathParam), PathParam, &buffer)
	buffer.WriteString("\"files\":[")
	for i := 0; i < entLen; i++ {
		e := ents[i]
		if filterDirNames(e, params.GetConfigFileFilter()) {
			writeSingleFileNameToJson(e, &buffer)
			if i < (entLen - 1) {
				buffer.WriteRune(',')
			}
		}
	}
	buffer.WriteString("]}")
	return buffer.Bytes()
}

func listDirectoriesAsJson(dir string, param *UrlRequestParts) []byte {
	list := &[]string{}
	listDirectoriesRec(dir, dir+string(os.PathSeparator), param.GetConfigFileFilter(), list)
	listLen := len(*list)
	var buffer bytes.Buffer
	buffer.WriteRune('{')
	writeJsonHeader(param, &buffer)
	buffer.WriteString(",\"paths\":[")
	for i, s := range *list {
		buffer.WriteString("{\"name\":\"")
		buffer.WriteString(s)
		buffer.WriteString("\",")
		buffer.WriteString("\"encName\":\"")
		buffer.WriteString(EncodePath(s))
		buffer.WriteString("\"}")
		if i < (listLen - 1) {
			buffer.WriteRune(',')
		}
	}
	buffer.WriteString("]}")
	return buffer.Bytes()
}

func listDirectoriesRec(path string, root string, filter []string, l *[]string) {
	entries, _ := os.ReadDir(path)
	dirCount := 0
	fileCount := 0
	for _, ent := range entries {
		n := ent.Name()
		if ent.IsDir() {
			if !strings.HasPrefix(n, ".") && !strings.HasPrefix(n, "_") {
				dirCount++
			}
		} else {
			if filterFileNames(ent.Name(), filter) {
				fileCount++
			}
		}
	}
	if (dirCount > 0 || fileCount > 0) && strings.HasPrefix(path, root) {
		*l = append(*l, path[len(root):])
	}
	for _, ent := range entries {
		if ent.IsDir() {
			p := filepath.Join(path, ent.Name())
			listDirectoriesRec(p, root, filter, l)
		}
	}
}

func filterDirNames(e fs.DirEntry, filter []string) bool {
	if e.IsDir() {
		return false
	}
	return filterFileNames(e.Name(), filter)
}

func filterFileNames(name string, filter []string) bool {
	n := strings.ToLower(name)
	if strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_") {
		return false
	}
	for i := 0; i < len(filter); i++ {
		if strings.HasSuffix(n, filter[i]) {
			return true
		}
	}
	return false
}

func writePathToJson(pathUnencoded string, key string, buffer *bytes.Buffer) {
	buffer.WriteRune('"')
	buffer.WriteString(key)
	buffer.WriteString("\":")
	if pathUnencoded != "" {
		buffer.WriteString("{\"name\":\"")
		buffer.WriteString(pathUnencoded)
		buffer.WriteString("\", \"encName\":\"")
		buffer.WriteString(EncodePath(pathUnencoded))
		buffer.WriteString("\"},")
	} else {
		buffer.WriteString("null,")
	}
}

func writeJsonHeader(param *UrlRequestParts, buffer *bytes.Buffer) {
	writeErrorAsJsonString(false, buffer)
	writeParamAsJsonString(param.GetOptionalParam(UserParam), UserParam, true, false, buffer)
	writeParamAsJsonString(param.GetOptionalParam(LocationParam), LocationParam, true, false, buffer)
}

func writeSingleFileNameToJson(file fs.DirEntry, buffer *bytes.Buffer) {
	inf, err := file.Info()
	if err != nil {
		return
	}
	buffer.WriteString("{\"size\":")
	buffer.WriteString(strconv.Itoa(int(inf.Size())))
	buffer.WriteString(",\"name\":{\"name\":\"")
	buffer.WriteString(file.Name())
	buffer.WriteString("\", \"encName\":\"")
	buffer.WriteString(EncodePath(file.Name()))
	buffer.WriteString("\"}}")
}

func writeParamAsJsonString(value string, key string, commaAtStart, commaAtEnd bool, buffer *bytes.Buffer) {
	if value != "" {
		if commaAtStart {
			buffer.WriteRune(',')
		}
		buffer.WriteRune('"')
		buffer.WriteString(key)
		buffer.WriteString("\":\"")
		buffer.WriteString(value)
		buffer.WriteRune('"')
		if commaAtEnd {
			buffer.WriteRune(',')
		}
	}
}

func writeErrorAsJsonString(error bool, buffer *bytes.Buffer) {
	buffer.WriteRune('"')
	buffer.WriteString(ErrorParam)
	buffer.WriteString("\":")
	if error {
		buffer.WriteString("true")
	} else {
		buffer.WriteString("false")
	}
}

func DecodePath(encodedPath string) string {
	if encodedPath == "" {
		return ""
	}
	if strings.HasPrefix(encodedPath, encodedPathPrefix) {
		decoded, err := base64.StdEncoding.DecodeString(encodedPath[len(encodedPathPrefix):])
		if err != nil {
			return encodedPath
		}
		return string(decoded)
	}
	return encodedPath
}

func EncodePath(unEncoded string) string {
	if unEncoded == "" {
		return ""
	}
	return encodedPathPrefix + base64.StdEncoding.EncodeToString([]byte(unEncoded))
}
