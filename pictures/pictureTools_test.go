package pictures

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const tdFileJson = "td1.json"
const originals = "../testdata"
const x1DataFileName = "xxx-1.log"

var td = []byte{0xff, 0x8, 0xff, 0x4, 0xaf, 0xc6, 0x45, 0x78}

func TestPictureScan(t *testing.T) {
	_, err := ScanDirectory("../tostdata", []string{})
	AssertErrContains(t, "TestPictureScan 1", err, []string{"no such file or directory"})
	_, err = ScanDirectory("../testdata/favicon.ico", []string{})
	AssertErrContains(t, "TestPictureScan 2", err, []string{"is not a directory"})

	removeDataFile(filepath.Join(originals, "dirScanData.json"))
	x1DataFile, _ := filepath.Abs(filepath.Join(originals, x1DataFileName))

	createDataFile(t, td, x1DataFile)
	defer removeDataFile(x1DataFile)

	// Initial scan crerates the dta file.
	// Current size is 65 with x1DataFile added
	// The data file is saved for next time
	sd3, err := ScanDirectory(originals, []string{})
	if err != nil {
		t.Fatalf("ScanDirectory 3 %v", err)
	}
	if sd3.OldStateCount != 65 {
		t.Fatalf("ScanDirectory 3 OldStateCount is %d. Should be 65", sd3.OldStateCount)
	}
	if sd3.NewStateCount != 0 {
		t.Fatalf("ScanDirectory 3 NewStateCount is %d. Should be 0", sd3.NewStateCount)
	}
	if sd3.NeedToCreate != nil {
		t.Fatalf("ScanDirectory 3 NeedToCreate Should be nil")
	}
	if sd3.NeedToCreateCount != 0 {
		t.Fatalf("ScanDirectory 3 NeedToCreateCount is %d. Should be 0", sd3.NeedToCreateCount)
	}
	if sd3.NeedToDelete != nil {
		t.Fatalf("ScanDirectory 3 NeedToDelete Should be nil")
	}
	if sd3.NeedToDeleteCount != 0 {
		t.Fatalf("ScanDirectory 3 NeedToDeleteCount is %d. Should be 0", sd3.NeedToDeleteCount)
	}

	// Second scan should read datafile in to OldState
	// New State is the new scan.
	// Should be nothing to do!
	sd4, err := ScanDirectory(originals, []string{})
	if err != nil {
		t.Fatalf("ScanDirectory 4 %v", err)
	}
	if sd4.OldStateCount != 0 {
		t.Fatalf("ScanDirectory 4 OldStateCount is %d. Should be 0", sd3.OldStateCount)
	}
	if sd4.NewStateCount != 65 {
		t.Fatalf("ScanDirectory 4 NewStateCount is %d. Should be 65", sd3.NewStateCount)
	}
	if sd4.NeedToCreate == nil {
		t.Fatalf("ScanDirectory 4 NeedToCreate Should not be nil")
	}
	if sd4.NeedToCreateCount != 0 {
		t.Fatalf("ScanDirectory 4 NeedToCreateCount is %d. Should be 0", sd3.NeedToCreateCount)
	}
	if sd4.OldStateCount != 0 {
		t.Fatalf("ScanDirectory 4 OldStateCount is %d. Should be 0", sd3.OldStateCount)
	}
	if sd4.NeedToDelete == nil {
		t.Fatalf("ScanDirectory 4 NeedToDelete Should not be nil")
	}
	if sd4.NeedToDeleteCount != 0 {
		t.Fatalf("ScanDirectory 4 NeedToDeleteCount is %d. Should be 0", sd3.NeedToDeleteCount)
	}
	if sd3.OldStateCount != sd4.NewStateCount {
		t.Fatalf("ScanDirectory 3 OldStateCount Should equal ScanDirectory 4 NewStateCount")
	}

	// remove a file
	removeDataFile(x1DataFile)

	sd5, err := ScanDirectory(originals, []string{})
	if err != nil {
		t.Fatalf("ScanDirectory 4 %v", err)
	}

	if sd5.NeedToDeleteCount != 1 {
		t.Fatalf("ScanDirectory 5 NeedToDeleteCount is %d. Should be 0", sd5.NeedToDeleteCount)
	}
	sd5.NeedToDelete.VisitEachFile(func(pp *PicPath, s string) bool {
		if s != x1DataFileName {
			t.Fatalf("ScanDirectory 5 File deleted was %s. Should be %s", s, x1DataFileName)
		}
		return true
	})
	sd5.NeedToCreate.VisitEachFile(func(pp *PicPath, s string) bool {
		t.Fatalf("ScanDirectory 5 File created was %s. Should be none", s)
		return true
	})

	if sd5.NeedToCreateCount != 0 {
		t.Fatalf("ScanDirectory 5 NeedToCreateCount is %d. Should be 0", sd5.NeedToCreateCount)
	}

	defer removeDataFile(filepath.Join(originals, "dirScanData.json"))
}

