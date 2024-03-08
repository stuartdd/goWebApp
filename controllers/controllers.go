package controllers

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"stuartdd.com/config"
)

type ResponseData struct {
	Status   int
	content  []byte
	Header   map[string][]string
	MimeType string
}

func (p *ResponseData) ToString() string {
	var buffer bytes.Buffer
	buffer.WriteString("ResponseData, ")
	buffer.WriteString(fmt.Sprintf("Status:%d", p.Status))
	buffer.WriteString(", ")
	buffer.WriteString(fmt.Sprintf("Content-Length:%d", p.ContentLength()))
	buffer.WriteString(", ")
	buffer.WriteString(fmt.Sprintf("Content-Type:%s", config.LookupContentType(p.MimeType, "")))
	return buffer.String()
}

func NewResponseData(status int) *ResponseData {
	return &ResponseData{
		Status:   status,
		Header:   make(map[string][]string),
		content:  make([]byte, 0),
		MimeType: "json",
	}
}
func (p *ResponseData) ContentLength() int {
	return len(p.content)
}

func (p *ResponseData) Content() []byte {
	return p.content
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

func (p *ResponseData) WithContentStatusJson(reason string) *ResponseData {
	p.content = StatusAsJson(p.Status, reason)
	return p
}

func (p *ResponseData) WithMimeType(mimeType string) *ResponseData {
	p.MimeType = mimeType
	return p
}

type Handler interface {
	Submit() *ResponseData
}

type FileHandler struct {
	parameters *config.Parameters
}

type PostFileHandler struct {
	parameters *config.Parameters
	request    *http.Request
}

type DirHandler struct {
	parameters *config.Parameters
}

type TreeHandler struct {
	parameters *config.Parameters
}

func NewFileHandler(parameters map[string]string, configData *config.ConfigData) Handler {
	return &FileHandler{
		parameters: config.NewParameters(parameters, configData),
	}
}

func (p *FileHandler) Submit() *ResponseData {
	file, err := p.parameters.UserDataFile()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("File not found")
	}

	// s, _ := filepath.Abs(file)
	stats, err := os.Stat(file)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("File not found")
	}

	if stats.IsDir() {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Is not a file")
	}

	fileContent, err := os.ReadFile(file)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("File could not be read")
	}
	return NewResponseData(http.StatusOK).WithContentBytesJson(fileContent).WithMimeType(p.parameters.GetName())
}

func NewTreeHandler(parameters map[string]string, configData *config.ConfigData) Handler {
	return &TreeHandler{
		parameters: config.NewParameters(parameters, configData),
	}
}

func (p *TreeHandler) Submit() *ResponseData {
	file, err := p.parameters.UserDataPath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Dir not found")
	}
	stats, err := os.Stat(file)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Dir not found")
	}
	if !stats.IsDir() {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Is not a dir")
	}

	root := newTreeNode("fs")
	err = filepath.Walk(file,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasPrefix(path, ".") && !strings.HasPrefix(path, "_") && info.IsDir() {
				err := root.AddPath(path)
				if err == nil {
					fmt.Printf("ADD:%s\n", path)
				} else {
					fmt.Printf("---:%s\n", path)
				}
			}
			return nil
		})
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Dir cannot be read")
	}
	return NewResponseData(http.StatusOK).WithContentBytesJson(root.ToJson(false)).WithMimeType("json")
}

func NewDirHandler(parameters map[string]string, configData *config.ConfigData) Handler {
	return &DirHandler{
		parameters: config.NewParameters(parameters, configData),
	}
}

func (p *DirHandler) Submit() *ResponseData {
	file, err := p.parameters.UserDataPath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Dir not found")
	}
	stats, err := os.Stat(file)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Dir not found")
	}
	if !stats.IsDir() {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Is not a dir")
	}
	entries, err := os.ReadDir(file)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Dir cannot be read")
	}
	return NewResponseData(http.StatusOK).WithContentBytesJson(DirAsJson(entries, p.parameters.FilterFiles()))
}

func NewFilePostHandler(parameters map[string]string, configData *config.ConfigData, r *http.Request) Handler {
	return &PostFileHandler{
		parameters: config.NewParameters(parameters, configData),
		request:    r,
	}
}

