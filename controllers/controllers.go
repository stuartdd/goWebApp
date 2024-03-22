package controllers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"stuartdd.com/config"
	"stuartdd.com/runCommand"
)

type ResponseData struct {
	Status    int
	content   []byte
	Header    map[string][]string
	MimeType  string
	shouldLog bool
}

func (p *ResponseData) ToString() string {
	var buffer bytes.Buffer
	buffer.WriteString("ResponseData, ")
	buffer.WriteString(fmt.Sprintf("Status:%d", p.Status))
	buffer.WriteString(", ")
	buffer.WriteString(fmt.Sprintf("Content-Length:%d", p.ContentLength()))
	buffer.WriteString(", ")
	buffer.WriteString(fmt.Sprintf("Content-Type:%s", config.LookupContentType(p.MimeType)))
	return buffer.String()
}

func NewResponseData(status int) *ResponseData {
	rd := &ResponseData{
		Status:    status,
		Header:    make(map[string][]string),
		content:   make([]byte, 0),
		shouldLog: false,
		MimeType:  "json",
	}
	if rd.IsError() {
		rd.SetShouldLog()
	}
	return rd
}

func (p *ResponseData) ContentLength() int {
	return len(p.content)
}

func (p *ResponseData) Content() []byte {
	return p.content
}

func (p *ResponseData) ContentLimit(n int) []byte {
	if len(p.content) <= n {
		return p.content
	}
	return p.content[0:n]
}

func (p *ResponseData) IsError() bool {
	if p.Status >= 200 && p.Status < 300 {
		return false
	}
	return true
}

func (p *ResponseData) WithContentBytesJson(content []byte) *ResponseData {
	p.content = content
	return p
}

func (p *ResponseData) SetShouldLog() *ResponseData {
	p.shouldLog = true
	return p
}

func (p *ResponseData) GetShouldLog() bool {
	return p.shouldLog
}

func (p *ResponseData) WithContentReasonAsJson(reason string, error bool) *ResponseData {
	p.content = statusAsJson(p.Status, reason, error)
	return p
}

func (p *ResponseData) WithContentMapJson(data map[string]interface{}) *ResponseData {
	jsonData, err := json.Marshal(data)
	if err != nil {
		p.content = statusAsJson(http.StatusUnprocessableEntity, "data mapping to json failed", true)
	} else {
		p.content = jsonData
	}
	return p
}

func (p *ResponseData) WithMimeType(mimeType string) *ResponseData {
	p.MimeType = mimeType
	return p
}

type Handler interface {
	Submit() *ResponseData
}

type ReadFileHandler struct {
	parameters *config.Parameters
}

type PostFileHandler struct {
	parameters *config.Parameters
	request    *http.Request
}

type DirHandler struct {
	parameters *config.Parameters
	listFiles  bool
}

type TreeHandler struct {
	parameters *config.Parameters
}

type ExecHandler struct {
	parameters *config.Parameters
	createMap  func([]byte, []byte, int) map[string]interface{}
}

func NewReadFileHandler(parameters map[string]string, configData *config.ConfigData) Handler {
	return &ReadFileHandler{
		parameters: config.NewParameters(parameters, configData),
	}
}

func (p *ReadFileHandler) Submit() *ResponseData {
	file, err := p.parameters.UserLocFilePath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("File not found", true)
	}

	// s, _ := filepath.Abs(file)
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

func NewDirHandler(parameters map[string]string, configData *config.ConfigData, listFiles bool) Handler {
	return &DirHandler{
		parameters: config.NewParameters(parameters, configData),
		listFiles:  listFiles,
	}
}