func TestPictureInAnotB(t *testing.T) {
	AA, err := WalkDir(originals, func(p string, n string) bool {
		return !strings.Contains(n, ".json") && !strings.Contains(n, ".log")
	})
	if err != nil {
		t.Fatalf("Failed to walk %v", err)
	}

	err = AA.Save(tdFileJson, false)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(tdFileJson)

	BB, err := newPicDir("Root").Load(tdFileJson)
	if err != nil {
		t.Fatal(err)
	}

	InAnotB(AA, BB, func(pp *PicPath) {
		t.Fatalf("There should not be differences %s", pp)
	})

	pFromB := removeFileFromPic(t, BB, "admin/diskSize.sh")
	notInB := false
	InAnotB(AA, BB, func(pp *PicPath) {
		if pp.Last() == pFromB.Last() {
			notInB = true
		}
	})
	if !notInB {
		t.Fatalf("InAnotB should report file %s is not in B", pFromB)
	}

	InAnotB(BB, AA, func(pp *PicPath) {
		t.Fatalf("InAnotB should NOT report file %s is not in A", pFromB)
	})

	pFromA := removeFileFromPic(t, AA, "bob/b-pics/favicon.ico")
	notInA := false
	InAnotB(BB, AA, func(pp *PicPath) {
		if pp.Last() == pFromA.Last() {
			notInA = true
		}
	})
	if !notInA {
		t.Fatalf("InAnotB should report file %s is not in A", pFromA)
	}
}

func removeFileFromPic(t *testing.T, pic *PicDir, file string) *PicPath {
	path := newPicPathFromFile(file)
	dir, fil := pic.Find(path)
	if dir == nil || fil == nil {
		t.Fatalf("Find should find file %s in pic", path)
	}
	nf := []*PicFile{}
	for _, fn := range dir.Files {
		if fn.N != path.Last() {
			nf = append(nf, fn)
		}
	}
	dir.Files = nf

	_, f1 := pic.Find(path)
	if f1 != nil {
		t.Fatalf("Find should NOT find file %s", path.Last())
	}
	return path
}

