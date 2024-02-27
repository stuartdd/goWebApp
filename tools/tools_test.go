package tools

import (
	"fmt"
	"html"
	"testing"
)

func TestUrlRequestParamsMap(t *testing.T) {
	p1 := NewUrlRequestParts("/a/b/*/*/c/*")
	l1 := NewUrlRequestParts("/a/b/1/2/c/3").UrlParamMap(p1)
	AssertEquals(t, "MX", fmt.Sprintf("%s", l1), "map[b:1 c:3 p3:2]")
	p1 = NewUrlRequestParts("/a/b/1/2/c/3")
	l1 = NewUrlRequestParts("/a/b/1/2/c/3").UrlParamMap(p1)
	AssertEquals(t, "MX", fmt.Sprintf("%s", l1), "map[]")
	p1 = NewUrlRequestParts("/*/b/1/2/c/3")
	l1 = NewUrlRequestParts("/a/b/1/2/c/3").UrlParamMap(p1)
	AssertEquals(t, "MX", fmt.Sprintf("%s", l1), "map[p0:a]")
}

func TestUrlRequestParamsList(t *testing.T) {
	p1 := NewUrlRequestParts("/a/b/*/*/c/*")
	l1 := NewUrlRequestParts("/a/b/1/2/c/3").UrlParamList(p1)
	AssertEquals(t, "MX", fmt.Sprintf("%s", l1), "[1 2 3]")
}

func TestUrlRequestParts(t *testing.T) {
	AssertEquals(t, "M1", NewUrlRequestParts(html.EscapeString("/A/B/C")).ToString(), "/A/B/C")
	AssertEquals(t, "M2", NewUrlRequestParts(html.EscapeString("/")).ToString(), "/")
	AssertEquals(t, "M3", NewUrlRequestParts(html.EscapeString("")).ToString(), "")
	AssertEquals(t, "M4", NewUrlRequestParts(html.EscapeString("A")).ToString(), "A")
	AssertEquals(t, "M4", NewUrlRequestParts(html.EscapeString("A/")).ToString(), "A/")

	t1 := NewUrlRequestParts("/A/B/C")
	t2 := NewUrlRequestParts("/A/*/C")

	AssertTrue(t, "T1", NewUrlRequestParts(html.EscapeString("/A/B/C")).Match(t1))
	AssertTrue(t, "T2", NewUrlRequestParts(html.EscapeString("/A/B/C")).Match(t2))
	AssertTrue(t, "T3", NewUrlRequestParts(html.EscapeString("/A/B/C")).MatchUrl("/A/B/C"))
	AssertTrue(t, "T4", NewUrlRequestParts(html.EscapeString("/A/B/C")).MatchUrl("/A/*/C"))
	AssertTrue(t, "T5", NewUrlRequestParts(html.EscapeString("/A/B/C")).MatchUrl("/A/*/*"))
	AssertTrue(t, "T6", NewUrlRequestParts(html.EscapeString("A/B/C")).MatchUrl("A/*/*"))
	AssertTrue(t, "T7", NewUrlRequestParts(html.EscapeString("A/B/C")).MatchUrl("A/B/C"))

	f1 := NewUrlRequestParts("/A/B")
	f2 := NewUrlRequestParts("/A/*")
	f3 := NewUrlRequestParts("/*/*")
	f4 := NewUrlRequestParts("/A/*/Z")

	AssertFalse(t, "F1", NewUrlRequestParts(html.EscapeString("/A/B/C")).Match(f1))
	AssertFalse(t, "F2", NewUrlRequestParts(html.EscapeString("/A/B/C")).Match(f2))
	AssertFalse(t, "F3", NewUrlRequestParts(html.EscapeString("/A/B/C")).Match(f3))
	AssertFalse(t, "F4", NewUrlRequestParts(html.EscapeString("/A/B/C")).Match(f4))
	AssertFalse(t, "F5", NewUrlRequestParts(html.EscapeString("/A/B/C")).MatchUrl("A/B"))
	AssertFalse(t, "F6", NewUrlRequestParts(html.EscapeString("/A/B/C")).MatchUrl("A/*/Z"))
	AssertFalse(t, "F7", NewUrlRequestParts(html.EscapeString("/A/B/C")).MatchUrl("A/B/Z"))
	AssertFalse(t, "F8", NewUrlRequestParts(html.EscapeString("/A/B/C")).MatchUrl("A/*"))
	AssertFalse(t, "F9", NewUrlRequestParts(html.EscapeString("/A/B/C")).MatchUrl("A"))

	f11 := NewUrlRequestParts("A/*/Z")

	AssertFalse(t, "F10", NewUrlRequestParts(html.EscapeString("A/B/C")).Match(t1))
	AssertFalse(t, "F11", NewUrlRequestParts(html.EscapeString("A/B/C")).Match(t2))
	AssertFalse(t, "F12", NewUrlRequestParts(html.EscapeString("A/B/C")).Match(f11))

	p1 := NewUrlRequestParts("A/*/Z").WithReqType("POST")
	p2 := NewUrlRequestParts("A/B/Z").WithReqType("PUT")

	AssertFalse(t, "P1", p2.Match(p1))

}

func AssertNil(t *testing.T, message string, err error) {
	if err != nil {
		t.Fatalf("%s.\nExpected:Nil\nActual:  %s", message, err)
	}
}

func AssertErr(t *testing.T, message string, err error, expected string) {
	if err.Error() != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s", message, expected, err)
	}
}

func AssertEquals(t *testing.T, message, actual, expected string) {
	if actual != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s", message, expected, actual)
	}
}

func AssertTrue(t *testing.T, message string, actual bool) {
	if actual == false {
		t.Fatalf("%s.\nExpected:true\nActual:  %t", message, actual)
	}
}
func AssertFalse(t *testing.T, message string, actual bool) {
	if actual == true {
		t.Fatalf("%s.\nExpected:false\nActual:  %t", message, actual)
	}
}
