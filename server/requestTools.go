package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RootUrlList struct {
	ids map[string]bool
}

func NewRootUrlList() *RootUrlList {
	return &RootUrlList{ids: map[string]bool{}}
}

func (rl *RootUrlList) AddUrlRequestMatcher(templateUrl string, reqType string, shouldLog bool) *urlRequestMatcher {
	return rl.add(newUrlRequestMatcher(templateUrl, reqType, shouldLog))
}

func (p *RootUrlList) HasRoot(name string) bool {
	_, ok := p.ids[name]
	return ok
}

func (p *RootUrlList) String() string {
	var buffer bytes.Buffer
	for n, _ := range p.ids {
		buffer.WriteString(n)
		buffer.WriteString(",")
	}
	return buffer.String()
}

func (p *RootUrlList) add(info *urlRequestMatcher) *urlRequestMatcher {
	if len(info.Parts) > 0 {
		p.ids[info.Parts[0]] = true
	}
	return info
}

type RequestInfo struct {
	method         string
	path           string
	query          string
	logFunc        func(string)
	verboseLogFunc func(string)
}

func NewRequestInfo(m, p, q string, lf, vlf func(string)) *RequestInfo {
	return &RequestInfo{
		method:         m,
		path:           p,
		query:          q,
		logFunc:        lf,
		verboseLogFunc: vlf,
	}
}

func (r *RequestInfo) String() string {
	if r.query == "" {
		return fmt.Sprintf("Req:  %s:%s", r.method, r.path)
	}
	return fmt.Sprintf("Req:  %s:%s?%s", r.method, r.path, r.query)
}

func (r *RequestInfo) Log(shouldLog bool) {
	if r.logFunc != nil && shouldLog {
		r.logFunc(r.String())
		return
	}
	if r.verboseLogFunc != nil {
		r.verboseLogFunc(r.String())
	}
}

type urlRequestMatcher struct {
	Parts     []string
	ReqType   string
	shouldLog bool
	Len       int
}

func newUrlRequestMatcher(templateUrl string, reqType string, shouldLog bool) *urlRequestMatcher {
	s := strings.Split(strings.TrimSpace(templateUrl), "/")
	if s[0] == "" {
		s = s[1:]
	}
	return &urlRequestMatcher{
		Parts:     s,
		ReqType:   strings.ToUpper(reqType),
		shouldLog: shouldLog,
		Len:       len(s),
	}
}

func (p *urlRequestMatcher) String() string {
	var buffer bytes.Buffer
	len := 0
	buffer.WriteRune('/')
	for _, v := range p.Parts {
		buffer.WriteString(v)
		len = buffer.Len()
		buffer.WriteRune('/')
	}
	buffer.Truncate(len)
	return fmt.Sprintf("Req:  %s:%s", p.ReqType, buffer.String())
}

func (p *urlRequestMatcher) Match(requestParts []string, reqType string, rqi *RequestInfo) (map[string]string, bool, bool) {
	if p.Len == 0 || p.Len != len(requestParts) {
		return nil, false, p.shouldLog
	}
	if p.Parts[0] != requestParts[0] {
		return nil, false, p.shouldLog
	}
	if p.ReqType != strings.ToUpper(reqType) {
		return nil, false, p.shouldLog
	}
	params := map[string]string{}
	for i := 1; i < p.Len; i++ {
		if p.Parts[i] != "*" {
			if p.Parts[i] != requestParts[i] {
				return params, false, p.shouldLog
			}
		} else {
			if p.Parts[i-1] != "*" {
				params[p.Parts[i-1]] = requestParts[i]
			}
		}
	}
	if rqi != nil {
		rqi.Log(p.shouldLog)
	}
	return params, true, p.shouldLog
}

func SendToHost(port string, path string, callback func(string, int, error)) {
	path = strings.TrimLeft(path, "/")
	url := fmt.Sprintf("http://localHost%s/%s", port, path)
	fmt.Printf("Client-Request:%s\n", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		callback("could not build new requet", -1, err)
		return
	}
	client := http.Client{Timeout: 5 * time.Second}
	// send the request
	res, err := client.Do(req)
	if err != nil {
		callback("could not GET data.", -2, err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 && res.StatusCode != 201 && res.StatusCode != 202 {
		callback(http.StatusText(res.StatusCode), res.StatusCode, nil)
		return
	}
	// read body
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		callback("could not read response data.", -3, err)
		return
	}
	callback(string(resBody), res.StatusCode, nil)
}
