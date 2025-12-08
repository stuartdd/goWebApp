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

// Cmd[0] is the script name (shell, cmd,,,). Each parameter is Cmd[n]
// All scripts M|UST| be inb the Exec Path. No Exec Path will return 404
type execData struct {
	Cmd          []string     // ScriptName (bash/cmd file) and args
	StdOutLog    string       // If defined StdOut returned to to process is written to this file
	StdErrLog    string       // If defined StdErr returned to to process is written to this file
	log          func(string) // Log details from this process
	StartLTSFile string       // File contains output from Cmd that is returned immedialty. EG Errors, proc info...
	id           string       // Identity uses to track Long Running Processes. Get PID via FindProcessIdWithName(id)
	detached     bool         // Detached indicates a  Long Running Processes
	canStop      bool         // If detached then it ncan be stopped using KillrocessWithPid
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
	m["msg"] = http.StatusText(ee.status)
	m["cause"] = ee.String()
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

func NewExecData(commands []string, stdOut string, stdErr string, id string, startLTSFile string, detached bool, canStop bool, logFunc func(string), substitute func([]byte) string) *execData {
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
		StdOutLog:    stdOut,
		StdErrLog:    stdErr,
		StartLTSFile: startLTSFile,
		log:          logFunc,
		id:           id,
		detached:     detached,
		canStop:      canStop,
	}
}

func (p *execData) Validate(addError func(string)) *execData {
	return p
}

func (p *execData) String() string {
	return fmt.Sprintf("CMD:%s, LogOut:%s, LogErr:%s", p.Cmd, p.StdOutLog, p.StdErrLog)
}

func (p *execData) RunSystemProcess(execDir string) ([]byte, []byte, int) {
	if execDir == "" {
		panic(NewExecError("Exec path is undefined", p.id, "Config error", http.StatusInternalServerError))
	}
	absExecDir, err := filepath.Abs(execDir)
	if err != nil {
		panic(NewExecError("Could not get absolute path of exec dir", p.id, fmt.Sprintf("Path error: filepath.Abs(%s). Error:%s", execDir, err.Error()), http.StatusInternalServerError))
	}
	stat, err := os.Stat(absExecDir)
	if err != nil {
		panic(NewExecError("Could find exec dir", p.id, fmt.Sprintf("Path error: os.Stat(%s). Error:%s", absExecDir, err.Error()), http.StatusFailedDependency))
	}
	if !stat.IsDir() {
		panic(NewExecError("Exec path is not a directory", p.id, fmt.Sprintf("Path error: os.Stat(%s). Error:%s", absExecDir, "Must be a directory"), http.StatusFailedDependency))
	}

	// Trim spaces from command and args then check there are some!
	cleanCmd := []string{}
	for _, v := range p.Cmd {
		vTrim := strings.TrimSpace(v)
		if vTrim != "" {
			cleanCmd = append(cleanCmd, vTrim)
		}
	}
	if len(cleanCmd) == 0 {
		panic(NewExecError("No command given", p.id, "Config error", http.StatusExpectationFailed))
	}

	if p.detached {
		if p.StdOutLog != "" {
			panic(NewExecError("Detached process cannot use StdOutLog", p.id, "Config error", http.StatusExpectationFailed))
		}
		if p.StdErrLog != "" {
			panic(NewExecError("Detached process cannot use StdErrLog", p.id, "Config error", http.StatusExpectationFailed))
		}
	}
	cmdX := filepath.Join(absExecDir, cleanCmd[0])
	var cmd *exec.Cmd
	if len(cleanCmd) == 1 {
		cmd = exec.Command(cmdX)
	} else {
		cmd = exec.Command(cmdX, p.Cmd[1:]...)
	}
	cmd.Dir = execDir
	stat, err = os.Stat(filepath.Join(absExecDir, cleanCmd[0]))
	if err != nil {
		panic(NewExecError("Could not find cmd script", p.id, fmt.Sprintf("Path error: os.Stat(%s). Error:%s", filepath.Join(absExecDir, cleanCmd[0]), err.Error()), http.StatusFailedDependency))
	}
	if stat.IsDir() {
		panic(NewExecError("Cmd script is not a file", p.id, fmt.Sprintf("Path error: os.Stat(%s). Error:%s", filepath.Join(absExecDir, cleanCmd[0]), "Exec is a directory"), http.StatusFailedDependency))
	}

	var stdout, stderr bytes.Buffer
	code := 0
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if p.detached {
		pidx := FindProcessIdWithName(cleanCmd[0])
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