func (p *DirHandler) Submit() *ResponseData {
	var path string
	var err error

	file, err := p.parameters.UserLocPath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Dir not found", true)
	}

	if p.parameters.HasParam("path") {
		pathByte, err := base64.StdEncoding.DecodeString(p.parameters.GetParam("path"))
		if err == nil {
			path = string(pathByte)
			file = filepath.Join(file, string(path))
		} else {
			return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Invalid path encoding", true)
		}
	} else {
		path = ""
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
		return NewResponseData(http.StatusOK).WithContentBytesJson(filesAsJson(entries, p.parameters.FilterFiles(), path))
	} else {

		return NewResponseData(http.StatusOK).WithContentBytesJson(listDirectoriesAsJson(file, p.parameters.FilterFiles()))
	}
}

func NewTreeHandler(parameters map[string]string, configData *config.ConfigData) Handler {
	return &TreeHandler{
		parameters: config.NewParameters(parameters, configData),
	}
}

func (p *TreeHandler) Submit() *ResponseData {
	file, err := p.parameters.UserLocPath()
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
	return NewResponseData(http.StatusOK).WithContentBytesJson(treeAsJson(root)).WithMimeType("json")
}

func NewPostFileHandler(parameters map[string]string, configData *config.ConfigData, r *http.Request) Handler {
	return &PostFileHandler{
		parameters: config.NewParameters(parameters, configData),
		request:    r,
	}
}

func (p *PostFileHandler) Submit() *ResponseData {
	dir, err := p.parameters.UserLocPath()
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
	file, err := p.parameters.UserLocFilePath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("File name not found", true)
	}
	err = os.WriteFile(file, body, 0644)
	if err != nil {
		return NewResponseData(http.StatusUnprocessableEntity).WithContentReasonAsJson("Failed to save data", true)
	}

	return NewResponseData(http.StatusAccepted).WithContentReasonAsJson("File saved", false)
}

func NewExecHandler(parameters map[string]string, configData *config.ConfigData, createMapFunc func([]byte, []byte, int) map[string]interface{}) Handler {
	return &ExecHandler{
		parameters: config.NewParameters(parameters, configData),
		createMap:  createMapFunc,
	}
}

