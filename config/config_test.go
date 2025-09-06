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
	assertEquals(t, "String 8", pm.String(), "Running process with ID;12345 could not be found Status:404")
	assertEquals(t, "String 8.1", pm.log, "")

	pm = NewConfigErrorFromString("running process with ID:12345 could not be found Status:404", 500)
	assertEquals(t, "String 9", pm.String(), "running process with ID;12345 could not be found Status:404")
	assertEquals(t, "String 9.1", pm.log, "")

	pm = NewConfigErrorFromString("ABC log:LM Status:404", 500)
	assertEquals(t, "Recover 2", pm.String(), "ABC Status:404")
	assertEquals(t, "Recover 2.1", pm.log, "LM Status:404")

	pm = NewConfigErrorFromString("ABC log:LM", 500)
	assertEquals(t, "Recover 3", pm.String(), "ABC Status:500 Log:LM")
	assertEquals(t, "Recover 3.1", pm.log, "LM")

	pm = NewConfigErrorFromString("ABC: log:LM Status:404", 500)
	assertEquals(t, "Recover 4", pm.String(), "ABC; Status:404")
	assertEquals(t, "Recover 4.1", pm.log, "LM Status:404")

	pm = NewConfigErrorFromString("ABC: Status:32768", 500)
	assertEquals(t, "Recover 5", pm.String(), fmt.Sprintf("ABC; Status:%d", math.MaxInt16))
	assertEquals(t, "Recover 5.1", pm.log, "")

	pm = NewConfigErrorFromString("ABC: Status:4.9 4", 500)
	assertEquals(t, "Recover 6", pm.String(), "ABC; 4 Status:49")
	assertEquals(t, "Recover 6.1", pm.log, "")

	pm = NewConfigErrorFromString("ABC: Status:4.0.4", 500)
	assertEquals(t, "Recover 7", pm.String(), "ABC; Status:404")
	assertEquals(t, "Recover 7.1", pm.log, "")

	pm = NewConfigErrorFromString("ABC: Status:404. log:LM", 500)
	assertEquals(t, "Recover 8", pm.String(), "ABC; Status:404 Log:LM")
	assertEquals(t, "Recover 8.1", pm.log, "LM")

	pm = NewConfigErrorFromString("ABC", 500)
	assertEquals(t, "Recover 9", pm.String(), "ABC Status:500")
	assertEquals(t, "Recover 9.1", pm.log, "")

	pm = NewConfigError("R:X", 400, "LM")
	assertEquals(t, "Simple 1", pm.String(), "R:X Status:400 Log:LM")

	pm = NewConfigError("R:X Status", 400, "LM")
	assertEquals(t, "Simple 2", pm.String(), "R:X Status Status:400 Log:LM")

	pm = NewConfigError("R:X Status", 400, "L Status:500")
	assertEquals(t, "Simple 3", pm.String(), "R:X Status Status:400")

	pm = NewConfigError("R:X", 400, "LM")
	assertEquals(t, "Simple 4", pm.String(), "R;X Status:400 Log:LM")
	assertEquals(t, "Simple 4.1", pm.log, "LM")

	pm = NewConfigError("R:X Status", 400, "LM")
	assertEquals(t, "Simple 5", pm.String(), "R;X Status Status:400 Log:LM")

	pm = NewConfigError("R:X Status", 400, "L Status:500")
	assertEquals(t, "Simple 6", pm.String(), "R;X Status Status:400")

}

func TestThumbNailTrim(t *testing.T) {
	conf := loadConfigData(t)
	assertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail(""), "")
	assertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail("fred"), "fred")
	assertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail("2024_09_21_12_22_11_HuwSig.jpg.jpg"), "HuwSig.jpg")
}

