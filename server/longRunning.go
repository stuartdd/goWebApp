package server

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LongRunningProcess struct {
	ID      string
	Started time.Time
	PID     int
	CanStop bool
}

type LongRunningManager struct {
	enabled            bool
	path               string
	file               string
	script             string
	longRunningProcess map[string]*LongRunningProcess
	logger             func(string)
}

func NewLongRunningManagerDisabled() *LongRunningManager {
	return &LongRunningManager{
		enabled:            false,
		path:               "",
		file:               "",
		script:             "",
		logger:             nil,
		longRunningProcess: map[string]*LongRunningProcess{},
	}
}

func NewLongRunningManager(path string, file string, script string, log func(string)) (*LongRunningManager, error) {
	if file == "" {
		return NewLongRunningManagerDisabled(), nil
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return NewLongRunningManagerDisabled(), err
	}
	st, err := os.Stat(path)
	if err != nil {
		return NewLongRunningManagerDisabled(), err
	}
	if !st.IsDir() {
		return NewLongRunningManagerDisabled(), fmt.Errorf("LongRunningManager:Path %s is not a directory", path)
	}

	lrm := &LongRunningManager{
		enabled:            true,
		path:               path,
		file:               filepath.Join(path, file),
		script:             script,
		logger:             log,
		longRunningProcess: map[string]*LongRunningProcess{},
	}
	return lrm, nil
}

func (p *LongRunningManager) AddLongRunningProcessData(id string, cmd []string, canStop bool) {
	p.longRunningProcess[id] = &LongRunningProcess{
		ID:      "",
		PID:     0,
		Started: time.Date(0, 1, 1, 0, 0, 0, 0, time.Local),
		CanStop: canStop,
	}
	var buf bytes.Buffer
	for i, c := range cmd {
		if strings.HasPrefix(c,"./") {
			c = c[2:]
		}
ggit		buf.WriteString(strings.TrimSpace(c))
		if i < (len(cmd) - 1) {
			buf.WriteString(" ")
		}
	}
}

func (p *LongRunningManager) Log(m string) {
	if p.logger != nil {
		p.logger(m)
	}
}

func (p *LongRunningManager) IsEnabled() bool {
	return p.enabled
}

func (p *LongRunningManager) Len() int {
	return len(p.longRunningProcess)
}

func (p *LongRunningManager) String() string {
	if p.enabled {
		return fmt.Sprintf("File:%s. Script:%s", p.file, p.script)
	}
	return "Long Running Process Manager is disabled"
}

func (p *LongRunningManager) ToJson() string {
	if len(p.longRunningProcess) > 0 {
		var b bytes.Buffer
		b.WriteRune('[')
		for n, v := range p.longRunningProcess {
			b.WriteString("{\"Name\":\"")
			b.WriteString(n)
			b.WriteString("\",\"Started\":\"")
			b.WriteString(v.GetStartTime())
			b.WriteString("\",\"PID\":")
			b.WriteString(strconv.Itoa(v.PID))
			b.WriteString(",\"CanStop\":")
			if v.CanStop {
				b.WriteString("true")
			} else {
				b.WriteString("false")
			}
			b.WriteString("},")
		}
		return b.String()[:b.Len()-1] + "]"
	}
	return "[]"
}

// func (p *LongRunningManager) loadX() {
// 	if p.enabled {
// 		content, err := os.ReadFile(p.file)
// 		if err != nil {
// 			p.longRunningProcess = map[string]*LongRunningProcess{}
// 			return
// 		}
// 		err = json.Unmarshal(content, &p.longRunningProcess)
// 		if err != nil {
// 			p.longRunningProcess = map[string]*LongRunningProcess{}
// 			return
// 		}
// 		p.Log("LongRunningManager Loaded: " + p.file)

// 	}
// }

// func (p *LongRunningManager) storeX() error {
// 	if p.enabled {
// 		data, err := json.MarshalIndent(p.longRunningProcess, "", "  ")
// 		if err != nil {
// 			return err
// 		}
// 		err = os.WriteFile(p.file, data, os.ModePerm)
// 		if err != nil {
// 			return err
// 		}
// 		p.Log("LongRunningManager Stored: " + p.file)
// 	}
// 	return nil
// }

// func (p *LongRunningManager) AddLongRunningProcess(id string, pid int, canStop bool, commit bool) bool {
// 	if p.enabled {
// 		p.longRunningMutex.Lock()
// 		defer p.longRunningMutex.Unlock()
// 		lrp := NewLongRunningProcess(id, pid, canStop)
// 		if p.longRunningProcess[lrp.key()] == nil {
// 			if commit {
// 				p.longRunningProcess[lrp.key()] = lrp
// 				p.Log("Process added. " + lrp.ID)
// 				p.store()
// 			}
// 			return true
// 		}
// 	} else {
// 		return true
// 	}
// 	return false
// }

// func (p *LongRunningManager) UpdateLongRunningProcess() {
// 	if p.enabled {
// 		p.longRunningMutex.Lock()
// 		defer p.longRunningMutex.Unlock()
// 		p.load()
// 		upd := false
// 		lrp := map[string]*LongRunningProcess{}
// 		for _, v := range p.longRunningProcess { // Example pid
// 			process, _ := os.FindProcess(v.PID) // Always succeeds on Unix systems
// 			err := process.Signal(syscall.Signal(0))
// 			if err != nil {
// 				lrp[v.key()] = v
// 			} else {
// 				upd = true
// 				p.Log(fmt.Sprintf("UpdateLongRunningProcess PID: %d NO longer running [%s]", v.PID, v.ID))
// 			}
// 		}
// 		if upd {
// 			p.longRunningProcess = lrp
// 			p.store()
// 		}
// 	}
// }

// func NewLongRunningProcess(id string, pid int, canStop bool) *LongRunningProcess {
// 	return &LongRunningProcess{
// 		ID:      id,
// 		PID:     pid,
// 		Started: time.Now(),
// 		CanStop: canStop,
// 	}
// }

// func (p *LongRunningProcess) key() string {
// 	return p.ID
// }

func (p *LongRunningProcess) GetStartTime() string {
	if p.Started.Year() == 0 {
		return ""
	}
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		p.Started.Year(), p.Started.Month(), p.Started.Day(),
		p.Started.Hour(), p.Started.Minute(), p.Started.Second())
}
