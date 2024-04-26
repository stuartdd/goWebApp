package image

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type image struct {
	name       string
	walker     *walker
	sof        string
	exif       bool
	littleE    bool
	app1Marker string
	app1Size   uint32 // APP1 data size
}

func (p *image) IsBigE() bool {
	return !p.littleE
}

func (p *image) IsExif() bool {
	return p.exif
}

func (p *image) String() string {
	return fmt.Sprintf("IMG: SOF[%s]  APP1 Size[%d] EXIF[%t] LittleE[%t] FileLen[%d] Name[%s]", p.sof, p.app1Size, p.IsExif(), p.IsBigE(), p.walker.len, p.name)
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

func (p *walker) Zstring(max int) string {
	var line bytes.Buffer
	count := 0
	for p.canAdvance() {
		b := p.Advance(1)
		if b > 0 && count < max {
			line.WriteByte(byte(b))
		} else {
			return line.String()
		}
		count++
	}
	return ""
}

func (p *walker) ZstringEquals(s string) bool {
	return (strings.ToLower(p.Zstring(len(s)+2)) == strings.ToLower(s))
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

func (p *walker) canAdvance() bool {
	return p.pos < p.len
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
		name:       path,
		walker:     walker,
		sof:        walker.Pos(0).Hex16(),
		app1Marker: walker.Hex16(),
		app1Size:   walker.Int32BE() - 2,
		exif:       walker.Pos(6).ZstringEquals("Exif"),
		littleE:    walker.Pos(12).ZstringEquals("II*"),
	}
	walker.SetBigE(image.IsBigE())
	walker.Reset()
	return image, nil
}

/*
254	00FE	NewSubfileType	A general indication of the kind of data contained in this subfile.
255	00FF	SubfileType	A general indication of the kind of data contained in this subfile.
256	0100	ImageWidth	The number of columns in the image, i.e., the number of pixels per row.
257	0101	ImageLength	The number of rows of pixels in the image.
258	0102	BitsPerSample	Number of bits per component.
259	0103	Compression	Compression scheme used on the image data.
262	0106	PhotometricInterpretation	The color space of the image data.
263 0107 	Threshholding 	For black and white TIFF files that represent shades of gray, the technique used to convert from gray to black and white pixels.
264 0108 	CellWidth 	The width of the dithering or halftoning matrix used to create a dithered or halftoned bilevel file.
265 0109 	CellLength 	The length of the dithering or halftoning matrix used to create a dithered or halftoned bilevel file.
266	010A	FillOrder	The logical order of bits within a byte.
270	010E	ImageDescription	A string that describes the subject of the image.
271	010F	Make	The scanner manufacturer.
272	0110	Model	The scanner model name or number.
273	0111	StripOffsets	For each strip, the byte offset of that strip.
274	0112	Orientation	The orientation of the image with respect to the rows and columns.
277	0115	SamplesPerPixel	The number of components per pixel.
278	0116	RowsPerStrip	The number of rows per strip.
279	0117	StripByteCounts	For each strip, the number of bytes in the strip after compression.
280	0118	MinSampleValue	The minimum component value used.
281	0119	MaxSampleValue	The maximum component value used.
282	011A	XResolution	The number of pixels per ResolutionUnit in the ImageWidth direction.
283	011B	YResolution	The number of pixels per ResolutionUnit in the ImageLength direction.
284	011C	PlanarConfiguration	How the components of each pixel are stored.
288	0120	FreeOffsets	For each string of contiguous unused bytes in a TIFF file, the byte offset of the string.
289 0121 	FreeByteCounts 	For each string of contiguous unused bytes in a TIFF file, the number of bytes in the string.
290	0122	GrayResponseUnit	The precision of the information contained in the GrayResponseCurve.
291	0123	GrayResponseCurve	For grayscale data, the optical density of each possible pixel value.
296	0128	ResolutionUnit	The unit of measurement for XResolution and YResolution.
305	0131	Software	Name and version number of the software package(s) used to create the image.
306	0132	DateTime	Date and time of image creation.
315	013B	Artist	Person who created the image.
316	013C	HostComputer	The computer and/or operating system in use at the time of image creation.
320	0140	ColorMap	A color map for palette color images.
338	0152	ExtraSamples	Description of extra components.
33432	8298	Copyright	Copyright notice.
*/
