package runCommand

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const PROC_NAME = "longRunTest1.sh"

func TestShellDetatchTwice(t *testing.T) {
	pidx := FindProcessIdWithName(PROC_NAME)
	if pidx != 0 {
		wrapKillrocessWithPid(t, "TestDetatch 0", pidx, []string{})
	}
	tc := NewExecData([]string{PROC_NAME}, "../testdata/exec", "", "", "lr1", "", true, true, nil, nil)
	wrapRunSystemProcess(t, "TestDetatchTwice 1", []string{}, tc)
	pid := FindProcessIdWithName(PROC_NAME)
	if pid == 0 {
		t.Fatalf("Process did not start")
	}
	time.Sleep(time.Millisecond * 1030)
	wrapRunSystemProcess(t, "TestDetatchTwice 2", []string{"Process 'lr1' already running. PID:", "Status:400"}, tc)
	wrapKillrocessWithPid(t, "TestDetatch 2", pid, []string{})

	os.Remove("../testdata/exec/LongRunTest1.txt")
	os.Remove("../testdata/exec/LongRunTest1Error.txt")

}

func TestShellDetatch(t *testing.T) {
	wd, _ := os.Getwd()

	pidx := FindProcessIdWithName(PROC_NAME)
	if pidx != 0 {
		wrapKillrocessWithPid(t, "TestDetatch 0", pidx, []string{})
	}

	tc := NewExecData([]string{PROC_NAME}, "../testdata/exec", "", "", "lr1", "", true, true, nil, nil)
	wrapRunSystemProcess(t, "TestDetatch 1", []string{}, tc)

	cd, _ := os.Getwd()
	if wd != cd {
		t.Fatalf("Working dir has changed. From %s To %s", wd, cd)
	}

	pid := FindProcessIdWithName(PROC_NAME)
	if pid == 0 {
		t.Fatalf("Process did not start")
	}

	var filSize int64 = 0
	for range 2 {
		time.Sleep(time.Millisecond * 1030)
		st, err := os.Stat("../testdata/exec/LongRunTest1.txt")
		if err != nil {
			t.Fatal(err.Error())
		}
		if st == nil {
			t.Fatal("os.Stat(\"../testdata/exec/LongRunTest1.txt\") returned nil")
		}
		if st.Size() > filSize {
			filSize = st.Size()
		} else {
			t.Fatal("../testdata/exec/LongRunTest1.txt is not increasing in size")
		}
	}

	wrapKillrocessWithPid(t, "TestDetatch 2", pid, []string{})

	pid = FindProcessIdWithName(PROC_NAME)
	if pid != 0 {
		t.Fatalf("Process was not stopped")
	}

	os.Remove("../testdata/exec/LongRunTest1.txt")
	os.Remove("../testdata/exec/LongRunTest1Error.txt")

	//"Exec Error. Status:424 ID:'kill'. Process could not be stopped. Kill 89476486749 failed with error:exit status 1"
	wrapKillrocessWithPid(t, "TestDetatch 3", 89476486749, []string{"Status:424", "Process could not be stopped", "Kill process 89476486749", "exit status 1"})
}

func TestNoCommands(t *testing.T) {
	tc := NewExecData([]string{}, "", "../testdata/logs/cmdout.txt", "../testdata/logs/cmderr.txt", "info", "", false, true, nil, nil)
	wrapRunSystemProcess(t, "TestNoCommands", []string{"No command were given", "Config error"}, tc)
}

func TestEmptyCommands(t *testing.T) {
	tc := NewExecData([]string{"", " "}, "", "../testdata/logs/cmdout.txt", "../testdata/logs/cmderr.txt", "info", "", false, false, nil, nil)
	wrapRunSystemProcess(t, "TestEmptyCommands", []string{"No command were given", "Config error"}, tc)
}

func TestZeroResp(t *testing.T) {
	tc := NewExecData([]string{"free"}, "", "../testdata/logs/cmdout-free.txt", "../testdata/logs/cmderr-free.txt", "info", "", false, false, nil, nil)
	stdOut, stdErr, rc := wrapRunSystemProcess(t, "TestZeroResp", []string{}, tc)
	AssertContains(t, "TestZeroResp", string(stdOut), []string{"Mem:", "Swap:"})
	if rc != 0 {
		t.Fatal("RC should be 0 not", rc)
	}
	if len(stdErr) > 0 {
		t.Fatal("stdErr should be empty")
	}
}

func TestLs(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta"}, "", "../testdata/logs/cmdout-ls.txt", "../testdata/logs/cmderr-ls.txt", "info", "", false, false, nil, nil)
	stdOut, stdErr, rc := wrapRunSystemProcess(t, "TestLs", []string{}, tc)
	AssertContains(t, "TestLs", string(stdOut), []string{"go.mod", "runCommand.go"})
	if rc != 0 {
		t.Fatal("RC should be 0 not", rc)
	}
	if len(stdErr) > 0 {
		t.Fatal("stdErr should be empty")
	}
}

