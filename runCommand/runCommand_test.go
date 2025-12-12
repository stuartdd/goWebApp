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
const TEST_PATH_PREF = "testdata"
const TEST_EXEC_PATH = "exec"
const TEST_LOGS_PATH = "logs"

func TestShellDetatchTwice(t *testing.T) {
	pidx := FindProcessIdWithName(PROC_NAME)
	if pidx != 0 {
		wrapKillrocessWithPid(t, "TestDetatch 0", pidx, []string{})
	}
	tc := NewExecData([]string{PROC_NAME}, "", "", "lr1", "", true, true, nil, nil)
	wrapRunSystemProcess(t, "TestDetatchTwice 1", []string{}, tc)
	pid := FindProcessIdWithName(PROC_NAME)
	if pid == 0 {
		t.Fatalf("Process did not start")
	}
	time.Sleep(time.Millisecond * 1030)
	wrapRunSystemProcess(t, "TestDetatchTwice 2", []string{"Process 'lr1' already running. PID:", "Status:400"}, tc)
	wrapKillrocessWithPid(t, "TestDetatch 2", pid, []string{})

	os.Remove(filepath.Join(getTestExecPath(), "LongRunTest1.txt"))
	os.Remove(filepath.Join(getTestExecPath(), "LongRunTest1Error.txt"))
}

func TestShellDetatch(t *testing.T) {
	wd, _ := os.Getwd()

	// If job already running we need to kill it!
	pidx := FindProcessIdWithName(PROC_NAME)
	if pidx != 0 {
		wrapKillrocessWithPid(t, "TestDetatch 0", pidx, []string{})
	}

	tc := NewExecData([]string{PROC_NAME}, "", "", "lr1", "", true, true, nil, nil)
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
		st, err := os.Stat(filepath.Join(getTestExecPath(), "LongRunTest1.txt"))
		if err != nil {
			t.Fatal(err.Error())
		}
		if st == nil {
			t.Fatal("os.Stat(", filepath.Join(getTestExecPath(), "/LongRunTest1.txt"), ") returned nil")
		}
		if st.Size() > filSize {
			filSize = st.Size()
		} else {
			t.Fatal(filepath.Join(getTestExecPath(), "/LongRunTest1.txt"), " is not increasing in size")
		}
	}

	wrapKillrocessWithPid(t, "TestDetatch 2", pid, []string{})

	pid = FindProcessIdWithName(PROC_NAME)
	if pid != 0 {
		t.Fatalf("Process was not stopped")
	}
	os.Remove(filepath.Join(getTestExecPath(), "LongRunTest1.txt"))
	os.Remove(filepath.Join(getTestExecPath(), "LongRunTest1Error.txt"))

	//"Exec Error. Status:424 ID:'kill'. Process could not be stopped. Kill 89476486749 failed with error:exit status 1"
	wrapKillrocessWithPid(t, "TestDetatch 3", 89476486749, []string{"Status:424", "Process could not be stopped", "Kill process 89476486749", "exit status 1"})
}

func TestNoCommands(t *testing.T) {
	tc := NewExecData([]string{}, getTestLogsPath("cmdout.txt"), getTestLogsPath("cmderr.txt"), "info", "", false, true, nil, nil)
	wrapRunSystemProcess(t, "TestNoCommands", []string{"No command given", "Config error"}, tc)
}

func TestEmptyCommands(t *testing.T) {
	tc := NewExecData([]string{"", " "}, getTestLogsPath("cmdout.txt"), getTestLogsPath("cmderr.txt"), "info", "", false, false, nil, nil)
	wrapRunSystemProcess(t, "TestEmptyCommands", []string{"No command given", "Config error"}, tc)
}

func TestZeroResp(t *testing.T) {
	tc := NewExecData([]string{"free.sh"}, getTestLogsPath("cmdout-free.txt"), getTestLogsPath("cmderr-free.txt"), "info", "", false, false, nil, nil)
	stdOut, stdErr, rc := wrapRunSystemProcess(t, "TestZeroResp", []string{}, tc)
	AssertContains(t, "TestZeroResp", string(stdOut), []string{"\"MemUsed\":", "\"SwapTotal\":"})
	if rc != 0 {
		t.Fatal("RC should be 0 not", rc)
	}
	if len(stdErr) > 0 {
		t.Fatal("stdErr should be empty")
	}
}

