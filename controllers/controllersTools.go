package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"stuartdd.com/config"
)

const UserParam = "user"
const LocationParam = "loc"
const PathParam = "path"
const NameParam = "name"
const ExecParam = "exec"
const ErrorParam = "error"

type UrlRequestParts struct {
	parameters map[string]string
	Query      map[string][]string
	Header     map[string][]string
	config     *config.ConfigData
}

func NewUrlRequestParts(config *config.ConfigData) *UrlRequestParts {
	return &UrlRequestParts{
		parameters: make(map[string]string),
		Query:      make(map[string][]string),
		Header:     make(map[string][]string),
		config:     config,
	}
}
func (p *UrlRequestParts) GetConfigFileFilter() []string {
	return p.config.GetFilesFilter()
}

func (p *UrlRequestParts) WithQuery(q map[string][]string) *UrlRequestParts {
	p.Query = q
	return p
}

func (p *UrlRequestParts) WithHeader(h map[string][]string) *UrlRequestParts {
	p.Header = h
	return p
}

func (p *UrlRequestParts) WithParameters(params map[string]string) *UrlRequestParts {
	p.parameters = params
	return p
}

func (p *UrlRequestParts) GetParam(key string) string {
	v, ok := p.parameters[key]
	if ok {
		return v
	}
	panic(fmt.Errorf("url parameter '%s' is missing", key))
}

func (p *UrlRequestParts) HasParam(key string) bool {
	_, ok := p.parameters[key]
	return ok
}

func (p *UrlRequestParts) SetParam(key string, value string) {
	p.parameters[key] = value
}

func (p *UrlRequestParts) GetOptionalParam(key string) string {
	v, ok := p.parameters[key]
	if ok {
		return v
	}
	return ""
}

func (p *UrlRequestParts) GetUser() string {
	return p.GetParam(UserParam)
}

func (p *UrlRequestParts) GetPath() string {
	return p.GetParam(PathParam)
}

func (p *UrlRequestParts) GetLocation() string {
	return p.GetParam(LocationParam)
}

func (p *UrlRequestParts) GetName() string {
	return p.GetParam(NameParam)
}

func (p *UrlRequestParts) GetExecId() string {
	return p.GetParam(ExecParam)
}

func (p *UrlRequestParts) SubstituteFromMap(cmd []rune, includeLocations bool) string {
	return p.config.SubstituteFromMap(cmd, p.config.GetUserEnv(p.GetUser(), includeLocations))
}

func (p *UrlRequestParts) GetUserLocNamePath() (file string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			file = ""
		}
	}()
	pa, err := p.GetUserLocPath()
	return filepath.Join(pa, p.GetName()), err
}

func (p *UrlRequestParts) GetUserLocPath() (path string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			path = ""
		}
	}()
	return p.config.GetUserLocPath(p.GetUser(), p.GetLocation())
}

func (p *UrlRequestParts) GetUserExecInfo() (path *config.ExecInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			path = nil
		}
	}()
	return p.config.GetUserExecInfo(p.GetUser(), p.GetExecId())
}

type ResponseData struct {
	Status    int
	content   []byte
	Header    map[string][]string
	MimeType  string
	shouldLog bool
}

func (p *ResponseData) String() string {
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
