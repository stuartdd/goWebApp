package image

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const OfsSOI = 0
const OfsAPP1Marker = 2
const OfsAPP1Size = 4
const OfsExifHeader = 6
const OfsEndianDef = 12
const OfsMainImageOffset = 16

const TiffRecordSize = 12
const TagExifVersion uint32 = 36864

const TagExifSubIFD uint32 = 34665
const TagGPSIFD uint32 = 34853
const TagInteroperabilityIFD uint32 = 40965

type Tag struct {
	TagNum         uint32
	Name           string
	LongDesc       string
	formatOverride TiffFormat
}

type TagFormat struct {
	tiffFormat TiffFormat
	byteLen    uint32
	desc       string
}

func (p *Tag) String() string {
	return fmt.Sprintf("%s: %s", p.Name, p.LongDesc)
}

func (p *TagFormat) String() string {
	return fmt.Sprintf("id:%d bytes:%d: type:%s", p.tiffFormat, p.byteLen, p.desc)
}

func newTagFormat(format TiffFormat, desc string, byteLen uint32) *TagFormat {
	if byteLen < 1 {
		byteLen = 1
	}
	return &TagFormat{
		tiffFormat: format,
		byteLen:    byteLen,
		desc:       desc,
	}
}

func newExifTagDetails(tag uint32, desc string, formatOverride TiffFormat, longD string) *Tag {
	return &Tag{
		TagNum:         tag,
		Name:           desc,
		LongDesc:       longD,
		formatOverride: formatOverride,
	}
}

type IFDEntry struct {
	itemCount   uint32
	location    []byte
	Value       string
	TagData     *Tag
	tagFormat   *TagFormat
	DataAddress uint32
	ByteCount   uint32
}

func newIFDEntry(walker *Walker) *IFDEntry {
	// Do these in the right order!
	tag := uint32(walker.BytesToUint(walker.Bytes(2)))
	fmt := uint16(walker.BytesToUint(walker.Bytes(2)))
	items := uint32(walker.BytesToUint(walker.Bytes(4)))

	addressOfData := walker.posit
	loc := walker.Bytes(4)

	tagData := ToTagData(tag)
	tagFmt := toTagFormat(fmt)
	if tagFmt.tiffFormat == FormatUndefined {
		tagFmt = toTagFormat(uint16(tagData.formatOverride))
	}
	byteCount := items * tagFmt.byteLen
	if byteCount > 4 {
		addressOfData = uint32(walker.BytesToUint(loc) + TiffRecordSize)
	}

	return &IFDEntry{
		itemCount:   items,
		location:    loc,
		Value:       "",
		TagData:     tagData,
		tagFormat:   tagFmt,
		DataAddress: addressOfData,
		ByteCount:   byteCount,
	}

}

func (p *IFDEntry) Diagnostics(m string) string {
	return fmt.Sprintf("IFD:%s TAG[%d:%s] FORMAT[%s] ITEM_COUNT[%d] LOCATION[%s] VALUE[%s] TAG_DESC[%s]", m, p.TagData.TagNum, p.TagData.Name, p.tagFormat, p.itemCount, bytesToHex(p.location), p.Value, p.TagData.LongDesc)
}

func (p *IFDEntry) Output() string {
	return fmt.Sprintf("%s=%s", p.TagData.Name, p.Value)
}

type Walker struct {
	data    *[]uint8
	posit   uint32
	len     uint32
	littleE bool
}

func NewWalker(b *[]uint8, len uint32) *Walker {
	return &Walker{
		data:    b,
		posit:   0,
		len:     len,
		littleE: false,
	}
}

func (p *Walker) SetLittleE(yes bool) {
	p.littleE = yes
}