func TestPicPath(t *testing.T) {
	pp1 := newPicPathFromFile("A")
	if pp1.Len() != 1 || pp1.paths[0] != "A" {
		t.Fatalf("pp1 newPicPathFromFile 'A' failed")
	}
	pp2 := newPicPathFromFile("A/B")
	if pp2.Len() != 2 || pp2.paths[0] != "A" || pp2.paths[1] != "B" {
		t.Fatalf("pp2 newPicPathFromFile 'A/B' failed")
	}
	pp3 := newPicPathFromFile("A")
	if pp3.Len() != 1 || pp3.paths[0] != "A" || !pp1.Equal(pp3) {
		t.Fatalf("pp1 newPicPathFromFile 'A=A' failed")
	}
	pp4 := newPicPathFromFile("B")
	if pp4.Len() != 1 || pp4.paths[0] != "B" || pp4.Equal(pp3) {
		t.Fatalf("pp1 newPicPathFromFile 'A=B' failed")
	}
	if !newPicPathFromFile("").Equal(newPicPath()) {
		t.Fatalf("pp1 newPicPathFromFile != newPicPath")
	}
	pp3.push("B")
	if !pp2.Equal(pp3) {
		t.Fatalf("pp2 != pp3 after push B")
	}
	pp2.push("X")
	if pp2.Equal(pp3) {
		t.Fatalf("pp3 == pp3 after push X")
	}

	if pp3.String() != "A/B" {
		t.Fatalf("pp3 string expected 'A/B' actual '%s'", pp3.String())
	}
	pp3.pop()
	if pp3.String() != "A" {
		t.Fatalf("pp3 string expected 'A' actual '%s'", pp3.String())
	}
	pp3.pop()
	if pp3.String() != "" {
		t.Fatalf("pp3.pop 1 string expected '' actual '%s'", pp3.String())
	}
	pp3.pop()
	if pp3.String() != "" {
		t.Fatalf("pp3.pop 2 string expected '' actual '%s'", pp3.String())
	}
	pp3.push("X")
	if pp3.String() != "X" {
		t.Fatalf("pp3.push(X) 1 string expected 'X' actual '%s'", pp3.String())
	}
	pp3.push("X")
	if pp3.String() != "X/X" {
		t.Fatalf("pp3.push(X) 2string expected 'X/X' actual '%s'", pp3.String())
	}
}

func TestPictureWalker(t *testing.T) {
	m := map[string]string{}
	c := 0
	l, err := WalkDir(originals, func(p string, n string) bool {
		fn, _ := filepath.Abs(fmt.Sprintf("%s/%s/%s", originals, p, n))
		_, ok := m[fn]
		if ok {
			t.Fatalf("Duplicate of walk! %s", fn)
		}
		m[p] = "."
		if strings.Contains(n, ".json") {
			return false
		}
		c++
		return true
	})

	if err != nil {
		t.Fatalf("Failed to walk %v", err)
	}

	err = l.Save(tdFileJson, false)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(tdFileJson)

	count := 0
	l.VisitEachFile(func(p *PicPath, n string) bool {
		fn, _ := filepath.Abs(fmt.Sprintf("%s/%s/%s", originals, p, n))
		_, err = os.Stat(fn)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := m[fn]
		if !ok {
			t.Fatalf("Node is NOT in the map! %s", fn)
		}
		count++
		return true
	})
	if count != c {
		t.Fatalf("Number of nodes added (%d) != nodes visited (%d)", c, count)
	}

	ll, err := newPicDir("Root").Load(tdFileJson)

	count = 0
	ll.VisitEachFile(func(p *PicPath, s string) bool {
		fn, _ := filepath.Abs(fmt.Sprintf("%s/%s/%s", originals, p, s))
		_, err = os.Stat(fn)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := m[fn]
		if !ok {
			t.Fatalf("Node is NOT in the map! %s", fn)
		}
		count++
		return true
	})

	if count != c {
		t.Fatalf("Number of nodes added (%d) != nodes visited (%d)", c, count)
	}
}

func removeDataFile(fil string) {
	os.Remove(fil)
}

func createDataFile(t *testing.T, data []byte, fil string) {
	err := os.WriteFile(fil, data, 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func AssertContains(t *testing.T, note string, actual string, expectedList []string) {
	for i := 0; i < len(expectedList); i++ {
		expected := expectedList[i]
		if !strings.Contains(actual, expected) {
			t.Fatalf("Value \n%s\nDoes NOT contain '%s'", actual, expected)
		}
	}
}

func AssertErrContains(t *testing.T, note string, actual error, expectedList []string) {
	if actual == nil {
		t.Fatalf("An error was expected containing %s", expectedList)
	}
	for i := 0; i < len(expectedList); i++ {
		expected := expectedList[i]
		if !strings.Contains(actual.Error(), expected) {
			t.Fatalf("Value \n%s\nDoes NOT contain '%s'", actual, expected)
		}
	}
}
