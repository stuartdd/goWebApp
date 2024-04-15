package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

func RunMain(configFileName string) {

	content, err := os.ReadFile(configFileName)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to read config: %s\n", configFileName))
		os.Exit(1)
	}
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
	err = json.Unmarshal(content, &properties)
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
			os.Stderr.WriteString(fmt.Sprintf("Failed to create output file: %s\n", properties.OutputFile))
			os.Exit(1)
		}
	}
	os.Stdout.WriteString(buff.String())
}

func main() {
	if len(os.Args) < 2 {
		os.Stderr.WriteString(fmt.Sprintf("Args:%s\n", os.Args))
		os.Exit(1)
	}
	RunMain(os.Args[1])
}

func contains(line string, cont []string) bool {
	if len(cont) == 0 {
		return true
	}
	for _, v := range cont {
		if strings.Contains(line, v) {
			return true
		}
	}
	return false
}

func tokenise(lineNum int, lines []*Line, line []byte) string {
	tokenLine := lines[lineNum%len(lines)]
	if tokenLine.Tokens == nil || len(tokenLine.Tokens) == 0 {
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
