package tools

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"
)

type Logger interface {
	Log(s string)
	Close()
}

type logFileData struct {
	fileName string   // The file name from the config data (used in map logLevelFileMap).
	logFile  *os.File // The actual file reference
}

type logger struct {
	file       *logFileData
	datePrefix string
	open       bool
}

func NewLogger(absFileName string) (Logger, error) {
	if absFileName == "" {
		return &logger{
			file: &logFileData{
				fileName: absFileName,
				logFile:  nil,
			},
			datePrefix: getDatePrefix(),
			open:       false,
		}, nil
	}

	f, err := os.OpenFile(absFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, errors.New("Logging Error:" + err.Error())
	}
	return &logger{
		file: &logFileData{
			fileName: absFileName,
			logFile:  f,
		},
		datePrefix: getDatePrefix(),
		open:       true,
	}, nil
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

func (l *logger) Close() {
	l.file.logFile.Close()
	l.open = false
}

func (l *logger) Log(msg string) {
	t := time.Now()
	h := t.Hour()
	m := t.Minute()
	s := t.Second()
	if l.open {
		l.file.logFile.WriteString(fmt.Sprintf("%s%2d:%2d:%2d %s\n", l.datePrefix, h, m, s, msg))
	}
	os.Stderr.WriteString(fmt.Sprintf("%s%2d:%2d:%2d %s\n", l.datePrefix, h, m, s, msg))
}
