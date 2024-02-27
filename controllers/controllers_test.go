package controllers

import (
	"testing"
)

func TestController(t *testing.T) {
}

func AssertEquals(t *testing.T, message, actual, expected string) {
	if actual != expected {
		t.Fatalf("%s.\nExpected:%s\nActual:  %s", message, expected, actual)
	}
}
