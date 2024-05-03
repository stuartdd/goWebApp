package image

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

var td = []byte{0xff, 0x8, 0xff, 0x4, 0xaf, 0xc6, 0x45, 0x78}

func TestWalker(t *testing.T) {
	m := map[string]string{}

	c := 0
	l, err := WalkDir("/home/stuart/git/originals", func(p string, n string) bool {
		_, ok := m[p]
		if ok {
			t.Fatalf("Duplicate of walk! %s", p)
		}
		m[p] = "."
		c++
		return strings.HasSuffix(n, ".jpg") || strings.HasSuffix(n, ".JPG")
	})

	if err != nil {
		t.Fatalf("Failed to walk %v", err)
	}
	fmt.Println(c)

	const tdFileJson = "td1.json"
	j, _ := json.Marshal(l)
	createDataFile(t, j, tdFileJson)
	defer removeDataFile(t, tdFileJson)

	count := 0
	l.VisitEachFile(func(p *PicPath, s string) {
		fn := fmt.Sprintf("/home/stuart/git/originals/%s%s", p, s)
		_, err = os.Stat(fn)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := m[fn]
		if !ok {
			t.Fatalf("Node is NOT in the map! %s", fn)
		}
		count++
	})
	if count != c {
		t.Fatalf("Number of nodes added (%d) != nodes visited (%d)", c, count)
	}
	fmt.Println(count)

	dd, err := os.ReadFile(tdFileJson)
	if err != nil {
		t.Fatal(err)
	}
	ll := newPicDir("Root")
	err = json.Unmarshal(dd, ll)
	if err != nil {
		t.Fatal(err)
	}

	count = 0
	ll.VisitEachFile(func(p *PicPath, s string) {
		fn := fmt.Sprintf("/home/stuart/git/originals/%s%s", p, s)
		_, err = os.Stat(fn)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := m[fn]
		if !ok {
			t.Fatalf("Node is NOT in the map! %s", fn)
		}
		count++
	})
	if count != c {
		t.Fatalf("Number of nodes added (%d) != nodes visited (%d)", c, count)
	}

	fmt.Println(count)
}

func TestWalkerInt(t *testing.T) {
	walker := NewWalker(&td, uint32(len(td)))
	walker.SetLittleE(false)

	if walker.BytesToUint(walker.Bytes(1)) != 255 {
		t.Fatal("1st byte != 255")
	}
	w1 := fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "8ff" {
		t.Fatal("int word as hex != 8ff")
	}
	w1 = fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "4af" {
		t.Fatal("int word as hex != 4af")
	}
	w1 = fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "c645" {
		t.Fatal("int word as hex != c645")
	}
	walker.Reset()
	walker.SetLittleE(true)
	w1 = fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "8ff" {
		t.Fatal("int word-hi as hex != 8ff")
	}
	w1 = fmt.Sprintf("%x", walker.Pos(6).BytesToUint(walker.Bytes(1)))
	if w1 != "45" {
		t.Fatal("int word-hi as hex != 45")
	}

	walker.Reset()
	walker.SetLittleE(false)
	w1 = fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "ff08" {
		t.Fatal("int word-hi as hex != ff08")
	}
	w1 = fmt.Sprintf("%x", walker.Pos(6).BytesToUint(walker.Bytes(1)))
	if w1 != "45" {
		t.Fatal("int word-hi as hex != 45")
	}

	walker.Reset()
	w1 = fmt.Sprintf("%x", walker.Pos(0).bytesToUintBE(walker.Bytes(4)))
	if w1 != "ff08ff04" {
		t.Fatal("int word-hi as hex != ff08ff04")
	}
	w1 = fmt.Sprintf("%x", walker.Pos(0).bytesToUintLE(walker.Bytes(4)))
	if w1 != "4ff08ff" {
		t.Fatal("int word-hi as hex != 4ff08ff")
	}

	defer func() {
		if r := recover(); r != nil {
			if r != "Pos was set past end: Max=7 Requested=8" {
				t.Fatalf("SetPos(8) Did not panic with correct message")
			}
		}
	}()
	walker.Pos(8)
	t.Fatal("Should not get here")

}

