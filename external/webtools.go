package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultthumbNailTimeStamp = "%y_%m_%d_%H_%M_%S_"
const defaultThumbnailExt = ".jpg"

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

type Users struct {
	ImageRoot  string
	ImagePaths []string
}

// func (u *Users) Path(index int) string {
// 	return filepath.Join(u.ImageRoot, u.ImagePaths[index])
// }

func (u *Users) ToUserPath(name string) string {
	var buff bytes.Buffer
	buff.WriteString("\n ##   Root  ")
	absRoot := absPath(u.ImageRoot)
	buff.WriteString(absRoot)
	for _, v := range u.ImagePaths {
		buff.WriteString("\n ##     Path  ")
		fullPath := filepath.Join(absRoot, name, v)
		buff.WriteString(fullPath)
		ok, s := fileExists(fullPath, true, 0)
		if !ok {
			buff.WriteString(" --> ")
			buff.WriteString(s)
		}
	}
	return buff.String()
}

type UserPathInfo struct {
	root  string
	iPath string
	user  string
}

func (upi *UserPathInfo) Joined() string {
	return filepath.Join(upi.root, upi.user, upi.iPath)
}

type ThumbnailInfo struct {
	ThumbNailsExec      string
	ThumbNailTimeStamp  string
	ThumbNailFileSuffix string
	ThumbNailsRoot      string
	ImageExtensions     []string
	MaxFiles            int
	DryRun              bool
	Verbose             bool
	Resources           map[string]*Users
	LogPath             string
	LogName             string

	pathList    []*UserPathInfo
	currentPath int
}

func (tni *ThumbnailInfo) Next() *UserPathInfo {
	if len(tni.pathList) == 0 {
		for n, u := range tni.Resources {
			for _, p := range u.ImagePaths {
				tni.pathList = append(tni.pathList, &UserPathInfo{root: absPath(u.ImageRoot), user: n, iPath: p})
			}
		}
		tni.currentPath = -1
	}
	tni.currentPath++
	if tni.currentPath >= len(tni.pathList) {
		return nil
	}
	return tni.pathList[tni.currentPath]
}

func (tni *ThumbnailInfo) Extensions() []string {
	l := []string{}
	for _, v := range tni.ImageExtensions {
		l = append(l, strings.ToLower(v))
	}
	return l
}

