package controllers

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stuartdd/goWebApp/config"
)

func TestToJson(t *testing.T) {
	conf, errlist := config.NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatalf("Config failed\n%s", errlist.String())
	}
	if conf == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errlist.String())
	}
	path := conf.GetUserLocPath("stuart", "home")
	params := NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "stuart", LocationParam: "home"})

	json := listDirectoriesAsJson(path, params, false, verboseNil, "")
	if !strings.HasPrefix(string(json), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"paths\":[{\"name\":") {
		t.Fatalf("listDirectoriesAsJson Invalid header in json. [%s]", string(json))
	}

	params = NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "stuart", LocationParam: "home", PathParam: path})
	files, _ := os.ReadDir(path)
	json = listFilesAsJson(files, params, false, verboseNil, "", -1)
	if !strings.HasPrefix(string(json), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"path\":{\"name\":\"") {
		t.Fatalf("filesAsJson with path Invalid header in json. [%s]", string(json))
	}

	params = NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "stuart", LocationParam: "home"})
	json = listFilesAsJson(files, params, false, verboseNil, "", -1)
	if !strings.HasPrefix(string(json), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"path\":null,\"files\":[") {
		t.Fatalf("filesAsJson without path Invalid header in json. [%s]", string(json))
	}
}

func TestExecFailRcNonZero(t *testing.T) {
	conf, errlist := config.NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatal(errlist.String())
	}
	if conf == nil {
		t.Fatal("Config is nil. Load failed")
	}
	os.Remove(conf.GetExecInfo("cat").GetOutLogFile())
	os.Remove(conf.GetExecInfo("cat").GetErrLogFile())
	params := NewUrlRequestParts(conf).WithParameters(map[string]string{ExecParam: "cat"})

	ex := NewExecHandler(params.AsAdmin(), conf, func(out, err []byte, ec int) map[string]interface{} {
		if ec == 0 {
			return map[string]interface{}{"error": false, "code": ec, "out": string(out), "err": string(err)}
		}
		return map[string]interface{}{"error": true, "code": ec, "out": string(out), "err": string(err)}
	}, func(s string) {
		// Log function
	},
		false,
		func(s string) {
			// Verbose function
		}, nil)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TestExecFailRcNonZero: Should NOT Panic; Error:%s", r)
		}
	}()

	res := ex.Submit()
	if res.Status != 200 {
		t.Fatalf("Exec status should be 200")
	}
	if !strings.Contains(string(res.Content()), "\"code\":1") {
		t.Fatalf("Return code should be 1")
	}
	if !strings.Contains(string(res.Content()), "\"err\":\"cat: fileThatDoesNotExist") {
		t.Fatalf("Return should have stdErr")
	}
	if !strings.Contains(string(res.Content()), "\"out\":\"\"") {
		t.Fatalf("Return should NOT have stdOut")
	}
	testDoesNotExist(t, conf.GetExecInfo("cat").GetOutLogFile())
	testFileContains(t, conf.GetExecInfo("cat").GetErrLogFile(), []string{"No such file or directory"})
}

func TestExecFailCommandNotFound(t *testing.T) {
	conf, errlist := config.NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatal(errlist.String())
	}
	if conf == nil {
		t.Fatal("Config is nil. Load failed")
	}
	os.Remove(conf.GetExecInfo("c2").GetOutLogFile())
	os.Remove(conf.GetExecInfo("c2").GetErrLogFile())
	params := NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "bob", ExecParam: "c2"})

	ex := NewExecHandler(params, conf, func(out, err []byte, ec int) map[string]interface{} {
		if ec == 0 {
			return map[string]interface{}{"error": false, "code": ec, "out": string(out), "err": string(err)}
		}
		return map[string]interface{}{"error": true, "code": ec, "out": string(out), "err": string(err)}
	}, func(s string) {
		// Log function
	},
		false,
		func(s string) {
			// Verbose function
		}, nil)

	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*config.PanicMessage)
			if !ok || pm == nil {
				t.Fatalf("TestExecFailCommandNotFound: Should have returned a PanicMessage")
			}
			if pm.Logged != "Exec: c2 RC:1 Error:exec failed: exec: \"cmd2\": executable file not found in $PATH" {
				t.Fatalf("TestExecFailCommandNotFound: Should have returned a PanicMessage == Exec: c2 RC:1 Error:exec failed: exec: \"cmd2\": executable file not found in $PATH | actual = %s", pm.String())
			}
		}
	}()
	testDoesNotExist(t, conf.GetExecInfo("c2").GetOutLogFile())
	testDoesNotExist(t, conf.GetExecInfo("c2").GetErrLogFile())

	ex.Submit()
}

func TestExecPass(t *testing.T) {
	conf, errlist := config.NewConfigData("../goWebAppTest.json", false, false, false)
	if errlist.ErrorCount() != 1 {
		t.Fatal(errlist.String())
	}
	if conf == nil {
		t.Fatal("Config is nil. Load failed")
	}
	os.Remove(conf.GetExecInfo("ls").GetOutLogFile())
	os.Remove(conf.GetExecInfo("ls").GetErrLogFile())

	params := NewUrlRequestParts(conf).WithParameters(map[string]string{ExecParam: "ls"})

	ex := NewExecHandler(params, conf, func(out, err []byte, ec int) map[string]interface{} {
		if ec == 0 {
			return map[string]interface{}{"error": false, "code": ec, "out": string(out), "err": string(err)}
		}
		return map[string]interface{}{"error": true, "code": ec, "out": string(out), "err": string(err)}
	}, func(s string) {
		// Log function
	},
		false,
		func(s string) {
			// Verbose function
		}, nil)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TestExecPass: Should NOT Panic")
		}
	}()
	res := ex.Submit()

	if !strings.Contains(string(res.Content()), "keepmefortesting") {
		t.Fatalf("Exec of ls -lta should contain keepmefortesting. Actual:\n%s", string(res.Content()))
	}

	testDoesNotExist(t, conf.GetExecInfo("ls").GetErrLogFile())
	testFileContains(t, conf.GetExecInfo("ls").GetOutLogFile(), []string{"keepmefortesting"})

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

func verboseNil(s string) {

}

func testFileContains(t *testing.T, name string, contains []string) {
	txt, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("file %s cannot be read", name)
	}
	txtStr := string(txt)
	for _, s := range contains {
		if !strings.Contains(txtStr, s) {
			t.Fatalf("File %s does not contain text %s", name, s)
		}
	}
}

func testDoesNotExist(t *testing.T, name string) {
	_, err := os.Stat(name)
	if err == nil {
		t.Fatalf("file %s should not exist", name)
	}
}
