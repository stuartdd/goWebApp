package config

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

func TestPanicMessage(t *testing.T) {
	pm := NewConfigErrorFromString("Status:404: Running process with ID:12345 could not be found", 500)
	AssertEquals(t, "String 8", pm.LogError(), "Config Error: Status:404. Running process with ID:12345 could not be found")
	AssertEquals(t, "String 8.1", pm.log, "")

	pm = NewConfigErrorFromString("running process with ID:12345 could not be found Status:404", 500)
	AssertEquals(t, "String 9", pm.LogError(), "Config Error: Status:404. running process with ID:12345 could not be found")
	AssertEquals(t, "String 9.1", pm.log, "")

	pm = NewConfigErrorFromString("ABC log:LM Status:404", 500)
	AssertEquals(t, "Recover 2", pm.Error(), "Config Error: Status:404. ABC")
	AssertEquals(t, "Recover 2.1", pm.log, "LM Status:404")

	pm = NewConfigErrorFromString("ABC log:LM", 500)
	AssertEquals(t, "Recover 3", pm.Error(), "Config Error: Status:500. ABC")
	AssertEquals(t, "Recover 3.1", pm.log, "LM")

	pm = NewConfigErrorFromString("ABC: log:LM Status:404", 500)
	AssertEquals(t, "Recover 4", pm.Error(), "Config Error: Status:404. ABC:")
	AssertEquals(t, "Recover 4.1", pm.log, "LM Status:404")

	pm = NewConfigErrorFromString("ABC: Status:32768", 500)
	AssertEquals(t, "Recover 5", pm.LogError(), fmt.Sprintf("Config Error: Status:%d. ABC:", math.MaxInt16))
	AssertEquals(t, "Recover 5.1", pm.log, "")

	pm = NewConfigErrorFromString("ABC: Status:4.9 4", 500)
	AssertEquals(t, "Recover 6", pm.LogError(), "Config Error: Status:49. ABC: 4")
	AssertEquals(t, "Recover 6.1", pm.log, "")

	pm = NewConfigErrorFromString("ABC: Status:4.0.4", 500)
	AssertEquals(t, "Recover 7", pm.LogError(), "Config Error: Status:404. ABC:")
	AssertEquals(t, "Recover 7.1", pm.log, "")

	pm = NewConfigErrorFromString("ABC: Status:404 log:LM", 500)
	AssertEquals(t, "Recover 8", pm.LogError(), "Config Error: Status:404. ABC: Log:LM")
	AssertEquals(t, "Recover 8.1", pm.log, "LM")

	pm = NewConfigErrorFromString("ABC", 500)
	AssertEquals(t, "Recover 9", pm.Error(), "Config Error: Status:500. ABC")
	AssertEquals(t, "Recover 9.1", pm.log, "")

	pm = NewConfigError("R:X", 400, "LM")
	AssertEquals(t, "Simple 1", pm.Error(), "Config Error: Status:400. R:X")
	AssertEquals(t, "Simple 1.1", pm.LogError(), "Config Error: Status:400. R:X Log:LM")

	pm = NewConfigError("R:X Status", 400, "LM")
	AssertEquals(t, "Simple 2", pm.Error(), "Config Error: Status:400. R:X Status")

	pm = NewConfigError("R:X Status", 400, "L Status:500")
	AssertEquals(t, "Simple 3", pm.Error(), "Config Error: Status:400. R:X Status")

	pm = NewConfigError("R:X", 400, "LM")
	AssertEquals(t, "Simple 4", pm.LogError(), "Config Error: Status:400. R:X Log:LM")
	AssertEquals(t, "Simple 4.1", pm.log, "LM")

	pm = NewConfigError("R:X Status", 400, "LM")
	AssertEquals(t, "Simple 5", pm.Error(), "Config Error: Status:400. R:X Status")

	pm = NewConfigError("R:X Status", 400, "L Status:500")
	AssertEquals(t, "Simple 6", pm.Error(), "Config Error: Status:400. R:X Status")

}

func TestThumbNailTrim(t *testing.T) {
	conf := LoadConfigData(t, "../goWebAppTest.json", nil)
	AssertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail("", true), "")
	AssertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail("", false), "")
	AssertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail("fred", true), "fred")
	AssertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail("fred", false), "fred")
	AssertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail("2024_09_21_12_22_11_HuwSig.jpg.jpg", true), "HuwSig.jpg")
	AssertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail("2024_09_21_12_22_11_HuwSig.jpg.jpg", false), "2024_09_21_12_22_11_HuwSig.jpg.jpg")
}

