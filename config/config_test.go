package config

import (
	"strings"
	"testing"
)

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
}

func assertSub(t *testing.T, id, sub, expected string, m1 map[string]string, m2 map[string]string) {
	r := SubstituteFromMap([]rune(sub), m1, m2)
	if r != expected {
		t.Fatalf("Substitution: %s, \nExpected [%s]\nActual   [%s]", id, expected, r)
	}
}

func TestUserExec(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json")
	if errlist.Len() != 1 {
		t.Fatal(errlist.ToString())
	}
	if conf == nil {
		t.Fatal("Config is nil. Load failed")
	}
	p := NewParameters(map[string]string{"user": "bob", "exec": "c2"}, conf)
	exec, err := p.UserExec()
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	if exec.ToString() != "CMD:[cmd2], Dir:, LogOut:/home/stuart/git/golang/goWebApp/testdata/logs/logOut.txt, LogErr:" {
		t.Fatalf("Did not find the correct exec! Actual '%s'", exec.ToString())
	}
	p = NewParameters(map[string]string{"user": "bob", "exec": "X2"}, conf)
	_, err = p.UserExec()
	if err == nil {
		t.Fatal("Should NOT find exec X2")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Error should contain 'not found'. Actual:'%s'", err.Error())
	}
	p = NewParameters(map[string]string{"user": "notbob", "exec": "c2"}, conf)
	_, err = p.UserExec()
	if err == nil {
		t.Fatal("Should NOT find notbob")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Error should contain 'not found'. Actual:'%s'", err.Error())
	}

}
func TestCommands(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json")
	if errlist.Len() != 1 {
		t.Fatal(errlist.ToString())
	}
	if conf == nil {
		t.Fatal("Config is nil. Load failed")
	}
	c1, err := conf.UserExec("bob", "ls")
	if err != nil {
		t.Fatal(err)
	}
	if c1.Cmd[0] != "ls" {
		t.Fatal("Command should be ls -lta")
	}
	if c1.Cmd[1] != "-lta" {
		t.Fatal("Command should be ls -lta")
	}
	if c1.Dir != "" {
		t.Fatal("Command Dir should be empty")
	}
	if c1.Log != "/home/stuart/git/golang/goWebApp/testdata/logs" {
		t.Fatal("Command Log should be ../testdata/logs/boblog1")
	}

}

func TestUserDataPath(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json")
	if errlist.Len() != 1 {
		t.Fatal(errlist.ToString())
	}

	_, e := conf.GetUserLocPathParams(NewParameters(map[string]string{"xxxx": "fred", "loc": "home"}, conf))
	if e == nil {
		t.Fatalf("Should have returned User error")
	}
	if e.Error() != "url parameter 'user' is missing" {
		t.Fatalf("Should have returned url parameter 'user' is missing")
	}

	_, e = conf.GetUserLocPathParams(NewParameters(map[string]string{"user": "fred", "xxx": "home"}, conf))
	if e == nil {
		t.Fatalf("Should have returned User error")
	}
	if e.Error() != "url parameter 'loc' is missing" {
		t.Fatalf("Should have returned url parameter 'loc' is missing")
	}

	_, e = conf.GetUserLocFilePathParams(NewParameters(map[string]string{"user": "bob", "loc": "home"}, conf))
	if e == nil {
		t.Fatalf("Should have returned User error")
	}
	if e.Error() != "url parameter 'name' is missing" {
		t.Fatalf("Should have returned url parameter 'name' is missing")
	}

	_, e = conf.GetUserLocPathParams(NewParameters(map[string]string{"user": "fred", "loc": "home"}, conf))
	if e == nil {
		t.Fatalf("Should have returned User error")
	}

	_, e = conf.GetUserLocPathParams(NewParameters(map[string]string{"user": "stuart", "loc": "xxx"}, conf))
	if e == nil {
		t.Fatalf("Should have returned Location error")
	}

	u, e := conf.GetUserLocPathParams(NewParameters(map[string]string{"user": "stuart", "loc": "home"}, conf))
	if e != nil {
		t.Fatalf(e.Error())
	}

	if !strings.HasSuffix(u, "/testdata") {
		t.Fatalf("Should return path to /testdata")
	}

	f, e := conf.GetUserLocFilePathParams(NewParameters(map[string]string{"user": "bob", "loc": "home", "name": "data.json"}, conf))
	if e != nil {
		t.Fatalf(e.Error())
	}
	if !strings.HasSuffix(f, "/testdata/data.json") {
		t.Fatalf("Should return path to /testdata/data.json")
	}

	f, e = conf.GetUserLocFilePathParams(NewParameters(map[string]string{"user": "stuart", "loc": "picsPlus", "name": "pics.json"}, conf))
	if e != nil {
		t.Fatalf(e.Error())
	}
	if !strings.HasSuffix(f, "/testdata/testfolder/pics.json") {
		t.Fatalf("Should return path to /testdata/testfolder/pics.json")
	}

}
