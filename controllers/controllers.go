package controllers

import (
	"bytes"
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

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/runCommand"
)

type Handler interface {
	Submit() *ResponseData
}

type StaticFileHandler struct {
	filePath   []string
	configData *config.ConfigData
	log        func(string)
	verbose    func(string)
}

func NewStaticFileHandler(file []string, configData *config.ConfigData, logFunc func(string), verboseFunc func(string)) *StaticFileHandler {
	return &StaticFileHandler{
		filePath:   file,
		configData: configData,
		log:        logFunc,
		verbose:    verboseFunc,
	}
}

func (p *StaticFileHandler) Submit() *ResponseData {
	list := []string{p.configData.GetServerStaticRoot()}
	list = append(list, p.filePath...)
	fullFile := filepath.Join(list...)

	stats, err := os.Stat(fullFile)
	if err != nil {
		panic(config.NewPanicMessage("File not found", http.StatusNotFound, fmt.Sprintf("Static File Error:%s", err.Error())))
	}
	if stats.IsDir() {
		panic(config.NewPanicMessage("Is a directory", http.StatusForbidden, fmt.Sprintf("Static file %s is a Directory", fullFile)))
	}
	fileContent, err := os.ReadFile(fullFile)
	if err != nil {
		panic(config.NewPanicMessage("File could not be read", http.StatusUnprocessableEntity, fmt.Sprintf("Static File Error:%s", err.Error())))
	}
	if p.verbose != nil {
		p.verbose(fmt.Sprintf("Static Read File:%s Mime[%s] Len[%d]", fullFile, config.LookupContentType(p.filePath[len(p.filePath)-1]), len(fileContent)))

	}
	if p.configData.IsTemplating() {
		td := p.configData.GetTemplateData()
		if td.ShouldTemplate(list[len(list)-1]) {
			fileContent = []byte(p.configData.SubstituteFromMap([]byte(string(fileContent)), td.Data))
		}
	}
	return NewResponseData(http.StatusOK).WithContentBytes(fileContent).WithMimeType(p.filePath[len(p.filePath)-1])
}

type ReadFileHandler struct {
	parameters *UrlRequestParts
	configData *config.ConfigData
	log        func(string)
	verbose    func(string)
	addLrp     func(string, string, int, bool) bool
}

func NewReadFileHandler(urlParts *UrlRequestParts, configData *config.ConfigData, logFunc func(string), verboseFunc func(string), addFunc func(string, string, int, bool) bool) Handler {
	return &ReadFileHandler{
		parameters: urlParts,
		configData: configData,
		log:        logFunc,
		verbose:    verboseFunc,
		addLrp:     addFunc,
	}
}

func (p *ReadFileHandler) Submit() *ResponseData {
	file := p.parameters.GetUserLocPath(true, p.parameters.GetQueryAsBool("thumbnail", false), p.parameters.GetQueryAsBool("base64", false))

	stats, err := os.Stat(file)
	if err != nil {
		panic(config.NewPanicMessage("File not found", http.StatusNotFound, err.Error()))
	}
	if stats.IsDir() {
		panic(config.NewPanicMessage("Is a directory", http.StatusForbidden, fmt.Sprintf("%s is a Directory", file)))
	}
	fileContent, err := os.ReadFile(file)
	if err != nil {
		panic(config.NewPanicMessage("File could not be read", http.StatusUnprocessableEntity, err.Error()))
	}
	if p.verbose != nil {
		p.verbose(fmt.Sprintf("Read File:%s Mime[%s] Len[%d]", file, config.LookupContentType(p.parameters.GetName()), len(fileContent)))
	}
	return NewResponseData(http.StatusOK).WithContentBytes(fileContent).WithMimeType(p.parameters.GetName())
}

type DirHandler struct {
	parameters *UrlRequestParts
	listFiles  bool
	log        func(string)
	verbose    func(string)
}

func NewDirHandler(urlRequestData *UrlRequestParts, configData *config.ConfigData, listFiles bool, logFunc func(string), verboseFunc func(string)) Handler {
	return &DirHandler{
		parameters: urlRequestData,
		listFiles:  listFiles,
		log:        logFunc,
		verbose:    verboseFunc,
	}
}

func (p *DirHandler) Submit() *ResponseData {
	var err error

	file := p.parameters.GetUserLocPath(false, false, p.parameters.GetQueryAsBool("base64", false))
	stats, err := os.Stat(file)
	if err != nil {
		panic(config.NewPanicMessage("Dir not found", http.StatusNotFound, err.Error()))
	}
	if !stats.IsDir() {
		panic(config.NewPanicMessage("Is NOT a directory", http.StatusForbidden, fmt.Sprintf("%s is NOT a Directory", file)))
	}
	if p.listFiles {
		entries, err := os.ReadDir(file)
		if err != nil {
			panic(config.NewPanicMessage("Dir could not be read", http.StatusUnprocessableEntity, err.Error()))
		}
		return NewResponseData(http.StatusOK).WithContentBytes(filesAsJson(entries, p.parameters, p.verbose, file)).WithMimeType("json")
	} else {
		return NewResponseData(http.StatusOK).WithContentBytes(listDirectoriesAsJson(file, p.parameters, p.verbose, file)).WithMimeType("json")
	}
}

