package controllers

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/runCommand"
)

type FileInfo struct {
	modTime time.Time
	path    string
	size    int64
}

type Handler interface {
	Submit() *ResponseData
}

type StaticFileHandler struct {
	filePath    []string
	urlParts    *UrlRequestParts
	verboseFunc func(string)
}

func NewStaticFileHandler(file []string, urlParts *UrlRequestParts, verboseFunc func(string)) *StaticFileHandler {
	return &StaticFileHandler{
		filePath:    file,
		urlParts:    urlParts,
		verboseFunc: verboseFunc,
	}
}

func (p *StaticFileHandler) Submit() *ResponseData {
	list := []string{p.urlParts.config.GetServerStaticRoot()}
	list = append(list, p.filePath...)
	fullFile := filepath.Join(list...)

	stats, err := os.Stat(fullFile)
	if err != nil {
		panic(NewControllerError("File not found", http.StatusNotFound, fmt.Sprintf("Static File Error:%s", err.Error())))
	}
	if stats.IsDir() {
		panic(NewControllerError("Is a directory", http.StatusForbidden, fmt.Sprintf("Static file %s is a Directory", fullFile)))
	}
	fileContent, err := os.ReadFile(fullFile)
	if err != nil {
		panic(NewControllerError("File could not be read", http.StatusUnprocessableEntity, fmt.Sprintf("Static File Error:%s", err.Error())))
	}
	if p.verboseFunc != nil { // Only do this if abs necessary as Sprintf does not need to be done
		p.verboseFunc(fmt.Sprintf("Static Read File:%s Mime[%s] Len[%d]", fullFile, config.LookupContentType(p.filePath[len(p.filePath)-1]), len(fileContent)))
	}
	if p.urlParts.config.IsTemplating() {
		td := p.urlParts.config.GetTemplateData()
		if td.ShouldTemplate(list[len(list)-1]) {
			fileContent = []byte(p.urlParts.config.SubstituteFromMap([]byte(string(fileContent)), td.Data(*p.urlParts.GetCachedMap())))
		}
	}
	return NewResponseData(http.StatusOK).WithContentBytes(fileContent).WithMimeType(p.filePath[len(p.filePath)-1])
}

type ReadFileHandler struct {
	parameters *UrlRequestParts
	configData *config.ConfigData
	verbose    func(string)
	delete     bool
}

func NewDeleteFileHandler(urlParts *UrlRequestParts, configData *config.ConfigData, verboseFunc func(string)) Handler {
	return &ReadFileHandler{
		parameters: urlParts,
		configData: configData,
		verbose:    verboseFunc,
		delete:     true,
	}
}

func NewReadFileHandler(urlParts *UrlRequestParts, configData *config.ConfigData, verboseFunc func(string)) Handler {
	return &ReadFileHandler{
		parameters: urlParts,
		configData: configData,
		verbose:    verboseFunc,
		delete:     false,
	}
}

func (p *ReadFileHandler) Submit() *ResponseData {
	file := p.parameters.GetUserLocPath(true, p.parameters.GetQueryAsBool("thumbnail", false), p.parameters.GetQueryAsBool("base64", false))

	stats, err := os.Stat(file)
	if err != nil {
		panic(NewControllerError("File not found", http.StatusNotFound, err.Error()))
	}
	if stats.IsDir() {
		panic(NewControllerError("Is a directory", http.StatusForbidden, fmt.Sprintf("%s is a Directory", file)))
	}
	if p.delete {
		err = os.Remove(file)
		if err != nil {
			panic(NewControllerError("File could not be deleted", http.StatusUnprocessableEntity, err.Error()))
		}
		_, err = os.Stat(file)
		if err == nil {
			panic(NewControllerError("File was not be deleted", http.StatusUnprocessableEntity, fmt.Sprintf("File %s was not deleted", file)))
		}
		return NewResponseData(http.StatusAccepted).WithContentWithCauseAsJson("File deleted OK")
	} else {
		fileContent, err := os.ReadFile(file)
		if err != nil {
			panic(NewControllerError("File could not be read", http.StatusUnprocessableEntity, err.Error()))
		}
		if p.verbose != nil { // Only do this if abs necessary as Sprintf does not need to be done
			p.verbose(fmt.Sprintf("Read File:%s Mime[%s] Len[%d]", file, config.LookupContentType(p.parameters.GetName()), len(fileContent)))
		}
		return NewResponseData(http.StatusOK).WithContentBytes(fileContent).WithMimeType(p.parameters.GetName())
	}
}

