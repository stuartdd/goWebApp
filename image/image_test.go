package image

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

var td = []byte{0xff, 0x8, 0xff, 0x4, 0xaf, 0xc6, 0x45, 0x78}

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

func TestWalkerHex(t *testing.T) {
	walker := NewWalker(&td, uint32(len(td)))
	if walker.Hex(walker.Bytes(1)) != "ff" {
		t.Fatal("1st byte != ff")
	}
	if walker.Hex(walker.Bytes(1)) != "08" {
		t.Fatal("byte != 08")
	}
	walker.Retard(1)
	if walker.Hex(walker.Bytes(1)) != "08" {
		t.Fatal("byte != 08")
	}
	if walker.Hex(walker.Bytes(2)) != "ff04" {
		t.Fatal("word != ff04")
	}
	walker.Retard(2)
	if walker.Hex(walker.Bytes(2)) != "ff04" {
		t.Fatal("word != ff04")
	}
	if walker.Hex(walker.Bytes(1)) != "af" {
		t.Fatal("byte != af")
	}
	if walker.Hex(walker.Bytes(2)) != "c645" {
		t.Fatal("word != c645")
	}
	defer func() {
		if r := recover(); r != nil {
			if r != "Advanced past end: Max=7 Requested=8" {
				t.Fatalf("Hex(2) Did not panic with correct message")
			}
		}
	}()
	walker.Hex(walker.Bytes(2)) // Will take it past end
}

func TestImage(t *testing.T) {
	image, err := GetImage("../testdata/testImage.jpg")
	if err != nil {
		t.Fatal(err)
	}
	walker := image.walker
	fmt.Printf("%s\n", image)
	fmt.Printf("%s\n", walker.LinePrint(0, 16, 2))
	fmt.Printf("%s\n", walker.LinePrint(22+2, 12, 20))

	for i, ifd := range image.IFDdata {
		fmt.Printf("%2d:%s [%s]\n", i, ifd, image.GetIDFData(i))
	}
	var output bytes.Buffer
	for i, ifd := range image.IFDdata {
		tag, ok := MapTags[ifd.tag]
		if ok {
			output.WriteString(fmt.Sprintf("%s=%s\n", tag.desc, image.GetIDFData(i)))
		} else {
			output.WriteString(fmt.Sprintf("Un-Registered Tag[0x%4x]=%s\n", ifd.tag, image.GetIDFData(i)))
		}
	}
	fmt.Println(output.String())
	a := strings.TrimSpace(output.String())
	b := strings.TrimSpace(golden)
	if b != a {
		t.Fatal("Not the same")
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
*/

const golden string = `ApertureValue=153/100
ColorSpace=65535
DateTime=2023:08:22 18:03:33
DateTimeDigitized=2023:08:22 18:03:33
DateTimeOriginal=2023:08:22 18:03:33
DigitalZoomRatio=229/100
ExifOffset=246
ExifVersion=0220
ExposureBiasValue=0/100
ExposureMode=0
ExposureProgram=2
ExposureTime=1/50
Flash=0
FNumber=170/100
FocalLength=630/100
FocalLengthIn35mmFilm=23
ImageLength=2252
ImageUniqueID=P12XLPE00NM
ImageWidth=4000
Make=samsung
MaxApertureValue=153/100
MeteringMode=2
Model=Galaxy S23 Ultra
OffsetTime=+01:00
OffsetTimeOriginal=+01:00
Orientation=6
ResolutionUnit=2
SceneCaptureType=0
ShutterSpeedValue=1/50
Software=S918BXXS3AWF7
SubSecTime=845
SubSecTimeDigitized=845
SubSecTimeOriginal=845
WhiteBalance=0
XResolution=72/1
YCbCrPositioning=1
YResolution=72/1`

const golden2 string = `ApertureValue=153/100
ColorSpace=65535
DateTime=2023:08:22 18:03:33
DateTimeDigitized=2023:08:22 18:03:33
DateTimeOriginal=2023:08:22 18:03:33
DigitalZoomRatio=229/100
ExifImageHeight=2252
ExifImageWidth=4000
ExifVersion=30323230
ExposureBiasValue=00
ExposureMode=0
ExposureProgram=2
ExposureTime=1/50
FNumber=170/100
Flash=0
FocalLength=630/100
FocalLengthIn35mmFilm=23
ImageLength=2252
ImageUniqueID=P12XLPE00NM
ImageWidth=4000
Make=samsung
MaxApertureValue=153/100
MeteringMode=2
Model=Galaxy S23 Ultra
OffsetTime=+01:00
OffsetTimeOriginal=+01:00
Orientation=6
ResolutionUnit=2
SceneCaptureType=0
ShutterSpeedValue=1/50
Software=S918BXXS3AWF7
SubsecTime=845
SubsecTimeDigitized=845
SubsecTimeOriginal=845
WhiteBalance=0
XResolution=72/1
YCbCrPositioning=1
YResolution=72/1`
