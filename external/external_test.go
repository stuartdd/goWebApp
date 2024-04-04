package main

import (
	"testing"
)

func TestUfsFileData(t *testing.T) {
	RunMain("ufsConfig.json")
}
func TestDsFileData(t *testing.T) {
	RunMain("dsConfig.json")
}
