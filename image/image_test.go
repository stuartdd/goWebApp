package image

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

const tdFileJson = "td1.json"
const tdFileJpeg = "td1.jpg"

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

	err = l.Save(tdFileJson)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(tdFileJson)

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

	ll, err := newPicDir("Root").Load(tdFileJson)

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
	createDataFile(t, td, tdFileJpeg)
	walker, err := NewWalker(createReader(t, tdFileJpeg), 100)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(tdFileJpeg)

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
	walker.Pos(0)
	walker.SetLittleE(true)
	w1 = fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "8ff" {
		t.Fatal("int word-hi as hex != 8ff")
	}
	w1 = fmt.Sprintf("%x", walker.Pos(6).BytesToUint(walker.Bytes(1)))
	if w1 != "45" {
		t.Fatal("int word-hi as hex != 45")
	}

	walker.Pos(0)
	walker.SetLittleE(false)
	w1 = fmt.Sprintf("%x", walker.BytesToUint(walker.Bytes(2)))
	if w1 != "ff08" {
		t.Fatal("int word-hi as hex != ff08")
	}
	w1 = fmt.Sprintf("%x", walker.Pos(6).BytesToUint(walker.Bytes(1)))
	if w1 != "45" {
		t.Fatal("int word-hi as hex != 45")
	}

	walker.Pos(0)
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
			if r != "Failed to extend buffer by 1. Only able to read 0. Error: EOF" {
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
	createDataFile(t, b, tdFileJpeg)
	walker, err := NewWalker(createReader(t, tdFileJpeg), 100)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(tdFileJpeg)

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
	createDataFile(t, td, tdFileJpeg)
	walker, err := NewWalker(createReader(t, tdFileJpeg), 100)
	if err != nil {
		t.Fatal(err)
	}
	defer removeDataFile(tdFileJpeg)

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
			if r != "Failed to extend buffer by 1. Only able to read 0. Error: EOF" {
				t.Fatalf("Hex(2) Did not panic with correct message")
			}
		}
	}()
	walker.Hex(walker.Bytes(2), "") // Will take it past end
}

// : FF D8 FF E1 8E 1F 45 78 69 66 00 00 4D 4D 00 2A 00 00 00 08 00 09 88 25 00 04 00 00 00 01 00 00 03 F9 01 10 00 02 00 00 00 08 00 00 00 7A 02 13 00 03 065496 065505 036383 017784 026982 000000 019789 000042 000000 000008 000009 034853 000004 000000 000001 000000 001017 000272 000002 000000 000008 000000 000122 000531 000003
// : ÿ  Ø  ÿ  á      F  x  i  f      M  M    *                %                    ù                        z          FFD8   FFE1   8E1F   4578   6966   0000   4D4D   002A   0000   0008   0009   8825   0004   0000   0001   0000   03F9   0110   0002   0000   0008   0000   007A   0213   0003
var td1 = []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x8E, 0x1F, 0x46, 0x78, 0x69, 0x66, 0x00, 0x00, 0x4D, 0x4D, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x08, 0x00, 0x09, 0x88, 0x25, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x03, 0xF9, 0x01, 0x10, 0x00, 0x02, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x7A, 0x02, 0x13, 0x0D, 0x03}
var td2 = []byte{0xFF, 0xD0, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x84, 0x00, 0x06, 0x04, 0x04, 0x05, 0x04, 0x03, 0x06, 0x05, 0x04, 0x05, 0x06, 0x06, 0x06, 0x07, 0x09, 0x0F, 0x09, 0x09, 0x08, 0x08, 0x09, 0x12, 0x0D, 0x0D, 0x0A}
var td3 = []byte{0xFF, 0xD8, 0xFF, 0xEF, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x84, 0x00, 0x06, 0x04, 0x04, 0x05, 0x04, 0x03, 0x06, 0x05, 0x04, 0x05, 0x06, 0x06, 0x06, 0x07, 0x09, 0x0F, 0x09, 0x09, 0x08, 0x08, 0x09, 0x12, 0x0D, 0x0D, 0x0A}
var td4 = []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x84, 0x00, 0x06, 0x04, 0x04, 0x05, 0x04, 0x03, 0x06, 0x05, 0x04, 0x05, 0x06, 0x06, 0x06, 0x07, 0x09, 0x0F, 0x09, 0x09, 0x08, 0x08, 0x09, 0x12, 0x0D, 0x0D, 0x0A}

