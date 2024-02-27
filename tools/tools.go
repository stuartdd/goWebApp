package tools

import (
	"bytes"
	"fmt"
	"strings"
)

type UrlRequestParts struct {
	Parts          []string
	Query          map[string][]string
	Header         map[string][]string
	StartWithSlash bool
	ReqType        string
}

func NewUrlRequestParts(url string) *UrlRequestParts {
	s := strings.Split(strings.TrimSpace(url), "/")
	if s[0] == "" {
		s = s[1:]
	}
	parts := &UrlRequestParts{
		Parts:          s,
		StartWithSlash: strings.HasPrefix(url, "/"),
		ReqType:        "GET",
		Query:          make(map[string][]string),
		Header:         make(map[string][]string),
	}
	return parts
}

func (p *UrlRequestParts) WithQuery(q map[string][]string) *UrlRequestParts {
	p.Query = q
	return p
}
func (p *UrlRequestParts) WithHeader(h map[string][]string) *UrlRequestParts {
	p.Header = h
	return p
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *UrlRequestParts) WithReqType(reqType string) *UrlRequestParts {
	p.ReqType = reqType
	return p
}

func (p *UrlRequestParts) UrlParamMap(match *UrlRequestParts) map[string]string {
	list := make(map[string]string)
	pLen := min(len(p.Parts), len(match.Parts))
	for i := 0; i < pLen; i++ {
		if match.Parts[i] == "*" {
			if i > 0 && match.Parts[i-1] != "*" {
				list[p.Parts[i-1]] = p.Parts[i]
			} else {
				list[fmt.Sprintf("p%d", i)] = p.Parts[i]
			}
		}
	}
	return list
}

func (p *UrlRequestParts) UrlParamList(match *UrlRequestParts) []string {
	list := make([]string, 0, 20)
	pLen := min(len(p.Parts), len(match.Parts))
	for i := 0; i < pLen; i++ {
		if match.Parts[i] == "*" {
			list = append(list, p.Parts[i])
		}
	}
	return list
}

func (p *UrlRequestParts) MatchUrl(ma string) bool {
	return p.Match(NewUrlRequestParts(ma))
}

func (p *UrlRequestParts) Match(match *UrlRequestParts) bool {
	if p.ReqType != match.ReqType {
		return false
	}
	pLen := len(p.Parts)
	if match.Len() != pLen {
		return false
	}
	if match.StartWithSlash != p.StartWithSlash {
		return false
	}
	for i := 0; i < pLen; i++ {
		if match.Parts[i] != "*" {
			if match.Parts[i] != p.Parts[i] {
				return false
			}
		}
	}
	return true
}

func (p *UrlRequestParts) Len() int {
	return len(p.Parts)
}

func (p *UrlRequestParts) ToString() string {
	var buffer bytes.Buffer
	pLen := len(p.Parts)
	if p.StartWithSlash {
		buffer.WriteByte('/')
	}
	for i := 0; i < pLen; i++ {
		buffer.WriteString(p.Parts[i])
		if i < (pLen - 1) {
			buffer.WriteByte('/')
		}
	}
	return buffer.String()
}
