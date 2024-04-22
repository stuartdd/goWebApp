package controllers

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"stuartdd.com/config"
)

func TestToJson(t *testing.T) {
	conf, errlist := config.NewConfigData("../goWebAppTest.json", false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}
	path, _ := conf.GetUserLocPath("stuart", "home")
	params := NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "stuart", LocationParam: "home"})

	json := listDirectoriesAsJson(path, params)
	if !strings.HasPrefix(string(json), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"paths\":[{\"name\":") {
		t.Fatalf("listDirectoriesAsJson Invalid header in json. [%s]", string(json))
	}

	params = NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "stuart", LocationParam: "home", PathParam: path})
	files, _ := os.ReadDir(path)
	json = filesAsJson(files, params)
	if !strings.HasPrefix(string(json), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"path\":{\"name\":\"") {
		t.Fatalf("filesAsJson with path Invalid header in json. [%s]", string(json))
	}

	params = NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "stuart", LocationParam: "home"})
	json = filesAsJson(files, params)
	if !strings.HasPrefix(string(json), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"path\":null,\"files\":[") {
		t.Fatalf("filesAsJson without path Invalid header in json. [%s]", string(json))
	}

}

func TestExec(t *testing.T) {
	conf, errlist := config.NewConfigData("../goWebAppTest.json", false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatal(errlist.String())
	}
	if conf == nil {
		t.Fatal("Config is nil. Load failed")
	}
	params := NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "bob", ExecParam: "ls"})

	ex := NewExecHandler(params, conf, func(out, err []byte, ec int) map[string]interface{} {
		if ec == 0 {
			return map[string]interface{}{"error": false, "code": ec, "out": string(out), "err": string(err)}
		}
		return map[string]interface{}{"error": true, "code": ec, "out": string(out), "err": string(err)}
	}, func(s string) {
		// Log function
	}, nil)

	res := ex.Submit()

	if !strings.Contains(string(res.Content()), "t1.JSON") {
		t.Fatal("Exec of ls -lta should contain t1.JSON")
	}
}

func TestMarshal(t *testing.T) {
	jj := "{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\",\"subs\":[{\"name\":\"a1\"}]},{\"name\":\"sub2\",\"subs\":[{\"name\":\"sub21\"}]},{\"name\":\"sub3\",\"subs\":[{\"name\":\"a1\",\"subs\":[{\"name\":\"a2\"}]}]}]}"
	tn := &TreeDirNode{}
	err := json.Unmarshal([]byte(jj), tn)
	if err != nil {
		t.Fatalf("failed to unmarshal the JSON. Error:%s", err.Error())
	}
	AssertEquals(t, "Unmarshal", tn.ToJson(false), jj)

	tim := time.Now().UnixMicro()
	jj2, err := json.Marshal(tn)
	tim2 := time.Now().UnixMicro()
	if err != nil {
		t.Fatalf("failed to marshal the JSON. Error:%s", err.Error())
	}
	AssertEquals(t, "Marshal", jj2, jj)

	tim3 := time.Now().UnixMicro()
	jj4 := tn.ToJson(false)
	tim4 := time.Now().UnixMicro()

	AssertEquals(t, "Marshal", jj4, jj)
	timMarshal := tim2 - tim
	timToJson := tim4 - tim3
	if timToJson > timMarshal {
		t.Fatalf("Time Marshal:%d Time ToJson:%d. Time ToJson should be faster!", timMarshal, timToJson)
	}
}

func TestTreeNode(t *testing.T) {
	params := NewUrlRequestParts(nil).WithParameters(map[string]string{UserParam: "stuart", LocationParam: "home"})
	root := NewTreeNode("root")
	AssertEquals(t, "root", treeAsJson(root, params), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"tree\":{\"name\":\"root\"}}")
	root.AddPath("sub1")
	AssertEquals(t, "sub1", treeAsJson(root, params), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"tree\":{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"}]}}")
	root.AddPath("sub2")
	AssertEquals(t, "sub2", treeAsJson(root, params), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"tree\":{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"},{\"name\":\"sub2\"}]}}")
	root.AddPath("sub2/sub21")
	AssertEquals(t, "sub21", treeAsJson(root, params), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"tree\":{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"},{\"name\":\"sub2\",\"subs\":[{\"name\":\"sub21\"}]}]}}")
	root.AddPath("sub1/a1")
	root.AddPath("sub1/a1")
	root.AddPath("sub3/a1/a2")
	AssertEquals(t, "added", treeAsJson(root, params), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"tree\":{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\",\"subs\":[{\"name\":\"a1\"}]},{\"name\":\"sub2\",\"subs\":[{\"name\":\"sub21\"}]},{\"name\":\"sub3\",\"subs\":[{\"name\":\"a1\",\"subs\":[{\"name\":\"a2\"}]}]}]}}")
}

func AssertEquals(t *testing.T, message string, actual []byte, expected string) {
	if string(actual) != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s", message, expected, string(actual))
	}
}
