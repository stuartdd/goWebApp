package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type Token struct {
	Pos    int
	Name   string
	Quoted bool
}

type Line struct {
	Split    string
	Skip     string
	TokenPos []int
	Tokens   []*Token
}

type Properties struct {
	InputFile    string
	OutputFile   string
	FilePrefix   string
	FilePostfix  string
	LinePrefix   string
	LinePostfix  string
	Infix        string
	SkipLines    int
	LineContains []string
	MaxLines     int
	Lines        []*Line
}

func (p *Properties) String() string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func RunTextToJson(content []byte, configFileName string) {

	properties := &Properties{
		SkipLines:   0,
		MaxLines:    999,
		InputFile:   "",
		OutputFile:  "",
		FilePrefix:  "",
		FilePostfix: "",
		LinePrefix:  "",
		LinePostfix: "",
		Infix:       ",",
	}
	err := json.Unmarshal(content, &properties)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to understand the config data in the file: %s\n", configFileName))
		os.Exit(1)
	}
	count := 0
	txt := ""
	var scanner *bufio.Scanner
	var buff bytes.Buffer
	if properties.FilePrefix != "" {
		buff.WriteString(properties.FilePrefix)
	}

	if properties.InputFile == "" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		file, err := os.Open(properties.InputFile)
		if err != nil {
			os.Stderr.WriteString(fmt.Sprintf("Failed to open data file: %s\n", properties.InputFile))
			os.Exit(1)
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	}

	for i := 0; i < properties.SkipLines; i++ {
		scanner.Scan()
		scanner.Text()
	}

	buffLen := buff.Len()
	for scanner.Scan() && count < properties.MaxLines {
		txt = scanner.Text()
		if contains(txt, properties.LineContains) {
			if properties.LinePrefix != "" {
				buff.WriteString(properties.LinePrefix)
			}
			if len(properties.Lines) > 0 {
				tokens := tokenise(count, properties.Lines, []byte(txt))
				buff.WriteString(tokens)
				buffLen = buff.Len()
				if tokens != "" {
					buff.WriteString(properties.Infix)
				}
			} else {
				if txt != "" {
					buff.WriteString(txt)
					buffLen = buff.Len()
				}
			}
			if properties.LinePostfix != "" {
				buff.WriteString(properties.LinePostfix)
				buffLen = buff.Len()
			}
			count++
		}
	}

	buff.Truncate(buffLen)
	if properties.FilePostfix != "" {
		buff.WriteString(properties.FilePostfix)
	}
	if properties.OutputFile != "" {
		err = os.WriteFile(properties.OutputFile, buff.Bytes(), 0644)
		if err != nil {
			log(fmt.Sprintf("Failed to create output file: %s\n", properties.OutputFile))
			os.Exit(1)
		}
	}
	os.Stdout.WriteString(buff.String())
}

func main() {
	if len(os.Args) < 2 {
		log("Requires configFileName")
		os.Exit(1)
	}
	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		log(fmt.Sprintf("Failed to read config: %s\n", os.Args[1]))
		os.Exit(1)
	}
	if findBytesInByteArray(0, content, []byte("\"FilePrefix\":")) >= 0 {
		RunTextToJson(content, os.Args[1])
	}
}

func findBytesInByteArray(start int, b []byte, f []byte) int {
	fl := len(f)
	bl := len(b)
	if fl == 0 || bl == 0 {
		return -1
	}
	m := (bl - fl) + 1
	if m < 1 {
		return -1
	}
	for i := start; i < m; i++ {
		if b[i] == f[0] {
			found := true
			for j := 0; j < len(f); j++ {
				if b[i+j] != f[j] {
					found = false
					break
				}
			}
			if found {
				return i
			}
		}
	}
	return -1
}

func contains(line string, cont []string) bool {
	if len(cont) == 0 {
		return true
	}
	for _, v := range cont {
		if findBytesInByteArray(0, []byte(line), []byte(v)) >= 0 {
			return true
		}
	}
	return false
}

func tokenise(lineNum int, lines []*Line, line []byte) string {
	tokenLine := lines[lineNum%len(lines)]
	if len(tokenLine.Tokens) == 0 {
		os.Stderr.WriteString(fmt.Sprintf("No Tokens are defined for Line[%d]", lineNum%len(lines)))
		os.Exit(1)
	}
	tokens := tokenLine.Tokens
	resp := []string{}
	var buff bytes.Buffer
	split := []byte(tokenLine.Split)
	skip := []byte(tokenLine.Skip)
	tokenNum := 0
	for _, b := range line {
		if isCharInString(b, split) {
			if buff.Len() > 0 {
				resp = append(resp, buff.String())
				tokenNum++
				buff.Reset()
			}
		} else {
			if !isCharInString(b, skip) {
				buff.WriteByte(b)
			}
		}
	}
	if buff.Len() > 0 {
		resp = append(resp, buff.String())
	}
	buff.Reset()
	buff.WriteRune('{')
	for i, tok := range tokens {
		p := tok.Pos
		if p >= 0 && p < len(resp) {
			buff.WriteString(finalToken(tok, resp[p]))
			if i < (len(tokens) - 1) {
				buff.WriteRune(',')
			}
		}
	}
	buff.WriteRune('}')
	return buff.String()
}

func finalToken(tokDesc *Token, value string) string {
	var buff bytes.Buffer
	buff.WriteRune('"')
	buff.WriteString(tokDesc.Name)
	buff.WriteRune('"')
	buff.WriteRune(':')
	if tokDesc.Quoted {
		buff.WriteRune('"')
		buff.WriteString(value)
		buff.WriteRune('"')
	} else {
		buff.WriteString(value)
	}

	return buff.String()
}

func isCharInString(b byte, splitters []byte) bool {
	for _, c := range splitters {
		if c == b {
			return true
		}
	}
	return false
}

func log(s string) {
	os.Stdout.WriteString(fmt.Sprintf(" ## %s", s))
}
