package image

import (
	"fmt"
	"testing"
)

var td = []byte{0xff, 0x8, 0xff, 0x4, 0xaf, 0xc6, 0x45, 0x78}
var sd = "12345abc"

var xx = []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x76, 0xD4, 0x45, 0x78, 0x69, 0x66, 0, 0}

func TestWalkerFind(t *testing.T) {
	bd := []byte(sd)
	walker := NewWalker(&bd, len(td))
	if walker.Search("5a") != 4 {
		t.Fatal("Should find 5a")
	}
	b := walker.Advance(1)
	if b != '5' {
		t.Fatal("Should find 5")
	}
	walker.Reset()
	if walker.Search("123") != 0 {
		t.Fatal("Should find 123")
	}
	b = walker.Advance(1)
	if b != '1' {
		t.Fatal("Should find 5")
	}

	if walker.Search("ab") != 5 {
		t.Fatal("Should find ab")
	}
	if walker.Search("abc") != 5 {
		t.Fatal("Should find abc")
	}
	walker.Reset()
	if walker.Search("abcd") != -1 {
		t.Fatal("Should not find abcd")
	}

}

func TestWalkerInt(t *testing.T) {
	walker := NewWalker(&td, len(td))
	if walker.Int8() != 255 {
		t.Fatal("1st byte != 255")
	}
	w1 := fmt.Sprintf("%x", walker.Int16())
	if w1 != "8ff" {
		t.Fatal("int word as hex != 8ff")
	}
	w1 = fmt.Sprintf("%x", walker.Int16())
	if w1 != "4af" {
		t.Fatal("int word as hex != 4af")
	}
	w1 = fmt.Sprintf("%x", walker.Int16())
	if w1 != "c645" {
		t.Fatal("int word as hex != c645")
	}
	walker.Reset()
	w1 = fmt.Sprintf("%x", walker.Int16())
	if w1 != "8ff" {
		t.Fatal("int word-hi as hex != 8ff")
	}
	w1 = fmt.Sprintf("%x", walker.SetPos(6))
	if w1 != "45" {
		t.Fatal("int word-hi as hex != 45")
	}
	defer func() {
		if r := recover(); r != nil {
			if r != "setPos past end: Max=7 Requested=8" {
				t.Fatalf("SetPos(8) Did not panic with correct message")
			}
		}
	}()
	walker.SetPos(8)
}

func TestWalkerHex(t *testing.T) {
	walker := NewWalker(&td, len(td))
	if walker.Hex8() != "ff" {
		t.Fatal("1st byte != ff")
	}
	if walker.Hex8() != "08" {
		t.Fatal("byte != 08")
	}
	walker.Retard(1)
	if walker.Hex8() != "08" {
		t.Fatal("byte != 08")
	}
	if walker.Hex16() != "ff04" {
		t.Fatal("word != ff04")
	}
	walker.Retard(2)
	if walker.Hex16() != "ff04" {
		t.Fatal("word != ff04")
	}
	if walker.Hex8() != "af" {
		t.Fatal("byte != af")
	}
	if walker.Hex16() != "c645" {
		t.Fatal("word != c645")
	}
	defer func() {
		if r := recover(); r != nil {
			if r != "Advanced past end: Max=7 Requested=8" {
				t.Fatalf("Hex16() Did not panic with correct message")
			}
		}
	}()
	walker.Hex16() // Will take it past end
}

func TestImage(t *testing.T) {
	image, err := GetImage("../testdata/testImage.jpg")
	if err != nil {
		t.Fatal(err)
	}
	walker := image.walker
	// ofs (065505 BE) 057855 LE
	fmt.Printf("%s", image)
	fmt.Printf(" %s\n", walker.Pos(4).Hex16())
	fmt.Printf("%s\n", walker.line16(0, 1))
	fmt.Printf("%s\n", walker.line16(16, 15))
	fmt.Printf("Offset to IFD0  %d\n", walker.Pos(16).Int32())
	fmt.Printf("Entries %d\n", walker.Int16())
	for i := 1; i < 10; i++ {
		fmt.Println()
		fmt.Printf("%2d TagNum   %d\n", i, walker.Int16())
		fmt.Printf("%2d TagType  %d\n", i, walker.Int16())
		fmt.Printf("%2d Data Len %d\n", i, walker.Int32())
		fmt.Printf("%2d Location %d\n", i, walker.Int32())
	}

	// for i := 0; i < 64; i++ {
	// 	fmt.Printf("%s,", walker.Hex8())
	// 	if (i % 16) == 0 {
	// 		fmt.Println()
	// 	}
	// }
	// fmt.Println()

	// walker.Reset()

	// h16 := walker.Hex16()
	// if h16 != "ffd8" {
	// 	t.Fatalf("Not a jpeg image. SOF = %s", h16)
	// }

	// h16 = walker.Hex16() // APP1 segment marker 0xFFE1
	// if h16 != "ffe1" {
	// 	t.Fatalf("Not a APP1 Marker . APP1 = %s", h16)
	// }
	// // len := walker.Int16Hi()
	// // ident := walker.Hex16()
	// // if len != 10 {
	// // 	t.Fatalf("Invalid len . APP1 = %d (%d) ident(%s)", len, walker.len, ident)
	// // }
	// pos := walker.Search("DateTimeOriginal")
	// if pos < 0 {
	// 	t.Fatalf("BOO HOO")
	// }
	// for i := 0; i < 40; i++ {
	// 	fmt.Printf("%c,", walker.Int8())
	// }
	// fmt.Println()

}