type TreeHandler struct {
	parameters *UrlRequestParts
	log        func(string)
	verbose    func(string)
}

func NewTreeHandler(urlParts *UrlRequestParts, configData *config.ConfigData, logFunc func(string), verboseFunc func(string)) Handler {
	return &TreeHandler{
		parameters: urlParts,
		log:        logFunc,
		verbose:    verboseFunc,
	}
}

func (p *TreeHandler) Submit() *ResponseData {
	file := p.parameters.GetUserLocPath(false, false, p.parameters.GetQueryAsBool("base64", false))
	stats, err := os.Stat(file)
	if err != nil {
		panic(config.NewPanicMessage("Dir not found", http.StatusNotFound, err.Error()))
	}
	if !stats.IsDir() {
		panic(config.NewPanicMessage("Is NOT a directory", http.StatusForbidden, fmt.Sprintf("%s is NOT a Directory", file)))
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
		panic(config.NewPanicMessage("Dir could not be read", http.StatusUnprocessableEntity, err.Error()))
	}
	return NewResponseData(http.StatusOK).WithContentBytes(treeAsJson(root, p.parameters)).WithMimeType("json")
}

type PostFileHandler struct {
	parameters *UrlRequestParts
	request    *http.Request
	log        func(string)
	verbose    func(string)
}

func NewPostFileHandler(urlParts *UrlRequestParts, configData *config.ConfigData, r *http.Request, logFunc func(string), verboseFunc func(string)) Handler {
	return &PostFileHandler{
		parameters: urlParts,
		request:    r,
		log:        logFunc,
		verbose:    verboseFunc,
	}
}

func (p *PostFileHandler) Submit() *ResponseData {
	dir := p.parameters.GetUserLocPath(false, false, p.parameters.GetQueryAsBool("base64", false))
	stats, err := os.Stat(dir)
	if err != nil {
		panic(config.NewPanicMessage("Dir not found", http.StatusNotFound, err.Error()))
	}
	if !stats.IsDir() {
		panic(config.NewPanicMessage("Is NOT a directory", http.StatusForbidden, fmt.Sprintf("%s is NOT a Directory", dir)))
	}
	body, err := io.ReadAll(p.request.Body)
	if err != nil {
		panic(config.NewPanicMessage("Failed to read POST data", http.StatusBadRequest, err.Error()))
	}
	file := p.parameters.GetUserLocPath(true, false, p.parameters.GetQueryAsBool("base64", false))
	err = os.WriteFile(file, body, 0644)
	if err != nil {
		panic(config.NewPanicMessage("Failed to save data", http.StatusInternalServerError, err.Error()))
	}
	if p.verbose != nil {
		p.verbose(fmt.Sprintf("File Stored:%s [%d]", file, len(body)))
	}

	return NewResponseData(http.StatusAccepted).WithContentReasonAsJson("File saved", false)
}

type ExecHandler struct {
	parameters *UrlRequestParts
	createMap  func([]byte, []byte, int) map[string]interface{}
	log        func(string)
	verbose    func(string)
	addLrp     func(string, string, int, bool) bool
}

func NewExecHandler(urlParts *UrlRequestParts, configData *config.ConfigData, createMapFunc func([]byte, []byte, int) map[string]interface{}, logFunc func(string), verboseFunc func(string), addFunc func(string, string, int, bool) bool) Handler {
	return &ExecHandler{
		parameters: urlParts,
		createMap:  createMapFunc,
		log:        logFunc,
		verbose:    verboseFunc,
		addLrp:     addFunc,
	}
}

