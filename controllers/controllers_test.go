package controllers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"stuartdd.com/config"
)

func TestExec(t *testing.T) {
	conf, errlist := config.NewConfigData("../goWebAppTest.json")
	if errlist.Len() != 1 {
		t.Fatal(errlist.ToString())
	}
	if conf == nil {
		t.Fatal("Config is nil. Load failed")
	}
	params := map[string]string{"user": "bob", "exec": "ls"}

	ex := NewExecHandler(params, conf, func(out, err []byte, ec int) map[string]interface{} {
		if ec == 0 {
			return map[string]interface{}{"error": false, "code": ec, "out": string(out), "err": string(err)}
		}
		return map[string]interface{}{"error": true, "code": ec, "out": string(out), "err": string(err)}
	})

	res := ex.Submit()

	if !strings.Contains(string(res.Content()), "go.mod") {
		t.Fatal("Exec of ls -lta should cintain go.mod")
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

	tim5 := time.Now().UnixMicro()
	tn.ToJson(true)
	tim6 := time.Now().UnixMicro()

	AssertEquals(t, "Marshal", jj4, jj)
	timMarshal := tim2 - tim
	timToJson := tim4 - tim3
	timToJsonInd := tim6 - tim5
	if timToJson > timMarshal {
		t.Fatalf("Time Marshal:%d Time ToJson:%d Time ToJsonIndent:%d. Time ToJson should be faster!", timMarshal, timToJson, timToJsonInd)
	}
	if timToJsonInd > timMarshal {
		t.Fatalf("Time Marshal:%d Time ToJson:%d Time ToJsonIndent:%d. Time ToJsonIndent should be faster!", timMarshal, timToJson, timToJsonInd)
	}
}

func TestTreeNode(t *testing.T) {
	root := NewTreeNode("root")
	AssertEquals(t, "root", TreeAsJson(root), "{\"error\":false, \"tree\":{\"name\":\"root\"}}")
	root.AddPath("sub1")
	AssertEquals(t, "sub1", TreeAsJson(root), "{\"error\":false, \"tree\":{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"}]}}")
	root.AddPath("sub2")
	AssertEquals(t, "sub2", TreeAsJson(root), "{\"error\":false, \"tree\":{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"},{\"name\":\"sub2\"}]}}")
	root.AddPath("sub2/sub21")
	AssertEquals(t, "sub21", TreeAsJson(root), "{\"error\":false, \"tree\":{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"},{\"name\":\"sub2\",\"subs\":[{\"name\":\"sub21\"}]}]}}")
	root.AddPath("sub1/a1")
	root.AddPath("sub1/a1")
	root.AddPath("sub3/a1/a2")
	AssertEquals(t, "added", TreeAsJson(root), "{\"error\":false, \"tree\":{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\",\"subs\":[{\"name\":\"a1\"}]},{\"name\":\"sub2\",\"subs\":[{\"name\":\"sub21\"}]},{\"name\":\"sub3\",\"subs\":[{\"name\":\"a1\",\"subs\":[{\"name\":\"a2\"}]}]}]}}")
}

func AssertEquals(t *testing.T, message string, actual []byte, expected string) {
	if string(actual) != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s\nActual:  %s", message, expected, string(actual), actual)
	}
}