func TestHex(t *testing.T) {
	b := []byte{}
	for i := 0; i < 256; i++ {
		b = append(b, byte(i))
	}
	walker := NewWalker(&b, uint32(len(b)))
	res := walker.Hex(walker.Bytes(256), "")
	for i := 0; i < 256; i++ {
		b := res[i*2 : (i*2)+2]
		d, err := strconv.ParseInt("0"+b, 16, 16)
		if err != nil {
			t.Fatalf("Conversion to bin failed for %s", b)
		}
		if i != int(d) {
			t.Fatalf("Conversion to hex failed for %s --> %d at %d", b, d, i)
		}
	}

}

func TestWalkerHex(t *testing.T) {
	walker := NewWalker(&td, uint32(len(td)))
	if walker.Hex(walker.Bytes(1), "0x") != "0xFF" {
		t.Fatal("1st byte != ff")
	}
	if walker.Hex(walker.Bytes(1), "0x") != "0x08" {
		t.Fatal("byte != 08")
	}
	if walker.Hex(walker.Bytes(2), "0x") != "0xFF04" {
		t.Fatal("word != ff04")
	}
	if walker.Hex(walker.Bytes(1), "") != "AF" {
		t.Fatal("byte != af")
	}
	if walker.Hex(walker.Bytes(2), "") != "C645" {
		t.Fatal("word != c645")
	}
	defer func() {
		if r := recover(); r != nil {
			if r != "Advanced past end: Max=7 Requested=8" {
				t.Fatalf("Hex(2) Did not panic with correct message")
			}
		}
	}()
	walker.Hex(walker.Bytes(2), "") // Will take it past end
}

// : FF D8 FF E1 8E 1F 45 78 69 66 00 00 4D 4D 00 2A 00 00 00 08 00 09 88 25 00 04 00 00 00 01 00 00 03 F9 01 10 00 02 00 00 00 08 00 00 00 7A 02 13 00 03 065496 065505 036383 017784 026982 000000 019789 000042 000000 000008 000009 034853 000004 000000 000001 000000 001017 000272 000002 000000 000008 000000 000122 000531 000003
// : ÿ  Ø  ÿ  á      F  x  i  f      M  M    *                %                    ù                        z          FFD8   FFE1   8E1F   4578   6966   0000   4D4D   002A   0000   0008   0009   8825   0004   0000   0001   0000   03F9   0110   0002   0000   0008   0000   007A   0213   0003
var td1 = []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x8E, 0x1F, 0x46, 0x78, 0x69, 0x66, 0x00, 0x00, 0x4D, 0x4D, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x08, 0x00, 0x09, 0x88, 0x25, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x03, 0xF9, 0x01, 0x10, 0x00, 0x02, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x7A, 0x02, 0x13, 0x00, 0x03}
var td2 = []byte{0xFF, 0xD0, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x84, 0x00, 0x06, 0x04, 0x04, 0x05, 0x04, 0x03, 0x06, 0x05, 0x04, 0x05, 0x06, 0x06, 0x06, 0x07, 0x09, 0x0F, 0x09, 0x09, 0x08, 0x08, 0x09, 0x12, 0x0D, 0x0D, 0x0A}
var td3 = []byte{0xFF, 0xD8, 0xFF, 0xEF, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x84, 0x00, 0x06, 0x04, 0x04, 0x05, 0x04, 0x03, 0x06, 0x05, 0x04, 0x05, 0x06, 0x06, 0x06, 0x07, 0x09, 0x0F, 0x09, 0x09, 0x08, 0x08, 0x09, 0x12, 0x0D, 0x0D, 0x0A}
var td4 = []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x84, 0x00, 0x06, 0x04, 0x04, 0x05, 0x04, 0x03, 0x06, 0x05, 0x04, 0x05, 0x06, 0x06, 0x06, 0x07, 0x09, 0x0F, 0x09, 0x09, 0x08, 0x08, 0x09, 0x12, 0x0D, 0x0D, 0x0A}

