package runCommand

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type execData struct {
	Cmd       []string
	Dir       string
	StdOutLog string
	StdErrLog string
	log       func(string)
	info      string
	detached  bool
	canStop   bool
}

func (p *execData) String() string {
	return fmt.Sprintf("CMD:%s, Dir:%s, LogOut:%s, LogErr:%s", p.Cmd, p.Dir, p.StdOutLog, p.StdErrLog)
}

func NewExecData(commands []string, dir string, stdOut string, stdErr string, info string, detached bool, canStop bool, logFunc func(string), substitute func([]byte) string) *execData {
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
		Cmd:       subCmd,
		Dir:       dir,
		StdOutLog: stdOut,
		StdErrLog: stdErr,
		log:       logFunc,
		info:      info,
		detached:  detached,
		canStop:   canStop,
	}
}

func (p *execData) RunSystemProcess() ([]byte, []byte, int, error) {
	if p.detached {
		if len(p.StdOutLog) != 0 {
			return nil, nil, -1, fmt.Errorf("exec detached cannot use stdOut")
		}
		if len(p.StdErrLog) != 0 {
			return nil, nil, -1, fmt.Errorf("exec detached cannot use stdErr")
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
		return nil, nil, -1, fmt.Errorf("exec failed: no commands")
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, nil, -1, fmt.Errorf("exec failed: Get Working Dir. %s", err.Error())
	}

	commandX := pruned[0]
	if p.Dir != "" {
		fp, err := filepath.Abs(p.Dir)
		if err != nil {
			return nil, nil, -1, fmt.Errorf("exec failed: could not make dir ABS. %s", p.Dir)
		}
		_, err = os.Stat(fp)
		if err != nil {
			return nil, nil, -1, fmt.Errorf("exec failed: %s", err.Error())
		}
		os.Chdir(fp)
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
		_, err := ForEachSystemProcess(func(cmd string, i int) (bool, error) {
			if strings.HasSuffix(cmd, pruned[0]) {
				return true, fmt.Errorf("process %s is already running", pruned[0])
			}
			return false, nil
		})
		if err != nil {
			return nil, nil, 429, err
		}
		err = cmd.Start()
		if err != nil {
			stdout.WriteString("{\"Error\":true")
			for i, v := range pruned {
				stdout.WriteString(fmt.Sprintf(", \"P%d\":\"%s\"", i, v))
			}
			stdout.WriteString("}")
			return stdout.Bytes(), stderr.Bytes(), -1, err
		}
		pid := cmd.Process.Pid
		cmd.Process.Release()
		stdout.WriteString(fmt.Sprintf("{\"Error\":false, \"pid\":%d", pid))
		for i, v := range pruned {
			stdout.WriteString(fmt.Sprintf(", \"P%d\":\"%s\"", i, v))
		}
		stdout.WriteString("}")
		return stdout.Bytes(), stderr.Bytes(), 0, nil
	}

	err = cmd.Run()
	if err != nil {
		if p.log != nil {
			p.log(fmt.Sprintf("Error Exec        :%s, %s", p.info, err.Error()))
		}
		_, ok := err.(*os.PathError)
		if ok {
			return nil, nil, -1, fmt.Errorf("exec failed: Invalid path to command. %s", err.Error())
		}
		ee, ok := err.(*exec.ExitError)
		if ok {
			code = ee.ExitCode()
		} else {
			return nil, nil, 1, fmt.Errorf("exec failed: %s", err.Error())
		}
	}
	sob := stdout.Bytes()
	if p.StdOutLog != "" {
		if len(sob) > 0 {
			err = os.WriteFile(p.StdOutLog, sob, 0644)
			if err != nil {
				return nil, nil, -1, fmt.Errorf("exec failed: could not write stdout %s", err.Error())
			}
		}
	}
	seb := stderr.Bytes()
	if p.StdErrLog != "" {
		if len(seb) > 0 {
			err = os.WriteFile(p.StdErrLog, seb, 0644)
			if err != nil {
				return nil, nil, -1, fmt.Errorf("exec failed: could not write stderr %s", err.Error())
			}
		}
	}
	return sob, seb, code, nil
}

func KillrocessWithName(path, execid string) {
	id := 0
	ForEachSystemProcess(func(cmd string, i int) (bool, error) {
		if strings.Contains(cmd, filepath.Join(path, execid)) {
			id = i
			return true, nil
		}
		return false, nil
	})
	if id == 0 {
		panic(fmt.Errorf("running process with ID:%s could not be found", execid))
	}
	 KillrocessWithPid(id)
}

func KillrocessWithPid(id int)  {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("kill", strconv.Itoa(id))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		panic(fmt.Errorf("process with PID:%d could not be stopped. Cmd error: %s", id, err))
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