func (p *Walker) Zstring(max int) string {
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

func (p *Walker) ZstringEquals(s string) bool {
	return (strings.EqualFold(p.Zstring(len(s)+2), s))
}

func (p *Walker) Bytes(n uint32) []byte {
	b := make([]byte, n)
	for i := 0; uint32(i) < n; i++ {
		b[i] = byte(p.Advance(1))
	}
	return b
}

func (p *Walker) Hex(b []byte, pre string) string {
	return pre + bytesToHex(b)
}

func (p *Walker) BytesToUint(b []byte) uint64 {
	if p.littleE {
		return p.bytesToUintLE(b)
	}
	return p.bytesToUintBE(b)
}

func (p *Walker) BytesToInt(b []byte) int64 {
	if p.littleE {
		return int64(p.bytesToUintLE(b))
	}
	return int64(p.bytesToUintBE(b))
}

func (p *Walker) bytesToUintLE(b []byte) uint64 {
	switch len(b) {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(binary.LittleEndian.Uint16(b))
	case 4:
		return uint64(binary.LittleEndian.Uint32(b))
	case 8:
		return uint64(binary.LittleEndian.Uint64(b))
	}
	return 0
}

func (p *Walker) bytesToUintBE(b []byte) uint64 {
	switch len(b) {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(binary.BigEndian.Uint16(b))
	case 4:
		return uint64(binary.BigEndian.Uint32(b))
	case 8:
		return uint64(binary.BigEndian.Uint64(b))
	}
	return 0
}

func (p *Walker) Char() string {
	b := p.Advance(1)
	if b < 32 {
		return string(rune(1))
	}
	return string(byte(b & 0xFF))
}

func (p *Walker) canNotAdvance() bool {
	return p.posit >= p.len
}

func (p *Walker) canAdvance() bool {
	return p.posit < p.len
}

func (p *Walker) Reset() *Walker {
	p.posit = 0
	return p
}

func (p *Walker) Advance(n uint32) uint32 {
	if p.canNotAdvance() {
		panic(fmt.Sprintf("Advanced past end: Max=%d Requested=%d", p.len-1, p.posit))
	}
	b := (*p.data)[p.posit]
	p.posit = p.posit + n
	return uint32(b) & 0xFF
}

func (p *Walker) Pos(n uint32) *Walker {
	if n >= p.len {
		panic(fmt.Sprintf("Pos was set past end: Max=%d Requested=%d", p.len-1, n))
	}
	p.posit = n
	return p
}

func (p *Walker) Clone() *Walker {
	return &Walker{
		data:    p.data,
		posit:   p.posit,
		len:     p.len,
		littleE: p.littleE,
	}
}

func (q *Walker) LinePrint(start uint32, count int, lines int) string {
	clone := q.Clone()
	var line bytes.Buffer
	clone.Pos(start)
	for j := 0; j < lines; j++ {
		line.WriteString(pad(uint32(clone.posit), 4))
		line.WriteRune(':')
		line.WriteRune(' ')
		p := clone.posit
		for i := 0; i < count; i++ {
			line.WriteString(clone.Hex(clone.Bytes(1), ""))
			line.WriteRune(' ')
		}
		clone.Pos(p)
		for i := 0; i < (count / 2); i++ {
			line.WriteString(pad(uint32(clone.BytesToUint(clone.Bytes(2))), 6))
			line.WriteRune(' ')
		}
		line.WriteString("\n")
		clone.Pos(p)
		line.WriteString("    : ")
		for i := 0; i < count; i++ {
			line.WriteString(clone.Char())
			line.WriteRune(' ')
			line.WriteRune(' ')
		}
		clone.Pos(p)
		for i := 0; i < count/2; i++ {
			line.WriteString(clone.Hex(clone.Bytes(2), ""))
			line.WriteString("   ")
		}
		line.WriteString("\n")
	}
	return line.String()
}

type image struct {
	name          string
	walker        *Walker
	soi           string
	exif          bool
	app1Marker    string
	app1Size      uint32 // APP1 data size
	mainDirOffset uint32
	IFDdata       []*IFDEntry
	debug         bool
	echo          bool
	sort          bool
	selector      func(*IFDEntry, *Walker) bool
}

func GetImage(path string, debug bool, echo bool, sort bool, sel func(*IFDEntry, *Walker) bool) (img *image, err error) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC:%s", r)
			os.Stderr.WriteString(msg)
			img = nil
			err = fmt.Errorf(msg)
		}
	}()
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
	walker := NewWalker(&byteArray, uint32(len(byteArray)))
	image := &image{
		debug:      debug,
		echo:       echo,
		sort:       sort,
		selector:   sel,
		name:       path,
		walker:     walker,
		soi:        walker.Pos(OfsSOI).Hex(walker.Bytes(2), ""),
		app1Marker: walker.Pos(OfsAPP1Marker).Hex(walker.Bytes(2), ""),
		// Always Big Endian
		// Size includes the size bytes so sub 2
		app1Size: uint32(walker.Pos(OfsAPP1Size).bytesToUintBE(walker.Bytes(2)) - 2),
		exif:     walker.Pos(OfsExifHeader).ZstringEquals("Exif"),
		IFDdata:  []*IFDEntry{},
	}

	if image.debug {
		os.Stdout.WriteString(image.Diagnostics("IMG"))
		os.Stdout.WriteString("\n")
		if image.selector != nil {
			if !image.selector(nil, walker.Clone().Pos(OfsMainImageOffset)) {
				os.Exit(1)
			}
		}
	}

	if image.soi != "FFD8" {
		panic(fmt.Sprintf("Jpeg marker 'FFD8' is missing (Offset %d) found %s", OfsSOI, image.soi))
	}
	if image.app1Marker != "FFE1" {
		panic(fmt.Sprintf("Jpeg APP1 marker 'FFE1' is missing (Offset %d) found %s", OfsAPP1Marker, image.app1Marker))
	}
	if !image.exif {
		panic(fmt.Sprintf("Jpeg 'Exif' data marker is missing (Offset %d) found %s", OfsExifHeader, bytesToZString(walker.Pos(OfsExifHeader).Bytes(6))))
	}

	image.walker.littleE = (walker.Pos(OfsEndianDef).ZstringEquals("II*"))
	/*
		The rest of the image data needs to know the littleE setting to work

		Calc the start if the tags Using TIFF Header offset
	*/
	image.mainDirOffset = uint32(walker.Pos(OfsMainImageOffset).BytesToUint(walker.Bytes(4)))

	mainTiffDir := OfsMainImageOffset + image.mainDirOffset - uint32(4)
	image.readDirectory(mainTiffDir, walker, 0)

	if image.sort {
		image.sortEntries()
	}

	if image.echo {
		for _, ifd := range image.IFDdata {
			os.Stdout.WriteString(ifd.Output())
			os.Stdout.WriteString("\n")
		}
	}
	return image, nil
}

