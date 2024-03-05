package tools

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
	IsOpen() bool
	Close()
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
	level          string
	file           *logFileData
	datePrefix     string
	nextFileCheck  time.Time
	queue          chan string
	mu             sync.Mutex
}

const padding0 = "000000000000"

func NewLogger(pPath string, pFileNameMask string, pMonitorSeconds int, pLevel string) (Logger, error) {
	l := &logger{
		fileNameMask:   pFileNameMask,
		path:           pPath,
		monitorSeconds: pMonitorSeconds,
		level:          pLevel,
		file:           newLogFileData(pPath, pFileNameMask, time.Now()),
		datePrefix:     newDatePrefix(time.Now()),
		nextFileCheck:  getNextMonitorTime(pMonitorSeconds),
		queue:          make(chan string, 20),
	}
	go l.deQueue()
	return l, l.file.err
}

func (l *logger) Close() {
	l.file.close()
	close(l.queue)
}

func (l *logger) IsOpen() bool {
	return l.file.isOpen()
}

func (l *logger) Log(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.queue <- msg
}

func (l *logger) deQueue() {
	for msg := range l.queue {
		t := time.Now()
		h := t.Hour()
		m := t.Minute()
		s := t.Second()
		if t.After(l.nextFileCheck) {
			l.nextFileCheck = getNextMonitorTime(l.monitorSeconds)
			l.datePrefix = newDatePrefix(t)
			l.file = l.file.reOpen(l.path, l.fileNameMask, t)
		}
		if l.IsOpen() {
			l.file.logFile.WriteString(fmt.Sprintf("%s%2d:%2d:%2d %s\n", l.datePrefix, h, m, s, msg))
		}
		os.Stderr.WriteString(fmt.Sprintf("%s%2d:%2d:%2d %s.\n", l.datePrefix, h, m, s, msg))
	}
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
