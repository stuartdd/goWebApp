package image

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type image struct {
	name    string
	walker  *walker
	exifPos int
	iiPos   int
	offset  uint32
}

func (p *image) IsBigE() bool {
	return p.iiPos < 4
}

func (p *image) String() string {
	return fmt.Sprintf("IMG: EXIF[%t] LittleE[%t] Offset[%d] FileLen[%d] Name[%s]", p.exifPos > 0, p.iiPos > 0, p.offset, p.walker.len, p.name)
}

type walker struct {
	data *[]uint8
	pos  int
	len  int
	bigE bool
}

func NewWalker(b *[]uint8, len int) *walker {
	return &walker{
		data: b,
		pos:  0,
		len:  len,
		bigE: false,
	}
}

func (p *walker) Hex8() string {
	s := fmt.Sprintf("%x", p.Advance(1))
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func (p *walker) Hex16() string {
	s := fmt.Sprintf("%s%s", p.Hex8(), p.Hex8())
	return s
}

func (p *walker) SetBigE(le bool) {
	p.bigE = le
}

func (p *walker) Char() string {
	b := p.Advance(1)
	if b < 32 {
		return string(rune(1))
	}
	return string(byte(b & 0xFF))
}

func (p *walker) Int8() uint32 {
	return p.Advance(1)
}

func (p *walker) Int16() uint32 {
	if p.bigE {
		return p.Int16BE()
	}
	return p.Int16LE()
}

func (p *walker) Int32() uint32 {
	if p.bigE {
		return p.Int32BE()
	}
	return p.Int32LE()
}

func (p *walker) Int32LE() uint32 {
	w1 := p.Int16LE()
	w2 := p.Int16LE()
	return (w2 << 16) | w1
}

func (p *walker) Int16LE() uint32 {
	b1 := p.Int8()
	b2 := p.Int8()
	return (b2 << 8) | b1
}

func (p *walker) Int32BE() uint32 {
	w1 := p.Int16BE()
	w2 := p.Int16BE()
	return (w1 << 16) | w2
}

func (p *walker) Int16BE() uint32 {
	b1 := p.Int8()
	b2 := p.Int8()
	return (b1 << 8) | b2
}

func (p *walker) canNotAdvance() bool {
	return p.pos >= p.len
}

func (p *walker) Reset() *walker {
	p.pos = 0
	return p
}

func (p *walker) matchChunk(byts []byte) bool {
	pos := p.pos
	for i := 0; i < len(byts); i++ {
		if pos+i >= p.len {
			return false
		}
		if byts[i] != (*p.data)[pos+i] {
			return false
		}
	}
	return true
}

func (p *walker) SearchFromStart(chars string) int {
	p.Reset()
	return p.Search(chars)
}

func (p *walker) Search(chars string) int {
	byts := []byte(chars)
	for !p.canNotAdvance() {
		if p.Advance(1) == uint32(byts[0]) {
			if p.matchChunk(byts[1:]) {
				p.Retard(1)
				return p.pos
			}
		}
	}
	return -1
}

func (p *walker) Retard(n int) uint32 {
	if p.pos < 0 {
		panic(fmt.Sprintf("Retard past start: Requested=%d", n))
	}
	b := (*p.data)[p.pos]
	p.pos = p.pos - n
	return uint32(b) & 0xFF
}

func (p *walker) Advance(n int) uint32 {
	if p.canNotAdvance() {
		panic(fmt.Sprintf("Advanced past end: Max=%d Requested=%d", p.len-1, p.pos))
	}
	b := (*p.data)[p.pos]
	p.pos = p.pos + n
	return uint32(b) & 0xFF
}

func (p *walker) Clone() *walker {
	return &walker{
		data: p.data,
		pos:  p.pos,
		len:  p.len,
		bigE: p.bigE,
	}
}

func (q *walker) line16(start int, lines int) string {
	clone := q.Clone()
	var line bytes.Buffer
	clone.SetPos(start)
	for j := 0; j < lines; j++ {
		line.WriteString(pad(uint32(clone.pos), 4))
		line.WriteRune(':')
		line.WriteRune(' ')
		p := clone.pos
		for i := 0; i < 16; i++ {
			line.WriteString(clone.Hex8())
			line.WriteRune(' ')
		}
		clone.SetPos(p)
		for i := 0; i < 8; i++ {
			line.WriteString(pad(clone.Int16(), 6))
			line.WriteRune(' ')
		}
		line.WriteString("\n")
		clone.SetPos(p)
		line.WriteString("    : ")
		for i := 0; i < 16; i++ {
			line.WriteString(clone.Char())
			line.WriteRune(' ')
			line.WriteRune(' ')
		}
		clone.SetPos(p)
		for i := 0; i < 8; i++ {
			line.WriteString(clone.Hex16())
			line.WriteString("   ")
		}
		line.WriteString("\n")
	}
	return line.String()
}

func (p *walker) Pos(n int) *walker {
	p.SetPos(n)
	return p
}

func (p *walker) SetPos(n int) uint32 {
	if n >= p.len {
		panic(fmt.Sprintf("setPos past end: Max=%d Requested=%d", p.len-1, n))
	}
	p.pos = n
	return uint32((*p.data)[p.pos]) & 0xFF
}

func pad(i uint32, n int) string {
	s := fmt.Sprintf("%d", i)
	return "00000"[0:n-len(s)] + s
}

func GetImage(path string) (*image, error) {
	p, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	reader, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	byteArray, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	walker := NewWalker(&byteArray, len(byteArray))
	image := &image{
		name:    path,
		walker:  walker,
		exifPos: walker.SearchFromStart("Exif"),
		iiPos:   walker.SearchFromStart("II*"),
		offset:  walker.Pos(4).Int16BE() - 2,
	}
	walker.SetBigE(image.IsBigE())
	walker.Reset()
	return image, nil
}
