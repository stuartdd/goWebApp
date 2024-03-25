package server

import (
	"bytes"
	"fmt"
	"strings"
)

type UrlRequestMatcher struct {
	Parts          []string
	ReqType        string
	isAbsolutePath bool
	Len            int
}

func NewUrlRequestMatcher(templateUrl string, reqType string) *UrlRequestMatcher {
	s := strings.Split(strings.TrimSpace(templateUrl), "/")
	if s[0] == "" {
		s = s[1:]
	}
	return &UrlRequestMatcher{
		Parts:          s,
		ReqType:        strings.ToUpper(reqType),
		isAbsolutePath: strings.HasPrefix(templateUrl, "/"),
		Len:            len(s),
	}
}

func (p *UrlRequestMatcher) String() string {
	var buffer bytes.Buffer
	if p.isAbsolutePath {
		buffer.WriteRune('/')
	}
	for _, v := range p.Parts {
		buffer.WriteString(v)
		buffer.WriteRune('/')
	}
	return fmt.Sprintf("Parts: '%s:%s'", p.ReqType, buffer.String())
}

func (p *UrlRequestMatcher) Match(requestParts []string, isAbsolutePath bool, reqType string) (map[string]string, bool) {
	params := map[string]string{}
	if p.ReqType != strings.ToUpper(reqType) {
		return params, false
	}
	if p.Len != len(requestParts) {
		return params, false
	}
	if p.Len == 0 {
		return params, true
	}
	if isAbsolutePath != p.isAbsolutePath {
		return params, false
	}
	if p.Parts[0] != requestParts[0] {
		return params, false
	}
	for i := 1; i < p.Len; i++ {
		if p.Parts[i] != "*" {
			if p.Parts[i] != requestParts[i] {
				return params, false
			}
		} else {
			if p.Parts[i-1] != "*" {
				params[p.Parts[i-1]] = requestParts[i]
			}
		}
	}
	return params, true
}