func (p *image) Diagnostics(m string) string {
	return fmt.Sprintf("DEBUG:%s SOI[%s]  APP1 Mark[%s] APP1 Size[%d] FileLen[%d]Name[%s] LittleE[%t] EXIF[%t] OffsetToMainDir[%d] Entries[%d]", m, p.soi, p.app1Marker, p.app1Size, p.walker.len, p.name, p.walker.littleE, p.IsExif(), p.mainDirOffset, len(p.IFDdata))
}

func (p *image) Output() string {
	var line bytes.Buffer
	for _, o := range p.IFDdata {
		line.WriteString(o.Output())
		line.WriteString("\n")
	}
	return line.String()
}

func (p *image) sortEntries() {
	m := map[string]*IFDEntry{}
	for i, x := range p.IFDdata {
		tag, ok := mapTags[x.TagData.TagNum]
		if ok {
			m[tag.Name] = x
		} else {
			m[fmt.Sprintf("x:%4x:%d", x.TagData.TagNum, i)] = x
		}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := make([]*IFDEntry, len(keys))
	for i, s := range keys {
		sorted[i] = m[s]
	}
	p.IFDdata = sorted
}

func (p *image) IsExif() bool {
	return p.exif
}

func (p *image) getValueBytes(ifd *IFDEntry) []byte {
	byteCount := ifd.itemCount * ifd.tagFormat.byteLen
	if (byteCount) > 4 {
		// Location is a pointer from the IDFBase
		// Clone the walker so we can use it to get the bytes without effecting the parser
		w := p.walker.Clone()
		return w.Pos(uint32(w.BytesToUint(ifd.location)) + TiffRecordSize).Bytes(byteCount)
	} else {
		// Location is the value
		return ifd.location
	}
}

func (p *image) GetIDFData(ifd *IFDEntry) string {
	var line bytes.Buffer
	bytes := p.getValueBytes(ifd)
	items := int(ifd.itemCount)
	tagFormat := ifd.tagFormat

	if tagFormat.tiffFormat == FormatString || ifd.TagData.TagNum == TagExifVersion {
		return bytesToZString(bytes)
	}

	bytePos := 0
	byteLen := int(tagFormat.byteLen)
	for i := 0; i < items; i++ {
		subBytes := bytes[bytePos : bytePos+byteLen]
		switch tagFormat.tiffFormat {
		case FormatUint8:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToUint(subBytes)))
		case FormatInt8:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToInt(subBytes)))
		case FormatUint16:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToUint(subBytes)))
		case FormatInt16:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToInt(subBytes)))
		case FormatUint32:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToUint(subBytes)))
		case FormatInt32:
			line.WriteString(fmt.Sprintf("%d", p.walker.BytesToInt(subBytes)))
		case FormatURational:
			n := p.walker.BytesToUint(subBytes[0:4])
			d := p.walker.BytesToUint(subBytes[4:])
			line.WriteString(fmt.Sprintf("%d/%d", n, d))
		case FormatRational:
			n := p.walker.BytesToInt(subBytes[0:4])
			d := p.walker.BytesToInt(subBytes[4:])
			line.WriteString(fmt.Sprintf("%d/%d", n, d))
		default:
			line.WriteString(p.walker.Hex(subBytes, "0x"))
		}
		bytePos = bytePos + byteLen
		if i < (items - 1) {
			line.WriteRune(',')
		}
	}
	return line.String()
}

func (p *image) readDirectory(base uint32, walker *Walker, dirId uint32) {
	dirCount := int(walker.Pos(base).BytesToUint(walker.Bytes(2)))
	if dirCount <= 0 || dirCount > 200 {
		panic(fmt.Sprintf("Image TIFF data count is invalid. Expected[1..200]. Actual=[%d]", dirCount))
	}
	for i := 0; i < dirCount; i++ {
		current := walker.posit
		ne := newIFDEntry(walker)
		ne.Value = p.GetIDFData(ne)
		if ne.TagData.TagNum == TagExifSubIFD || ne.TagData.TagNum == TagGPSIFD || ne.TagData.TagNum == TagInteroperabilityIFD {
			ofs := walker.BytesToUint(ne.location)
			p.readDirectory(uint32(ofs+12), walker.Clone(), ne.TagData.TagNum)
		} else {
			if p.debug {
				var n string
				n, ok := dirTagNames[dirId]
				if ok {
					os.Stdout.WriteString(ne.Diagnostics(fmt.Sprintf("[%d]%s", i, n)))
				} else {
					os.Stdout.WriteString(ne.Diagnostics(fmt.Sprintf("[%d]%d", i, ne.TagData.TagNum)))
				}
				os.Stdout.WriteString("\n")
			}
			if (p.selector != nil && p.selector(ne, walker.Clone().Pos(current))) || p.selector == nil {
				p.IFDdata = append(p.IFDdata, ne)
			}
		}
	}
}

