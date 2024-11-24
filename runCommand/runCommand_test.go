package runCommand

import (
	"strings"
	"testing"
)

func TestNoCommands(t *testing.T) {
	tc := NewExecData([]string{}, "..", "../testdata/logs/cmdout.txt", "../testdata/logs/cmderr.txt", "info", false, true, nil, nil, nil)
	_, _, _, err := tc.Run()
	if err == nil {
		t.Fatal("Should throw no commands error")
	}
	if !strings.Contains(err.Error(), "no commands") {
		t.Fatal("Error contain 'no commands'")
	}
}

func TestEmptyCommands(t *testing.T) {
	tc := NewExecData([]string{"", " "}, "..", "../testdata/logs/cmdout.txt", "../testdata/logs/cmderr.txt", "info", false, false, nil, nil, nil)
	_, _, _, err := tc.Run()
	if err == nil {
		t.Fatal("Should throw no commands error")
	}
	if !strings.Contains(err.Error(), "no commands") {
		t.Fatal("Error contain 'no commands'")
	}
}

func TestLs(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta"}, "..", "../testdata/logs/cmdout-ls.txt", "../testdata/logs/cmderr-ls.txt", "info", false, false, nil, nil, nil)
	stdOut, _, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}
	if !strings.Contains(string(stdOut), "go.mod") {
		t.Fatal(string(stdOut))
	}
}

func TestLsErr(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta", "x"}, "..", "../testdata/logs/cmdout-ls-x.txt", "../testdata/logs/cmderr-ls-x.txt", "info", false, false, nil, nil, nil)
	stdOut, stdErr, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}
	if !strings.Contains(string(stdErr), "'x': No such") {
		t.Fatal(string(stdErr))
	}
	if len(stdOut) > 0 {
		t.Fatal("StdOut should be empty")
	}
}

func TestPWD(t *testing.T) {

	tc := NewExecData([]string{"pwd"}, "..", "../testdata/logs/cmdout-pwd.txt", "../testdata/logs/cmderr-pwd.txt", "info", false, false, nil, nil, nil)
	stdOut, _, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}

	if !strings.Contains(string(stdOut), "/goWebApp") {
		t.Fatal(string(stdOut))
	}

}
func TestGo(t *testing.T) {

	tc := NewExecData([]string{"go", "version"}, "", "../testdata/logs/cmdout-go.txt", "../testdata/logs/cmderr-go.txt", "info", false, false, nil, nil, nil)
	stdOut, _, _, err := tc.Run()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}

	if !strings.Contains(string(stdOut), "go version") {
		t.Fatal(string(stdOut))
	}
}