func TestWalkerRead2(t *testing.T) {
	createDataFile(t, td1, tdFileJpeg)
	w, err := NewWalker(createReader(t, tdFileJpeg), 2)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		b := w.Advance(1)
		if b != uint32(td1[i]) {
			t.Fatalf("Byte should be %d actual %d", td1[i], b)
		}
	}
	w.Pos(17)
	for i := 17; i < 27; i++ {
		b := w.Advance(1)
		if b != uint32(td1[i]) {
			t.Fatalf("Byte should be %d actual %d", td1[i], b)
		}
	}

}
func TestWalkerRead10(t *testing.T) {
	createDataFile(t, td1, tdFileJpeg)
	w, err := NewWalker(createReader(t, tdFileJpeg), 10)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		b := w.Advance(1)
		if b != uint32(td1[i]) {
			t.Fatalf("Byte should be %d actual %d", td1[i], b)
		}
	}
	w.Pos(17)
	for i := 17; i < 27; i++ {
		b := w.Advance(1)
		if b != uint32(td1[i]) {
			t.Fatalf("Byte should be %d actual %d", td1[i], b)
		}
	}
}

func TestWalkerAdvanceOffEnd(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if r != "Failed to extend buffer by 1. Only able to read 0. Error: EOF" {
				t.Fatalf("Advance(1) Did not panic with correct message")
			}
		}
	}()
	createDataFile(t, td1, tdFileJpeg)
	w, err := NewWalker(createReader(t, tdFileJpeg), 10)
	if err != nil {
		t.Fatal(err)
	}
	w.Pos(37)
	for i := 37; i < len(td1); i++ {
		b := w.Advance(1)
		if b != uint32(td1[i]) {
			t.Fatalf("Byte should be %d actual %d", td1[i], b)
		}
	}
	// Following should panic EOF
	w.Advance(1)
}

func TestWalkerLastByted(t *testing.T) {
	createDataFile(t, td1, tdFileJpeg)
	for p := 31; p < 49; p++ {
		w, err := NewWalker(createReader(t, tdFileJpeg), 10)
		if err != nil {
			t.Fatal(err)
		}
		w.Pos(uint32(p))
		b := uint32(0)
		for i := p; i < len(td1); i++ {
			b = w.Advance(1)
			if b != uint32(td1[i]) {
				t.Fatalf("Byte should be %d actual %d", td1[i], b)
			}
		}
		if b != 3 {
			t.Fatalf("Last Byte should be 3 actual %d", b)
		}

	}

}

func TestWalkerPosPastEnd(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if r != "Failed to extend buffer by 4. Only able to read 2." {
				t.Fatalf("w.Pos(51) Did not panic with correct message")
			}
		}
	}()
	createDataFile(t, td1, tdFileJpeg)
	w, err := NewWalker(createReader(t, tdFileJpeg), 10)
	if err != nil {
		t.Fatal(err)
	}
	w.Pos(37)
	// Length of buffer is not 37+10
	//
	// Following should panic EOF. File only has 3 to go we wanted 4
	w.Pos(51)
}

func TestBadExifMarker(t *testing.T) {
	createDataFile(t, td1, tdFileJpeg)
	defer removeDataFile(tdFileJpeg)
	_, err := GetImage(tdFileJpeg, true, true, true, nil)
	if err.Error() != "PANIC:Jpeg 'Exif' data marker is missing (Offset 6) found Fxif" {
		t.Fatalf("TD1 %s", err.Error())
	}
}

func TestBadSOI(t *testing.T) {
	createDataFile(t, td2, tdFileJpeg)
	defer removeDataFile(tdFileJpeg)
	_, err := GetImage(tdFileJpeg, true, true, true, nil)
	if err.Error() != "PANIC:Jpeg marker 'FFD8' is missing (Offset 0) found FFD0" {
		t.Fatalf("BadSOI %s", err.Error())
	}
}

func TestBadA001(t *testing.T) {
	createDataFile(t, td3, tdFileJpeg)
	defer removeDataFile(tdFileJpeg)
	_, err := GetImage(tdFileJpeg, true, true, true, nil)
	if err.Error() != "PANIC:Jpeg APP1 marker 'FFE1' is missing (Offset 2) found FFEF" {
		t.Fatalf("BadA001 %s", err.Error())
	}
}

