package runCommand

import (
	"bytes"
	"fmt"
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

func (p *execData) Run(done func(string, string)) error {
	cmd := exec.Command(p.Cmd[0], p.Cmd[1:]...)
	if p.Dir != "" {
		cmd.Dir = p.Dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command failed %e", err)
	}
	done(stdout.String(), stderr.String())
	return err
}
