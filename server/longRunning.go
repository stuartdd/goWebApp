package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"stuartdd.com/runCommand"
)

type LongRunningProcess struct {
	User    string
	ID      string
	Started time.Time
	PID     int
}

type LongRunningManager struct {
	enabled          bool
	path             string
	file             string
	script           string
	longRunning      map[string]*LongRunningProcess
	logger           func(string)
	message          string
	longRunningMutex sync.Mutex
}

func NewLongRunningManagerDisabled() *LongRunningManager {
	return &LongRunningManager{
		enabled:     false,
		path:        "",
		file:        "",
		script:      "",
		message:     "",
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
		message:     "",
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
	}
	return nil
}

func (p *LongRunningManager) AddLongRunningProcess(user string, id string, pid int, commit bool) bool {
	if p.enabled {
		p.longRunningMutex.Lock()
		defer p.longRunningMutex.Unlock()
		lrp := NewLongRunningProcess(user, id, pid)
		if p.longRunning[lrp.key()] == nil {
			if commit {
				p.longRunning[lrp.key()] = lrp
				p.Log("Process added. " + lrp.Strings()[1])
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
		for _, v := range p.longRunning {
			ex := runCommand.NewExecData([]string{p.script, strconv.Itoa(v.PID)}, "", "", "", "", false, nil, nil, nil)
			stdout, _, _, err := ex.Run()
			if err != nil {
				p.message = strings.ReplaceAll(fmt.Sprintf("UpdateLongRunningProcess ERROR: %v", err), "\"", "'")
				p.Log(p.message)
				return
			}
			output := strings.TrimSpace(string(stdout))
			running := output != "" && !strings.Contains(output, "defunct")
			if running {
				lrp[v.key()] = v
			} else {
				upd = true
				p.Log(fmt.Sprintf("UpdateLongRunningProcess PID: %d NO longer running [%s]", v.PID, output))
			}
		}
		if upd {
			p.longRunning = lrp
			p.store()
		}
	}
}

func (p *LongRunningManager) LongRunningMap() map[string]string {
	list := map[string]string{}
	if p.enabled {
		if p.message != "" {
			list["error"] = p.message
		}
		for _, v := range p.longRunning {
			vl := v.Strings()
			list[vl[0]] = vl[1]
		}
	}
	return list
}

func NewLongRunningProcess(user, id string, pid int) *LongRunningProcess {
	return &LongRunningProcess{
		User:    user,
		ID:      id,
		PID:     pid,
		Started: time.Now(),
	}
}

func (p *LongRunningProcess) key() string {
	return p.User + "-" + p.ID
}

func (p *LongRunningProcess) GetStartTime() string {
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		p.Started.Year(), p.Started.Month(), p.Started.Day(),
		p.Started.Hour(), p.Started.Minute(), p.Started.Second())
}

func (p *LongRunningProcess) Strings() []string {
	return []string{p.key(), fmt.Sprintf("User:%s ExecId:%s Run:%s PID:%d", p.User, p.ID, p.GetStartTime(), p.PID)}
}
