package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type Token struct {
	Name   string
	Quoted bool
}

type Line struct {
	Split     string
	Skip      string
	MaxTokens int
	Tokens    []*Token
}

type Properties struct {
	InputFile  string
	OutputFile string
	Prefix     string
	Postfix    string
	Infix      string
	SkipLines  int
	MaxLines   int
	Lines      []*Line
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
		SkipLines:  0,
		MaxLines:   999,
		InputFile:  "",
		OutputFile: "",
		Prefix:     "",
		Postfix:    "",
		Infix:      ",",
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
	if properties.Prefix != "" {
		buff.WriteString(properties.Prefix)
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
		tokens := tokenise(count, properties.Lines, []byte(txt))
		buff.WriteString(tokens)
		buffLen = buff.Len()
		if tokens != "" {
			buff.WriteString(properties.Infix)
		}
		count++
	}

	buff.Truncate(buffLen)
	if properties.Postfix != "" {
		buff.WriteString(properties.Postfix)
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

func tokenise(lineNum int, lines []*Line, line []byte) string {
	tokenLine := lines[lineNum%len(lines)]
	resp := []string{}
	var buff bytes.Buffer
	split := []byte(tokenLine.Split)
	skip := []byte(tokenLine.Skip)
	tokenNum := 0
	maxTokens := tokenLine.MaxTokens
	if maxTokens < 1 {
		return ""
	}
	for _, b := range line {
		if isCharInString(b, split) {
			if buff.Len() > 0 {
				if tokenNum < maxTokens {
					resp = append(resp, finalToken(tokenLine, tokenNum, buff.String()))
				}
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
		if tokenNum < maxTokens {
			resp = append(resp, finalToken(tokenLine, tokenNum, buff.String()))
		}
	}
	buff.Reset()
	buff.WriteRune('{')
	for i, t := range resp {
		buff.WriteString(t)
		if i < (len(resp) - 1) {
			buff.WriteRune(',')
		}
	}
	buff.WriteRune('}')
	return buff.String()
}

func finalToken(line *Line, tn int, token string) string {
	var buff bytes.Buffer
	tokenDesc := line.Tokens[tn]
	buff.WriteRune('"')
	buff.WriteString(tokenDesc.Name)
	buff.WriteRune('"')
	buff.WriteRune(':')
	if tokenDesc.Quoted {
		buff.WriteRune('"')
		buff.WriteString(token)
		buff.WriteRune('"')
	} else {
		buff.WriteString(token)
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