func TestBadJpg(t *testing.T) {
	createDataFile(t, td4, tdFileJpeg)
	defer removeDataFile(tdFileJpeg)
	_, err := GetImage(tdFileJpeg, true, true, true, nil)
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

func createReader(t *testing.T, fil string) *bufio.Reader {
	f, err := os.Open(fil)
	if err != nil {
		t.Fatal(err)
	}
	return bufio.NewReader(f)
}

func removeDataFile(fil string) {
	os.Remove(fil)
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

func TestImage03(t *testing.T) {
	_, err := GetImage("../testdata/test_data_01.ti", true, false, true, func(i *IFDEntry, w *Walker) bool {
		if i != nil {
			_, ok := sampleFromMagick[i.TagData.Name]
			if !ok {
				t.Fatalf("Tan name [%s] not found in sampleFromMagick", i.TagData.Name)
			}
		}
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
}

var sampleFromMagick = map[string]string{
	"ColorSpace":                        "1",
	"ComponentConfiguration":            "0x01,0x02,0x03,0x00",
	"DateTimeDigitized":                 "2016:11:06 11:29:18",
	"DateTimeOriginal":                  "2016:11:06 11:29:18",
	"DigitalZoomRatio":                  "100/100",
	"ExifOffset":                        "161",
	"ExifVersion":                       "0220",
	"ExifImageHeight":                   "2340",
	"ExifImageWidth":                    "4160",
	"ExposureBiasValue":                 "0/1",
	"ExposureTime":                      "1/24",
	"Flash":                             "0",
	"FlashPixVersion":                   "0100",
	"FNumber":                           "240/100",
	"FocalLength":                       "3970/1000",
	"GPSAltitude":                       "0/1000",
	"GPSAltitudeRef":                    "0",
	"GPSInfo":                           "1017",
	"GPSLatitudeRef":                    "R98",
	"InteroperabilityOffset":            "987",
	"Make":                              "LG Electronics",
	"MeteringMode":                      "2",
	"Model":                             "LG-D855",
	"Orientation":                       "6",
	"PhotographicSensitivity":           "100",
	"PixelXDimension":                   "4160",
	"PixelYDimension":                   "2340",
	"ResolutionUnit":                    "2",
	"thumbnail:InteroperabilityIndex":   "R98",
	"thumbnail:InteroperabilityVersion": "0100",
	"UserComment":                       "    FocusArea=111111111",
	"WhiteBalance":                      "0",
	"XResolution":                       "72/1",
	"YCbCrPositioning":                  "1",
	"YResolution":                       "72/1",
}

/*
ColorSpace=1
ComponentConfiguration=0x01,0x02,0x03,0x00
DateTimeDigitized=2016:11:06 11:29:18
DateTimeOriginal=2016:11:06 11:29:18
DigitalZoomRatio=100/100
ExifImageHeight=2340
ExifImageWidth=4160
ExifVersion=0220
ExposureBiasValue=0/1
ExposureTime=1/24
FNumber=240/100
Flash=0
FlashPixVersion=0100
FocalLength=3970/1000
GPSAltitude=0/1000
GPSAltitudeRef=0
GPSLatitude=0x30,0x31,0x30,0x30
GPSLatitudeRef=R98
ISOSpeedRatings=100
Make=LG Electronics
MeteringMode=2
Model=LG-D855
Orientation=6
ResolutionUnit=2
UserComment=    FocusArea=111111111
WhiteBalance=0
XResolution=72/1
YCbCrPositioning=1
YResolution=72/1

DEBUG:IMG SOI[FFD8]  APP1 Mark[FFE1] APP1 Size[36381] FileLen[1024]Name[../testdata/test_data_01.ti] LittleE[false] EXIF[true] OffsetToMainDir[0] Entries[0]
IFD:[0]GPSIFD TAG[6:GPSAltitude] FORMAT[id:5 bytes:8: type:n/d URational] ITEM_COUNT[1] LOCATION[00000417] VALUE[0/1000] TAG_DESC[Indicates the altitude based on the reference in GPSAltitudeRef.]
IFD:[1]GPSIFD TAG[5:GPSAltitudeRef] FORMAT[id:1 bytes:1: type:Byte Uint8] ITEM_COUNT[1] LOCATION[00000000] VALUE[0] TAG_DESC[Indicates the altitude used as the reference altitude.]
IFD:[1]MainIFD TAG[272:Model] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[8] LOCATION[0000007A] VALUE[LG-D855] TAG_DESC[The scanner model name or number.]
IFD:[2]MainIFD TAG[531:YCbCrPositioning] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00010000] VALUE[1] TAG_DESC[Specifies the positioning of subsampled chrominance components relative to luminance samples.]
IFD:[3]MainIFD TAG[296:ResolutionUnit] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00020000] VALUE[2] TAG_DESC[The unit of measurement for XResolution and YResolution.]
IFD:[4]MainIFD TAG[283:YResolution] FORMAT[id:5 bytes:8: type:n/d URational] ITEM_COUNT[1] LOCATION[00000082] VALUE[72/1] TAG_DESC[The number of pixels per ResolutionUnit in the ImageLength direction.]
IFD:[5]MainIFD TAG[274:Orientation] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00060000] VALUE[6] TAG_DESC[The orientation of the image with respect to the rows and columns.]
IFD:[0]ExifSubIFD TAG[40961:ColorSpace] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00010000] VALUE[1] TAG_DESC[Value is '1'. ]
IFD:[1]ExifSubIFD TAG[36868:DateTimeDigitized] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[20] LOCATION[0000018B] VALUE[2016:11:06 11:29:18] TAG_DESC[Date/Time of image digitized. Usually, it contains the same value of DateTimeOriginal(0x9003). ]
IFD:[2]ExifSubIFD TAG[33437:FNumber] FORMAT[id:5 bytes:8: type:n/d URational] ITEM_COUNT[1] LOCATION[0000019F] VALUE[240/100] TAG_DESC[The actual F-number(F-stop) of lens when the image was taken. ]
IFD:[3]ExifSubIFD TAG[37386:FocalLength] FORMAT[id:5 bytes:8: type:n/d URational] ITEM_COUNT[1] LOCATION[000001A7] VALUE[3970/1000] TAG_DESC[Focal length of lens used to take image. Unit is millimeter. ]
IFD:[4]ExifSubIFD TAG[41987:WhiteBalance] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00000000] VALUE[0] TAG_DESC[Indicates the white balance mode set when the image was shot.]
IFD:[5]ExifSubIFD TAG[40962:ExifImageWidth] FORMAT[id:4 bytes:4: type:Long Uint32] ITEM_COUNT[1] LOCATION[00001040] VALUE[4160] TAG_DESC[Size of main image. ]
IFD:[6]ExifSubIFD TAG[37383:MeteringMode] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00020000] VALUE[2] TAG_DESC[Exposure metering method. '1' means average, '2' center weighted average, '3' spot, '4' multi-spot, '5' multi-segment. ]
IFD:[7]ExifSubIFD TAG[36867:DateTimeOriginal] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[20] LOCATION[000001AF] VALUE[2016:11:06 11:29:18] TAG_DESC[Date/Time of original image taken. This value should not be modified by user program. ]
IFD:[8]ExifSubIFD TAG[37510:UserComment] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[512] LOCATION[000001C3] VALUE[    FocusArea=111111111] TAG_DESC[Stores user comment. ]
IFD:[9]ExifSubIFD TAG[37121:ComponentConfiguration] FORMAT[id:7 bytes:1: type:Undefined] ITEM_COUNT[4] LOCATION[01020300] VALUE[0x01,0x02,0x03,0x00] TAG_DESC[It seems value 0x00,0x01,0x02,0x03 always. ]
IFD:[10]ExifSubIFD TAG[40963:ExifImageHeight] FORMAT[id:4 bytes:4: type:Long Uint32] ITEM_COUNT[1] LOCATION[00000924] VALUE[2340] TAG_DESC[ExifImageHeight ]
IFD:[11]ExifSubIFD TAG[37385:Flash] FORMAT[id:3 bytes:2: type:Short Uint16] ITEM_COUNT[1] LOCATION[00000000] VALUE[0] TAG_DESC['1' means flash was used, '0' means not used. ]
IFD:[12]ExifSubIFD TAG[36864:ExifVersion] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[4] LOCATION[30323230] VALUE[0220] TAG_DESC[Exif version number. Stored as 4bytes of ASCII character (like 0210) ]
IFD:[0]Interoperability TAG[1:GPSLatitudeRef] FORMAT[id:2 bytes:1: type:ASCII String] ITEM_COUNT[4] LOCATION[52393800] VALUE[R98] TAG_DESC[Indicates whether the latitude is north or south latitude]
IFD:[1]Interoperability TAG[2:GPSLatitude] FORMAT[id:7 bytes:1: type:Undefined] ITEM_COUNT[4] LOCATION[30313030] VALUE[0x30,0x31,0x30,0x30] TAG_DESC[Indicates the latitude]


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