type DirHandler struct {
	parameters *UrlRequestParts
	listFiles  bool
	verbose    func(string)
}

func NewDirHandler(urlRequestData *UrlRequestParts, configData *config.ConfigData, listFiles bool, verboseFunc func(string)) Handler {
	return &DirHandler{
		parameters: urlRequestData,
		listFiles:  listFiles,
		verbose:    verboseFunc,
	}
}

func (p *DirHandler) Submit() *ResponseData {
	var err error
	file := p.parameters.GetUserLocPath(false, false, p.parameters.GetQueryAsBool("base64", false))
	stats, err := os.Stat(file)
	if err != nil {
		panic(NewControllerError("Dir not found", http.StatusNotFound, err.Error()))
	}
	if !stats.IsDir() {
		panic(NewControllerError("Is NOT a directory", http.StatusForbidden, fmt.Sprintf("%s is NOT a Directory", file)))
	}
	index := p.parameters.GetQueryAsInt("index", -1)
	if p.listFiles {
		entries, err := os.ReadDir(file)
		if err != nil {
			panic(NewControllerError("Dir could not be read", http.StatusUnprocessableEntity, err.Error()))
		}
		// Panic Check Done
		return NewResponseData(http.StatusOK).WithContentBytes(listFilesAsJson(entries, p.parameters, p.verbose, file, index)).WithMimeType("json")
	} else {
		// Panic Check Done
		return NewResponseData(http.StatusOK).WithContentBytes(listDirectoriesAsJson(file, p.parameters, p.verbose, file)).WithMimeType("json")
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
	file := p.parameters.GetUserLocPath(false, false, p.parameters.GetQueryAsBool("base64", false))
	stats, err := os.Stat(file)
	if err != nil {
		panic(NewControllerError("Dir not found", http.StatusNotFound, err.Error()))
	}
	if !stats.IsDir() {
		panic(NewControllerError("Is NOT a directory", http.StatusForbidden, fmt.Sprintf("%s is NOT a Directory", file)))
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
		panic(NewControllerError("Dir could not be read", http.StatusUnprocessableEntity, err.Error()))
	}
	return NewResponseData(http.StatusOK).WithContentBytes(treeAsJson(root, p.parameters)).WithMimeType("json")
}

type PostFileHandler struct {
	parameters *UrlRequestParts
	request    *http.Request
	postToLog  bool
	verbose    func(string)
}

func NewPostFileHandler(urlParts *UrlRequestParts, configData *config.ConfigData, r *http.Request, postToLog bool, verboseFunc func(string)) Handler {
	return &PostFileHandler{
		parameters: urlParts,
		request:    r,
		verbose:    verboseFunc,
		postToLog:  postToLog,
	}
}

func (p *PostFileHandler) Submit() *ResponseData {
	dir := p.parameters.GetUserLocPath(false, false, p.parameters.GetQueryAsBool("base64", false))
	stats, err := os.Stat(dir)
	if err != nil {
		panic(NewControllerError("Dir not found", http.StatusNotFound, err.Error()))
	}
	if !stats.IsDir() {
		panic(NewControllerError("Is NOT a directory", http.StatusForbidden, fmt.Sprintf("PostFileHandler: %s is NOT a Directory", dir)))
	}
	body, err := io.ReadAll(p.request.Body)
	if err != nil {
		panic(NewControllerError("Failed to read posted data", http.StatusBadRequest, err.Error()))
	}
	file := p.parameters.GetUserLocPath(true, false, p.parameters.GetQueryAsBool("base64", false))

	action := p.parameters.GetOptionalQuery("action", "save")
	switch action {
	case "append":
		err = AppendFile(file, body, 0644)
		if err != nil {
			panic(NewControllerError("Failed to append data", http.StatusInternalServerError, fmt.Sprintf("File:%s Error:%s", file, err.Error())))
		}
	case "replace":
		err = os.WriteFile(file, body, 0644)
		if err != nil {
			panic(NewControllerError("Failed to save data", http.StatusInternalServerError, err.Error()))
		}
	default:
		_, err := os.Stat(file)
		if err == nil {
			panic(NewControllerError("File exists", http.StatusPreconditionFailed, fmt.Sprintf("File:%s already exists", file)))
		}
		err = os.WriteFile(file, body, 0644)
		if err != nil {
			panic(NewControllerError("Failed to save data", http.StatusInternalServerError, err.Error()))
		}
	}
	if p.verbose != nil { // Only do this if abs necessary as Sprintf does not need to be done
		p.verbose(fmt.Sprintf("File action[%s]:%s [%d] bytes", action, file, len(body)))
	}
	return NewResponseData(http.StatusAccepted).WithContentWithCauseAsJson(fmt.Sprintf("File:Action:%s %s", action, file))
}

func AppendFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

type ExecHandler struct {
	parameters       *UrlRequestParts
	makeExecResponse func(string, string, []byte, []byte, int) []byte
	verbose          func(string)
	isVerbose        bool
	log              func(string)
	execInfo         *config.ExecInfo
}

func NewExecHandler(urlParts *UrlRequestParts, configData *config.ConfigData, makeExecResponse func(string, string, []byte, []byte, int) []byte, logFunc func(string), verboseFunc func(string)) Handler {
	return &ExecHandler{
		parameters:       urlParts,
		makeExecResponse: makeExecResponse,
		verbose:          verboseFunc,
		log:              logFunc,
	}
}

func (p *ExecHandler) Submit() *ResponseData {
	userId := p.parameters.GetOptionalUser(AdminName)
	execId := p.parameters.GetExecId()
	p.execInfo = p.parameters.GetExecInfo()
	action := p.parameters.GetOptionalQuery("action", "start")
	if action == "stop" {
		pid := runCommand.FindProcessIdWithName(p.execInfo.Cmd[0])
		if pid < 100 {
			panic(NewControllerError("Process not found", http.StatusExpectationFailed, fmt.Sprintf("Command:%s", p.execInfo.Cmd[0])))
		}
		runCommand.KillrocessWithPid(pid)
		dataMap := map[string]interface{}{"error": false, "id": execId, "status": http.StatusOK, "msg": http.StatusText(http.StatusOK), "rc": 0, "stdOut": "Stop process complete", "stdErr": ""}
		return NewResponseData(http.StatusOK).WithContentMapAsJson(dataMap).SetHasErrors(false)
	}

	execData := runCommand.NewExecData(p.execInfo.Cmd, p.execInfo.Dir, p.execInfo.GetOutLogFile(), p.execInfo.GetErrLogFile(), execId, p.execInfo.StartLTSFile, p.execInfo.Detached, p.execInfo.CanStop, p.log, func(r []byte) string {
		return p.parameters.SubstituteFromCachedMap(r)
	})

	if p.isVerbose { // Only do this if abs necessary as execData.String() does not need to be done
		p.verbose(execData.String())
	}

	stdOut, stdErr, code := execData.RunSystemProcess()

	if p.execInfo.LogOutFile != "" && len(stdOut) > 0 {
		of := p.parameters.config.SubstituteFromMap([]byte(p.execInfo.LogOutFile), p.parameters.config.GetUserEnv(userId))
		err := os.WriteFile(filepath.Join(p.execInfo.LogDir, string(of)), stdOut, 0644)
		if err != nil {
			panic(NewControllerError("Failed to write stdOut to log", http.StatusInternalServerError, fmt.Sprintf("Failed to write stdOut. RC:%d Error:%s", code, err.Error())))
		}
	}

	if p.execInfo.LogErrFile != "" && len(stdErr) > 0 {
		of := p.parameters.config.SubstituteFromMap([]byte(p.execInfo.LogErrFile), p.parameters.config.GetUserEnv(userId))
		err := os.WriteFile(filepath.Join(p.execInfo.LogDir, string(of)), stdErr, 0644)
		if err != nil {
			panic(NewControllerError("Failed to write stdErr to log", http.StatusInternalServerError, fmt.Sprintf("Failed to write stdErr. RC:%d Error:%s", code, err.Error())))
		}
	}

	if p.makeExecResponse == nil {
		return NewResponseData(p.execInfo.NzCodeReturns).WithContentFromExecAsJson(execId, code, p.execInfo.NzCodeReturns, stdOut, stdErr)
	}
	return NewResponseData(p.execInfo.NzCodeReturns).WithContentBytes(p.makeExecResponse(execId, p.execInfo.StdOutType, stdOut, stdErr, code))

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
	if len(l1) == 0 {
		panic(NewControllerError("No users are defined", http.StatusNotFound, "GetUsersAsMap: No un-hidden users found"))
	}
	m1["users"] = l1
	return m1
}

func GetLog(configData *config.ConfigData, offsetString string) *ResponseData {
	ld := configData.GetLogData()
	list := []*FileInfo{}
	filepath.Walk(ld.Path, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".log") {
			list = append(list, &FileInfo{modTime: info.ModTime(), path: path, size: info.Size()})
		}
		return nil
	})
	if len(list) == 0 {
		panic(NewControllerError("No log files were found", http.StatusNotFound, "GetLog: No log files were found"))
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].modTime.Before(list[j].modTime)
	})

	var b bytes.Buffer

	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		offset = 0
		b.WriteString("##! Offset '")
		b.WriteString(offsetString)
		b.WriteString("' is not an integer. Offset set to 0")
		b.WriteString("\n")
	}

	for i, v := range list {
		b.WriteString("##! Offset[")
		b.WriteString(fmt.Sprintf("%2d", len(list)-(i+1)))
		b.WriteString("] ")
		b.WriteString(v.path[len(ld.Path):])
		b.WriteString(" (")
		b.WriteString(strconv.Itoa(int(v.size)))
		b.WriteRune(')')
		b.WriteString("\n")
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= (len(list)) {
		offset = len(list) - 1
	}
	fileName := list[len(list)-(offset+1)].path
	b.WriteString("##! Displaying log File at Offset[")
	b.WriteString(fmt.Sprintf("%2d", offset))
	b.WriteString("] ")
	b.WriteString(fileName[len(ld.Path):])
	b.WriteString("\n##! -- --\n\n")

	fileContent, err := os.ReadFile(fileName)
	if err != nil {
		panic(NewControllerError("Log file read error", http.StatusUnprocessableEntity, fmt.Sprintf("GetLog: Log file could not be read. Error:%s", err.Error())))
	}
	b.WriteString(string(fileContent))
	return NewResponseData(http.StatusOK).WithContentBytes(b.Bytes()).WithMimeType("log")
}

func GetFaveIcon(configData *config.ConfigData) *ResponseData {
	if configData.GetFaviconIcoPath() == "" {
		panic(NewControllerError("favicon.ico not configured", http.StatusNotFound, "FaviconIcoPath is not defined in config file"))
	}
	fileContent, err := os.ReadFile(configData.GetFaviconIcoPath())
	if err != nil {
		panic(NewControllerError("favicon.ico not found", http.StatusNotFound, err.Error()))
	}
	return NewResponseData(http.StatusOK).WithContentBytes(fileContent).WithMimeType("ico")
}

func GetOSFreeData(configData *config.ConfigData) (res string) {
	execInfo := configData.GetExecInfo("free")
	execData := runCommand.NewExecData(execInfo.Cmd, execInfo.Dir, execInfo.GetOutLogFile(), execInfo.GetErrLogFile(), "Run system 'free' command", execInfo.StartLTSFile, false, false, nil, nil)
	stdOut, _, code := execData.RunSystemProcess()
	if code != 0 {
		panic(NewControllerError("Get System status via 'free' exec returned nz code", http.StatusFailedDependency, fmt.Sprintf("RC:%d", code)))
	}
	return string(stdOut)
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