func TestJoinPath(t *testing.T) {
	conf := loadConfigData(t)
	var f string
	pre := conf.GetServerDataRoot()

	f = conf.resolvePaths("", conf.GetServerDataRoot(), "/dir")
	assertEquals(t, "file 3", f, filepath.Join(pre, "dir"))

	f = conf.resolvePaths("", conf.GetServerDataRoot(), "***/dir")
	assertEquals(t, "file 4", f, "/dir")

	f = conf.resolvePaths("stuart", conf.GetServerDataRoot(), "***/dir")
	assertEquals(t, "file 5", f, "/dir")

	f = conf.resolvePaths("bob", conf.GetServerDataRoot(), "***dir")
	assertEquals(t, "file 6", f, "dir")

	f = conf.resolvePaths("stuart", conf.GetServerDataRoot(), "dir")
	assertEquals(t, "file 7", f, filepath.Join(pre, "stuart/dir"))

	f = conf.resolvePaths("john", conf.GetServerDataRoot(), "")
	assertEquals(t, "file 8", f, filepath.Join(pre, "john"))

	f = conf.resolvePaths("", conf.GetServerDataRoot(), "dir")
	assertEquals(t, "file 9", f, filepath.Join(pre, "dir"))

	f = conf.resolvePaths("", conf.GetServerDataRoot(), "")
	assertEquals(t, "file 9", f, pre)

}

func TestSubstitute(t *testing.T) {
	m1 := map[string]string{"A": "X", "b": "Y"}
	m2 := map[string]string{"UA": "UX", "Ub": "UY"}
	m3 := map[string]string{"UA": "UX", "Ub": "UX", "A": "UA"}

	assertSub(t, "Ab8", "-%{UA}-%{A}-%{b}-%{Ub}-%{A}-", "-UX-UA-Y-UX-UA-", m1, m3)
	assertSub(t, "Ab9", "-%%{%%{A}%{b}}-", "-%%{%XY}-", m1, m2)

	assertSub(t, "A2", "-%%{%%{A}}-", "-%%{%X}-", m1, m2)
	assertSub(t, "A3", "-%{%%{A}}-", "-%{%X}-", m1, m2)
	assertSub(t, "A4", "-%{%{A}}-", "-%{X}-", m1, m2)
	assertSub(t, "A5", "-{%%{A}}-", "-{%X}-", m1, m2)
	assertSub(t, "A6", "-{%{A}}-", "-{X}-", m1, m2)
	assertSub(t, "A7", "-%{A}}-", "-X}-", m1, m2)
	assertSub(t, "A8", "-%{A}-", "-X-", m1, m2)
	assertSub(t, "A9", "-%%{A}-", "-%X-", m1, m2)

	assertSub(t, "A5", "-%{A-", "-%{A-", m1, m2)
	assertSub(t, "A6", "-%%{A-", "-%%{A-", m1, m2)
	assertSub(t, "A8", "-%%A-", "-%%A-", m1, m2)
	assertSub(t, "A9", "-%A-", "-%A-", m1, m2)

	assertSub(t, "Z4", "-%{Z}-", "-%{Z}-", m1, m2)
	assertSub(t, "Z5", "-%{Z-", "-%{Z-", m1, m2)
	assertSub(t, "Z6", "-%%{Z-", "-%%{Z-", m1, m2)
	assertSub(t, "Z7", "-%%{Z}-", "-%%{Z}-", m1, m2)
	assertSub(t, "Z8", "-%%Z-", "-%%Z-", m1, m2)
	assertSub(t, "Z9", "-%Z-", "-%Z-", m1, m2)

	assertEquals(t, "empty", string(SubstituteFromMap([]byte(""), m1, m2)), "")
	assertEquals(t, "1 ch", string(SubstituteFromMap([]byte("%"), m1, m2)), "%")
	assertEquals(t, "2 ch", string(SubstituteFromMap([]byte("%{"), m1, m2)), "%{")
	assertEquals(t, "3 ch", string(SubstituteFromMap([]byte("%{}"), m1, m2)), "%{}")
	assertEquals(t, "4 chA", string(SubstituteFromMap([]byte("%{A}"), m1, m2)), "X")
	assertEquals(t, "4 chX", string(SubstituteFromMap([]byte("%{Z}"), m1, m2)), "%{Z}")
}

