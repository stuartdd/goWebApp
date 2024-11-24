package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateStaticFiles(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}

}
func TestThumbNailTrim(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}
	assertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail(""), "")
	assertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail("fred"), "fred")
	assertEquals(t, "ConvertToThumbnail ", conf.ConvertToThumbnail("2024_09_21_12_22_11_HuwSig.jpg.jpg"), "HuwSig.jpg")
}
func TestJoinPath(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}
	var f string
	pre := conf.GetServerDataRoot()

	f = conf.resolvePaths("", "/dir")
	assertEquals(t, "file 3", f, filepath.Join(pre, "dir"))

	f = conf.resolvePaths("", "***/dir")
	assertEquals(t, "file 4", f, "/dir")

	f = conf.resolvePaths("stuart", "***/dir")
	assertEquals(t, "file 5", f, "/dir")

	f = conf.resolvePaths("bob", "***dir")
	assertEquals(t, "file 6", f, "dir")

	f = conf.resolvePaths("stuart", "dir")
	assertEquals(t, "file 7", f, filepath.Join(pre, "stuart/dir"))

	f = conf.resolvePaths("john", "")
	assertEquals(t, "file 8", f, filepath.Join(pre, "john"))

	f = conf.resolvePaths("", "dir")
	assertEquals(t, "file 9", f, filepath.Join(pre, "dir"))

	f = conf.resolvePaths("", "")
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

	assertEquals(t, "empty", SubstituteFromMap([]byte(""), m1, m2), "")
	assertEquals(t, "1 ch", SubstituteFromMap([]byte("%"), m1, m2), "%")
	assertEquals(t, "2 ch", SubstituteFromMap([]byte("%{"), m1, m2), "%{")
	assertEquals(t, "3 ch", SubstituteFromMap([]byte("%{}"), m1, m2), "%{}")
	assertEquals(t, "4 chA", SubstituteFromMap([]byte("%{A}"), m1, m2), "X")
	assertEquals(t, "4 chX", SubstituteFromMap([]byte("%{Z}"), m1, m2), "%{Z}")
}

func assertSub(t *testing.T, id, sub, expected string, m1 map[string]string, m2 map[string]string) {
	r := SubstituteFromMap([]byte(sub), m1, m2)
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
	conf, errlist := NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}

	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*PanicMessage)
			if !ok || pm == nil {
				t.Fatalf("TestUserExecBadExecId: Should have returned a PanicMessage")
			}
			if pm.Reason != "exec ID not found" {
				t.Fatalf("TestUserExecBadExecId: Should have returned a PanicMessage == exec ID not found | actual = %s", pm.String())
			}
		}
	}()

	conf.GetExecInfo("notid")
	t.Fatalf("Should have panicked")
}

func TestUserExec(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}

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
 	assertContains(t, "TestUserExec ", exec.String(), []string{pre, "[cmd2]", "/logs/logOut.txt"})

}
func TestGetUserExecInfo(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}
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
	conf, errlist := NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}

	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*PanicMessage)
			if !ok || pm == nil {
				t.Fatalf("Should have returned a PanicMessage")
			}
			if pm.String() != "user not found:404:User=fred" {
				t.Fatalf("Should have returned a PanicMessage == user not found:404:User=fred | actual = %s", pm.String())

			}
		}
	}()

	conf.GetUserLocPath("fred", "home")
	t.Fatalf("Should have panicked")
}

func TestGetUserLocPathBadLoc(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}

	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*PanicMessage)
			if !ok || pm == nil {
				t.Fatalf("Should have returned a PanicMessage")
			}
			if pm.String() != "location not found:404:User=stuart Location=nothome" {
				t.Fatalf("Should have returned a PanicMessage == location not found:404:User=stuart Location=nothome | actual = %s", pm.String())
			}
		}
	}()

	conf.GetUserLocPath("stuart", "nothome")
	t.Fatalf("Should have panicked")
}

func TestGetUserLocPath(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}

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
