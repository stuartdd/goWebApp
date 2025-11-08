package controllers

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/runCommand"
)

func TestToJson(t *testing.T) {
	conf := loadConfigData(t)
	path := conf.GetUserLocPath("stuart", "home")
	params := NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "stuart", LocationParam: "home"})

	json := listDirectoriesAsJson(path, params, nil, "")
	if !strings.HasPrefix(string(json), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"paths\":[{\"name\":") {
		t.Fatalf("listDirectoriesAsJson Invalid header in json. [%s]", string(json))
	}

	params = NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "stuart", LocationParam: "home", PathParam: path})
	files, _ := os.ReadDir(path)
	json = listFilesAsJson(files, params, nil, "", -1)
	if !strings.HasPrefix(string(json), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"path\":{\"name\":\"") {
		t.Fatalf("filesAsJson with path Invalid header in json. [%s]", string(json))
	}

	params = NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "stuart", LocationParam: "home"})
	json = listFilesAsJson(files, params, nil, "", -1)
	if !strings.HasPrefix(string(json), "{\"error\":false,\"user\":\"stuart\",\"loc\":\"home\",\"path\":null,\"files\":[") {
		t.Fatalf("filesAsJson without path Invalid header in json. [%s]", string(json))
	}
}

func TestExecFailRcNonZero(t *testing.T) {
	conf := loadConfigData(t)
	os.Remove(conf.GetExecInfo("cat").GetOutLogFile())
	os.Remove(conf.GetExecInfo("cat").GetErrLogFile())
	params := NewUrlRequestParts(conf).WithParameters(map[string]string{ExecParam: "cat"})

	ex := NewExecHandler(params.AsAdmin(), conf, func(id string, respType string, out, err []byte, ec int, q map[string][]string) []byte {
		return []byte(fmt.Sprintf("{\"error\": %t, \"code\": %d, \"out\": \"%s\", \"err\": \"%s\"}", ec != 0, ec, out, err))
	}, func(s string) {
		// Log function
	}, func(s string) {
		// Verbose function
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TestExecFailRcNonZero: Should NOT Panic; Error:%s", r)
		}
	}()

	res := ex.Submit()
	if res.Status != 403 {
		t.Fatalf("Exec status should be 403")
	}
	if !strings.Contains(string(res.Content()), "\"code\": 1") {
		t.Fatalf("Return code should be 1")
	}
	if !strings.Contains(string(res.Content()), "\"err\": \"cat: fileThatDoesNotExist") {
		t.Fatalf("Return should have stdErr")
	}
	if !strings.Contains(string(res.Content()), "\"out\": \"\"") {
		t.Fatalf("Return should NOT have stdOut")
	}
	testDoesNotExist(t, conf.GetExecInfo("cat").GetOutLogFile())
	testFileContains(t, conf.GetExecInfo("cat").GetErrLogFile(), []string{"No such file or directory"})
}

func TestExecFailCommandNotFound(t *testing.T) {
	conf := loadConfigData(t)
	os.Remove(conf.GetExecInfo("c2").GetOutLogFile())
	os.Remove(conf.GetExecInfo("c2").GetErrLogFile())
	params := NewUrlRequestParts(conf).WithParameters(map[string]string{UserParam: "bob", ExecParam: "c2"})

	ex := NewExecHandler(params, conf, func(id string, respType string, out, err []byte, ec int, q map[string][]string) []byte {
		return []byte(fmt.Sprintf("\"error\": %t, \"code\": %d, \"out\": \"%s\", \"err\": \"%s\"", ec != 0, ec, out, err))
	}, func(s string) {
		// Log function
	}, func(s string) {
		// Verbose function
	})
	defer func() {
		if r := recover(); r != nil {
			pm, ok := r.(*runCommand.ExecError)
			if !ok || pm == nil {
				t.Fatalf("TestExecFailCommandNotFound: Should have returned a runCommand.ExecError")
			}
			if !strings.Contains(pm.LogError(), "exec/cmd2: no such file or directory") {
				t.Fatalf("TestExecFailCommandNotFound: Should contain: 'exec/cmd2: no such file or directory'")
			}
		}
	}()
	testDoesNotExist(t, conf.GetExecInfo("c2").GetOutLogFile())
	testDoesNotExist(t, conf.GetExecInfo("c2").GetErrLogFile())

	ex.Submit()
}

func TestExecPass(t *testing.T) {
	conf := loadConfigData(t)
	os.Remove(conf.GetExecInfo("ls").GetOutLogFile())
	os.Remove(conf.GetExecInfo("ls").GetErrLogFile())

	params := NewUrlRequestParts(conf).WithParameters(map[string]string{ExecParam: "ls"})

	ex := NewExecHandler(params, conf, func(id string, respType string, out, err []byte, ec int, q map[string][]string) []byte {
		return []byte(fmt.Sprintf("\"error\": %t, \"code\": %d, \"out\": \"%s\", \"err\": \"%s\"", ec != 0, ec, out, err))
	}, func(s string) {
		// Log function
	}, func(s string) {
		// Verbose function
	})

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

// func TestMarshal(t *testing.T) {
// 	jj := "{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\",\"subs\":[{\"name\":\"a1\"}]},{\"name\":\"sub2\",\"subs\":[{\"name\":\"sub21\"}]},{\"name\":\"sub3\",\"subs\":[{\"name\":\"a1\",\"subs\":[{\"name\":\"a2\"}]}]}]}"
// 	tn := &TreeDirNode{}
// 	err := json.Unmarshal([]byte(jj), tn)
// 	if err != nil {
// 		t.Fatalf("failed to unmarshal the JSON. Error:%s", err.Error())
// 	}
// 	AssertEquals(t, "Unmarshal", tn.ToJson(false), jj)

// 	jj2 := []byte{}
// 	tim1 := time.Now().UnixMicro()
// 	for i := 0; i < 50; i++ {
// 		jj2, err = json.Marshal(tn)
// 	}
// 	tim2 := time.Now().UnixMicro()
// 	if err != nil {
// 		t.Fatalf("failed to marshal the JSON. Error:%s", err.Error())
// 	}
// 	AssertEquals(t, "Marshal", jj2, jj)

// 	jj4 := []byte{}
// 	tim3 := time.Now().UnixMicro()
// 	for i := 0; i < 50; i++ {
// 		jj4 = tn.ToJson(false)
// 	}
// 	tim4 := time.Now().UnixMicro()

// 	AssertEquals(t, "Marshal", jj4, jj)
// 	timMarshal := tim2 - tim1
// 	timToJson := tim4 - tim3
// 	if timToJson > (timMarshal + 10) {
// 		t.Fatalf("Time Marshal:%d Time ToJson:%d. Time ToJson should be faster!", timMarshal, timToJson)
// 	}
// }

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

func writeToFile(t *testing.T, name string, bytes []byte) {
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = f.Write(bytes)
	if err != nil {
		t.Fatal(err)
	}
}

func loadConfigData(t *testing.T) *config.ConfigData {
	errList := config.NewConfigErrorData()
	configData := config.NewConfigData("../goWebAppTest.json", "goWebApp", false, false, false, errList)
	if errList.ErrorCount() > 1 || configData == nil {
		t.Fatal(errList.String())
	}
	if configData == nil {
		t.Fatalf("Config is nil. Load failed\n%s", errList.String())
	}
	return configData
}
