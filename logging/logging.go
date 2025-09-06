package logging

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Logger interface {
	Log(string)
	VerboseFunction() func(string)
	IsOpen() bool
	Close()
	LogFileName() string
}

type logFileData struct {
	fileName string   // The file name from the config data (used in map logLevelFileMap).
	logFile  *os.File // The actual file reference
	err      error
}

func newLogFileData(path, fileNameMask string, t time.Time) *logFileData {
	fn := deriveFileName(fileNameMask, t)

	stats, err := os.Stat(path)
	if err != nil {
		return &logFileData{
			logFile:  nil,
			fileName: fn,
			err:      fmt.Errorf("log directory '%s' does not exist", path),
		}
	}

	if !stats.IsDir() {
		return &logFileData{
			logFile:  nil,
			fileName: fn,
			err:      fmt.Errorf("log directory '%s' is not a directory", path),
		}
	}

	f, err := os.OpenFile(filepath.Join(path, fn), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return &logFileData{
			logFile:  nil,
			fileName: fn,
			err:      err,
		}
	}
	return &logFileData{
		logFile:  f,
		fileName: fn,
		err:      nil,
	}
}

func (p *logFileData) reOpen(path, fileNameMask string, t time.Time) *logFileData {
	if p.isOpen() && p.fileName != deriveFileName(fileNameMask, t) {
		p.close()
		return newLogFileData(path, fileNameMask, t)
	}
	return p
}

func (p *logFileData) isOpen() bool {
	return (p.logFile != nil)
}

func (p *logFileData) close() {
	if p.isOpen() {
		p.logFile.Sync()
		p.logFile.Close()
		p.logFile = nil
	}
}

type logger struct {
	fileNameMask   string
	path           string
	monitorSeconds int
	logFileData    *logFileData
	datePrefix     string
	nextFileCheck  time.Time
	consoleOut     bool
	queue          chan string
	mu1            sync.Mutex
	mu2            sync.Mutex
	noMoreCalls    bool
	verboseLog     bool
}

const padding0 = "000000000000"

func NewLogger(pPath string, pFileNameMask string, pMonitorSeconds int, consoleOut bool, isVerbose bool) (Logger, error) {
	if pPath == "" || pFileNameMask == "" {
		return &logger{
			logFileData: &logFileData{
				fileName: "",
				logFile:  nil,
				err:      nil,
			},
			fileNameMask: pFileNameMask,
			noMoreCalls:  true,
			path:         pPath,
			datePrefix:   newDatePrefix(time.Now()),
			verboseLog:   isVerbose,
		}, nil
	}
	l := &logger{
		fileNameMask:   pFileNameMask,
		path:           pPath,
		monitorSeconds: pMonitorSeconds,
		logFileData:    newLogFileData(pPath, pFileNameMask, time.Now()),
		datePrefix:     newDatePrefix(time.Now()),
		nextFileCheck:  getNextMonitorTime(pMonitorSeconds),
		consoleOut:     consoleOut,
		queue:          make(chan string, 20),
		noMoreCalls:    false,
		verboseLog:     isVerbose,
	}
	go l.deQueue()
	return l, l.logFileData.err
}

func (l *logger) VerboseFunction() func(string) {
	if l.verboseLog {
		return l.verbose
	}
	return nil
}

func (l *logger) verbose(s string) {
	if l.verboseLog {
		l.Log(s)
	}
}

func (l *logger) Close() {
	l.mu2.Lock()
	defer l.mu2.Unlock()
	if l.noMoreCalls {
		return
	}
	l.noMoreCalls = true
	count := 2000
	for len(l.queue) > 0 && count > 0 {
		time.Sleep(time.Millisecond)
		count--
	}
	l.logFileData.close()
	close(l.queue)
}

func (l *logger) IsOpen() bool {
	return l.logFileData.isOpen()
}

func (l *logger) Log(msg string) {
	l.mu1.Lock()
	defer l.mu1.Unlock()
	if l.noMoreCalls {
		os.Stdout.WriteString(buildLogLine(msg, l.datePrefix, time.Now()))
		return
	}
	l.queue <- msg
}

func (l *logger) LogFileName() string {
	if l.logFileData.fileName == "" {
		return l.fileNameMask
	}
	return l.logFileData.fileName
}

func (l *logger) deQueue() {
	for msg := range l.queue {
		t := time.Now()
		if t.After(l.nextFileCheck) {
			l.nextFileCheck = getNextMonitorTime(l.monitorSeconds)
			l.datePrefix = newDatePrefix(t)
			l.logFileData = l.logFileData.reOpen(l.path, l.fileNameMask, t)
		}

		out := buildLogLine(msg, l.datePrefix, t)
		if l.IsOpen() {
			l.logFileData.logFile.WriteString(out)
			if l.consoleOut {
				os.Stderr.WriteString(out)
			}
		} else {
			os.Stderr.WriteString(out)
		}
	}
}

func buildLogLine(msg string, datePrefix string, t time.Time) string {
	h := t.Hour()
	hs := strconv.Itoa(h)
	if len(hs) < 2 {
		hs = "0" + hs
	}
	m := t.Minute()
	ms := strconv.Itoa(m)
	if len(ms) < 2 {
		ms = "0" + ms
	}

	s := t.Second()
	ss := strconv.Itoa(s)
	if len(ss) < 2 {
		ss = "0" + ss
	}
	return fmt.Sprintf("%s%s:%s:%s %s\n", datePrefix, hs, ms, ss, msg)
}

func getNextMonitorTime(n int) time.Time {
	t := time.Now()
	t = t.Add(time.Second * time.Duration(n))
	return t
}

func newDatePrefix(t time.Time) string {
	y := t.Year()
	m := t.Month()
	d := t.Day()
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprint(y))
	buffer.WriteRune('/')
	if m < 10 {
		buffer.WriteRune('0')
		buffer.WriteRune(rune(48 + m))
	} else {
		buffer.WriteString(strconv.Itoa(int(m)))
	}
	buffer.WriteRune('/')
	if d < 10 {
		buffer.WriteRune('0')
		buffer.WriteRune(rune(48 + d))
	} else {
		buffer.WriteString(strconv.Itoa(d))
	}
	buffer.WriteRune(' ')
	return buffer.String()
}

func deriveFileName(m string, t time.Time) string {
	m = strings.ReplaceAll(m, "%y", fixedLenInt(t.Year(), 4))
	m = strings.ReplaceAll(m, "%m", fixedLenInt(int(t.Month()), 2))
	m = strings.ReplaceAll(m, "%d", fixedLenInt(t.Day(), 2))
	m = strings.ReplaceAll(m, "%H", fixedLenInt(t.Hour(), 2))
	m = strings.ReplaceAll(m, "%M", fixedLenInt(t.Minute(), 2))
	m = strings.ReplaceAll(m, "%S", fixedLenInt(t.Second(), 2))
	return m
}

func fixedLenInt(i int, l int) string {
	s := strconv.Itoa(i)
	dif := l - len(s)
	if dif <= 0 || dif > 10 {
		return s
	}
	return fmt.Sprintf("%s%s", padding0[0:dif], s)
}
