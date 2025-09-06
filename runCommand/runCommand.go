package runCommand

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type execData struct {
	Cmd          []string
	Dir          string
	StdOutLog    string
	StdErrLog    string
	StartLTSFile string
	log          func(string)
	id           string
	detached     bool
	canStop      bool
}

type ExecError struct {
	message string
	id      string
	status  int
	log     string
}

func NewExecError(msg, id, log string, status int) *ExecError {
	return &ExecError{
		message: msg,
		id:      id,
		log:     log,
		status:  status,
	}
}

func (ee *ExecError) Error() string {
	return fmt.Sprintf("Exec Error. Status:%d ID:'%s'. %s", ee.status, ee.id, ee.message)
}

func (ee *ExecError) Map() map[string]interface{} {
	m := make(map[string]interface{})
	m["error"] = true
	m["status"] = ee.Status()
	m["id"] = ee.id
	m["msg"] = ee.String()
	return m
}

func (ee *ExecError) LogError() string {
	return fmt.Sprintf("%s. %s", ee.Error(), ee.log)
}

func (ee *ExecError) String() string {
	return ee.message
}
func (ee *ExecError) Status() int {
	return ee.status
}

func (p *execData) String() string {
	return fmt.Sprintf("CMD:%s, Dir:%s, LogOut:%s, LogErr:%s", p.Cmd, p.Dir, p.StdOutLog, p.StdErrLog)
}

func NewExecData(commands []string, dir string, stdOut string, stdErr string, id string, startLTSFile string, detached bool, canStop bool, logFunc func(string), substitute func([]byte) string) *execData {
	var subCmd []string
	if substitute != nil {
		subCmd = make([]string, len(commands))
		for pos, cmd := range commands {
			subCmd[pos] = substitute([]byte(cmd))
		}
	} else {
		subCmd = commands
	}

	return &execData{
		Cmd:          subCmd,
		Dir:          dir,
		StdOutLog:    stdOut,
		StdErrLog:    stdErr,
		StartLTSFile: startLTSFile,
		log:          logFunc,
		id:           id,
		detached:     detached,
		canStop:      canStop,
	}
}

func (p *execData) RunSystemProcess() ([]byte, []byte, int) {
	if p.detached {
		if p.StdOutLog != "" {
			panic(NewExecError("Detached process cannot use StdOutLog", p.id, "Config error", http.StatusExpectationFailed))
		}
		if p.StdErrLog != "" {
			panic(NewExecError("Detached process cannot use StdErrLog", p.id, "Config error", http.StatusExpectationFailed))
		}
	}
	pruned := []string{}
	for _, v := range p.Cmd {
		vTrim := strings.TrimSpace(v)
		if vTrim != "" {
			pruned = append(pruned, vTrim)
		}
	}
	if len(pruned) == 0 {
		panic(NewExecError("No command were given", p.id, "Config error", http.StatusExpectationFailed))
	}

	currentDir, err := os.Getwd()
	if err != nil {
		panic(NewExecError("Could not read working dir", p.id, fmt.Sprintf("Path error: os.Getwd(). Error:%s", err.Error()), http.StatusInternalServerError))
	}

	commandX := pruned[0]
	if p.Dir != "" {
		fp, err := filepath.Abs(p.Dir)
		if err != nil {
			panic(NewExecError("Could not get absolute path of exec dir", p.id, fmt.Sprintf("Path error: filepath.Abs(%s). Error:%s", p.Dir, err.Error()), http.StatusInternalServerError))
		}
		_, err = os.Stat(fp)
		if err != nil {
			panic(NewExecError("Could find exec dir", p.id, fmt.Sprintf("Path error: os.Stat(%s). Error:%s", fp, err.Error()), http.StatusFailedDependency))
		}
		err = os.Chdir(fp)
		if err != nil {
			panic(NewExecError("Could select exec dir", p.id, fmt.Sprintf("Path error: os.Chdir(%s). Error:%s", fp, err.Error()), http.StatusFailedDependency))
		}
		defer os.Chdir(currentDir)

		commandX = filepath.Join(fp, pruned[0])
	}

	var cmd *exec.Cmd
	if len(pruned) == 1 {
		cmd = exec.Command(commandX)
	} else {
		cmd = exec.Command(commandX, p.Cmd[1:]...)
	}

	var stdout, stderr bytes.Buffer
	code := 0
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if p.detached {
		pidx := FindProcessIdWithName(pruned[0])
		if pidx != 0 {
			panic(NewExecError("Process is already running", p.id, fmt.Sprintf("Process '%s' already running. PID:%d", p.id, pidx), http.StatusBadRequest))
		}

		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		err = cmd.Start()
		if err != nil {
			panic(NewExecError("Detached Process could not be started", p.id, fmt.Sprintf("Config error: cmd.Start(). Error:%s", err.Error()), http.StatusFailedDependency))
		}
		pid := cmd.Process.Pid
		cmd.Process.Release()

		startLTSText := p.readStartLTSFile("<br>")

		m := make(map[string]interface{})
		m["error"] = false
		m["pid"] = pid
		m["id"] = p.id
		if startLTSText != "" {
			m["StartLTSText"] = startLTSText
		}
		v, err := json.Marshal(m)
		if err != nil {
			panic(NewExecError("Failed to marshal response JSON", p.id, fmt.Sprintf("System error: runCommand:RunSystemProcess:Marshal(m). Error:%s", err.Error()), http.StatusInternalServerError))
		}
		return v, stderr.Bytes(), 0
	}

	err = cmd.Run()
	if err != nil {
		_, ok := err.(*os.PathError)
		if ok {
			panic(NewExecError("Could find exec dir", p.id, fmt.Sprintf("Path error:%s", err.Error()), http.StatusFailedDependency))
		}
		ee, ok := err.(*exec.ExitError)
		if ok {
			code = ee.ExitCode()
		} else {
			panic(NewExecError("Exec failed", p.id, fmt.Sprintf("Exec error:%s", err.Error()), http.StatusFailedDependency))
		}
	}
	sob := stdout.Bytes()
	if p.StdOutLog != "" {
		if len(sob) > 0 {
			err = os.WriteFile(p.StdOutLog, sob, 0644)
			if err != nil {
				panic(NewExecError("Could not write to StdOut log", p.id, fmt.Sprintf("Config error:%s", err.Error()), http.StatusFailedDependency))
			}
		}
	}
	seb := stderr.Bytes()
	if p.StdErrLog != "" {
		if len(seb) > 0 {
			err = os.WriteFile(p.StdErrLog, seb, 0644)
			if err != nil {
				panic(NewExecError("Could not write to StdErr log", p.id, fmt.Sprintf("Config error:%s", err.Error()), http.StatusFailedDependency))
			}
		}
	}
	return sob, seb, code
}

