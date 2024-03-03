package config

import (
	"testing"
)

func TestUserDataPath(t *testing.T) {
	conf, err := NewConfigData("../goWebAppTest.json")
	if err != nil {
		t.Fatal(err)
	}

	_, e := conf.UserDataPath(NewParameters(map[string]string{"user": "fred", "loc": "home"}, conf))
	if e == nil {
		t.Fatalf("Should have returned User error")
	}

	_, e = conf.UserDataPath(NewParameters(map[string]string{"user": "stuart", "loc": "xxx"}, conf))
	if e == nil {
		t.Fatalf("Should have returned Location error")
	}

	u, e := conf.UserDataPath(NewParameters(map[string]string{"user": "stuart", "loc": "home"}, conf))
	if e != nil {
		t.Fatalf(e.Error())
	}

	if u != "../sHome" {
		t.Fatalf("Should return ../sHome")
	}

	f, e := conf.UserDataFile(NewParameters(map[string]string{"user": "bob", "loc": "home", "name": "data.json"}, conf))
	if e != nil {
		t.Fatalf(e.Error())
	}
	if f != "../bHome/data.json" {
		t.Fatalf("Should return ../bHome/data.json")
	}

	f, e = conf.UserDataFile(NewParameters(map[string]string{"user": "stuart", "loc": "pics", "name": "data.json"}, conf))
	if e != nil {
		t.Fatalf(e.Error())
	}
	if f != "../testdata/data.json" {
		t.Fatalf("Should return ../testdata/data.json. actual%s", f)
	}

}

func TrialConfigUnMarshal(t *testing.T) {
	ld := &LogData{
		FileNameMask:   "fileNameMask",
		Path:           "logs",
		MonitorSeconds: 20,
		LogLevel:       "quiet",
	}

	conf := &ConfigData{
		Port:               8080,
		Users:              make(map[string]UserData),
		UserDataRoot:       "~/",
		LogData:            ld,
		ContentTypeCharset: "utf-8",
		ServerName:         "serverName",
		PanicResponseCode:  500,
		Debugging:          true,
		ModuleName:         "moduleName",
		ConfigName:         "ConfigName.json",
	}

	conf.Users["stuart"] = NewUserData("Stuart", map[string]string{"home": "/home", "usr": "stuarts"})
	conf.Users["bob"] = NewUserData("Bob", map[string]string{"home": "/home", "usr": "bobs"})

	s, e := conf.ToString()
	if e != nil {
		t.Fatalf(e.Error())
	}
	t.Fatal(s)

}