func (tni *ThumbnailInfo) String() string {
	var buff bytes.Buffer
	buff.WriteString("\n ## Run At:               ")
	buff.WriteString(time.Now().Format(time.RFC3339))
	buff.WriteString("\n ## ThumbNailsExec:       ")
	buff.WriteString(tni.ThumbNailsExec)
	buff.WriteString("\n ## ThumbNailsRoot:       ")
	buff.WriteString(tni.ThumbNailsRoot)
	buff.WriteString(" --> ")
	buff.WriteString(absPath(tni.ThumbNailsRoot))
	buff.WriteString("\n ## ThumbNailTimeStamp:   ")
	buff.WriteString(tni.ThumbNailTimeStamp)
	buff.WriteString("\n ## ThumbNailFileSuffix:  ")
	buff.WriteString(tni.ThumbNailFileSuffix)
	buff.WriteString("\n ## MaxFiles:               ")
	buff.WriteString(strconv.Itoa(tni.MaxFiles))
	buff.WriteString("\n ## DryRun:               ")
	buff.WriteString(fmt.Sprintf("%t", tni.DryRun))
	buff.WriteString("\n ## Verbose:              ")
	buff.WriteString(fmt.Sprintf("%t", tni.Verbose))
	buff.WriteString("\n ## Log:                  ")
	logJoin := filepath.Join(tni.LogPath, tni.LogName)
	buff.WriteString(logJoin)
	buff.WriteString(" --> ")
	buff.WriteString(absPath(absPath(logJoin)))
	for n, v := range tni.Resources {
		buff.WriteString("\n ## User:")
		buff.WriteString(n)
		buff.WriteString(v.ToUserPath(n))
	}
	return buff.String()
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

// Generate 1 line for each /usr/bin/convert -thumbnail 200 "$i" thumbnails/thumb.$(basename "$i")
func RunThumbnails(content []byte, configFileName string) {
	thumbnailInfo := &ThumbnailInfo{
		ThumbNailsExec:      "Undefined",
		ThumbNailsRoot:      "",
		ThumbNailTimeStamp:  "%y_%m_%d_%H_%M_%S_",
		ThumbNailFileSuffix: ".json",
		ImageExtensions:     make([]string, 0),
		MaxFiles:            math.MaxInt,
		Verbose:             false,
		DryRun:              true,
		Resources:           make(map[string]*Users),
		LogPath:             "",
		LogName:             "",
		currentPath:         0,
		pathList:            make([]*UserPathInfo, 0),
	}

	err := json.Unmarshal(content, &thumbnailInfo)
	if err != nil {
		log(fmt.Sprintf("Failed to understand the config data in the file: %s\n", configFileName))
		os.Exit(1)
	}
	fileExists(thumbnailInfo.ThumbNailsRoot, true, 1)
	fileExists(thumbnailInfo.LogPath, true, 1)
	if thumbnailInfo.Verbose {
		log(fmt.Sprintf("Config Data:%s\n", thumbnailInfo.String()))
	}

	count := 0
	thumbNailRoot := absPath(thumbnailInfo.ThumbNailsRoot)
	extensions := thumbnailInfo.Extensions()
	p := thumbnailInfo.Next()
	for p != nil {
		err := WalkDir(thumbnailInfo, p, func(name string) bool {
			if len(thumbnailInfo.ImageExtensions) == 0 {
				return true
			}
			for _, v := range extensions {
				if strings.HasSuffix(strings.ToLower(name), v) {
					return true
				}
			}
			return false
		}, func(info *UserPathInfo, path string, fileName string, tn string) {
			p2 := filepath.Join(thumbNailRoot, filepath.Dir(path)[len(info.root):], tn)
			p2Dir := filepath.Dir(p2)
			okDir, _ := fileExists(p2Dir, true, 0)
			if !okDir {
				if thumbnailInfo.DryRun {
					logLn(fmt.Sprintf("DryRun: MkdirAll %s", p2Dir))
				} else {
					err := os.MkdirAll(p2Dir, 0755)
					if err != nil {
						logLn(fmt.Sprintf("Could not crerate directories %s", p2Dir))
						os.Exit(1)
					}
					logLn(fmt.Sprintf("MkdirAll %s", p2Dir))
				}
			}
			ok, _ := fileExists(p2, false, 0)
			if !ok {
				exec := strings.ReplaceAll(thumbnailInfo.ThumbNailsExec, "%in", path)
				exec = strings.ReplaceAll(exec, "%out", p2)
				os.Stdout.WriteString(exec + "\n")
				count++
				if count >= thumbnailInfo.MaxFiles {
					os.Exit(0)
				}
			}
		})
		if err != nil {
			logLn(err.Error())
		}

		p = thumbnailInfo.Next()
	}
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
	if findBytesInByteArray(0, content, []byte("\"thumbNailsRoot\":")) >= 0 {
		RunThumbnails(content, os.Args[1])
	}
}

/*
Find bytes in the byte array. Return position of first byte in found bytes

	start = 0
	b = []byte("abcdef")
	f = []byte("cd")
	result = 2
	so b[result:] would return 'cdef'
	result = -1 if not found
*/
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

func fileExists(filename string, shouldBeDir bool, abort int) (bool, string) {
	fn, err := filepath.Abs(filename)
	if err != nil {
		s := "Invalid Path: " + err.Error()
		if abort > 0 {
			log(fmt.Sprintf("File %s %s", filename, s))
			os.Exit(abort)
		}
		return false, s
	}
	info, err := os.Stat(fn)
	if os.IsNotExist(err) {
		s := "Does NOT exist"
		if abort > 0 {
			log(fmt.Sprintf("File %s %s", fn, s))
			os.Exit(abort)
		}
		return false, s
	}
	if shouldBeDir {
		if info.IsDir() {
			return true, ""
		}
		s := "Should be a directory"
		if abort > 0 {
			log(fmt.Sprintf("File %s %s", fn, s))
			os.Exit(abort)
		}
		return false, s
	}
	return true, ""
}

func absPath(p string) string {
	s, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return s
}

func log(s string) {
	os.Stdout.WriteString(fmt.Sprintf(" ## %s", s))
}
func logLn(s string) {
	os.Stdout.WriteString(fmt.Sprintf(" ## %s\n", s))
}

func WalkDir(config *ThumbnailInfo, userPathInfo *UserPathInfo, onFile func(string) bool, onLog func(*UserPathInfo, string, string, string)) error {
	absBasePath, err := filepath.Abs(userPathInfo.Joined())
	if err != nil {
		return err
	}
	_, err = os.Stat(absBasePath)
	if err != nil {
		return err
	}

	filepath.Walk(absBasePath, func(path string, info fs.FileInfo, err error) error {
		if info != nil {
			if !info.IsDir() {
				add := true
				if onFile != nil {
					add = onFile(info.Name())
				}
				if add {
					var dt *FileDateTime
					NewImage(path, false, func(i *IFDEntry, w *Walker) bool {
						if i != nil {
							if i.TagData.Name == "DateTimeOriginal" && dt == nil {
								dt, _ = NewFileDateTimeFromSpec(i.Value, 1)
							}
							if i.TagData.Name == "DateTime" && dt == nil {
								dt, _ = NewFileDateTimeFromSpec(i.Value, 2)
							}
							if i.TagData.Name == "DateTimeDigitized" && dt == nil {
								dt, _ = NewFileDateTimeFromSpec(i.Value, 3)
							}
						}
						return true
					}, "Scan EXIF image data")

					if dt == nil {
						dt, _ = NewFileDateTimeFromSpec(info.Name(), 4)
						if dt == nil {
							dt = NewFileDateTimeFromTime(info.ModTime())
						}
					}

					onLog(userPathInfo, path, info.Name(), dt.Format(config.ThumbNailTimeStamp, info.Name()))
				}
			}
		}
		return nil
	})
	return nil
}

type FileDateTime struct {
	y, m, d, hh, mm, ss, src int
}

func (dt *FileDateTime) Format(formatString string, imageFileName string) string {
	// var b bytes.Buffer
	// b.WriteRune('T')
	if len(formatString) < 12 {
		formatString = defaultthumbNailTimeStamp
	}

	s := strings.ReplaceAll(formatString, "%y", strconv.Itoa(dt.y))
	s = strings.ReplaceAll(s, "%m", pad2(dt.m))
	s = strings.ReplaceAll(s, "%d", pad2(dt.d))
	s = strings.ReplaceAll(s, "%H", pad2(dt.hh))
	s = strings.ReplaceAll(s, "%M", pad2(dt.mm))
	s = strings.ReplaceAll(s, "%S", pad2(dt.ss))
	s = strings.ReplaceAll(s, "%?", pad2(dt.src))

	if imageFileName == "" {
		return s
	}
	return s + imageFileName + defaultThumbnailExt
}

func pad2(i int) string {
	if i < 10 {
		return "0" + strconv.Itoa(i)
	}
	return strconv.Itoa(i)
}

func NewFileDateTimeFromSpec(spec string, src int) (*FileDateTime, error) {
	spec1 := []byte(spec)
	spec2 := make([]byte, 18)
	specPos := 0
	for _, c := range spec1 {
		if c >= '0' && c <= '9' {
			spec2[specPos] = c
			specPos++
			if specPos > 17 {
				return nil, fmt.Errorf("character buffer overrun")
			}
		}
	}
	specPos = 0
	y, spespecPos := readIntFromSpec(spec2, 0, 4)
	if y < 1970 {
		return nil, fmt.Errorf("year '%d' before 1970", y)
	}
	if y > 2100 {
		return nil, fmt.Errorf("year '%d' after 2070", y)
	}
	m, spespecPos := readIntFromSpec(spec2, spespecPos, 2)
	if m < 1 {
		return nil, fmt.Errorf("month '%d' is 0", m)
	}
	if m > 12 {
		return nil, fmt.Errorf("month '%d' above 12", m)
	}
	d, spespecPos := readIntFromSpec(spec2, spespecPos, 2)
	if d < 1 {
		return nil, fmt.Errorf("day Of Month '%d' is 0", d)
	}
	if m > 31 {
		return nil, fmt.Errorf("day Of Month '%d' above 31", d)
	}
	hh, spespecPos := readIntFromSpec(spec2, spespecPos, 2)
	if hh > 23 {
		return nil, fmt.Errorf("hour '%d' is above 23", hh)
	}
	mm, spespecPos := readIntFromSpec(spec2, spespecPos, 2)
	if mm > 59 {
		return nil, fmt.Errorf("min '%d' is above 59", mm)
	}
	ss, _ := readIntFromSpec(spec2, spespecPos, 2)
	if ss > 59 {
		return nil, fmt.Errorf("seconds '%d' is above 59", ss)
	}
	return &FileDateTime{y: y, m: m, d: d, hh: hh, mm: mm, ss: ss, src: src}, nil
}

func readIntFromSpec(spec []byte, from, len int) (int, int) {
	acc := 0
	for i := from; i < (from + len); i++ {
		acc = acc * 10
		si := int(spec[i])
		if si >= '0' && si <= '9' {
			acc = acc + (int(spec[i] - '0'))
		}
	}
	return acc, from + len
}

func NewFileDateTimeFromTime(t time.Time) *FileDateTime {
	return &FileDateTime{y: t.Year(), m: int(t.Month()), d: t.Day(), hh: t.Hour(), mm: t.Minute(), ss: t.Second(), src: 0}
}
