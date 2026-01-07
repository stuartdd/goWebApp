package config

import (
	"encoding/json"
	"os"
	"testing"
)

const configRef = "../goWebAppTest.json"
const configTmp = "../goWebAppTmp.json"

func TestLoad(t *testing.T) {
	LoadConfigData(t, configRef, nil)
}
func TestLoadNilStaticWebData(t *testing.T) {
	c := UpdateConfigAndLoad(t, func(cdff *ConfigDataFromFile) {
		cdff.StaticWebData = nil
	}, nil)
	if c.HasStaticWebData {
		t.Fatal("Config should NOT have static data")
	}
	if c.IsTemplating {
		t.Fatal("Failed Config is templating")
	}
}

func TestLoadNoHomePage(t *testing.T) {
	errList := NewConfigErrorData()
	c := UpdateConfigAndLoad(t, func(cdff *ConfigDataFromFile) {
		cdff.StaticWebData.HomePage = ""
	}, errList)
	if c.HasStaticWebData {
		t.Fatal("Config should NOT have static data")
	}
	if c.IsTemplating {
		t.Fatal("Failed Config is templating")
	}
	AssertContains(t, "TestLoadNoHomePage", errList.String(), []string{
		"StaticWebData 'Home' page is undefined",
		"/missingfolder] Not found",
	})
}

func TestLoadNoPaths(t *testing.T) {
	errList := NewConfigErrorData()
	c := UpdateConfigAndLoad(t, func(cdff *ConfigDataFromFile) {
		cdff.StaticWebData.Paths = nil
	}, errList)
	if c.HasStaticWebData {
		t.Fatal("Config should NOT have static data")
	}
	if c.IsTemplating {
		t.Fatal("Failed Config is templating")
	}
	AssertErrors(t, "TestLoadNoPaths", errList, []string{
		"StaticWebData 'Paths' is empty",
		"/missingfolder] Not found",
	}, 2)
}

func TestLoadNoStaticPath(t *testing.T) {
	errList := NewConfigErrorData()
	c := UpdateConfigAndLoad(t, func(cdff *ConfigDataFromFile) {
		cdff.StaticWebData.Paths = map[string]string{"images": "x123"}
	}, errList)
	if c.HasStaticWebData {
		t.Fatal("Config should NOT have static data")
	}
	if c.IsTemplating {
		t.Fatal("Failed Config is templating")
	}
	AssertErrors(t, "TestLoadNoStaticPath", errList, []string{
		"StaticWebData 'Paths[static]' was not found",
		"x123: no such file or directory",
		"/missingfolder] Not found",
	}, 3)
}

func TestLoadNoStaticNotFound(t *testing.T) {
	errList := NewConfigErrorData()
	c := UpdateConfigAndLoad(t, func(cdff *ConfigDataFromFile) {
		cdff.StaticWebData.Paths = map[string]string{"static": "../testdata/missing1", "images": "../testdata/missing2"}
	}, errList)
	if c.HasStaticWebData {
		t.Fatal("Config should NOT have static data")
	}
	if c.IsTemplating {
		t.Fatal("Failed Config is templating")
	}
	AssertErrors(t, "TestLoadNoStaticNotFound", errList, []string{
		"StaticWebData 'Paths[static]'",
		"/missing1: no such file or directory",
		"StaticWebData 'Paths[images]'",
		"/missing2: no such file or directory",
		"/missingfolder] Not found",
	}, 3)
}

func TestTemplateDataNotFound(t *testing.T) {
	errList := NewConfigErrorData()
	c := UpdateConfigAndLoad(t, func(cdff *ConfigDataFromFile) {
		cdff.StaticWebData.TemplateStaticFiles.DataFile = "missing.json"
	}, errList)
	if c.HasStaticWebData {
		t.Fatal("Config should NOT have static data")
	}
	if c.IsTemplating {
		t.Fatal("Failed Config is templating")
	}
	AssertErrors(t, "TestTemplateDataNotFound", errList, []string{
		"failed to read template data file",
		"/missing.json: no such file or directory",
		"/missingfolder] Not found",
	}, 2)
}
func TestTemplateDataNotJson(t *testing.T) {
	errList := NewConfigErrorData()
	c := UpdateConfigAndLoad(t, func(cdff *ConfigDataFromFile) {
		cdff.StaticWebData.TemplateStaticFiles.DataFile = "simple.html"
	}, errList)
	if c.HasStaticWebData {
		t.Fatal("Config should NOT have static data")
	}
	if c.IsTemplating {
		t.Fatal("Failed Config is templating")
	}
	AssertErrors(t, "TestTemplateDataNotJson", errList, []string{
		"failed to parse template json file",
		"/missingfolder] Not found",
	}, 2)
}

func TestTemplateDataUndefined(t *testing.T) {
	errList := NewConfigErrorData()
	c := UpdateConfigAndLoad(t, func(cdff *ConfigDataFromFile) {
		cdff.StaticWebData.TemplateStaticFiles.Files = []string{}
	}, errList)
	if c.HasStaticWebData {
		t.Fatal("Config should NOT have static data")
	}
	if c.IsTemplating {
		t.Fatal("Failed Config is templating")
	}
	AssertErrors(t, "TestTemplateDataUndefined", errList, []string{
		"No template 'TemplateStaticFiles.Files' have been defined",
		"/missingfolder] Not found",
	}, 2)
}
func TestTemplateFileNotFound(t *testing.T) {
	errList := NewConfigErrorData()
	c := UpdateConfigAndLoad(t, func(cdff *ConfigDataFromFile) {
		cdff.StaticWebData.TemplateStaticFiles.Files = []string{"missing.html"}
	}, errList)
	if c.HasStaticWebData {
		t.Fatal("Config should NOT have static data")
	}
	if c.IsTemplating {
		t.Fatal("Failed Config is templating")
	}
	AssertErrors(t, "TestTemplateFileNotFound", errList, []string{
		"/missing.html: no such file or directory",
		"/missingfolder] Not found",
	}, 2)
}

func TestTemplateFileIsDir(t *testing.T) {
	errList := NewConfigErrorData()
	c := UpdateConfigAndLoad(t, func(cdff *ConfigDataFromFile) {
		cdff.StaticWebData.TemplateStaticFiles.Files = []string{"images"}
	}, errList)
	if c.HasStaticWebData {
		t.Fatal("Config should NOT have static data")
	}
	if c.IsTemplating {
		t.Fatal("Failed Config is templating")
	}
	AssertErrors(t, "TestTemplateFileIsDir", errList, []string{
		"/images is a directory",
		"/missingfolder] Not found",
	}, 2)
}

func UpdateConfigAndLoad(t *testing.T, callBack func(*ConfigDataFromFile), errList *ConfigErrorData) *ConfigData {
	content, err := os.ReadFile(configRef)
	if err != nil {
		t.Fatalf("Failed to read config data file:%s. Error:%s", configRef, err.Error())
	}
	config := &ConfigDataFromFile{
		ThumbnailTrim: []int{thumbnailTrimPrefix, thumbnailTrimSuffix},
	}
	err = json.Unmarshal(content, &config)
	if err != nil {
		t.Fatalf("Failed to understand the config data in the file:%s. Error:%s", configRef, err.Error())
	}
	callBack(config)
	cf2, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to martial new json. Error:%s", err.Error())
	}
	err = os.WriteFile(configTmp, cf2, os.ModePerm)
	if err != nil {
		t.Fatalf("Failed to write new json file %s. Error:%s", configTmp, err.Error())
	}
	defer os.Remove(configTmp)
	return LoadConfigData(t, configTmp, errList)
}
