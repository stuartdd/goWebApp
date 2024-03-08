package controllers

import (
	"bytes"
	"strings"
	"testing"
)

func TestTreeNode(t *testing.T) {
	root := newTreeNode("root")
	AssertEquals(t, "root", root.ToJson(false), "{\"name\":\"root\"}")
	root.add("sub1")
	AssertEquals(t, "sub1", root.ToJson(false), "{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"}]}")
	s2 := root.add("sub2")
	AssertEquals(t, "sub2", root.ToJson(false), "{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"},{\"name\":\"sub2\"}]}")
	s2.add("sub21")
	AssertEquals(t, "sub21", root.ToJson(false), "{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\"},{\"name\":\"sub2\",\"subs\":[{\"name\":\"sub21\"}]}]}")
	root.addPath([]string{"sub1", "a1"})
	root.addPath([]string{"sub1", "a1"})
	root.addPath([]string{"sub3", "a1", "a2"})
	AssertEquals(t, "added", root.ToJson(false), "{\"name\":\"root\",\"subs\":[{\"name\":\"sub1\",\"subs\":[{\"name\":\"a1\"}]},{\"name\":\"sub2\",\"subs\":[{\"name\":\"sub21\"}]},{\"name\":\"sub3\",\"subs\":[{\"name\":\"a1\",\"subs\":[{\"name\":\"a2\"}]}]}]}")
}

func AssertEquals(t *testing.T, message, actual, expected string) {
	if trimString(actual) != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s\nActual:  %s", message, expected, trimString(actual), actual)
	}
}

func trimString(res string) string {
	var buffer bytes.Buffer
	spaceCount := 0
	for i := 0; i < len(res); i++ {
		c := res[i]
		if c >= 32 {
			if c == 32 {
				spaceCount++
			} else {
				spaceCount = 0
			}
			if spaceCount <= 1 {
				buffer.WriteByte(c)
			}
		}
	}
	return strings.Trim(buffer.String(), " ")
}
