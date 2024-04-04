package runCommand

import (
	"strings"
	"testing"
)

func TestNoCommands(t *testing.T) {
	tc := NewExecData([]string{}, "..", "../testdata/logs/cmdout.txt", "../testdata/logs/cmderr.txt", "info", nil, nil)
	_, _, _, err := tc.Run()
	if err == nil {
		t.Fatalf("Should throw no commands error")
	}
	if !strings.Contains(err.Error(), "no commands") {
		t.Fatalf("Error contain 'no commands'")
	}
}

func TestEmptyCommands(t *testing.T) {
	tc := NewExecData([]string{"", " "}, "..", "../testdata/logs/cmdout.txt", "../testdata/logs/cmderr.txt", "info", nil, nil)
	_, _, _, err := tc.Run()
	if err == nil {
		t.Fatalf("Should throw no commands error")
	}
	if !strings.Contains(err.Error(), "no commands") {
		t.Fatalf("Error contain 'no commands'")
	}
}

func TestLs(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta"}, "..", "../testdata/logs/cmdout-ls.txt", "../testdata/logs/cmderr-ls.txt", "info", nil, nil)
	stdOut, _, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}
	if !strings.Contains(string(stdOut), "go.mod") {
		t.Fatalf(string(stdOut))
	}
}

func TestLsErr(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta", "x"}, "..", "../testdata/logs/cmdout-ls-x.txt", "../testdata/logs/cmderr-ls-x.txt", "info", nil, nil)
	stdOut, stdErr, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}
	if !strings.Contains(string(stdErr), "'x': No such") {
		t.Fatalf(string(stdErr))
	}
	if len(stdOut) > 0 {
		t.Fatalf("StdOut should be empty")
	}
}

func TestPWD(t *testing.T) {

	tc := NewExecData([]string{"pwd"}, "..", "../testdata/logs/cmdout-pwd.txt", "../testdata/logs/cmderr-pwd.txt", "info", nil, nil)
	stdOut, _, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}

	if !strings.Contains(string(stdOut), "/goWebApp") {
		t.Fatalf(string(stdOut))
	}

}
func TestGo(t *testing.T) {

	tc := NewExecData([]string{"go", "version"}, "", "../testdata/logs/cmdout-go.txt", "../testdata/logs/cmderr-go.txt", "info", nil, nil)
	stdOut, _, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}

	if !strings.Contains(string(stdOut), "go version") {
		t.Fatalf(string(stdOut))
	}
}