func (p *execData) readStartLTSFile(lineSep string) string {
	if p.StartLTSFile != "" {
		time.Sleep(time.Second)
		file, err := os.Open(p.StartLTSFile)
		if err != nil {
			return "" // If no error file then no error
		}
		defer file.Close()
		var buff bytes.Buffer
		scanner := bufio.NewScanner(file)
		len := buff.Len()
		for scanner.Scan() {
			buff.WriteString(strings.TrimSpace(scanner.Text()))
			len = buff.Len()
			if len > 100 {
				break
			}
			buff.WriteRune(' ')
			buff.WriteString(lineSep)
		}
		buff.Truncate(len)
		if err := scanner.Err(); err != nil {
			return fmt.Sprintf("readStartLTSFile:Scanner:Error:%s: %s", err.Error(), buff.String())
		}
		if buff.Len() > 0 {
			return buff.String()
		}
	}
	return ""
}

func FindProcessIdWithName(execCmd string) int {
	id := 0
	ForEachSystemProcess(func(cmd string, i int) (bool, error) {
		if strings.HasSuffix(cmd, execCmd) {
			id = i
			return true, nil
		}
		return false, nil
	})
	return id
}

func KillrocessWithPid(id int) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("kill", strconv.Itoa(id))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		panic(NewExecError("Process could not be stopped", "KillrocessWithPid", fmt.Sprintf("Kill process %d failed with error:%s", id, err.Error()), http.StatusFailedDependency))
	}
}

func ForEachSystemProcess(fe func(string, int) (bool, error)) (int, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("ps", "-eo", "pid,start,cmd")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return 0, err
	}
	count := 0
	lc := 0
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		l := scanner.Text()
		id := 0
		idEnd := 0
		// Skip lines with ] at the end of the command and line 0
		if !strings.HasSuffix(l, "]") && lc > 0 {
			for i, c := range l {
				if c >= '0' && c <= '9' {
					id = id*10 + int(c) - '0'
				} else {
					if id > 0 && i < 10 {
						idEnd = i
						break
					}
				}
			}
			if id > 100 {
				found, err := fe(l[idEnd:], id)
				if err != nil {
					return count, err
				}
				if found {
					count++
				}
			}
		}
		lc++
	}
	return count, nil
}
