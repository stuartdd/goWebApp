package runCommand

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type LongRunningProcess struct {
	ID          string // The execId in the config exec list
	Exec        string // Cmd[0] from config exec
	Description string // Description from config exec list
	PID         int    // The process id PID from the running process
	CanStop     bool   // The ability to terminate the process while running
}

type LongRunningManager struct {
	enabled            bool
	path               string
	longRunningProcess map[string]*LongRunningProcess
	logger             func(string)
}

func NewLongRunningManagerDisabled() *LongRunningManager {
	return &LongRunningManager{
		enabled:            false,
		path:               "",
		logger:             nil,
		longRunningProcess: map[string]*LongRunningProcess{},
	}
}

func NewLongRunningManager(path string, log func(string)) (*LongRunningManager, error) {
	if path == "" {
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
		logger:             log,
		longRunningProcess: map[string]*LongRunningProcess{},
	}
	return lrm, nil
}

func (p *LongRunningManager) AddLongRunningProcessData(id string, desc string, cmdList []string, canStop bool) error {
	if len(cmdList) == 0 {
		return fmt.Errorf("detached command: Exec:%s is empty", id)
	}
	exec := strings.TrimSpace(cmdList[0])
	if exec == "" {
		return fmt.Errorf("detached command: Exec:%s has no command: %s", id, cmdList)
	}
	if strings.HasPrefix(exec, ".") || strings.HasPrefix(exec, string(os.PathSeparator)) {
		return fmt.Errorf("detached command: Exec:%s command MUST be in the ExecManager:Path dir and not hidden: %s", id, cmdList)
	}
	for _, v := range p.longRunningProcess {
		if v.ID == exec {
			return fmt.Errorf("detached command: Exec:%s has duplicate executable. Cannot reliably identify the process: %s", id, exec)
		}
	}
	execFile := filepath.Join(p.path, exec)
	st, err := os.Stat(execFile)
	if err != nil {
		return fmt.Errorf("detached command: Exec:%s Not found in the ExecManager:Path dir: %s", id, exec)
	}
	if st.IsDir() {
		return fmt.Errorf("detached command: Exec:%s Exec is a directory in the ExecManager:Path dir: %s", id, exec)
	}
	if desc == "" {
		desc = fmt.Sprintf("ID[%s] Cmd:%s", id, exec)
	}
	p.longRunningProcess[id] = &LongRunningProcess{
		ID:          id,
		Exec:        exec,
		PID:         0,
		Description: desc,
		CanStop:     canStop,
	}
	return nil
}

func (p *LongRunningManager) IsEnabled() bool {
	return p.enabled
}

func (p *LongRunningManager) Update() {
	for _, v := range p.longRunningProcess {
		v.PID = FindProcessIdWithName(v.ID)
	}
}

func (p *LongRunningManager) Len() int {
	return len(p.longRunningProcess)
}

func (p *LongRunningManager) String() string {
	if p.enabled {
		return fmt.Sprintf("ExecPath:%s", p.path)
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
			b.WriteString("\",\"Desc\":\"")
			b.WriteString(v.Description)
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
