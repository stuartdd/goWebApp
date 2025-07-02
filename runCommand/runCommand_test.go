package runCommand

import (
	"os"
	"strings"
	"testing"
	"time"
)

const PROC_NAME = "longRunTest.sh"
const PROC_PATH = "exec/" + PROC_NAME

func TestDetatch(t *testing.T) {
	wd, _ := os.Getwd()

	ForEachSystemProcess(func(cmd string, p int) bool {
		if strings.HasSuffix(cmd, PROC_NAME) {
			KillrocessWithId(p)
			return true
		}
		return false
	})

	tc := NewExecData([]string{PROC_NAME}, "../testdata/exec", "", "", "info", true, true, nil, nil)
	_, _, _, err := tc.RunSystemProcess()
	if err != nil {
		t.Fatalf("Run Should NOT throw error %s", err.Error())
	}
	cd, _ := os.Getwd()
	if wd != cd {
		t.Fatalf("Working dis has changed. From %s To %s", wd, cd)
	}
	s := ""
	pid := 0

	count, err := ForEachSystemProcess(func(cmd string, p int) bool {
		if strings.HasSuffix(cmd, PROC_NAME) {
			s = cmd
			pid = p
			return true
		}
		return false
	})
	if err != nil {
		t.Fatal("FindProcessWithName Should NOT throw an error")
	}
	if count != 1 {
		t.Fatalf("FindProcessWithName Should find 1 process. Not : %d", count)
	}
	if !strings.Contains(s, PROC_PATH) {
		t.Fatalf("FindProcessWithName should contain '%s'", PROC_PATH)
	}
	if pid < 100 {
		t.Fatalf("FindProcessWithName should return a valid pid. Not %d", pid)
	}

	var filSize int64 = 0
	for range 2 {
		time.Sleep(time.Millisecond * 1030)
		st, err := os.Stat("../testdata/exec/LongRunTest.txt")
		if err != nil {
			t.Fatal(err.Error())
		}
		if st == nil {
			t.Fatal("os.Stat(\"../testdata/exec/LongRunTest.txt\") returned nil")
		}
		if st.Size() > filSize {
			filSize = st.Size()
		} else {
			t.Fatal("../testdata/exec/LongRunTest.txt is not increasing in size")
		}
	}

	err = KillrocessWithId(pid)
	if err != nil {
		t.Fatalf("Kill Should NOT throw error %s", err.Error())
	}

	pid = 0
	s = ""
	count, err = ForEachSystemProcess(func(cmd string, p int) bool {
		if strings.HasSuffix(cmd, PROC_NAME) {
			s = cmd
			pid = p
			return true
		}
		return false
	})
	if err != nil {
		t.Fatal("ForEachSystemProcess Should NOT throw error")
	}
	if count != 0 {
		t.Fatalf("ForEachSystemProcess Should find 0 process. Not : %d", count)
	}
	if pid != 0 {
		t.Fatalf("ForEachSystemProcess should return a ZERO pid. Not %d", pid)
	}
	if s != "" {
		t.Fatalf("ForEachSystemProcess should return an empty string. Not %s", s)
	}

	err = KillrocessWithId(89476486749)
	if err == nil {
		t.Fatalf("Kill Should throw an error:")
	}
}

func TestNoCommands(t *testing.T) {
	tc := NewExecData([]string{}, "", "../testdata/logs/cmdout.txt", "../testdata/logs/cmderr.txt", "info", false, true, nil, nil)
	_, _, _, err := tc.RunSystemProcess()
	if err == nil {
		t.Fatal("Should throw no commands error")
	}
	if !strings.Contains(err.Error(), "no commands") {
		t.Fatal("Error contain 'no commands'")
	}
}

func TestEmptyCommands(t *testing.T) {
	tc := NewExecData([]string{"", " "}, "", "../testdata/logs/cmdout.txt", "../testdata/logs/cmderr.txt", "info", false, false, nil, nil)
	_, _, _, err := tc.RunSystemProcess()
	if err == nil {
		t.Fatal("Should throw no commands error")
	}
	if !strings.Contains(err.Error(), "no commands") {
		t.Fatal("Error contain 'no commands'")
	}
}

func TestLs(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta"}, "", "../testdata/logs/cmdout-ls.txt", "../testdata/logs/cmderr-ls.txt", "info", false, false, nil, nil)
	stdOut, _, _, err := tc.RunSystemProcess()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}
	if !strings.Contains(string(stdOut), "go.mod") {
		t.Fatal(string(stdOut))
	}
}

func TestLsErr(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta", "x"}, "", "../testdata/logs/cmdout-ls-x.txt", "../testdata/logs/cmderr-ls-x.txt", "info", false, false, nil, nil)
	stdOut, stdErr, _, err := tc.RunSystemProcess()
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

	tc := NewExecData([]string{"pwd"}, "", "../testdata/logs/cmdout-pwd.txt", "../testdata/logs/cmderr-pwd.txt", "info", false, false, nil, nil)
	stdOut, _, _, err := tc.RunSystemProcess()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}

	if !strings.Contains(string(stdOut), "/goWebApp") {
		t.Fatal(string(stdOut))
	}

}
func TestGo(t *testing.T) {

	tc := NewExecData([]string{"go", "version"}, "", "../testdata/logs/cmdout-go.txt", "../testdata/logs/cmderr-go.txt", "info", false, false, nil, nil)
	stdOut, _, _, err := tc.RunSystemProcess()
	if err != nil {
		t.Fatalf("Error:%e", err)
	}

	if !strings.Contains(string(stdOut), "go version") {
		t.Fatal(string(stdOut))
	}
}
