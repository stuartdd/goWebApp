package runCommand

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

type execData struct {
	Cmd    []string
	Dir    string
	StdOut string
}

func (p *execData) ToString() string {
	return fmt.Sprintf("CMD:%s, Dir:%s, Log:%s", p.Cmd, p.Dir, p.StdOut)
}

func NewExecData(commands []string, dir string, stdOut string) *execData {
	return &execData{
		Cmd:    commands,
		Dir:    dir,
		StdOut: stdOut,
	}
}

func (p *execData) Run() ([]byte, []byte, int, error) {
	cmd := exec.Command(p.Cmd[0], p.Cmd[1:]...)
	if p.Dir != "" {
		cmd.Dir = p.Dir
	}
	var stdout, stderr bytes.Buffer
	code := 0
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		_, ok := err.(*os.PathError)
		if ok {
			return nil, nil, -1, fmt.Errorf("exec failed to change dir")
		}
		ee, ok := err.(*exec.ExitError)
		if ok {
			code = ee.ExitCode()
		} else {
			ex, ok := err.(*exec.Error)
			if ok {
				return nil, nil, 1, fmt.Errorf("exec cmd %s not on Path", ex.Name)
			}
			return nil, nil, 1, fmt.Errorf("exec failed %s", err.Error())
		}
	}
	if p.StdOut != "" {
		err = os.WriteFile(p.StdOut, stdout.Bytes(), 0644)
		if err != nil {
			return nil, nil, -1, fmt.Errorf("exec failed to write stdout %s", err.Error())
		}
	}
	return stdout.Bytes(), stderr.Bytes(), code, nil
}
