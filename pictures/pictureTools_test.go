package pictures

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const tdFileJson = "td1.json"
const originals = "../testdata"
const x1DataFileName = "xxx-1.log"
const x2DataFileName = "xxx-2.log"

var td = []byte{0xff, 0x8, 0xff, 0x4, 0xaf, 0xc6, 0x45, 0x78}
var ext = []string{"json", "LOG", "KeepMe"}

func TestPictureScan(t *testing.T) {
	dirDataScanFile, _ := filepath.Abs(filepath.Join(originals, DirDataScanFileName))
	x1DataFile, _ := filepath.Abs(filepath.Join(originals, x1DataFileName))
	x2DataFile, _ := filepath.Abs(filepath.Join(originals, x2DataFileName))

	_, err := ScanDirectory("../tostdata", ext, DirDataScanFileName)
	AssertErrContains(t, "TestPictureScan 1", err, []string{"no such file or directory"})
	_, err = ScanDirectory("../testdata/favicon.ico", ext, DirDataScanFileName)
	AssertErrContains(t, "TestPictureScan 2", err, []string{"is not a directory"})

	removeDataFile(t, dirDataScanFile)
	removeDataFile(t, x2DataFile)
	removeDataFile(t, x1DataFile)
	defer removeDataFile(t, dirDataScanFile)
	defer removeDataFile(t, x2DataFile)
	defer removeDataFile(t, x1DataFile)

	createDataFile(t, td, x1DataFile)
	_, referenceCount, _ := createScanData(originals, ext, DirDataScanFileName)

	// Initial scan crerates the dta file.
	// Current size is 65 with x1DataFile added
	// The data file is saved for next time
	sd1, err := ScanDirectory(originals, ext, DirDataScanFileName)
	if err != nil {
		t.Fatalf("ScanDirectory 1 %v", err)
	}

	// sd1.ScanState.VisitEachFile(func(pp *PicPath, s string) bool {
	// 	fmt.Printf("File:%s\n", s)
	// 	return true
	// })

	asserrtExpected(t, "Scan 1", sd1, 0, referenceCount, 0, 0, "", "")
	assertDiff(t, "ListNewAddDel 1", sd1, "NEW:xxx-1.log")
	sd1.Commit(true)
	// Second scan should read datafile in to OldState
	// New State is the new scan.
	// Should be nothing to do!
	sd2, err := ScanDirectory(originals, ext, DirDataScanFileName)
	if err != nil {
		t.Fatalf("ScanDirectory 2 %v", err)
	}
	asserrtExpected(t, "Scan 2", sd2, referenceCount, referenceCount, 0, 0, "", "")
	assertDiff(t, "ListNewAddDel 2", sd2, "!")

	// remove a file
	removeDataFile(t, x1DataFile)

	sd3, err := ScanDirectory(originals, ext, DirDataScanFileName)
	if err != nil {
		t.Fatalf("ScanDirectory 3 %v", err)
	}
	asserrtExpected(t, "Scan 3", sd3, referenceCount, referenceCount-1, 0, 1, "", x1DataFileName)
	assertContains(t, "Scan 3, OldState", sd3.DataFileState, x1DataFileName)
	assertNotContains(t, "Scan 3, NewState", sd3.ScanState, x1DataFileName, false)
	assertDiff(t, "ListNewAddDel 3", sd3, "DEL:xxx-1.log")

	err = sd3.Commit(true)
	if err != nil {
		t.Fatalf("ScanDirectory 3 %v", err)
	}

	sd4, err := ScanDirectory(originals, ext, DirDataScanFileName)
	if err != nil {
		t.Fatalf("ScanDirectory 4 %v", err)
	}
	asserrtExpected(t, "Scan 4", sd4, referenceCount-1, referenceCount-1, 0, 0, "", "")
	assertNotContains(t, "Scan 4, OldState", sd4.DataFileState, x1DataFileName, false)
	assertNotContains(t, "Scan 4, NewState", sd4.ScanState, x1DataFileName, false)
	assertDiff(t, "ListNewAddDel 4", sd4, "!")

	createDataFile(t, td, x2DataFile)

	sd5, err := ScanDirectory(originals, ext, DirDataScanFileName)
	if err != nil {
		t.Fatalf("ScanDirectory 3 %v", err)
	}
	asserrtExpected(t, "Scan 5", sd5, referenceCount-1, referenceCount, 1, 0, x2DataFileName, "")
	assertNotContains(t, "Scan 5, OldState", sd5.DataFileState, x2DataFileName, false)
	assertContains(t, "Scan 5, NewState", sd5.ScanState, x2DataFileName)
	assertNotContains(t, "Scan 5, OldState", sd5.DataFileState, x1DataFileName, false)
	assertNotContains(t, "Scan 5, NewState", sd5.ScanState, x1DataFileName, false)
	assertDiff(t, "ListNewAddDel 5", sd5, "ADD:xxx-2.log")

	err = sd5.Commit(true)
	if err != nil {
		t.Fatalf("ScanDirectory 5 %v", err)
	}

	sd6, err := ScanDirectory(originals, ext, DirDataScanFileName)
	if err != nil {
		t.Fatalf("ScanDirectory 6 %v", err)
	}
	asserrtExpected(t, "Scan 6", sd6, referenceCount, referenceCount, 0, 0, "", "")
	assertContains(t, "Scan 6, OldState", sd6.DataFileState, x2DataFileName)
	assertContains(t, "Scan 6, NewState", sd6.ScanState, x2DataFileName)
	assertNotContains(t, "Scan 6, OldState", sd6.DataFileState, x1DataFileName, false)
	assertNotContains(t, "Scan 6, NewState", sd6.ScanState, x1DataFileName, false)
	assertDiff(t, "ListNewAddDel 5", sd6, "!")

}