func (p *ExecHandler) Submit() *ResponseData {
	if p.addLrp != nil {
		ok := p.addLrp(p.parameters.GetUser(), p.parameters.GetExecId(), 0, false)
		if !ok {
			panic(config.NewPanicMessage("Exec already running", http.StatusForbidden, fmt.Sprintf("User:%s Exec:%s Lon running process is already running", p.parameters.GetUser(), p.parameters.GetExecId())))
		}
	}
	execInfo := p.parameters.GetUserExecInfo()
	info := fmt.Sprintf("User:%s Exec:%s", p.parameters.GetUser(), p.parameters.GetExecId())
	execData := runCommand.NewExecData(execInfo.Cmd, execInfo.Dir, execInfo.GetOutLogFile(), execInfo.GetErrLogFile(), info, execInfo.Detached, p.log, func(r []byte) string {
		sq := p.parameters.SubstituteFromCachedMap(r)
		return p.parameters.SubstituteFromUserEnv([]byte(sq))
	}, func(pid int) {
		if p.addLrp != nil {
			p.addLrp(p.parameters.GetUser(), p.parameters.GetExecId(), pid, true)
		}
	})

	if p.verbose != nil {
		p.verbose(execData.String())
	}

	stdOut, stdErr, code, err := execData.Run()
	if err != nil {
		panic(config.NewPanicMessage("Exec Failed", http.StatusFailedDependency, fmt.Sprintf("RC:%d Error:%s", code, err.Error())))
	}
	if execInfo.LogOut != "" {
		of := p.parameters.config.SubstituteFromMap([]byte(execInfo.LogOut), p.parameters.config.GetUserEnv(p.parameters.GetUser()))
		err = os.WriteFile(filepath.Join(execInfo.Log, string(of)), stdOut, 0644)
		if err != nil {
			panic(config.NewPanicMessage("Failed to write stdOut to log", http.StatusInternalServerError, fmt.Sprintf("Failed to write stdOut. RC:%d Error:%s", code, err.Error())))
		}
	}
	if execInfo.LogErr != "" {
		of := p.parameters.config.SubstituteFromMap([]byte(execInfo.LogErr), p.parameters.config.GetUserEnv(p.parameters.GetUser()))
		err = os.WriteFile(filepath.Join(execInfo.Log, string(of)), stdErr, 0644)
		if err != nil {
			panic(config.NewPanicMessage("Failed to write stdErr to log", http.StatusInternalServerError, fmt.Sprintf("Failed to write stdErr. RC:%d Error:%s", code, err.Error())))
		}
	}
	if code > 0 && execInfo.NzCodeReturns >= http.StatusMultipleChoices {
		return NewResponseData(execInfo.NzCodeReturns).WithContentReasonAsJson(fmt.Sprintf("Exec returned %d", code), true)
	}
	if execInfo.StdOutType != "" && len(stdOut) > 0 {
		return NewResponseData(http.StatusOK).WithContentBytes(stdOut).WithMimeType(execInfo.StdOutType).SetHasErrors(code != 0)
	}
	var dataMap map[string]interface{}
	if p.createMap != nil {
		dataMap = p.createMap(stdOut, stdErr, code)
	} else {
		dataMap = map[string]interface{}{"error": code > 0, "exitCode": code, "stdOut": string(stdOut), "stdErr": string(stdErr)}
	}
	if code == 0 {
		return NewResponseData(http.StatusOK).WithContentMapJson(dataMap).SetHasErrors(false)
	} else {
		return NewResponseData(execInfo.NzCodeReturns).WithContentMapJson(dataMap).SetHasErrors(true)
	}
}

