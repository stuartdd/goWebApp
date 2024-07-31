package logging

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testLogDir = "../testdata/logs"
const dtStringLen = len("2024/03/05 15:34:51 ")

func TestLogFileData(t *testing.T) {
	cleanLogs(t)
	lfd1 := newLogFileData("p", "m", time.Date(2021, 4, 15, 14, 30, 45, 100, time.Local))
	if lfd1.err == nil {
		t.Fatalf("Error: p should not exist")
	}
	AssertEquals(t, "Fn2", lfd1.err.Error(), "log directory 'p' does not exist")
	lfd1 = newLogFileData("../testdata/t1.JSON", "m", time.Date(2021, 4, 15, 14, 30, 45, 100, time.Local))
	if lfd1.err == nil {
		t.Fatalf("Error:../testdata/t1.JSON is a file")
	}
	AssertEquals(t, "Fn2", lfd1.err.Error(), "log directory '../testdata/t1.JSON' is not a directory")

	lfd1 = newLogFileData(testLogDir, "m.log", time.Date(2021, 4, 15, 14, 30, 45, 100, time.Local))
	if lfd1.err != nil {
		t.Fatalf("Error: %s", lfd1.err.Error())
	}
	AssertEquals(t, "Fn2", lfd1.fileName, "m.log")
	_, err := os.Stat(filepath.Join(testLogDir, "m.log"))
	if err != nil {
		t.Fatalf("Log file m.log not created: %s", err.Error())
	}
}

func TestLoggingFunctions(t *testing.T) {
	dp := newDatePrefix(time.Date(2021, 4, 15, 14, 30, 45, 100, time.Local))
	AssertEquals(t, "04", dp, "2021/04/15 ")
	dp = newDatePrefix(time.Date(2021, 10, 24, 14, 30, 45, 100, time.Local))
	AssertEquals(t, "24", dp, "2021/10/24 ")
	dp = newDatePrefix(time.Date(2021, 10, 2, 14, 30, 45, 100, time.Local))
	AssertEquals(t, "05", dp, "2021/10/02 ")

	fn := deriveFileName("", time.Date(2021, 10, 2, 14, 30, 45, 100, time.Local))
	AssertEquals(t, "Fn1", fn, "")
	fn = deriveFileName("goWebServer-%y-%m-%d-%H-%M-%S.log", time.Date(2021, 10, 2, 14, 30, 45, 100, time.Local))
	AssertEquals(t, "Fn2", fn, "goWebServer-2021-10-02-14-30-45.log")

	AssertEquals(t, "FLi0", fixedLenInt(0, 4), "0000")
	AssertEquals(t, "FLi1", fixedLenInt(1, 2), "01")
	AssertEquals(t, "FLi0", fixedLenInt(0, 2), "00")
	AssertEquals(t, "FLi999", fixedLenInt(999, 0), "999")
	AssertEquals(t, "FLi999", fixedLenInt(999, 1), "999")
	AssertEquals(t, "FLi999", fixedLenInt(999, 2), "999")
	AssertEquals(t, "FLi999", fixedLenInt(999, 3), "999")
	AssertEquals(t, "FLi0999", fixedLenInt(999, 4), "0999")
}

func TestLoggingSink(t *testing.T) {
	l, err := NewLogger("", "goWebServer-test.log", 10, false, false)
	if err != nil {
		t.Fatalf("Error: Empty path should not produce an error: %s", err.Error())
	}
	l.Log("Should log to console")
	if l.LogFileName() != "goWebServer-test.log" {
		t.Fatalf("Error: Empty path should not effect file name")
	}
	if l.IsOpen() {
		t.Fatalf("Error: Empty path should not Open log")
	}
	_, err = NewLogger(testLogDir, "", 10, false, false)
	if err != nil {
		t.Fatalf("Error: Empty path file name should produce an error:")
	}
}
func TestLogging(t *testing.T) {
	cleanLogs(t)

	l, err := NewLogger(testLogDir, "goWebServer-test.log", 10, false, false)
	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}
	if !l.IsOpen() {
		t.Fatalf("Error: log is is not open")
	}
	l.Log("1")
	l.Log("2")
	l.Close()
	l, err = NewLogger(testLogDir, "goWebServer-test.log", 10, false, false)
	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}
	if !l.IsOpen() {
		t.Fatalf("Error: log is is not open")
	}
	l.Log("3")
	l.Log("4")
	l.Close()
	s := readLog(t)

	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Scan()
	s1 := scanner.Text()
	AssertEquals(t, "M1", string(s1[dtStringLen:]), "1")
	scanner.Scan()
	s2 := scanner.Text()
	AssertEquals(t, "M2", string(s2[dtStringLen:]), "2")
	scanner.Scan()
	s3 := scanner.Text()
	AssertEquals(t, "M3", string(s3[dtStringLen:]), "3")
	scanner.Scan()
	s4 := scanner.Text()
	AssertEquals(t, "M4", string(s4[dtStringLen:]), "4")

}

func readLog(t *testing.T) string {
	entries, err := os.ReadDir(testLogDir)
	if err != nil {
		t.Fatalf("Dir: %s", err.Error())
	}
	if len(entries) != 1 {
		t.Fatalf("Read:Failed. Multiple log files exist: %s", testLogDir)
	}
	log, err := os.ReadFile(filepath.Join(testLogDir, entries[0].Name()))
	if len(entries) != 1 {
		t.Fatalf("Read:Failed: %s", err.Error())
	}
	return string(log)
}

func cleanLogs(t *testing.T) {
	entries, err := os.ReadDir(testLogDir)
	if err != nil {
		t.Fatalf("Dir: %s", err.Error())
	}
	if len(entries) == 0 {
		return
	}
	for i := 0; i < len(entries); i++ {
		os.Remove(filepath.Join(testLogDir, entries[i].Name()))
	}
	entries, err = os.ReadDir(testLogDir)
	if err != nil {
		t.Fatalf("Dir: %s", err.Error())
	}
	if len(entries) != 0 {
		t.Fatalf("Dir:Failed to clean test log dir: %s", testLogDir)
	}
}

func AssertNil(t *testing.T, message string, err error) {
	if err != nil {
		t.Fatalf("%s.\nExpected:Nil\nActual:  %s", message, err)
	}
}

func AssertErr(t *testing.T, message string, err error, expected string) {
	if err.Error() != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s", message, expected, err)
	}
}

func AssertEquals(t *testing.T, message, actual, expected string) {
	if actual != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s", message, expected, actual)
	}
}

func AssertTrue(t *testing.T, message string, actual bool) {
	if actual == false {
		t.Fatalf("%s.\nExpected:true\nActual:  %t", message, actual)
	}
}
func AssertFalse(t *testing.T, message string, actual bool) {
	if actual == true {
		t.Fatalf("%s.\nExpected:false\nActual:  %t", message, actual)
	}
}
