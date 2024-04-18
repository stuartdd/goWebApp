package runCommand

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
}

func (p *execData) String() string {
	return fmt.Sprintf("CMD:%s, Dir:%s, LogOut:%s, LogErr:%s", p.Cmd, p.Dir, p.StdOutLog, p.StdErrLog)
}

func NewExecData(commands []string, dir string, stdOut string, stdErr string, info string, detached bool, logFunc func(string), substitute func([]byte) string) *execData {
	var subCmd []string
	if substitute != nil {
		subCmd = make([]string, len(commands))
		for pos, cmd := range commands {
			subCmd[pos] = substitute([]byte(cmd))
		}
	} else {
		subCmd = commands
	}
	// if detached {
	// 	subCmd = append(subCmd, "&")
	// }
	return &execData{
		Cmd:       subCmd,
		Dir:       dir,
		StdOutLog: stdOut,
		StdErrLog: stdErr,
		log:       logFunc,
		info:      info,
		detached:  detached,
	}
}

func (p *execData) Run() ([]byte, []byte, int, error) {
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

	var cmd *exec.Cmd
	if len(pruned) == 1 {
		cmd = exec.Command(pruned[0])
	} else {
		cmd = exec.Command(pruned[0], p.Cmd[1:]...)
	}
	if p.Dir != "" {
		cmd.Dir = p.Dir
	}

	var stdout, stderr bytes.Buffer
	code := 0
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if p.detached {
		err := cmd.Start()
		if err != nil {
			stdout.WriteString("Error:")
			stdout.WriteString(err.Error())
			return nil, nil, -1, err
		}
		cmd.Process.Release()
		return stdout.Bytes(), stderr.Bytes(), 0, nil
	}

	err := cmd.Run()
	if err != nil {
		if p.log != nil {
			p.log(fmt.Sprintf("Error Exec        :%s, %s", p.info, err.Error()))
		}
		_, ok := err.(*os.PathError)
		if ok {
			return nil, nil, -1, fmt.Errorf("exec failed: Invalid path to command")
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