//-------------------------------------------------------------------
/*
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
		if !ud.IsHidden() {
			l1 = append(l1, map[string]string{"id": id, "name": ud.Name})
		}
	}
	m1["users"] = l1
	return m1
}

func GetFaveIcon(configData *config.ConfigData) *ResponseData {
	if configData.GetFaviconIcoPath() == "" {
		panic(config.NewPanicMessage("favicon.ico not configured", http.StatusNotFound, "FaviconIcoPath is not defined in config file"))
	}
	fileContent, err := os.ReadFile(configData.GetFaviconIcoPath())
	if err != nil {
		panic(config.NewPanicMessage("favicon.ico not found", http.StatusNotFound, err.Error()))
	}
	return NewResponseData(http.StatusOK).WithContentBytes(fileContent).WithMimeType("ico")
}

// "{\"Alloc\":\"2 MiB (2309672 B)\",\"Sys\":\"12 MiB (12672016 B)\",\"TotalAlloc\":\"2 MiB (2309672 B)\",\"configName\":\"goWebApp.json\",\"error\":false,\"reloadConfig\":3080.27,\"upSince\":\"Fri Apr  5 12:48:19 2024\",\"upTime\":\"00:08:39\"}"
// "[{\"error\":false,}{\"Alloc\":\"1 MiB (1368424 B)\"}]"
func GetServerStatusAsJson(configData *config.ConfigData, logFileName string, upSince time.Time, longRunning map[string]string) []byte {
	var b bytes.Buffer
	var st runtime.MemStats
	runtime.ReadMemStats(&st)
	b.WriteRune('{')
	writeParamAsJsonString("error", "false", false, false, true, &b)
	b.WriteString("\"status\": {")
	writeParamAsJsonString("UpSince", upSince.Format(time.ANSIC), true, false, true, &b)
	writeParamAsJsonString("UpTime", fmtDuration(time.Since(upSince)), true, false, true, &b)
	writeParamAsJsonString("Reload Config in", fmt.Sprintf("%.0f seconds", configData.GetTimeToReloadConfig()), true, false, true, &b)
	writeParamAsJsonString("Alloc", fmtAlloc(st.Alloc), true, false, true, &b)
	writeParamAsJsonString("TotalAlloc", fmtAlloc(st.TotalAlloc), true, false, true, &b)
	writeParamAsJsonString("Sys", fmtAlloc(st.Sys), true, false, true, &b)
	writeParamAsJsonString("configName", configData.ConfigName, true, false, true, &b)
	writeParamAsJsonString("Log Dir", configData.GetLogDataPath(), true, false, true, &b)
	for n, v := range longRunning {
		if n == "error" {
			writeParamAsJsonString("Long Running Process", v, true, false, true, &b)
		} else {
			writeParamAsJsonString(n, v, true, false, true, &b)
		}
	}
	writeParamAsJsonString("Log File", logFileName, true, false, false, &b)
	b.WriteRune('}')
	b.WriteRune('}')
	return b.Bytes()
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
	b.WriteString(cleanStringForJson(reason))
	b.WriteString("\"}")
	return b.Bytes()
}

func cleanStringForJson(s string) string {
	var buffer bytes.Buffer
	for _, c := range s {
		if c == '"' {
			buffer.WriteRune('\'')
		} else {
			if c >= ' ' && c <= 127 && c != '$' {
				buffer.WriteRune(c)
			}
		}
	}
	return buffer.String()
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

func filesAsJson(ents []fs.DirEntry, params *UrlRequestParts, verbose func(string), path string) []byte {
	var buffer bytes.Buffer
	entLen := len(ents)
	buffer.WriteRune('{')
	writeJsonHeader(params, &buffer)
	buffer.WriteRune(',')
	writePathToJson(params.GetOptionalParam(PathParam, ""), PathParam, &buffer)
	buffer.WriteString("\"files\":[")
	bufLen := buffer.Len()
	count := 0
	for i := 0; i < entLen; i++ {
		e := ents[i]
		if filterDirNames(e, params.GetConfigFileFilter()) {
			writeSingleFileNameToJson(e, &buffer)
			bufLen = buffer.Len()
			buffer.WriteRune(',')
			count++
		}
	}
	buffer.Truncate(bufLen)
	buffer.WriteString("]}")
	if verbose != nil {
		verbose(fmt.Sprintf("List Files:%s Returned[%d]", path, count))
	}

	return buffer.Bytes()
}

func listDirectoriesAsJson(dir string, param *UrlRequestParts, verbose func(string), path string) []byte {
	list := &[]string{}
	listDirectoriesRec(dir, dir+string(os.PathSeparator), param.GetConfigFileFilter(), list)
	listLen := len(*list)
	var buffer bytes.Buffer
	buffer.WriteRune('{')
	writeJsonHeader(param, &buffer)
	buffer.WriteString(",\"paths\":[")
	count := 0
	for i, s := range *list {
		buffer.WriteString("{\"name\":\"")
		buffer.WriteString(s)
		buffer.WriteString("\",")
		buffer.WriteString("\"encName\":\"")
		buffer.WriteString(encodeValue(s))
		buffer.WriteString("\"}")
		if i < (listLen - 1) {
			buffer.WriteRune(',')
		}
		count++
	}
	buffer.WriteString("]}")
	if verbose != nil {
		verbose(fmt.Sprintf("List Directories:%s Returned[%d]", path, count))
	}
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
	if (fileCount > 0) && strings.HasPrefix(path, root) {
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
		buffer.WriteString(encodeValue(pathUnencoded))
		buffer.WriteString("\"},")
	} else {
		buffer.WriteString("null,")
	}
}

func writeJsonHeader(param *UrlRequestParts, buffer *bytes.Buffer) {
	writeErrorAsJsonString(false, buffer)
	writeParamAsJsonString(UserParam, param.GetOptionalParam(UserParam, ""), true, true, false, buffer)
	writeParamAsJsonString(LocationParam, param.GetOptionalParam(LocationParam, ""), true, true, false, buffer)
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
	buffer.WriteString(encodeValue(file.Name()))
	buffer.WriteString("\"}}")
}

func writeParamAsJsonString(key string, value string, quoted bool, commaAtStart, commaAtEnd bool, buffer *bytes.Buffer) {
	if value != "" {
		if commaAtStart {
			buffer.WriteRune(',')
		}
		buffer.WriteRune('"')
		buffer.WriteString(key)
		buffer.WriteString("\":")
		if quoted {
			buffer.WriteRune('"')
		}
		buffer.WriteString(value)
		if quoted {
			buffer.WriteRune('"')
		}
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
