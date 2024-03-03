package controllers

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
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
	buffer.WriteString(fmt.Sprintf("Content-Type:%s", config.LookupContentType(p.MimeType)))
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