func TestLs(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta"}, getTestLogsPath("cmdout-ls.txt"), getTestLogsPath("cmderr-ls.txt"), "info", "", false, false, nil, nil)
	stdOut, stdErr, rc := wrapRunSystemProcess(t, "TestLs", []string{}, tc)
	AssertContains(t, "TestLs", string(stdOut), []string{"webtools", "ls"})
	if rc != 0 {
		t.Fatal("RC should be 0 not", rc)
	}
	if len(stdErr) > 0 {
		t.Fatal("stdErr should be empty")
	}
}

func TestLsErr(t *testing.T) {
	tc := NewExecData([]string{"ls", "-lta", "x"}, getTestLogsPath("cmdout-ls-x.txt"), getTestLogsPath("cmderr-ls-x.txt"), "info", "", false, false, nil, nil)
	stdOut, stdErr, rc := wrapRunSystemProcess(t, "TestLsErr", []string{}, tc)
	AssertContains(t, "TestLsErr", string(stdErr), []string{"'x': No such"})
	if rc != 2 {
		t.Fatal("RC should be 2 not", rc)
	}
	if len(stdOut) > 0 {
		t.Fatal("StdOut should be empty")
	}
}
func TestLsRC_NZ(t *testing.T) {
	tc := NewExecData([]string{"ls_nonzero"}, getTestLogsPath("cmdout-ls-nz.txt"), getTestLogsPath("cmderr-ls-nz.txt"), "info", "", false, false, nil, nil)
	stdOut, stdErr, rc := wrapRunSystemProcess(t, "TestLsErr", []string{}, tc)
	AssertContains(t, "TestLsErr", string(stdErr), []string{"'fred': No such file"})
	if rc != 2 {
		t.Fatal("RC should be 2 not", rc)
	}
	if len(stdOut) > 0 {
		t.Fatal("StdOut should be empty")
	}

}
func TestPWD(t *testing.T) {
	tc := NewExecData([]string{"pwd-test"}, getTestLogsPath("cmdout-pwd.txt"), getTestLogsPath("cmderr-pwd.txt"), "info", "", false, false, nil, nil)
	stdOut, stdErr, rc := wrapRunSystemProcess(t, "TestPWD", []string{}, tc)
	AssertContains(t, "TestPWD", string(stdOut), []string{getTestExecPath()})
	if rc != 0 {
		t.Fatal("RC should be 0 not", rc)
	}
	if len(stdErr) > 0 {
		t.Fatal("stdErr should be empty")
	}
}

func TestGo(t *testing.T) {
	tc := NewExecData([]string{"go-version", "version"}, getTestLogsPath("cmdout-go.txt"), getTestLogsPath("cmderr-go.txt"), "go", "", false, false, nil, nil)
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
	tc := NewExecData([]string{"ls_nonzero"}, getTestLogsPath("cmdout-ls_nonzero.txt"), getTestLogsPath("cmderr-ls_nonzero.txt"), "ls_nonzero", "", false, false, nil, nil)
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

	o, r, c := tc.RunSystemProcess(getTestExecPath())
	if len(panic) != 0 {
		t.Fatal(name, ": Should have panicked:")
	}
	checkFiles(t, tc, o, r)
	return o, r, c
}

func getTestExecPath() string {
	cp, _ := os.Getwd()
	cp = filepath.Dir(cp)
	xx, _, found := strings.Cut(TEST_PATH_PREF, cp)
	if found {
		return filepath.Join(xx, TEST_PATH_PREF, TEST_EXEC_PATH)
	}
	return filepath.Join(cp, TEST_PATH_PREF, TEST_EXEC_PATH)
}

func getTestLogsPath(file string) string {
	cp, _ := os.Getwd()
	cp = filepath.Dir(cp)
	xx, _, found := strings.Cut(TEST_PATH_PREF, cp)
	if found {
		return filepath.Join(xx, TEST_PATH_PREF, TEST_LOGS_PATH, file)
	}
	return filepath.Join(cp, TEST_PATH_PREF, TEST_LOGS_PATH, file)
}

func checkFiles(t *testing.T, tc *execData, o []byte, e []byte) {
	name := tc.StdOutLog
	if name != "" {
		checkFile(t, name, o)
	}
	name = tc.StdErrLog
	if name != "" {
		checkFile(t, name, e)
	}
	cleanFiles(tc)
}

func cleanFiles(tc *execData) {
	name := tc.StdOutLog
	if name != "" {
		os.Remove(filepath.Join(getTestExecPath(), name))
	}
	name = tc.StdErrLog
	if name != "" {
		os.Remove(filepath.Join(getTestExecPath(), name))
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
