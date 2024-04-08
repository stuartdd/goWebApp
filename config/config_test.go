package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestJoinPath(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json")
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

func TestUserExec(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json")
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}
	pre := conf.GetServerDataRoot()
	exec, err := conf.GetUserExecInfo("bob", "c2")
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	assertContains(t, "UserExec", exec.String(), []string{pre, "[cmd2]", "/bob/logs/logOut.txt"})

	_, err = conf.GetUserExecInfo("bob", "X2")
	if err == nil {
		t.Fatal("Should NOT find exec X2")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Error should contain 'not found'. Actual:'%s'", err.Error())
	}
	_, err = conf.GetUserExecInfo("notbob", "x2")
	if err == nil {
		t.Fatal("Should NOT find notbob")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Error should contain 'not found'. Actual:'%s'", err.Error())
	}

}
func TestGetUserExecInfo(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json")
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}
	c1, err := conf.GetUserExecInfo("bob", "ls")
	if err != nil {
		t.Fatal(err)
	}
	if c1.Cmd[0] != "ls" {
		t.Fatal("Command should be ls -lta")
	}
	if c1.Cmd[1] != "-lta" {
		t.Fatal("Command should be ls -lta")
	}
	pre := conf.GetServerDataRoot()

	assertEquals(t, "TestGetUserExecInfo", c1.Dir, pre+"/bob")

}

func TestGetUserLocPath(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json")
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}

	_, e := conf.GetUserLocPath("fred", "home")
	if e == nil {
		t.Fatalf("Should have returned User error")
	}

	_, e = conf.GetUserLocPath("stuart", "xxxx")
	if e == nil {
		t.Fatalf("Should have returned Location error")
	}

	u, e := conf.GetUserLocPath("stuart", "home")
	if e != nil {
		t.Fatalf(e.Error())
	}

	if !strings.HasSuffix(u, "/testdata/stuart") {
		t.Fatalf("Should return path to /testdata/stuart")
	}

}