func (p *PostFileHandler) Submit() *ResponseData {
	dir, err := p.parameters.UserDataPath()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Dir not found")
	}
	stats, err := os.Stat(dir)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Dir not found")
	}
	if !stats.IsDir() {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("Is not a dir")
	}
	body, err := io.ReadAll(p.request.Body)
	if err != nil {
		return NewResponseData(http.StatusUnprocessableEntity).WithContentStatusJson("Failed to read input")
	}
	file, err := p.parameters.UserDataFile()
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("File name not found")
	}
	err = os.WriteFile(file, body, 0644)
	if err != nil {
		return NewResponseData(http.StatusUnprocessableEntity).WithContentStatusJson("Failed to save data")
	}

	return NewResponseData(http.StatusAccepted).WithContentStatusJson("File saved")
}

func GetFaveIcon(configData *config.ConfigData) *ResponseData {
	if configData.FaviconIcoPath == "" {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("favicon.ico not defined")
	}
	fileContent, err := os.ReadFile(configData.FaviconIcoPath)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("favicon.ico not found")
	}
	return NewResponseData(http.StatusOK).WithContentBytesJson(fileContent).WithMimeType("ico")
}

func StatusAsJson(status int, reason string) []byte {
	var b bytes.Buffer
	b.WriteString("{\"status\":")
	b.WriteString(strconv.Itoa(status))
	b.WriteString(", \"msg\":\"")
	b.WriteString(http.StatusText(status))
	b.WriteString("\", \"reason\":\"")
	b.WriteString(reason)
	b.WriteString("\"}")
	return b.Bytes()
}

func DirAsJson(ent []fs.DirEntry, filter []string) []byte {
	var buffer bytes.Buffer
	entLen := len(ent)
	count := 0
	buffer.WriteString("{")
	for i := 0; i < entLen; i++ {
		e := ent[i]
		if filterDirNames(e, filter) {
			buffer.WriteString("\"file\":\"")
			buffer.WriteString(e.Name())
			buffer.WriteString("\"")
			buffer.WriteRune(',')
			count++
		}
	}
	if count > 0 {
		buffer.Truncate(buffer.Len() - 1)
	}
	buffer.WriteString("}")
	return buffer.Bytes()
}

func filterDirNames(e fs.DirEntry, filter []string) bool {
	if e.IsDir() {
		return false
	}
	n := strings.ToLower(e.Name())
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

type treeDirNode struct {
	name string
	subs []*treeDirNode
}

func newTreeNode(name string) *treeDirNode {
	return &treeDirNode{
		name: name,
		subs: make([]*treeDirNode, 0),
	}
}

func (p *treeDirNode) ToJson(indented bool) []byte {
	return p.toJson(0, indented)
}

func (p *treeDirNode) AddPath(path string) error {
	return p.addPath(strings.Split(path, "/"))
}

func (p *treeDirNode) Len() int {
	return len(p.subs)
}

// --- 120 -- 012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789
const tabs = "                                                                                                                        "

func (p *treeDirNode) toJson(tab int, indented bool) []byte {
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
	subC := len(p.subs)
	if indented {
		buffer.WriteString(tabStr)
	}
	buffer.WriteString("{\"name\":\"")
	buffer.WriteString(p.name)
	if subC > 0 {
		buffer.WriteString("\",")
		if indented {
			buffer.WriteString(tabStr)
			buffer.WriteString(pad)
		}
		buffer.WriteString("\"subs\":[")
		for i := 0; i < len(p.subs); i++ {
			buffer.Write(p.subs[i].toJson(tab+1, indented))
			if i <= len(p.subs)-2 {
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

func findInSubs(subs []*treeDirNode, name string) *treeDirNode {
	for i := 0; i < len(subs); i++ {
		if subs[i].name == name {
			return subs[i]
		}
	}
	return nil
}

func (p *treeDirNode) addPath(names []string) error {
	pp := p
	for i := 0; i < len(names); i++ {
		n := names[i]
		if len(n) > 0 {
			if strings.HasPrefix(n, ".") {
				return fmt.Errorf("not added")
			}
			su := findInSubs(pp.subs, n)
			if su == nil {
				su = newTreeNode(n)
				pp.subs = append(pp.subs, su)
				pp = su
			} else {
				pp = su
			}
		}
	}
	return nil
}