func TestJoinPath(t *testing.T) {
	conf := LoadConfigData(t, "../goWebAppTest.json", nil)
	var f string
	pre := conf.GetServerDataRoot()

	f = conf.resolvePaths("", conf.GetServerDataRoot(), "/dir")
	AssertEquals(t, "file 3", f, filepath.Join(pre, "dir"))

	f = conf.resolvePaths("", conf.GetServerDataRoot(), "***/dir")
	AssertEquals(t, "file 4", f, "/dir")

	f = conf.resolvePaths("stuart", conf.GetServerDataRoot(), "***/dir")
	AssertEquals(t, "file 5", f, "/dir")

	f = conf.resolvePaths("bob", conf.GetServerDataRoot(), "***dir")
	AssertEquals(t, "file 6", f, "dir")

	f = conf.resolvePaths("stuart", conf.GetServerDataRoot(), "dir")
	AssertEquals(t, "file 7", f, filepath.Join(pre, "stuart/dir"))

	f = conf.resolvePaths("john", conf.GetServerDataRoot(), "")
	AssertEquals(t, "file 8", f, filepath.Join(pre, "john"))

	f = conf.resolvePaths("", conf.GetServerDataRoot(), "dir")
	AssertEquals(t, "file 9", f, filepath.Join(pre, "dir"))

	f = conf.resolvePaths("", conf.GetServerDataRoot(), "")
	AssertEquals(t, "file 9", f, pre)

}

func TestSubstitute(t *testing.T) {
	m1 := map[string]string{"A": "X", "b": "Y"}
	m2 := map[string]string{"UA": "UX", "Ub": "UY"}
	m3 := map[string]string{"UA": "UX", "Ub": "UX", "A": "UA"}

	AssertSub(t, "Ab8", "-%{UA}-%{A}-%{b}-%{Ub}-%{A}-", "-UX-UA-Y-UX-UA-", m1, m3)
	AssertSub(t, "Ab9", "-%%{%%{A}%{b}}-", "-%%{%XY}-", m1, m2)

	AssertSub(t, "A2", "-%%{%%{A}}-", "-%%{%X}-", m1, m2)
	AssertSub(t, "A3", "-%{%%{A}}-", "-%{%X}-", m1, m2)
	AssertSub(t, "A4", "-%{%{A}}-", "-%{X}-", m1, m2)
	AssertSub(t, "A5", "-{%%{A}}-", "-{%X}-", m1, m2)
	AssertSub(t, "A6", "-{%{A}}-", "-{X}-", m1, m2)
	AssertSub(t, "A7", "-%{A}}-", "-X}-", m1, m2)
	AssertSub(t, "A8", "-%{A}-", "-X-", m1, m2)
	AssertSub(t, "A9", "-%%{A}-", "-%X-", m1, m2)

	AssertSub(t, "A5", "-%{A-", "-%{A-", m1, m2)
	AssertSub(t, "A6", "-%%{A-", "-%%{A-", m1, m2)
	AssertSub(t, "A8", "-%%A-", "-%%A-", m1, m2)
	AssertSub(t, "A9", "-%A-", "-%A-", m1, m2)

	AssertSub(t, "Z4", "-%{Z}-", "-%{Z}-", m1, m2)
	AssertSub(t, "Z5", "-%{Z-", "-%{Z-", m1, m2)
	AssertSub(t, "Z6", "-%%{Z-", "-%%{Z-", m1, m2)
	AssertSub(t, "Z7", "-%%{Z}-", "-%%{Z}-", m1, m2)
	AssertSub(t, "Z8", "-%%Z-", "-%%Z-", m1, m2)
	AssertSub(t, "Z9", "-%Z-", "-%Z-", m1, m2)

	AssertEquals(t, "empty", string(SubstituteFromMap([]byte(""), m1, m2)), "")
	AssertEquals(t, "1 ch", string(SubstituteFromMap([]byte("%"), m1, m2)), "%")
	AssertEquals(t, "2 ch", string(SubstituteFromMap([]byte("%{"), m1, m2)), "%{")
	AssertEquals(t, "3 ch", string(SubstituteFromMap([]byte("%{}"), m1, m2)), "%{}")
	AssertEquals(t, "4 chA", string(SubstituteFromMap([]byte("%{A}"), m1, m2)), "X")
	AssertEquals(t, "4 chX", string(SubstituteFromMap([]byte("%{Z}"), m1, m2)), "%{Z}")
}