func TestLsErr(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta", "x"}, "", "../testdata/logs/cmdout-ls-x.txt", "../testdata/logs/cmderr-ls-x.txt", "info", "", false, false, nil, nil)
	stdOut, stdErr, rc := wrapRunSystemProcess(t, "TestLsErr", []string{}, tc)
	AssertContains(t, "TestLsErr", string(stdErr), []string{"'x': No such"})
	if rc != 2 {
		t.Fatal("RC should be 2 not", rc)
	}
	if len(stdOut) > 0 {
		t.Fatal("StdOut should be empty")
	}
}

func TestPWD(t *testing.T) {
	tc := NewExecData([]string{"pwd"}, "", "../testdata/logs/cmdout-pwd.txt", "../testdata/logs/cmderr-pwd.txt", "info", "", false, false, nil, nil)
	stdOut, stdErr, rc := wrapRunSystemProcess(t, "TestPWD", []string{}, tc)
	AssertContains(t, "TestPWD", string(stdOut), []string{"/goWebApp"})
	if rc != 0 {
		t.Fatal("RC should be 0 not", rc)
	}
	if len(stdErr) > 0 {
		t.Fatal("stdErr should be empty")
	}
}

func TestGo(t *testing.T) {
	tc := NewExecData([]string{"go", "version"}, "", "../testdata/logs/cmdout-go.txt", "../testdata/logs/cmderr-go.txt", "go", "", false, false, nil, nil)
	stdOut, stdErr, rc := wrapRunSystemProcess(t, "TestGo", []string{}, tc)
	AssertContains(t, "TestGo", string(stdOut), []string{"go version"})
	if rc != 0 {
		t.Fatal("RC should be 0")
	}
	if len(stdErr) > 0 {
		t.Fatal("stdErr should be empty")
	}
}

func TestShellNonZeroResp(t *testing.T) {
	tc := NewExecData([]string{"ls_nonzero"}, "../testdata/exec", "cmdout-ls_nonzero.txt", "cmderr-ls_nonzero.txt", "ls_nonzero", "", false, false, nil, nil)
	stdOut, _, rc := wrapRunSystemProcess(t, "TestZeroResp", []string{}, tc)
	AssertContains(t, "TestZeroResp", string(stdOut), []string{})
	if rc != 2 {
		t.Fatal("RC should be 2 not", rc)
	}
	if len(stdOut) > 0 {
		t.Fatal("stdErr should be empty")
	}
	cleanFiles(tc)
}

//             total        used       free      shared  buff/cache   available
// Mem:        32274420     3944936    23557504     1140792     6380212    28329484
// Swap:              0           0           0
//

func wrapRunSystemProcess(t *testing.T, name string, panic []string, tc *execData) ([]byte, []byte, int) {
	defer func() {
		if r := recover(); r != nil {
			if len(panic) == 0 {
				t.Fatal(name, fmt.Sprintf(": Should Not Panic. Error:%v", r))
			}
			switch x := r.(type) {
			case *ExecError:
				AssertContains(t, name, x.LogError(), panic)
			default:
				t.Fatal(name, ": Panic Message type should be ExecError")
			}
		}
	}()
	cleanFiles(tc)
	o, r, c := tc.RunSystemProcess()
	if len(panic) != 0 {
		t.Fatal(name, ": Should have panicked:")
	}
	checkFiles(t, tc, o, r)
	return o, r, c
}

func checkFiles(t *testing.T, tc *execData, o []byte, e []byte) {
	name := tc.StdOutLog
	if name != "" {
		if tc.Dir != "" {
			name = filepath.Join(tc.Dir, name)
		}
		checkFile(t, name, o)
	}
	name = tc.StdErrLog
	if name != "" {
		if tc.Dir != "" {
			name = filepath.Join(tc.Dir, name)
		}
		checkFile(t, name, e)
	}
	cleanFiles(tc)
}

func cleanFiles(tc *execData) {
	name := tc.StdOutLog
	if name != "" {
		if tc.Dir != "" {
			name = filepath.Join(tc.Dir, name)
		}
		os.Remove(name)
	}
	name = tc.StdErrLog
	if name != "" {
		if tc.Dir != "" {
			name = filepath.Join(tc.Dir, name)
		}
		os.Remove(name)
	}
}

func checkFile(t *testing.T, name string, b []byte) {
	if len(b) > 0 && name != "" {
		_, err := os.Stat(name)
		if err != nil {
			t.Fatal(name, "Should have been created")
		}
		var c []byte
		c, err = os.ReadFile(name)
		if err != nil {
			t.Fatal(name, "Could not be read")
		}
		if len(c) != len(b) {
			t.Fatal(name, "Should have been", len(b), "bytes. Found", len(c))
		}
	}
}

func wrapKillrocessWithPid(t *testing.T, name string, id int, panic []string) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case *ExecError:
				AssertContains(t, name, x.LogError(), panic)
			default:
				t.Fatal(name, ": Panic Message type should be ExecError")
			}
		}
	}()
	KillrocessWithPid(id)
	if len(panic) != 0 {
		t.Fatal(name, ": Should have panicked:")
	}
}

func AssertContains(t *testing.T, name, actual string, expectedList []string) {
	for _, expected := range expectedList {
		if !strings.Contains(actual, expected) {
			t.Fatalf("%s: Value \n%s\nDoes NOT contain '%s'", name, actual, expected)
		}
	}
}
