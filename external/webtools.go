package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type JsonToken struct {
	Pos    int    // The position of the value from the tokenised input line
	Name   string // JSON value name
	Quoted bool   // JSON value is quoted "name":"value" or "name":value
}

type Line struct {
	Split  string // Split line on these tokens. Split chars NOT written to output
	Skip   string // Skip chars NOT written to output
	Tokens []*JsonToken
}

type Properties struct {
	InputFile    string   // Input file. If not input from stdin
	OutputFile   string   // Output file. If not output to stdout
	FilePrefix   string   // Write at strat of output
	FilePostfix  string   // Write at end of output
	LinePrefix   string   // At the start of each input line write to the output
	LinePostfix  string   // At the end of each input line write to the output
	LineInfix    string   // At the end of each input line write to the output except the last line
	SkipLines    int      // Skip n lines from input
	MaxLines     int      // Max number of input lines to be read
	LineContains []string // Include input line if it contains ANY of these values
	Lines        []*Line  // Multiple line spec applied for each line processed
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
		LinePrefix:  "",  // At the start of each input line write to the output
		LinePostfix: "",  // At the end of each input line write to the output
		LineInfix:   ",", // At the end of each input line write to the output except the last line
	}
	err := json.Unmarshal(content, &properties)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to understand the config data in the file: %s\n", configFileName))
		os.Exit(1)
	}
	count := 0
	theLine := ""
	var scanner *bufio.Scanner
	var outputBuff bytes.Buffer

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

	outputBuff.WriteString(properties.FilePrefix)
	outputBuffLen := outputBuff.Len()
	// For each input line
	for scanner.Scan() && count < properties.MaxLines {
		theLine = scanner.Text()
		if contains(theLine, properties.LineContains) {

			outputBuff.WriteString(properties.LinePrefix)
			if len(properties.Lines) > 0 {
				tokensOnLine := tokeniseLineToJson(count, properties, []byte(theLine))
				outputBuff.WriteString(tokensOnLine)
			} else {
				if theLine != "" {
					outputBuff.WriteString(theLine)
				}
			}
			outputBuff.WriteString(properties.LinePostfix)
			outputBuffLen = outputBuff.Len()

			outputBuff.WriteString(properties.LineInfix)
			count++
		}
	}

	outputBuff.Truncate(outputBuffLen)
	outputBuff.WriteString(properties.FilePostfix)
	if properties.OutputFile != "" {
		err = os.WriteFile(properties.OutputFile, outputBuff.Bytes(), 0644)
		if err != nil {
			log(fmt.Sprintf("Failed to create output file: %s\n", properties.OutputFile))
			os.Exit(1)
		}
	} else {
		os.Stdout.WriteString(outputBuff.String())
	}
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

func tokeniseLineToJson(lineNum int, p *Properties, line []byte) string {
	//
	// get the tokens for a line. Repeat tokens for n lines using mod function.
	// so for eg. 3 line tokens are applied 3 times for 9 lines...
	//
	tokenLine := p.Lines[lineNum%len(p.Lines)]
	if len(tokenLine.Tokens) == 0 {
		os.Stderr.WriteString(fmt.Sprintf("No Tokens are defined for Line[%d]", lineNum%len(p.Lines)))
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
	// buff.WriteString(p.LinePrefix)
	for i, tok := range tokens {
		p := tok.Pos
		if p >= 0 && p < len(resp) {
			buff.WriteString(finalToken(tok, resp[p]))
			if i < (len(tokens) - 1) {
				buff.WriteRune(',')
			}
		}
	}
	// buff.WriteString(p.LinePostfix)
	return buff.String()
}

func finalToken(tokDesc *JsonToken, value string) string {
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
