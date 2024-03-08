package controllers

import (
	"testing"
)

func TestTreeNode(t *testing.T) {
	root := newTreeNode("root")
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
