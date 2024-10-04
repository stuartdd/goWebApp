package main

import (
	"os"
	"testing"
)

func TestUfsFileData(t *testing.T) {
	RunTextToJson(readContent(t, "ufsConfig.json"), "ufsConfig.json")
}
func TestDsFileData(t *testing.T) {
	RunTextToJson(readContent(t, "ufsConfig.json"), "ufsConfig.json")
}

func TestContains(t *testing.T) {
	if !contains("Abc123", []string{"c12", "123", "abc"}) {
		t.Fatalf("FAIL:Does not contaimn any!")
	}
	if !contains("Abc123", []string{"c1x2", "123", "abc"}) {
		t.Fatalf("FAIL:Does not contaimn any!")
	}
	if !contains("Abc123", []string{"c1x2", "1xx23", "Abc"}) {
		t.Fatalf("FAIL:Does not contaimn any!")
	}
	if contains("Abc123", []string{"c1x2", "1xx23", "abxxc"}) {
		t.Fatalf("FAIL:Should not contaimn any!")
	}
}

func assertIntEqual(t *testing.T, info string, exp, act int) {
	if exp != act {
		t.Fatalf("FAIL:assertIntEqual: Expected(%d) Actual(%d)", exp, act)
	}
}

func readContent(t *testing.T, filename string) []byte {
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read config: %s\n", filename)
	}
	return content
}