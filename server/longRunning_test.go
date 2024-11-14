package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestInitial(t *testing.T) {
	testPid := 12345670910
	testPidStr := fmt.Sprintf("%d", testPid)
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

	if !lrm.AddLongRunningProcess("lrp1", testPid, false) {
		t.Fatal("Should not find lrp1 and dont add")
	}
	if !lrm.AddLongRunningProcess("lrp1", testPid, true) {
		t.Fatal("Should not find lrp1 and add")
	}
	if lrm.Len() != 1 {
		t.Fatal("Len should be 1")
	}
	if lrm.AddLongRunningProcess("lrp1", testPid, true) {
		t.Fatal("Should find lrp1")
	}
	s := stringMap(lrm)
	if len(s) != 1 {
		t.Fatal("Map should be len 1")
	}
	st := fmt.Sprintf("%s", s)
	AssertContains(t, st, []string{" PID:" + testPidStr, "ExecId:lrp1"})
	lrm.store()

	logMessage := ""
	lrm2, err := NewLongRunningManager("../testdata/exec", "lrm.json", "./longRunCheck.sh", func(s string) {
		logMessage = logMessage + " : " + s
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	AssertContains(t, logMessage, []string{"Loaded:", "PID: " + testPidStr + " NO longer running", "Stored:"})

	s2 := stringMap(lrm2)
	if len(s2) != 0 {
		t.Fatal("Map should be len 0")
	}
	st2 := fmt.Sprintf("%s", s)

	AssertContains(t, st2, []string{" PID:" + testPidStr, "ExecId:lrp"})

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

func stringMap(p *LongRunningManager) map[string]string {
	list := map[string]string{}
	if p.enabled {
		for k, v := range p.longRunning {
			list[k] = fmt.Sprintf("ExecId:%s Run:%s PID:%d", v.ID, v.GetStartTime(), v.PID)
		}
	}
	return list
}
