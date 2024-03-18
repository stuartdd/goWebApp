package runCommand

import (
	"strings"
	"testing"
)

func TestNoCommands(t *testing.T) {
	tc := NewExecData([]string{}, "..", "cmdout.txt", nil)
	_, _, _, err := tc.Run()
	if err == nil {
		t.Fatalf("Should throw no commands error")
	}
	if !strings.Contains(err.Error(), "no commands") {
		t.Fatalf("Error contain 'no commands'")
	}
}
func TestEmptyCommands(t *testing.T) {
	tc := NewExecData([]string{"", " "}, "..", "cmdout.txt", nil)
	_, _, _, err := tc.Run()
	if err == nil {
		t.Fatalf("Should throw no commands error")
	}
	if !strings.Contains(err.Error(), "no commands") {
		t.Fatalf("Error contain 'no commands'")
	}
}

func TestSingleCmd(t *testing.T) {
	tc := NewExecData([]string{"pwd", ""}, "..", "cmdout.txt", nil)
	stdOut, _, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}
	if !strings.Contains(string(stdOut), "/goWebApp") {
		t.Fatalf(string(stdOut))
	}
}

func TestLs(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta"}, "..", "cmdout.txt", nil)
	stdOut, _, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}
	if !strings.Contains(string(stdOut), "go.mod") {
		t.Fatalf(string(stdOut))
	}
}

func TestPWD(t *testing.T) {

	tc := NewExecData([]string{"pwd"}, "..", "cmdout.txt", nil)
	stdOut, _, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}

	if !strings.Contains(string(stdOut), "/goWebApp") {
		t.Fatalf(string(stdOut))
	}

}
func TestGo(t *testing.T) {

	tc := NewExecData([]string{"go", "version"}, "", "cmdout.txt", nil)
	stdOut, _, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}

	if !strings.Contains(string(stdOut), "go version") {
		t.Fatalf(string(stdOut))
	}
}