func (p *ExecHandler) Submit() *ResponseData {
	execInfo, err := p.parameters.UserExec()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Exec not found", true)
	}
	execData := runCommand.NewExecData(execInfo.Cmd, execInfo.Dir, execInfo.GetOutLogFile(), execInfo.GetErrLogFile(), func(r []rune) string {
		userData := p.parameters.GetUserData()
		if userData == nil {
			return p.parameters.SubstituteFromMap(r, map[string]string{})
		}
		return p.parameters.SubstituteFromMap(r, userData.Env)
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

func treeAsJson(root *TreeDirNode) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("{\"error\":false, \"tree\":")
	buffer.WriteString(string(root.ToJson(false)))
	buffer.WriteString("}")
	return buffer.Bytes()
}

func filesAsJson(ents []fs.DirEntry, filter []string, path string) []byte {
	var buffer bytes.Buffer
	entLen := len(ents)
	buffer.WriteString("{\"error\":false, ")
	buffer.WriteString("\"path\":")
	pathToJson(path, &buffer)
	buffer.WriteString("\"files\":[")
	for i := 0; i < entLen; i++ {
		e := ents[i]
		if filterDirNames(e, filter) {
			singleFileToJson(e, &buffer)
			if i < (entLen - 1) {
				buffer.WriteRune(',')
			}
		}
	}
	buffer.WriteString("]}")
	return buffer.Bytes()
}

func pathToJson(path string, buffer *bytes.Buffer) {
	if path == "" {
		buffer.WriteString("null,")
	} else {
		buffer.WriteString("{\"name\":\"")
		buffer.WriteString(path)
		buffer.WriteString("\", \"encName\":\"")
		buffer.WriteString(base64.StdEncoding.EncodeToString([]byte(path)))
		buffer.WriteString("\"},")
	}
}

func singleFileToJson(file fs.DirEntry, buffer *bytes.Buffer) {
	buffer.WriteString("{\"size\": ")
	buffer.WriteString(strconv.Itoa(0))
	buffer.WriteString(",\"name\":{\"name\":\"")
	buffer.WriteString(file.Name())
	buffer.WriteString("\", \"encName\":\"")
	buffer.WriteString(base64.StdEncoding.EncodeToString([]byte(file.Name())))
	buffer.WriteString("\"}}")
}

func listDirectoriesAsJson(dir string, filter []string) []byte {
	list := &[]string{}
	listDirectoriesRec(dir, dir+string(os.PathSeparator), filter, list)
	listLen := len(*list)
	var buffer bytes.Buffer
	buffer.WriteString("{\"error\":false, \"paths\":[")
	for i, s := range *list {
		buffer.WriteString("{\"name\":\"")
		buffer.WriteString(s)
		buffer.WriteString("\",")
		buffer.WriteString("\"encName\":\"")
		buffer.WriteString(base64.StdEncoding.EncodeToString([]byte(s)))
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

type TreeDirNode struct {
	Name string         `json:"name"`
	Subs []*TreeDirNode `json:"subs,omitempty"`
}

func NewTreeNode(name string) *TreeDirNode {
	return &TreeDirNode{
		Name: name,
		Subs: nil,
	}
}

func (p *TreeDirNode) ToJson(indented bool) []byte {
	return p.toJson(0, indented)
}

func (p *TreeDirNode) AddPath(path string) error {
	return p.addPath(strings.Split(path, "/"))
}

func (p *TreeDirNode) Len() int {
	if p.Subs == nil {
		return 0
	}
	return len(p.Subs)
}

/*
	  Could use json.Marshal(tn) to serialise but this is faster
	    Marshal 5..8 microseconds
		ToJson 0..1 microseconds
		See controllers_test
*/

// --- 120 -- 012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789
const tabs = "                                                                                                                        "
const namePrefix = "{\"name\":\""
const subsPrefix = "\"subs\":["

func (p *TreeDirNode) toJson(tab int, indented bool) []byte {
	var buffer bytes.Buffer
	tabStr := ""
	pad := ""
	if indented {
		if (tab * 2) < len(tabs) {
			tabStr = "\n" + tabs[0:tab*2]
		} else {
			tabStr = "\n" + tabs
		}
		pad = " "
	}
	if indented {
		buffer.WriteString(tabStr)
	}
	buffer.WriteString(namePrefix)
	buffer.WriteString(p.Name)

	subC := p.Len()
	if subC > 0 {
		buffer.WriteString("\",")
		if indented {
			buffer.WriteString(tabStr)
			buffer.WriteString(pad)
		}
		buffer.WriteString(subsPrefix)
		for i := 0; i < subC; i++ {
			buffer.Write(p.Subs[i].toJson(tab+1, indented))
			if i <= subC-2 {
				buffer.WriteString(",")
			}
		}
		if indented {
			buffer.WriteString(tabStr)
			buffer.WriteString(pad)
		}
		buffer.WriteString("]")
		if indented {
			buffer.WriteString(tabStr)
		}
		buffer.WriteString("}")
	} else {
		buffer.WriteString("\"}")
	}
	return buffer.Bytes()

}

func findInSubs(subs []*TreeDirNode, name string) *TreeDirNode {
	if subs == nil {
		return nil
	}
	for i := 0; i < len(subs); i++ {
		if subs[i].Name == name {
			return subs[i]
		}
	}
	return nil
}

func (p *TreeDirNode) addPath(names []string) error {
	pp := p
	for i := 0; i < len(names); i++ {
		n := names[i]
		if len(n) > 0 {
			if strings.HasPrefix(n, ".") {
				return fmt.Errorf("not added")
			}
			su := findInSubs(pp.Subs, n)
			if su == nil {
				su = NewTreeNode(n)
				if pp.Subs == nil {
					pp.Subs = make([]*TreeDirNode, 0)
				}
				pp.Subs = append(pp.Subs, su)
				pp = su
			} else {
				pp = su
			}
		}
	}
	return nil
}
