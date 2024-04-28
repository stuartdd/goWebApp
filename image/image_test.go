package image

import (
	"fmt"
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
	fmt.Printf("%s\n", walker.LinePrint(OfsTiffEntries, 12, 20))

	for i, ifd := range image.IFDdata {
		fmt.Printf("%2d:%s [%s]\n", i, ifd, image.GetIDFData(i))
	}
	for i, ifd := range image.IFDdata {
		tag, ok := MapTags[ifd.tag]
		if ok {
			fmt.Printf("%s=%s\n", tag.desc, image.GetIDFData(i))
		}

	}
}

/*
exif:ApertureValue=153/100
exif:ColorSpace=65535
exif:DateTime=2023:08:22 18:03:33
exif:DateTimeDigitized=2023:08:22 18:03:33
exif:DateTimeOriginal=2023:08:22 18:03:33
exif:DigitalZoomRatio=229/100
exif:ExifOffset=246
exif:ExifVersion=0220
exif:ExposureBiasValue=0/100
exif:ExposureMode=0
exif:ExposureProgram=2
exif:ExposureTime=1/50
exif:Flash=0
exif:FNumber=170/100
exif:FocalLength=630/100
exif:FocalLengthIn35mmFilm=23
exif:GPSAltitude=87/1
exif:GPSAltitudeRef=0
exif:GPSInfo=720
exif:GPSLatitude=51/1,32/1,4224480/1000000
exif:GPSLatitudeRef=N
exif:GPSLongitude=3/1,6/1,43447320/1000000
exif:GPSLongitudeRef=W
exif:ImageLength=2252
exif:ImageUniqueID=P12XLPE00NM
exif:ImageWidth=4000
exif:Make=samsung
exif:MaxApertureValue=153/100
exif:MeteringMode=2
exif:Model=Galaxy S23 Ultra
exif:OffsetTime=+01:00
exif:OffsetTimeOriginal=+01:00
exif:Orientation=6
exif:PhotographicSensitivity=800
exif:PixelXDimension=4000
exif:PixelYDimension=2252
exif:ResolutionUnit=2
exif:SceneCaptureType=0
exif:ShutterSpeedValue=1/50
exif:Software=S918BXXS3AWF7
exif:SubSecTime=845
exif:SubSecTimeDigitized=845
exif:SubSecTimeOriginal=845
exif:thumbnail:Compression=6
exif:thumbnail:ImageLength=288
exif:thumbnail:ImageWidth=512
exif:thumbnail:JPEGInterchangeFormat=972
exif:thumbnail:JPEGInterchangeFormatLength=44018
exif:thumbnail:ResolutionUnit=2
exif:thumbnail:XResolution=72/1
exif:thumbnail:YResolution=72/1
exif:WhiteBalance=0
exif:XResolution=72/1
exif:YCbCrPositioning=1
exif:YResolution=72/1
*/
