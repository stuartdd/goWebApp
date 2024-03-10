package controllers

import (
	"encoding/json"
	"testing"
	"time"
)

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
	xx := tn.ToJson(true)
	tim6 := time.Now().UnixMicro()

	AssertEquals(t, "Marshal", jj4, jj)
	timMarshal := tim2 - tim
	timToJson := tim4 - tim3
	timToJsonInd := tim6 - tim5
	if timToJson >= timMarshal {
		t.Fatalf("Time Marshal:%d Time ToJson:%d Time ToJsonIndent:%d. Time ToJson should be faster!", timMarshal, timToJson, timToJsonInd)
	}
	t.Fatalf("Time Marshal:%d Time ToJson:%d Time ToJsonIndent:%d. Time ToJson should be faster!\n%s", timMarshal, timToJson, timToJsonInd, xx)
}

func TestTreeNode(t *testing.T) {
	root := NewTreeNode("root")
	AssertEquals(t, "root", root.ToJson(false), "{\"name\":\"root\"}")
	root.AddPath("sub1")
	AssertEquals(t, "sub1", root.ToJson(false), "{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"}]}")
	root.AddPath("sub2")
	AssertEquals(t, "sub2", root.ToJson(false), "{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"},{\"name\":\"sub2\"}]}")
	root.AddPath("sub2/sub21")
	AssertEquals(t, "sub21", root.ToJson(false), "{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"},{\"name\":\"sub2\",\"subs\":[{\"name\":\"sub21\"}]}]}")
	root.AddPath("sub1/a1")
	root.AddPath("sub1/a1")
	root.AddPath("sub3/a1/a2")
	AssertEquals(t, "added", root.ToJson(false), "{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\",\"subs\":[{\"name\":\"a1\"}]},{\"name\":\"sub2\",\"subs\":[{\"name\":\"sub21\"}]},{\"name\":\"sub3\",\"subs\":[{\"name\":\"a1\",\"subs\":[{\"name\":\"a2\"}]}]}]}")
}

func AssertEquals(t *testing.T, message string, actual []byte, expected string) {
	if string(actual) != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s\nActual:  %s", message, expected, string(actual), actual)
	}
}