func assertSub(t *testing.T, id, sub, expected string, m1 map[string]string, m2 map[string]string) {
	r := string(SubstituteFromMap([]byte(sub), m1, m2))
	if r != expected {
		t.Fatalf("Substitution: %s, \nExpected [%s]\nActual   [%s]", id, expected, r)
	}
}

func assertEquals(t *testing.T, message string, actual string, expected string) {
	if actual != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s", message, expected, string(actual))
	}
}

func assertContains(t *testing.T, message string, actual string, contains []string) {
	for _, c := range contains {
		if !strings.Contains(actual, c) {
			t.Fatalf("%s.\nDoes Not Contain:%s\nActual:  %s", message, c, string(actual))
		}
	}
}

func TestUserExecBadExecId(t *testing.T) {
	conf := loadConfigData(t)

	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*ConfigError)
			if !ok || pm == nil {
				t.Fatalf("TestUserExecBadExecId: Should have returned a PanicMessage")
			}
			if pm.String() != "exec ID not found" {
				t.Fatalf("TestUserExecBadExecId: Should have returned a PanicMessage == exec ID not found | actual = %s", pm.String())
			}
		}
	}()

	conf.GetExecInfo("notid")
	t.Fatalf("Should have panicked")
}

func TestUserExec(t *testing.T) {
	conf := loadConfigData(t)

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
	assertContains(t, "TestUserExec ", exec.String(), []string{pre, "[cmd2]", "/logs/stdOutC2.txt"})

}
func TestGetUserExecInfo(t *testing.T) {
	conf := loadConfigData(t)
	c1 := conf.GetExecInfo("ls")

	if c1.Cmd[0] != "ls" {
		t.Fatal("Command should be ls -lta")
	}
	if c1.Cmd[1] != "-lta" {
		t.Fatal("Command should be ls -lta")
	}
	pre := conf.GetServerDataRoot()

	assertEquals(t, "TestGetUserExecInfo", c1.Dir, pre+"/exec")

}

func TestGetUserLocPathBadUser(t *testing.T) {
	conf := loadConfigData(t)

	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*ConfigError)
			if !ok || pm == nil {
				t.Fatalf("Should have returned a PanicMessage")
			}
			// {Status: 404, Reason: "user not found", Logged: "User=fred"}
			if pm.String() != "user not found Status:404 Log:User=fred" {
				t.Fatalf("Should have returned a PanicMessage == user not found:404:User=fred | actual = %s", pm.String())

			}
		}
	}()

	conf.GetUserLocPath("fred", "home")
	t.Fatalf("Should have panicked")
}

func TestGetUserLocPathBadLoc(t *testing.T) {
	conf := loadConfigData(t)

	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*ConfigError)
			if !ok || pm == nil {
				t.Fatalf("Should have returned a PanicMessage")
			}
			if pm.String() != "location not found Status:404 Log:User=stuart Location=nothome" {
				t.Fatalf("Should have returned a PanicMessage == location not found Status:404 Log:User=stuart Location=nothome | actual = %s", pm.String())
			}
		}
	}()

	conf.GetUserLocPath("stuart", "nothome")
	t.Fatalf("Should have panicked")
}

func TestGetUserLocPath(t *testing.T) {
	conf := loadConfigData(t)

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

func loadConfigData(t *testing.T) *ConfigData {
	errList := NewConfigErrorData()
	configData := NewConfigData("../goWebAppTest.json", "goWebApp", false, false, false, errList)
	if errList.ErrorCount() > 1 || configData == nil {
		t.Fatal(errList.String())
	}
	if configData == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errList.String())
	}
	return configData
}
