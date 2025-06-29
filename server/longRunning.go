package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type LongRunningProcess struct {
	ID      string
	Started time.Time
	PID     int
	CanStop bool
}

type LongRunningManager struct {
	enabled          bool
	path             string
	file             string
	script           string
	longRunning      map[string]*LongRunningProcess
	logger           func(string)
	longRunningMutex sync.Mutex
}

func NewLongRunningManagerDisabled() *LongRunningManager {
	return &LongRunningManager{
		enabled:     false,
		path:        "",
		file:        "",
		script:      "",
		logger:      nil,
		longRunning: map[string]*LongRunningProcess{},
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
		enabled:     true,
		path:        path,
		file:        filepath.Join(path, file),
		script:      script,
		logger:      log,
		longRunning: map[string]*LongRunningProcess{},
	}
	lrm.UpdateLongRunningProcess()
	return lrm, nil
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
	return len(p.longRunning)
}

func (p *LongRunningManager) String() string {
	if p.enabled {
		return fmt.Sprintf("File:%s. Script:%s", p.file, p.script)
	}
	return "Long Running Process Manager is disabled"
}

func (p *LongRunningManager) ToJson() string {
	p.UpdateLongRunningProcess()
	if len(p.longRunning) > 0 {
		var b bytes.Buffer
		b.WriteRune('[')
		for n, v := range p.longRunning {
			b.WriteString(fmt.Sprintf("{\"Name\":\"%s\", \"Started\":\"%s\", \"PID\":%d, \"CanStop\":%t},", n, v.GetStartTime(), v.PID, v.CanStop))
		}
		return b.String()[:b.Len()-1] + "]"
	}
	return "[]"
}

func (p *LongRunningManager) load() {
	if p.enabled {
		content, err := os.ReadFile(p.file)
		if err != nil {
			p.longRunning = map[string]*LongRunningProcess{}
			return
		}
		err = json.Unmarshal(content, &p.longRunning)
		if err != nil {
			p.longRunning = map[string]*LongRunningProcess{}
			return
		}
		p.Log("LongRunningManager Loaded: " + p.file)

	}
}

func (p *LongRunningManager) store() error {
	if p.enabled {
		data, err := json.MarshalIndent(p.longRunning, "", "  ")
		if err != nil {
			return err
		}
		err = os.WriteFile(p.file, data, os.ModePerm)
		if err != nil {
			return err
		}
		p.Log("LongRunningManager Stored: " + p.file)
	}
	return nil
}

func (p *LongRunningManager) AddLongRunningProcess(id string, pid int, canStop bool, commit bool) bool {
	if p.enabled {
		p.longRunningMutex.Lock()
		defer p.longRunningMutex.Unlock()
		lrp := NewLongRunningProcess(id, pid, canStop)
		if p.longRunning[lrp.key()] == nil {
			if commit {
				p.longRunning[lrp.key()] = lrp
				p.Log("Process added. " + lrp.ID)
				p.store()
			}
			return true
		}
	} else {
		return true
	}
	return false
}

func (p *LongRunningManager) UpdateLongRunningProcess() {
	if p.enabled {
		p.longRunningMutex.Lock()
		defer p.longRunningMutex.Unlock()
		p.load()
		upd := false
		lrp := map[string]*LongRunningProcess{}
		for _, v := range p.longRunning { // Example pid
			process, _ := os.FindProcess(v.PID) // Always succeeds on Unix systems
			err := process.Signal(syscall.Signal(0))
			if err != nil {
				lrp[v.key()] = v
			} else {
				upd = true
				p.Log(fmt.Sprintf("UpdateLongRunningProcess PID: %d NO longer running [%s]", v.PID, v.ID))
			}
		}
		if upd {
			p.longRunning = lrp
			p.store()
		}
	}
}

func NewLongRunningProcess(id string, pid int, canStop bool) *LongRunningProcess {
	return &LongRunningProcess{
		ID:      id,
		PID:     pid,
		Started: time.Now(),
		CanStop: canStop,
	}
}

func (p *LongRunningProcess) key() string {
	return p.ID
}

func (p *LongRunningProcess) GetStartTime() string {
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		p.Started.Year(), p.Started.Month(), p.Started.Day(),
		p.Started.Hour(), p.Started.Minute(), p.Started.Second())
}