func assertContains(t *testing.T, info string, state *PicDir, file string) {
	if state.FindFile(file) != nil {
		return
	}
	t.Fatalf("ScanDirectory (%s). File %s was not found", info, file)
}

func assertDiff(t *testing.T, info string, sd *ScannedData, contains string) {
	var buff bytes.Buffer
	sd.ListNewAddDel(func(fct FileChangeType, s string) {
		switch fct {
		case FileAdd:
			buff.WriteString(fmt.Sprintf("ADD:%s", s))
		case FileNew:
			buff.WriteString(fmt.Sprintf("NEW:%s", s))
		case FileDel:
			buff.WriteString(fmt.Sprintf("DEL:%s", s))
		}
		buff.WriteString("\n")
	})
	if contains == "!" {
		if strings.TrimSpace(buff.String()) != "" {
			t.Fatalf("ScanDirectory (%s). List Shoulkd be empty\nActual:'%s'", info, buff.String())
		}
		return
	}
	if strings.Contains(buff.String(), contains) {
		return
	}
	t.Fatalf("ScanDirectory (%s). List does not contain'%s'\nActual:'%s'", info, contains, buff.String())
}

func assertNotContains(t *testing.T, info string, state *PicDir, file string, echo bool) {
	if echo {
		s, _ := state.toJson(true)
		fmt.Println(string(s))
	}
	if state.FindFile(file) == nil {
		return
	}
	t.Fatalf("ScanDirectory (%s). File %s was found", info, file)
}

