package config

import (
	"strings"
	"testing"
)

func TestUserExec(t *testing.T) {
	conf, errlist := NewConfigData("../goWebAppTest.json")
	if errlist.Len() != 1 {
		t.Fatal(errlist.ToString())
	}
	if conf == nil {
		t.Fatal("Config is nil. Load failed")
	}
	p := NewParameters(map[string]string{"user": "bob", "sync": "c2"}, conf)
	exec, err := p.UserExec()
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	if exec.ToString() != "CMD:[cmd2], Dir:testdata, Log:../testdata/boblog2" {
		t.Fatal("Did not find the correct exec!")
	}
	p = NewParameters(map[string]string{"user": "bob", "sync": "X2"}, conf)
	_, err = p.UserExec()
	if err == nil {
		t.Fatal("Should NOT find exec X2")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Error should contain 'not found'. Actual:'%s'", err.Error())
	}
	p = NewParameters(map[string]string{"user": "notbob", "sync": "c2"}, conf)
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
	if c1.Log != "../testdata/logs/boblog1" {
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

func TrialConfigUnMarshal(t *testing.T) {
	ld := &LogData{
		FileNameMask:   "fileNameMask",
		Path:           "logs",
		MonitorSeconds: 20,
		LogLevel:       "quiet",
	}

	conf := &ConfigDataInternal{
		Port:               8080,
		Users:              make(map[string]UserData),
		UserDataRoot:       "~/",
		LogData:            ld,
		ContentTypeCharset: "utf-8",
		ServerName:         "serverName",
		PanicResponseCode:  500,
		FaviconIcoPath:     "",
	}

	conf.Users["stuart"] = NewUserData("Stuart", map[string]string{"home": "/home", "usr": "stuarts"})
	conf.Users["bob"] = NewUserData("Bob", map[string]string{"home": "/home", "usr": "bobs"})

	s, e := conf.toString()
	if e != nil {
		t.Fatalf(e.Error())
	}
	t.Fatal(s)

}