func AssertSub(t *testing.T, id, sub, expected string, m1 map[string]string, m2 map[string]string) {
	r := string(SubstituteFromMap([]byte(sub), m1, m2))
	if r != expected {
		t.Fatalf("Substitution: %s, \nExpected [%s]\nActual   [%s]", id, expected, r)
	}
}

func AssertEquals(t *testing.T, message string, actual string, expected string) {
	if actual != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s", message, expected, string(actual))
	}
}

func AssertContains(t *testing.T, message string, actual string, contains []string) {
	for _, c := range contains {
		if !strings.Contains(actual, c) {
			t.Fatalf("%s.\nDoes Not Contain:%s\nActual:  %s", message, c, string(actual))
		}
	}
}

func AssertErrors(t *testing.T, message string, actual *ConfigErrorData, contains []string, count int) {
	if actual.ErrorCount() != count {
		t.Fatalf("%s.\nExpected:%d errors \nActual:  %d errors", message, count, actual.ErrorCount())
	}
	AssertContains(t, message, actual.String(), contains)
}

func TestUserExecBadExecId(t *testing.T) {
	conf := LoadConfigData(t, "../goWebAppTest.json", nil)

	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*ConfigError)
			if !ok || pm == nil {
				t.Fatalf("TestUserExecBadExecId: Should have returned a ConfigError")
			}
			AssertEquals(t, "TestUserExecBadExecId", pm.LogError(), "Config Error: Status:404. Exec ID not found Log:exec-id=notid")
			AssertEquals(t, "TestUserExecBadExecId", pm.String(), "Exec ID not found")
		}
	}()

	conf.GetExecInfo("notid")
	t.Fatalf("Should have panicked")
}

func TestUserExec(t *testing.T) {
	conf := LoadConfigData(t, "../goWebAppTest.json", nil)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TestUserExec Should not panic")
		}
	}()
	pre := conf.GetServerDataRoot()
	exec := conf.GetExecInfo("c2")
	if exec.CanStop {
		t.Fatalf("Exec canstop should default to false")
	}
	es := exec.String()
	AssertContains(t, "TestUserExec ", es, []string{pre, "[cmd2]", "/logs/stdOutC2.txt"})

}
func TestGetUserExecInfo(t *testing.T) {
	conf := LoadConfigData(t, "../goWebAppTest.json", nil)
	c1 := conf.GetExecInfo("ls")

	if c1.Cmd[0] != "ls" {
		t.Fatal("Command should be ls -lta")
	}
	if c1.Cmd[1] != "-lta" {
		t.Fatal("Command should be ls -lta")
	}
}

func TestGetUserLocPathBadUser(t *testing.T) {
	conf := LoadConfigData(t, "../goWebAppTest.json", nil)

	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*ConfigError)
			if !ok || pm == nil {
				t.Fatalf("Should have returned a ConfigError")
			}
			AssertEquals(t, "TestGetUserLocPathBadUser", pm.Error(), "Config Error: Status:404. User not found")
			AssertEquals(t, "TestGetUserLocPathBadUser", pm.LogError(), "Config Error: Status:404. User not found Log:User=fred")
		}
	}()

	conf.GetUserLocPath("fred", "home")
	t.Fatalf("Should have panicked")
}

func TestGetUserLocPathBadLoc(t *testing.T) {
	conf := LoadConfigData(t, "../goWebAppTest.json", nil)

	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*ConfigError)
			if !ok || pm == nil {
				t.Fatalf("Should have returned a PanicMessage")
			}
			AssertEquals(t, "TestGetUserLocPathBadLoc", pm.Error(), "Config Error: Status:404. Location not found")
			AssertEquals(t, "TestGetUserLocPathBadLoc", pm.LogError(), "Config Error: Status:404. Location not found Log:User=stuart Location=nothome")
		}
	}()

	conf.GetUserLocPath("stuart", "nothome")
	t.Fatalf("Should have panicked")
}

func TestGetUserLocPath(t *testing.T) {
	conf := LoadConfigData(t, "../goWebAppTest.json", nil)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Should not panic")
		}
	}()

	u := conf.GetUserLocPath("stuart", "home")

	if !strings.HasSuffix(u, "/testdata/stuart") {
		t.Fatalf("Should return path to /testdata/stuart")
	}

}

func LoadConfigData(t *testing.T, name string, errList *ConfigErrorData) *ConfigData {
	maxErr := 9
	if errList == nil {
		maxErr = 1
		errList = NewConfigErrorData()
	}
	configData := NewConfigData(name, "goWebApp", false, false, false, errList)
	if errList.ErrorCount() > maxErr || configData == nil {
		t.Fatal(errList.String())
	}
	return configData
}