const tdFile = "td1.jpg"

func TestBadExifMarker(t *testing.T) {
	createDataFile(t, td1, tdFile)
	defer removeDataFile(t, tdFile)
	_, err := GetImage(tdFile, true, true, true, nil)
	if err.Error() != "PANIC:Jpeg 'Exif' data marker is missing (Offset 6) found Fxif" {
		t.Fatalf("TD1 %s", err.Error())
	}
}

func TestBadSOI(t *testing.T) {
	createDataFile(t, td2, tdFile)
	defer removeDataFile(t, tdFile)
	_, err := GetImage(tdFile, true, true, true, nil)
	if err.Error() != "PANIC:Jpeg marker 'FFD8' is missing (Offset 0) found FFD0" {
		t.Fatalf("BadSOI %s", err.Error())
	}
}

func TestBadA001(t *testing.T) {
	createDataFile(t, td3, tdFile)
	defer removeDataFile(t, tdFile)
	_, err := GetImage(tdFile, true, true, true, nil)
	if err.Error() != "PANIC:Jpeg APP1 marker 'FFE1' is missing (Offset 2) found FFEF" {
		t.Fatalf("BadA001 %s", err.Error())
	}
}

func TestBadJpg(t *testing.T) {
	createDataFile(t, td4, tdFile)
	defer removeDataFile(t, tdFile)
	_, err := GetImage(tdFile, true, true, true, nil)
	if err.Error() != "PANIC:Jpeg 'Exif' data marker is missing (Offset 6) found JFIF" {
		t.Fatalf("%s", err.Error())
	}
}

func createDataFile(t *testing.T, data []byte, fil string) {
	err := os.WriteFile(fil, data, 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func removeDataFile(t *testing.T, fil string) {
	err := os.Remove(fil)
	if err != nil {
		t.Fatal(err)
	}
}

const golden string = `DateTimeDigitized=2016:11:06 11:29:18
DateTimeOriginal=2016:11:06 11:29:18
`

func TestImage01(t *testing.T) {
	im, err := GetImage("../testdata/test_data_01.ti", false, false, true, func(ifd *IFDEntry, w *Walker) bool {
		return strings.Contains(ifd.TagData.Name, "Date")
	})
	if err != nil {
		t.Fatal(err)
	}
	if im.Output() != golden {
		t.Fatalf("Output is not same a golden:\n'%s'\n'%s'", im.Output(), golden)
	}
}

func TestImage02(t *testing.T) {
	_, err := GetImage("../testdata/test_data_02.ti", false, false, true, func(ifd *IFDEntry, w *Walker) bool {
		return strings.Contains(ifd.TagData.Name, "Date")
	})
	if err.Error() != "PANIC:Jpeg APP1 marker 'FFE1' is missing (Offset 2) found FFE0" {
		t.Fatal(err)
	}
}

/*

0000 FFD8 				SOI Start of Image
0002 FFE1 				APP 1 Marker {EXIF}
0004 SSSS 				Size of APP1 (44998 AF C6)
0006 45 78 69 66 00 00 	'Exif00' OfsExifHeader
0012 49 49 2a 00 		'II*0' or 'MM*0' OfsEndianDef
0016 08 00 00 00 		Offset to TIFF (Main Image) OfsMainImageOffset
XXXX 0D 00    			Number of entries (12 bytes each) * BASE

XXXX FFD9 EOI End of Image

Value 				1 				2 				3 				4 				5 					6
Format 				unsigned byte 	ascii strings 	unsigned short 	unsigned long 	unsigned rational 	signed byte
Bytes/component 	1 				1 				2 				4 				8 					1

Value 				7 				8 				9 				10 				11 					12
Format 				undefined 		signed short 	signed long 	signed rational single float 		double float
Bytes/component 	1 				2 				4 				8 				4 					8

magick testImage.jpg -print "%[EXIF:*]\n" info:
*/