func asserrtExpected(t *testing.T, info string, sd *ScannedData, oldCount, newCount, addedCount, deletedCount int, addedFile string, deletedFile string) {
	if sd.DataFileStateCount != oldCount && oldCount >= 0 {
		t.Fatalf("ScanDirectory (%s) OldStateCount is %d. Should be %d", info, sd.DataFileStateCount, oldCount)
	}
	sc := countScannedFiles(sd.DataFileState)
	if sd.DataFileStateCount != sc && oldCount >= 0 {
		t.Fatalf("ScanDirectory (%s) OldStateCount is %d does not equal actual OldStateCount count %d", info, sd.DataFileStateCount, sc)
	}

	if sd.ScanStateCount != newCount && newCount >= 0 {
		t.Fatalf("ScanDirectory (%s) NewStateCount is %d. Should be %d", info, sd.ScanStateCount, newCount)
	}
	sc = countScannedFiles(sd.ScanState)
	if sd.ScanStateCount != sc && newCount >= 0 {
		t.Fatalf("ScanDirectory (%s) NewStateCount is %d does not equal actual NewStateCount count %d", info, sd.ScanStateCount, sc)
	}

	if sd.FilesDeletedCount != deletedCount && deletedCount >= 0 {
		t.Fatalf("ScanDirectory (%s) FilesDeletedCount is %d. Should be %d", info, sd.FilesDeletedCount, deletedCount)
	}
	sc = countScannedFiles(sd.FilesDeleted)
	if sd.FilesDeletedCount != sc && deletedCount >= 0 {
		t.Fatalf("ScanDirectory (%s) FilesDeletedCount is %d does not equal actual FilesDeletedCount count %d", info, sd.FilesDeletedCount, sc)
	}

	if sd.FilesAddedCount != addedCount && addedCount >= 0 {
		t.Fatalf("ScanDirectory (%s) FilesAddedCount is %d. Should be %d", info, sd.FilesAddedCount, addedCount)
	}
	sc = countScannedFiles(sd.FilesAdded)
	if sd.FilesAddedCount != sc && addedCount >= 0 {
		t.Fatalf("ScanDirectory (%s) FilesAddedCount is %d does not equal actual FilesAddedCount count %d", info, sd.FilesAddedCount, sc)
	}

	if sd.FilesDeleted != nil {
		count := 0
		sd.FilesDeleted.VisitEachFile(func(pp *PicPath, s string) bool {
			if s != deletedFile {
				t.Fatalf("ScanDirectory (%s) File deleted is %s. Should be %s", info, s, deletedFile)
			}
			count++
			return true
		})
		if deletedFile == "" {
			if count > 0 {
				t.Fatalf("ScanDirectory (%s) File deleted should not be found", info)
			}
		} else {
			if count == 0 {
				t.Fatalf("ScanDirectory (%s) File deleted '%s' should be found", info, deletedFile)
			}
		}
	}

	if sd.FilesAdded != nil {
		count := 0
		sd.FilesAdded.VisitEachFile(func(pp *PicPath, s string) bool {
			if s != addedFile {
				t.Fatalf("ScanDirectory (%s) File added is %s. Should be %s", info, s, addedFile)
			}
			count++
			return true
		})
		if addedFile == "" {
			if count > 0 {
				t.Fatalf("ScanDirectory (%s) File added should not be found", info)
			}
		} else {
			if count == 0 {
				t.Fatalf("ScanDirectory (%s) File added '%s' should be found", info, addedFile)
			}
		}
	}

}

func countScannedFiles(state *PicDir) int {
	count := 0
	if state != nil {
		state.VisitEachFile(func(pp *PicPath, s string) bool {
			count++
			return true
		})
	}
	return count
}

func TestPictureInAnotB(t *testing.T) {
	AA, err := WalkDir(originals, func(p string, n string) bool {
		return !strings.Contains(n, ".json") && !strings.Contains(n, ".log")
	})
	if err != nil {
		t.Fatalf("Failed to walk %v", err)
	}

	err = AA.save(tdFileJson, false)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(t, tdFileJson)

	BB, _, err := newPicDir("Root").load(tdFileJson)
	if err != nil {
		t.Fatal(err)
	}

	inAnotB(AA, BB, func(pp *PicPath) {
		t.Fatalf("There should not be differences %s", pp)
	})

	pFromB := removeFileFromPic(t, BB, "admin/diskSize.sh")
	notInB := false
	inAnotB(AA, BB, func(pp *PicPath) {
		if pp.Last() == pFromB.Last() {
			notInB = true
		}
	})
	if !notInB {
		t.Fatalf("InAnotB should report file %s is not in B", pFromB)
	}

	inAnotB(BB, AA, func(pp *PicPath) {
		t.Fatalf("InAnotB should NOT report file %s is not in A", pFromB)
	})

	pFromA := removeFileFromPic(t, AA, "bob/b-pics/favicon.ico")
	notInA := false
	inAnotB(BB, AA, func(pp *PicPath) {
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

	err = l.save(tdFileJson, false)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(t, tdFileJson)

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

	ll, _, err := newPicDir("Root").load(tdFileJson)

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

func removeDataFile(t *testing.T, fil string) {
	_, err := os.Stat(fil)
	if err != nil {
		return
	}
	err = os.Remove(fil)
	if err != nil {
		t.Fatalf("File %s was not deleted. Error:%s", fil, err.Error())
	}
	_, err = os.Stat(fil)
	for err == nil {
		_, err = os.Stat(fil)
	}

}

func createDataFile(t *testing.T, data []byte, fil string) {
	err := os.WriteFile(fil, data, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(fil)
	for err != nil {
		_, err = os.Stat(fil)
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
