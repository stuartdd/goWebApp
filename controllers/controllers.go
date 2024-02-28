package controllers

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

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

func (p *ResponseData) Content() string {
	return string(p.content)
}

func (p *ResponseData) IsError() bool {
	if p.Status >= 200 && p.Status < 300 {
		return false
	}
	return true
}

func (p *ResponseData) WithContent(content string) *ResponseData {
	p.content = []byte(content)
	return p
}

func (p *ResponseData) WithContentBytes(content []byte) *ResponseData {
	p.content = content
	return p
}

func (p *ResponseData) WithContentStatusJson(reason string) *ResponseData {
	p.content = []byte(StatusAsJson(p.Status, reason))
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
	configData *config.ConfigData
}

func NewFileHandler(parameters map[string]string, configData *config.ConfigData) *FileHandler {
	return &FileHandler{
		parameters: config.NewParameters(parameters),
		configData: configData,
	}
}

func (p *FileHandler) Submit() *ResponseData {
	file, err := p.configData.UserDataFile(p.parameters)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("File not found")
	}
	pwd, _ := os.Getwd()
	fmt.Printf("PWD:%s%s", pwd, file)

	if _, err := os.Stat(file); err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("File not found")
	}

	fileContent, err := os.ReadFile(file)
	if err != nil {
		return NewResponseData(http.StatusNotFound).WithContentStatusJson("File could not be read")
	}
	return NewResponseData(http.StatusOK).WithContentBytes(fileContent).WithMimeType(p.parameters.GetName())
}

func StatusAsJson(status int, reason string) string {
	return fmt.Sprintf("{\"status\":%d, \"msg\":\"%s\", \"reason\":\"%s\"}", status, http.StatusText(status), reason)
}