/*
Because I want to see hev values in upper case!
*/
var hexDigits = []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}

func bytesToHex(b []byte) string {
	var line bytes.Buffer
	for i := 0; i < len(b); i++ {
		line.WriteString(byteToHex(b[i]))
	}
	return line.String()
}

func byteToHex(b byte) string {
	var l bytes.Buffer
	l.WriteRune(hexDigits[b>>4])
	l.WriteRune(hexDigits[b&0x0f])
	return l.String()
}

func bytesToZString(b []byte) string {
	var line bytes.Buffer
	for _, c := range b {
		if c == 0 {
			break
		}
		line.WriteByte(c)
	}
	return strings.TrimSpace(line.String())
}

func pad(i uint32, n int) string {
	s := fmt.Sprintf("%d", i)
	if len(s) >= n {
		return s
	}
	return "00000000000"[0:n-len(s)] + s
}

var dirTagNames = map[uint32]string{0: "MainIFD", TagExifSubIFD: "ExifSubIFD", TagGPSIFD: "GPSIFD", TagInteroperabilityIFD: "Interoperability"}

type TiffFormat uint16

func toTagFormat(formatId uint16) *TagFormat {
	ta, ok := mapTiffFormats[formatId]
	if !ok {
		return mapTiffFormats[uint16(FormatUndefined)]
	}
	return ta
}

/*
Enum of format types
*/
const (
	FormatUint8 TiffFormat = iota + 1
	FormatString
	FormatUint16
	FormatUint32
	FormatURational
	FormatInt8
	FormatUndefined
	FormatInt16
	FormatInt32
	FormatRational
	FormatFloat32
	FormatFloat64
)

func ToTagData(tag uint32) *Tag {
	ta, ok := mapTags[tag]
	if !ok {
		ta = &Tag{
			TagNum:   tag,
			Name:     fmt.Sprintf("Undefined. Tag 0x%4x", tag),
			LongDesc: "",
		}
	}
	return ta
}

/*
Format type to type enum, name and bytes per entry
*/
var mapTiffFormats = map[uint16]*TagFormat{
	1:  newTagFormat(FormatUint8, "Byte Uint8", 1),
	2:  newTagFormat(FormatString, "ASCII String", 1),
	3:  newTagFormat(FormatUint16, "Short Uint16", 2),
	4:  newTagFormat(FormatUint32, "Long Uint32", 4),
	5:  newTagFormat(FormatURational, "n/d URational", 8),
	6:  newTagFormat(FormatInt8, "Byte Int8", 1),
	7:  newTagFormat(FormatUndefined, "Undefined", 1),
	8:  newTagFormat(FormatInt8, "Short Int8", 1),
	9:  newTagFormat(FormatInt16, "Long Int16", 2),
	10: newTagFormat(FormatRational, "n/d Rational", 8),
	11: newTagFormat(FormatFloat32, "Single Float32", 2),
	12: newTagFormat(FormatFloat64, "Double Float64", 4),
}

