package controllers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/pictures"
)

const UserParam = "user"
const LocationParam = "loc"
const PathParam = "path"
const NameParam = "name"
const ExecParam = "exec"
const ScriptParam = "script"
const ErrorParam = "error"
const AdminName = "admin"
const encodedValuePrefix = "X0X"

type UrlRequestParts struct {
	parameters map[string]string
	Query      map[string][]string
	Header     map[string][]string
	cache      *map[string]string
	config     *config.ConfigData
}

func NewUrlRequestParts(config *config.ConfigData) *UrlRequestParts {
	return &UrlRequestParts{
		parameters: make(map[string]string),
		Query:      make(map[string][]string),
		Header:     make(map[string][]string),
		cache:      nil,
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

func (p *UrlRequestParts) AsAdmin() *UrlRequestParts {
	p.parameters[UserParam] = AdminName
	return p
}

func (p *UrlRequestParts) WithUser(user string) *UrlRequestParts {
	p.parameters[UserParam] = user
	return p
}

func (p *UrlRequestParts) WithParameters(params map[string]string) *UrlRequestParts {
	p.parameters = params
	return p
}

func (p *UrlRequestParts) RenameParameter(old, new string) *UrlRequestParts {
	v := p.GetParam(old)
	p.RemoveParameter(old)
	p.parameters[new] = v
	return p
}

func (p *UrlRequestParts) RemoveParameter(name string) *UrlRequestParts {
	delete(p.parameters, name)
	return p
}

func (p *UrlRequestParts) WithParam(name string, value string) *UrlRequestParts {
	p.parameters[name] = value
	return p
}

func (p *UrlRequestParts) WithFile(file string) *UrlRequestParts {
	p.parameters["file"] = file
	return p
}

func (p *UrlRequestParts) GetCachedMap() *map[string]string {
	if p.cache == nil {
		m := map[string]string{}
		for n, v := range p.Header {
			if len(v[0]) != 0 {
				m[n] = decodeValue(v[0])
			}
		}
		for n, v := range p.parameters {
			m[n] = decodeValue(v)
		}
		for n, v := range p.Query {
			if len(v[0]) != 0 {
				m[n] = v[0] // Dont decrypt (convert from base64) query values.
			}
		}
		p.cache = &m
	}
	return p.cache
}

func (p *UrlRequestParts) GetQueryAsBool(key string, fallback bool) bool {
	var v string
	if fallback {
		v = p.GetOptionalQuery(key, "true")
	} else {
		v = p.GetOptionalQuery(key, "false")
	}
	s := strings.ToLower(v)
	if s == "true" || strings.HasPrefix(s, "y") {
		return true
	}
	if s == "false" || strings.HasPrefix(s, "n") {
		return false
	}
	return fallback
}

func (p *UrlRequestParts) GetOptionalQuery(key string, fallback string) string {
	v, ok := p.Query[key]
	if ok {
		if len(v) > 0 {
			return decodeValue(v[0])
		}
	}
	return fallback
}

func (p *UrlRequestParts) GetParam(key string) string {
	v, ok := p.parameters[key]
	if ok {
		return decodeValue(v)
	}
	panic(fmt.Errorf("url parameter not found: Name: '%s'", key))
}

func (p *UrlRequestParts) GetOptionalParam(key string, fallback string) string {
	v, ok := p.parameters[key]
	if ok {
		return decodeValue(v)
	}
	return fallback
}

func (p *UrlRequestParts) HasParam(key string) bool {
	_, ok := p.parameters[key]
	return ok
}

func (p *UrlRequestParts) SetParam(key string, value string) {
	p.parameters[key] = value
}

func (p *UrlRequestParts) GetUser() string {
	return p.GetParam(UserParam)
}

func (p *UrlRequestParts) GetScript() string {
	return p.GetParam(ScriptParam)
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

func (p *UrlRequestParts) GetUserExecInfo() *config.ExecInfo {
	return p.config.GetUserExecInfo(p.GetUser(), p.GetExecId())
}

func (p *UrlRequestParts) GetExecId() string {
	return p.GetParam(ExecParam)
}

func (p *UrlRequestParts) SubstituteFromUserEnv(cmd []byte) string {
	return p.config.SubstituteFromMap(cmd, p.config.GetUserEnv(p.GetUser()))
}

func (p *UrlRequestParts) SubstituteFromCachedMap(cmd []byte) string {
	return p.config.SubstituteFromMap(cmd, *p.GetCachedMap())
}

func (p *UrlRequestParts) GetUserLocPath(withName bool, asThumbnail bool, isBase64 bool) string {
	ulp := p.config.GetUserLocPath(p.GetUser(), p.GetLocation())
	if p.HasParam(PathParam) {
		pat := p.GetParam(PathParam)
		if isBase64 {
			patBytes, err := base64.StdEncoding.DecodeString(pat)
			if err != nil {
				ulp = filepath.Join(ulp, pat)
			} else {
				ulp = filepath.Join(ulp, string(patBytes))
			}
		} else {
			ulp = filepath.Join(ulp, pat)
		}
	}
	if withName {
		if p.HasParam(NameParam) {
			np := p.GetParam(NameParam)
			if isBase64 {
				npBytes, err := base64.StdEncoding.DecodeString(np)
				if err == nil {
					np = string(npBytes)
				}
			}
			if asThumbnail {
				ulp = filepath.Join(ulp, pictures.UnFormatThumbNail(p.config.GetThumbnailFormat(), np))
			} else {
				ulp = filepath.Join(ulp, np)
			}
		}
	}
	return ulp
}

type ResponseData struct {
	Status    int
	content   []byte
	Header    map[string][]string
	MimeType  string
	hasErrors bool
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
		hasErrors: false,
		MimeType:  "json",
	}
	if rd.IsError() {
		rd.SetHasErrors(true)
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

func (p *ResponseData) WithContentBytes(content []byte) *ResponseData {
	p.content = content
	return p
}

func (p *ResponseData) SetHasErrors(should bool) *ResponseData {
	p.hasErrors = should
	return p
}

func (p *ResponseData) GetHasErrors() bool {
	return p.hasErrors
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

func decodeValue(encodedValue string) string {
	if encodedValue == "" {
		return ""
	}
	if strings.HasPrefix(encodedValue, encodedValuePrefix) {
		decoded, err := base64.StdEncoding.DecodeString(encodedValue[len(encodedValuePrefix):])
		if err != nil {
			return encodedValue
		}
		return string(decoded)
	}
	return encodedValue
}

func encodeValue(unEncoded string) string {
	if unEncoded == "" {
		return ""
	}
	return encodedValuePrefix + base64.StdEncoding.EncodeToString([]byte(unEncoded))
}
