package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestInitial(t *testing.T) {
	file := filepath.Join("../testdata/admin", "lrm.json")
	os.Remove(file)
	defer os.Remove(file)

	lrm, err := NewLongRunningManager("../testdata/admin", "lrm.json", "./checkLrp.sh", nil)
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

	if !lrm.AddLongRunningProcess("adnin", "lrp1", 123, false) {
		t.Fatal("Should not find lrp1 and dont add")
	}
	if !lrm.AddLongRunningProcess("adnin", "lrp1", 123, true) {
		t.Fatal("Should not find lrp1 and add")
	}
	if lrm.Len() != 1 {
		t.Fatal("Len should be 1")
	}
	if lrm.AddLongRunningProcess("adnin", "lrp1", 123, true) {
		t.Fatal("Should find lrp1")
	}
	s := lrm.LongRunningMap()
	if len(s) != 1 {
		t.Fatal("Map should be len 1")
	}
	st := fmt.Sprintf("%s", s)
	AssertContains(t, st, []string{" PID:123", "ExecId:lrp1", "User:adnin"})
	lrm.store()

	lrm2, err := NewLongRunningManager("../testdata/admin", "lrm.json", "./checkLrp.sh", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	lrm2.load()

	s2 := lrm2.LongRunningMap()
	if len(s2) != 2 {
		t.Fatal("Map should be len 2")
	}
	_, ok := s2["error"]
	if !ok {
		t.Fatal("Map should have an error entry")
	}
	st2 := fmt.Sprintf("%s", s)

	AssertContains(t, st2, []string{" PID:123", "ExecId:lrp", "User:adnin"})

	if !lrm2.AddLongRunningProcess("bob", "lrp2", 999, false) {
		t.Fatal("Should not find lrp1 and dont add")
	}
	if !lrm2.AddLongRunningProcess("bob", "lrp2", 999, true) {
		t.Fatal("Should not find lrp2 and add")
	}
	if lrm2.Len() != 2 {
		t.Fatal("Len should be 2")
	}
	if lrm2.AddLongRunningProcess("bob", "lrp2", 999, true) {
		t.Fatal("Should find lrp2")
	}

	lrm2.store()
	lrm.load()

	st = fmt.Sprintf("%s", lrm.LongRunningMap())
	lrm2Map := lrm2.LongRunningMap()
	delete(lrm2Map, "error") // Purge the error message so they can match
	st2 = fmt.Sprintf("%s", lrm2Map)

	if st != st2 {
		t.Fatal("Should be the same")
	}

}