/*
Map of tags to tag,name and a long desc.
*/
var mapTags = map[uint32]*Tag{
	254:   newExifTagDetails(254, "NewSubfileType", FormatUndefined, "A general indication of the kind of data contained in this subfile."),
	255:   newExifTagDetails(255, "SubfileType", FormatUndefined, "A general indication of the kind of data contained in this subfile."),
	256:   newExifTagDetails(256, "ImageWidth", FormatUndefined, "The number of columns in the image, i.e., the number of pixels per row."),
	257:   newExifTagDetails(257, "ImageLength", FormatUndefined, "The number of rows of pixels in the image."),
	258:   newExifTagDetails(258, "BitsPerSample", FormatUndefined, "Number of bits per component."),
	259:   newExifTagDetails(259, "Compression", FormatUndefined, "Compression scheme used on the image data."),
	262:   newExifTagDetails(262, "PhotometricInterpretation", FormatUndefined, "The color space of the image data."),
	263:   newExifTagDetails(263, "Threshholding", FormatUndefined, "For black and white TIFF files that represent shades of gray, the technique used to convert from gray to black and white pixels."),
	264:   newExifTagDetails(264, "CellWidth", FormatUndefined, "The width of the dithering or halftoning matrix used to create a dithered or halftoned bilevel file."),
	265:   newExifTagDetails(265, "CellLength", FormatUndefined, "The length of the dithering or halftoning matrix used to create a dithered or halftoned bilevel file."),
	266:   newExifTagDetails(266, "FillOrder", FormatUndefined, "The logical order of bits within a byte."),
	270:   newExifTagDetails(270, "ImageDescription", FormatUndefined, "A string that describes the subject of the image."),
	271:   newExifTagDetails(271, "Make", FormatUndefined, "The scanner manufacturer."),
	272:   newExifTagDetails(272, "Model", FormatUndefined, "The scanner model name or number."),
	273:   newExifTagDetails(273, "StripOffsets", FormatUndefined, "For each strip, the byte offset of that strip."),
	274:   newExifTagDetails(274, "Orientation", FormatUndefined, "The orientation of the image with respect to the rows and columns."),
	277:   newExifTagDetails(277, "SamplesPerPixel", FormatUndefined, "The number of components per pixel."),
	278:   newExifTagDetails(278, "RowsPerStrip", FormatUndefined, "The number of rows per strip."),
	279:   newExifTagDetails(279, "StripByteCounts", FormatUndefined, "For each strip, the number of bytes in the strip after compression."),
	280:   newExifTagDetails(280, "MinSampleValue", FormatUndefined, "The minimum component value used."),
	281:   newExifTagDetails(281, "MaxSampleValue", FormatUndefined, "The maximum component value used."),
	282:   newExifTagDetails(282, "XResolution", FormatUndefined, "The number of pixels per ResolutionUnit in the ImageWidth direction."),
	283:   newExifTagDetails(283, "YResolution", FormatUndefined, "The number of pixels per ResolutionUnit in the ImageLength direction."),
	284:   newExifTagDetails(284, "PlanarConfiguration", FormatUndefined, "How the components of each pixel are stored."),
	288:   newExifTagDetails(288, "FreeOffsets", FormatUndefined, "For each string of contiguous unused bytes in a TIFF file, the byte offset of the string."),
	289:   newExifTagDetails(289, "FreeByteCounts", FormatUndefined, "For each string of contiguous unused bytes in a TIFF file, the number of bytes in the string."),
	290:   newExifTagDetails(290, "GrayResponseUnit", FormatUndefined, "The precision of the information contained in the GrayResponseCurve."),
	291:   newExifTagDetails(291, "GrayResponseCurve", FormatUndefined, "For grayscale data, the optical density of each possible pixel value."),
	296:   newExifTagDetails(296, "ResolutionUnit", FormatUndefined, "The unit of measurement for XResolution and YResolution."),
	305:   newExifTagDetails(305, "Software", FormatUndefined, "Name and version number of the software package(s) used to create the image."),
	306:   newExifTagDetails(306, "DateTime", FormatUndefined, "Date and time of image creation."),
	315:   newExifTagDetails(315, "Artist", FormatUndefined, "Person who created the image."),
	316:   newExifTagDetails(316, "HostComputer", FormatUndefined, "The computer and/or operating system in use at the time of image creation."),
	320:   newExifTagDetails(320, "ColorMap", FormatUndefined, "A color map for palette color images."),
	338:   newExifTagDetails(338, "ExtraSamples", FormatUndefined, "Description of extra components."),
	33432: newExifTagDetails(33432, "Copyright", FormatUndefined, "Copyright notice."),
	269:   newExifTagDetails(269, "DocumentName", FormatUndefined, "The name of the document from which this image was scanned."),
	285:   newExifTagDetails(285, "PageName", FormatUndefined, "The name of the page from which this image was scanned."),
	286:   newExifTagDetails(286, "XPosition", FormatUndefined, "X position of the image."),
	287:   newExifTagDetails(287, "YPosition", FormatUndefined, "Y position of the image."),
	292:   newExifTagDetails(292, "T4Options", FormatUndefined, "Options for Group 3 Fax compression"),
	293:   newExifTagDetails(293, "T6Options", FormatUndefined, "Options for Group 4 Fax compression"),
	297:   newExifTagDetails(297, "PageNumber", FormatUndefined, "The page number of the page from which this image was scanned."),
	301:   newExifTagDetails(301, "TransferFunction", FormatUndefined, "Describes a transfer function for the image in tabular style."),
	317:   newExifTagDetails(317, "Predictor", FormatUndefined, "A mathematical operator that is applied to the image data before an encoding scheme is applied."),
	318:   newExifTagDetails(318, "WhitePoint", FormatUndefined, "The chromaticity of the white point of the image."),
	319:   newExifTagDetails(319, "PrimaryChromaticities", FormatUndefined, "The chromaticities of the primaries of the image."),
	321:   newExifTagDetails(321, "HalftoneHints", FormatUndefined, "Conveys to the halftone function the range of gray levels within a colorimetrically-specified image that should retain tonal detail."),
	322:   newExifTagDetails(322, "TileWidth", FormatUndefined, "The tile width in pixels. This is the number of columns in each tile."),
	323:   newExifTagDetails(323, "TileLength", FormatUndefined, "The tile length (height) in pixels. This is the number of rows in each tile."),
	324:   newExifTagDetails(324, "TileOffsets", FormatUndefined, "For each tile, the byte offset of that tile, as compressed and stored on disk."),
	325:   newExifTagDetails(325, "TileByteCounts", FormatUndefined, "For each tile, the number of (compressed) bytes in that tile."),
	326:   newExifTagDetails(326, "BadFaxLines", FormatUndefined, "Used in the TIFF-F standard, denotes the number of 'bad' scan lines encountered by the facsimile device."),
	327:   newExifTagDetails(327, "CleanFaxData", FormatUndefined, "Used in the TIFF-F standard, indicates if 'bad' lines encountered during reception are stored in the data, or if 'bad' lines have been replaced by the receiver."),
	328:   newExifTagDetails(328, "ConsecutiveBadFaxLines", FormatUndefined, "Used in the TIFF-F standard, denotes the maximum number of consecutive 'bad' scanlines received."),
	330:   newExifTagDetails(330, "SubIFDs", FormatUndefined, "Offset to child IFDs."),
	332:   newExifTagDetails(332, "InkSet", FormatUndefined, "The set of inks used in a separated (PhotometricInterpretation=5) image."),
	333:   newExifTagDetails(333, "InkNames", FormatUndefined, "The name of each ink used in a separated image."),
	334:   newExifTagDetails(334, "NumberOfInks", FormatUndefined, "The number of inks."),
	336:   newExifTagDetails(336, "DotRange", FormatUndefined, "The component values that correspond to a 0% dot and 100% dot."),
	337:   newExifTagDetails(337, "TargetPrinter", FormatUndefined, "A description of the printing environment for which this separation is intended."),
	339:   newExifTagDetails(339, "SampleFormat", FormatUndefined, "Specifies how to interpret each data sample in a pixel."),
	340:   newExifTagDetails(340, "SMinSampleValue", FormatUndefined, "Specifies the minimum sample value."),
	341:   newExifTagDetails(341, "SMaxSampleValue", FormatUndefined, "Specifies the maximum sample value."),
	342:   newExifTagDetails(342, "TransferRange", FormatUndefined, "Expands the range of the TransferFunction."),
	343:   newExifTagDetails(343, "ClipPath", FormatUndefined, "Mirrors the essentials of PostScript's path creation functionality."),
	344:   newExifTagDetails(344, "XClipPathUnits", FormatUndefined, "The number of units that span the width of the image, in terms of integer ClipPath coordinates."),
	345:   newExifTagDetails(345, "YClipPathUnits", FormatUndefined, "The number of units that span the height of the image, in terms of integer ClipPath coordinates."),
	346:   newExifTagDetails(346, "Indexed", FormatUndefined, "Aims to broaden the support for indexed images to include support for any color space."),
	347:   newExifTagDetails(347, "JPEGTables", FormatUndefined, "JPEG quantization and/or Huffman tables."),
	351:   newExifTagDetails(351, "OPIProxy", FormatUndefined, "OPI-related."),
	400:   newExifTagDetails(400, "GlobalParametersIFD", FormatUndefined, "Used in the TIFF-FX standard to point to an IFD containing tags that are globally applicable to the complete TIFF file."),
	401:   newExifTagDetails(401, "ProfileType", FormatUndefined, "Used in the TIFF-FX standard, denotes the type of data stored in this file or IFD."),
	402:   newExifTagDetails(402, "FaxProfile", FormatUndefined, "Used in the TIFF-FX standard, denotes the 'profile' that applies to this file."),
	403:   newExifTagDetails(403, "CodingMethods", FormatUndefined, "Used in the TIFF-FX standard, indicates which coding methods are used in the file."),
	404:   newExifTagDetails(404, "VersionYear", FormatUndefined, "Used in the TIFF-FX standard, denotes the year of the standard specified by the FaxProfile field."),
	405:   newExifTagDetails(405, "ModeNumber", FormatUndefined, "Used in the TIFF-FX standard, denotes the mode of the standard specified by the FaxProfile field."),
	433:   newExifTagDetails(433, "Decode", FormatUndefined, "Used in the TIFF-F and TIFF-FX standards, holds information about the ITULAB (PhotometricInterpretation = 10) encoding."),
	434:   newExifTagDetails(434, "DefaultImageColor", FormatUndefined, "Defined in the Mixed Raster Content part of RFC 2301, is the default color needed in areas where no image is available."),
	512:   newExifTagDetails(512, "JPEGProc", FormatUndefined, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	513:   newExifTagDetails(513, "JPEGInterchangeFormat", FormatUndefined, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	514:   newExifTagDetails(514, "JPEGInterchangeFormatLength", FormatUndefined, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	515:   newExifTagDetails(515, "JPEGRestartInterval", FormatUndefined, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	517:   newExifTagDetails(517, "JPEGLosslessPredictors", FormatUndefined, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	518:   newExifTagDetails(518, "JPEGPointTransforms", FormatUndefined, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	519:   newExifTagDetails(519, "JPEGQTables", FormatUndefined, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	520:   newExifTagDetails(520, "JPEGDCTables", FormatUndefined, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	521:   newExifTagDetails(521, "JPEGACTables", FormatUndefined, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	529:   newExifTagDetails(529, "YCbCrCoefficients", FormatUndefined, "The transformation from RGB to YCbCr image data."),
	530:   newExifTagDetails(530, "YCbCrSubSampling", FormatUndefined, "Specifies the subsampling factors used for the chrominance components of a YCbCr image."),
	531:   newExifTagDetails(531, "YCbCrPositioning", FormatUndefined, "Specifies the positioning of subsampled chrominance components relative to luminance samples."),
	532:   newExifTagDetails(532, "ReferenceBlackWhite", FormatUndefined, "Specifies a pair of headroom and footroom image data values (codes) for each pixel component."),
	559:   newExifTagDetails(559, "StripRowCounts", FormatUndefined, "Defined in the Mixed Raster Content part of RFC 2301, used to replace RowsPerStrip for IFDs with variable-sized strips."),
	700:   newExifTagDetails(700, "XMP", FormatUndefined, "XML packet containing XMP metadata"),
	32781: newExifTagDetails(32781, "ImageID", FormatUndefined, "OPI-related."),
	34732: newExifTagDetails(34732, "ImageLayer", FormatUndefined, "Defined in the Mixed Raster Content part of RFC 2301, used to denote the particular function of this Image in the mixed raster scheme."),
	33434: newExifTagDetails(33434, "ExposureTime", FormatUndefined, "Exposure time (reciprocal of shutter speed). Unit is second. "),
	33437: newExifTagDetails(33437, "FNumber", FormatUndefined, "The actual F-number(F-stop) of lens when the image was taken. "),
	34850: newExifTagDetails(34850, "ExposureProgram", FormatUndefined, "Exposure program that the camera used when image was taken. '1' means manual control, '2' program normal, '3' aperture priority, '4' shutter priority, '5' program creative (slow program), '6' program action(high-speed program), '7' portrait mode, '8' landscape mode. "),
	34855: newExifTagDetails(34855, "ISOSpeedRatings", FormatUndefined, "CCD sensitivity equivalent to Ag-Hr film speedrate. "),
	36880: newExifTagDetails(36800, "OffsetTime", FormatUndefined, "Get time zone from 'Offset Time'"),
	36881: newExifTagDetails(36800, "OffsetTimeOriginal", FormatUndefined, "Get time zone from 'Offset Time Original'"),
	36864: newExifTagDetails(36864, "ExifVersion", FormatUndefined, "Exif version number. Stored as 4bytes of ASCII character (like 0210) "),
	36867: newExifTagDetails(36867, "DateTimeOriginal", FormatUndefined, "Date/Time of original image taken. This value should not be modified by user program. "),
	36868: newExifTagDetails(36868, "DateTimeDigitized", FormatUndefined, "Date/Time of image digitized. Usually, it contains the same value of DateTimeOriginal(0x9003). "),
	37121: newExifTagDetails(37121, "ComponentConfiguration", FormatUndefined, "It seems value 0x00,0x01,0x02,0x03 always. "),
	37122: newExifTagDetails(37122, "CompressedBitsPerPixel", FormatUndefined, "The average compression ratio of JPEG. "),
	37377: newExifTagDetails(37377, "ShutterSpeedValue", FormatUndefined, "Shutter speed. To convert this value to ordinary 'Shutter Speed'; calculate this value's power of 2, then reciprocal. For example, if value is '4', shutter speed is 1/(2^4)=1/16 second. "),
	37378: newExifTagDetails(37378, "ApertureValue", FormatUndefined, "The actual aperture value of lens when the image was taken. To convert this value to ordinary F-number(F-stop), calculate this value's power of root 2 (=1.4142). For example, if value is '5', F-number is 1.4142^5 = F5.6. "),
	37379: newExifTagDetails(37379, "BrightnessValue", FormatUndefined, "Brightness of taken subject, unit is EV. "),
	37380: newExifTagDetails(37380, "ExposureBiasValue", FormatUndefined, "Exposure bias value of taking picture. Unit is EV. "),
	37381: newExifTagDetails(37381, "MaxApertureValue", FormatUndefined, "Maximum aperture value of lens. You can convert to F-number by calculating power of root 2 (same process of ApertureValue(0x9202). "),
	37382: newExifTagDetails(37382, "SubjectDistance", FormatUndefined, "Distance to focus point, unit is meter. "),
	37383: newExifTagDetails(37383, "MeteringMode", FormatUndefined, "Exposure metering method. '1' means average, '2' center weighted average, '3' spot, '4' multi-spot, '5' multi-segment. "),
	37384: newExifTagDetails(37384, "LightSource", FormatUndefined, "Light source, actually this means white balance setting. '0' means auto, '1' daylight, '2' fluorescent, '3' tungsten, '10' flash. "),
	37385: newExifTagDetails(37385, "Flash", FormatUndefined, "'1' means flash was used, '0' means not used. "),
	37386: newExifTagDetails(37386, "FocalLength", FormatUndefined, "Focal length of lens used to take image. Unit is millimeter. "),
	37500: newExifTagDetails(37500, "MakerNote", FormatUndefined, "Maker dependent internal data. Some of maker such as Olympus/Nikon/Sanyo etc. uses IFD format for this area. "),
	37510: newExifTagDetails(37510, "UserComment", FormatString, "Stores user comment. "),
	40960: newExifTagDetails(40960, "FlashPixVersion", FormatUndefined, "Stores FlashPix version. Unknown but 4bytes of ASCII characters '0100' exists. "),
	40961: newExifTagDetails(40961, "ColorSpace", FormatUndefined, "Value is '1'. "),
	40962: newExifTagDetails(40962, "ExifImageWidth", FormatUndefined, "Size of main image. "),
	40963: newExifTagDetails(40963, "ExifImageHeight", FormatUndefined, "ExifImageHeight "),
	40964: newExifTagDetails(40964, "RelatedSoundFile", FormatUndefined, "If this digicam can record audio data with image, shows name of audio data. "),
	40965: newExifTagDetails(40965, "ExifInteroperabilityOffset", FormatUndefined, "Extension of 'ExifR98', detail is unknown. This value is offset to IFD format data. Currently there are 2 directory entries, first one is Tag0x0001, value is 'R98', next is Tag0x0002, value is '0100'. "),
	41486: newExifTagDetails(41486, "FocalPlaneXResolution", FormatUndefined, "CCD's pixel density. "),
	41487: newExifTagDetails(41487, "FocalPlaneYResolution", FormatUndefined, "FocalPlaneYResolution "),
	41488: newExifTagDetails(41488, "FocalPlaneResolutionUnit", FormatUndefined, "Unit of FocalPlaneXResoluton/FocalPlaneYResolution. '1' means no-unit, '2' inch, '3' centimeter. "),
	41495: newExifTagDetails(41495, "SensingMethod", FormatUndefined, "Shows type of image sensor unit. '2' means 1 chip color area sensor, most of all digicam use this type. "),
	41728: newExifTagDetails(41728, "FileSource", FormatUndefined, "Unknown but value is '3'. "),
	41729: newExifTagDetails(41729, "SceneType", FormatUndefined, "Unknown but value is '1'. "),
	37520: newExifTagDetails(41729, "SubsecTime", FormatUndefined, "Used to record fractions of seconds for the DateTime tag"),
	37521: newExifTagDetails(37521, "SubsecTimeOriginal", FormatUndefined, "Used to record fractions of seconds for the DateTimeOriginal tag."),
	37522: newExifTagDetails(37522, "SubsecTimeDigitized", FormatUndefined, "Used to record fractions of seconds for the DateTimeDigitized tag."),
	41986: newExifTagDetails(41986, "ExposureMode", FormatUndefined, "Indicates the exposure mode set when the image was shot."),
	41987: newExifTagDetails(41987, "WhiteBalance", FormatUndefined, "Indicates the white balance mode set when the image was shot."),
	41988: newExifTagDetails(41988, "DigitalZoomRatio", FormatUndefined, "Indicates the digital zoom ratio when the image was shot."),
	41989: newExifTagDetails(41989, "FocalLengthIn35mmFilm", FormatUndefined, "Indicates the equivalent focal length assuming a 35mm film camera, in mm."),
	41990: newExifTagDetails(41990, "SceneCaptureType", FormatUndefined, "Indicates the type of scene that was shot."),
	42016: newExifTagDetails(42016, "ImageUniqueID", FormatUndefined, "Indicates an identifier assigned uniquely to each image"),

	0: newExifTagDetails(0, "GPSVersionID", FormatUndefined, "Indicates the version of GPSInfoIFD."),
	1: newExifTagDetails(1, "GPSLatitudeRef", FormatUndefined, "Indicates whether the latitude is north or south latitude"),
	2: newExifTagDetails(2, "GPSLatitude", FormatUndefined, "Indicates the latitude"),
	3: newExifTagDetails(3, "GPSLongitudeRef", FormatUndefined, "Indicates whether the longitude is east or west longitude."),
	4: newExifTagDetails(4, "GPSLongitude", FormatUndefined, "Indicates the longitude."),
	5: newExifTagDetails(5, "GPSAltitudeRef", FormatUndefined, "Indicates the altitude used as the reference altitude."),
	6: newExifTagDetails(6, "GPSAltitude", FormatUndefined, "Indicates the altitude based on the reference in GPSAltitudeRef."),
}
