package controllers

import (
	"bytes"
	"fmt"
)

type ResponseData struct {
	Status      int
	content     []byte
	Header      map[string][]string
	ContentType string
}

func (p *ResponseData) ToString() string {
	var buffer bytes.Buffer
	buffer.WriteString("ResponseData, ")
	buffer.WriteString(fmt.Sprintf("Status:%d", p.Status))
	buffer.WriteString(", ")
	buffer.WriteString(fmt.Sprintf("Content-Type:%s", p.ContentType))
	return buffer.String()
}

func NewResponseData(status int) *ResponseData {
	return &ResponseData{
		Status:      status,
		Header:      make(map[string][]string),
		content:     make([]byte, 0),
		ContentType: "application/json",
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

func (p *ResponseData) WithContent(content string, contentType string) *ResponseData {
	p.content = []byte(content)
	if contentType != "" {
		p.ContentType = contentType
	}
	return p
}

type Handler interface {
	Submit() *ResponseData
}

type SimpleResponse struct {
	status  int
	reason  string
	content string
}

func NewErrorResponse(status int, reason string) *SimpleResponse {
	return &SimpleResponse{
		status:  status,
		reason:  reason,
		content: fmt.Sprintf("{\"status\":%d, \"error\":\"%s\"}", status, reason),
	}
}

func NewSimpleResponse(status int, content string) *SimpleResponse {
	return &SimpleResponse{
		status:  status,
		reason:  "",
		content: content,
	}
}

func (p *SimpleResponse) Submit() *ResponseData {
	return NewResponseData(p.status).WithContent(p.content, "")
}
