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
	Close()
}

type logFileData struct {
	fileName string   // The file name from the config data (used in map logLevelFileMap).
	logFile  *os.File // The actual file reference
	err      error
}

func newLogFileData(path, fileNameMask string) *logFileData {
	fn := deriveFileName(fileNameMask)

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

	ffn := filepath.Join(path, fn)
	f, err := os.OpenFile(ffn, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
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

func (p *logFileData) restart(path, fileNameMask string) *logFileData {
	if p.fileName != deriveFileName(fileNameMask) {
		p.close()
		return newLogFileData(path, fileNameMask)
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

	file          *logFileData
	datePrefix    string
	open          bool
	nextFileCheck time.Time
	queue         chan string
	mu            sync.Mutex
}

const padding0 = "000000000000"

func NewLogger(pPath string, pFileNameMask string, pMonitorSeconds int, pLevel string) (Logger, error) {

	l := &logger{
		fileNameMask:   pFileNameMask,
		path:           pPath,
		monitorSeconds: pMonitorSeconds,
		level:          pLevel,
		file:           newLogFileData(pPath, pFileNameMask),
		datePrefix:     getDatePrefix(),
		nextFileCheck:  nextMonitorTime(pMonitorSeconds),
		queue:          make(chan string, 10),
		open:           false,
	}

	l.open = l.file.isOpen()

	go l.deQueue()
	return l, l.file.err
}

func (l *logger) Close() {
	l.file.close()
	l.open = false
	close(l.queue)
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
		if l.file.isOpen() {
			l.file.logFile.WriteString(fmt.Sprintf("%s%2d:%2d:%2d %s\n", l.datePrefix, h, m, s, msg))
		}
		os.Stderr.WriteString(fmt.Sprintf("%s%2d:%2d:%2d %s.\n", l.datePrefix, h, m, s, msg))

		if t.After(l.nextFileCheck) {
			l.nextFileCheck = nextMonitorTime(l.monitorSeconds)
			l.datePrefix = getDatePrefix()
			l.file = l.file.restart(l.path, l.fileNameMask)
		}
	}
}

func nextMonitorTime(n int) time.Time {
	t := time.Now()
	t = t.Add(time.Second * time.Duration(n))
	return t
}

func getDatePrefix() string {
	t := time.Now()
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
		buffer.WriteString(fmt.Sprint(m))
	}
	buffer.WriteRune('/')
	if d < 10 {
		buffer.WriteRune('0')
		buffer.WriteRune(rune(48 + d))
	} else {
		buffer.WriteString(fmt.Sprint(d))
	}
	buffer.WriteRune(' ')
	return buffer.String()
}

func deriveFileName(m string) string {
	t := time.Now()
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
