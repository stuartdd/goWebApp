package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestInitial(t *testing.T) {
	file, _ := filepath.Abs(filepath.Join("../testdata/exec", "lrm.json"))
	os.Remove(file)
	defer os.Remove(file)

	lrm, err := NewLongRunningManager("../testdata/exec", "lrm.json", "./longRunCheck.sh", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	if lrm.Len() != 0 {
		t.Fatal("Len should be 0")
	}
	lrm.store()

	lrm.load()
	if lrm.Len() != 0 {
		t.Fatal("Len should be 0")
	}

	if !lrm.AddLongRunningProcess("lrp1", 123, false) {
		t.Fatal("Should not find lrp1 and dont add")
	}
	if !lrm.AddLongRunningProcess("lrp1", 123, true) {
		t.Fatal("Should not find lrp1 and add")
	}
	if lrm.Len() != 1 {
		t.Fatal("Len should be 1")
	}
	if lrm.AddLongRunningProcess("lrp1", 123, true) {
		t.Fatal("Should find lrp1")
	}
	s := lrm.LongRunningMap()
	if len(s) != 1 {
		t.Fatal("Map should be len 1")
	}
	st := fmt.Sprintf("%s", s)
	AssertContains(t, st, []string{" PID:123", "ExecId:lrp1"})
	lrm.store()

	logMessage := ""
	lrm2, err := NewLongRunningManager("../testdata/exec", "lrm.json", "./longRunCheck.sh", func(s string) {
		logMessage = logMessage + " : " + s
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	AssertContains(t, logMessage, []string{"Loaded:", "PID: 123 NO longer running", "Stored:"})

	s2 := lrm2.LongRunningMap()
	if len(s2) != 0 {
		t.Fatal("Map should be len 0")
	}
	st2 := fmt.Sprintf("%s", s)

	AssertContains(t, st2, []string{" PID:123", "ExecId:lrp"})

	if !lrm2.AddLongRunningProcess("lrp2", 999, false) {
		t.Fatal("Should not find lrp1 and dont add")
	}
	if !lrm2.AddLongRunningProcess("lrp2", 999, true) {
		t.Fatal("Should not find lrp2 and add")
	}
	if lrm2.Len() != 1 {
		t.Fatal("Len should be 1")
	}
	if lrm2.AddLongRunningProcess("lrp2", 999, true) {
		t.Fatal("Should find lrp2")
	}
	if lrm2.Len() != 1 {
		t.Fatal("Len should be 1")
	}
}
