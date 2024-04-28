package image

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const OfsSOI = 0
const OfsAPP1Marker = 2
const OfsAPP1Size = 4
const OfsExifHeader = 6
const OfsTiffEndian = 12
const OfsTiffHeader = 16
const OfsTiffCount = 20
const OfsTiffEntries = 22

type Tag struct {
	id       uint32
	desc     string
	longDesc string
}

type TagFormat struct {
	format TiffFormat
	length uint32
	desc   string
}

func (p *Tag) String() string {
	return fmt.Sprintf("%s: %s", p.desc, p.longDesc)
}

func (p *TagFormat) String() string {
	return fmt.Sprintf("%d:%d:%s", p.format, p.length, p.desc)
}

func newTagFormat(format TiffFormat, desc string, len uint32) *TagFormat {
	if len < 1 {
		len = 1
	}
	return &TagFormat{
		format: format,
		length: len,
		desc:   desc,
	}
}

func newExifTagDetails(id uint32, desc string, longD string) *Tag {
	return &Tag{
		id:       id,
		desc:     desc,
		longDesc: longD,
	}
}

type image struct {
	name       string
	walker     *walker
	sof        string
	exif       bool
	app1Marker string
	app1Size   uint32 // APP1 data size
	IFDOffset  uint32
	IDFBase    uint32
	IFDdata    []*IFDEntry
}

func (p *image) IsExif() bool {
	return p.exif
}

func (p *image) GetValueBytes(ifd *IFDEntry, tt *TagFormat) []byte {
	byteCount := ifd.length * tt.length
	if (byteCount) > 4 {
		// Location is a pointer from the IDFBase
		return p.walker.Pos(uint32(p.walker.BytesToUint(ifd.location)) + p.IDFBase).Bytes(byteCount)
	} else {
		// Location is the value
		return ifd.location
	}
}

func (p *image) GetIDFData(index int) (resp string) {
	defer func() {
		if r := recover(); r != nil {
			resp = fmt.Sprintf("%s", r)
		}
	}()
	if index < 0 || index >= len(p.IFDdata) {
		panic(fmt.Sprintf("GetIDFData out or range: max=%d requested=%d", len(p.IFDdata), index))
	}
	ifd := p.IFDdata[index]
	tiffType := MapTiffFormats[ifd.format]
	bytes := p.GetValueBytes(ifd, tiffType)
	switch tiffType.format {
	case FormatString:
		return string(bytes[0 : len(bytes)-1])
	case FormatUint8:
		return fmt.Sprintf("%d", p.walker.BytesToUint(bytes[0:1]))
	case FormatUint16:
		return fmt.Sprintf("%d", p.walker.BytesToUint(bytes[0:2]))
	case FormatUint32:
		return fmt.Sprintf("%d", p.walker.BytesToUint(bytes[0:4]))
	case FormatURational:
		n := p.walker.BytesToUint(bytes[:4])
		d := p.walker.BytesToUint(bytes[4:])
		return fmt.Sprintf("%d/%d", n, d)
	}
	return p.walker.Hex(bytes[0:ifd.length])
}

func (p *image) String() string {
	return fmt.Sprintf("IMG: SOF[%s]  APP1 Mark[%s] APP1 Size[%d] FileLen[%d] LittleE[%t] Name[%s] EXIF[%t] Entries[%d]", p.sof, p.app1Marker, p.app1Size, p.walker.len, p.walker.littleE, p.name, p.IsExif(), len(p.IFDdata))
}

type IFDEntry struct {
	tag      uint32
	format   uint16
	length   uint32
	location []byte
}

func NewIDFEntry(walker *walker) *IFDEntry {
	return &IFDEntry{
		tag:      uint32(walker.BytesToUint(walker.Bytes(2))),
		format:   uint16(walker.BytesToUint(walker.Bytes(2))),
		length:   uint32(walker.BytesToUint(walker.Bytes(4))),
		location: walker.Bytes(4),
	}
}

func (p *IFDEntry) String() string {
	tf := MapTiffFormats[p.format]
	return fmt.Sprintf("IDF: TAG[%d]  FORMAT[%s] LEN[%d * %d] Location[%s] Desc[%s]", p.tag, tf, p.length, tf.length, fmt.Sprintf("%x", p.location), MapTags[p.tag])
}

type walker struct {
	data    *[]uint8
	pos     uint32
	len     uint32
	littleE bool
}

func NewWalker(b *[]uint8, len uint32) *walker {
	return &walker{
		data:    b,
		pos:     0,
		len:     len,
		littleE: false,
	}
}

func (p *walker) SetLittleE(yes bool) {
	p.littleE = yes
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
	return (strings.EqualFold(p.Zstring(len(s)+2), s))
}

func (p *walker) Bytes(n uint32) []byte {
	b := make([]byte, n)
	for i := 0; uint32(i) < n; i++ {
		b[i] = byte(p.Advance(1))
	}
	return b
}

func (p *walker) Hex(b []byte) string {
	var line bytes.Buffer
	for i := 0; i < len(b); i++ {
		line.WriteString(p.hex8(b[i]))
	}
	return line.String()
}

func (p *walker) hex8(b byte) string {
	s := fmt.Sprintf("%x", b)
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func (p *walker) BytesToUint(b []byte) uint64 {
	if p.littleE {
		return p.bytesToUintLE(b)
	}
	return p.bytesToUintBE(b)
}

func (p *walker) bytesToUintLE(b []byte) uint64 {
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

func (p *walker) bytesToUintBE(b []byte) uint64 {
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

func (p *walker) Char() string {
	b := p.Advance(1)
	if b < 32 {
		return string(rune(1))
	}
	return string(byte(b & 0xFF))
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

func (p *walker) Retard(n uint32) uint32 {
	if p.pos == 0 {
		panic(fmt.Sprintf("Retard past start: Requested=%d", n))
	}
	b := (*p.data)[p.pos]
	p.pos = p.pos - n
	return uint32(b) & 0xFF
}

func (p *walker) Advance(n uint32) uint32 {
	if p.canNotAdvance() {
		panic(fmt.Sprintf("Advanced past end: Max=%d Requested=%d", p.len-1, p.pos))
	}
	b := (*p.data)[p.pos]
	p.pos = p.pos + n
	return uint32(b) & 0xFF
}

func (p *walker) Clone() *walker {
	return &walker{
		data:    p.data,
		pos:     p.pos,
		len:     p.len,
		littleE: p.littleE,
	}
}

func (q *walker) LinePrint(start uint32, count int, lines int) string {
	clone := q.Clone()
	var line bytes.Buffer
	clone.Pos(start)
	for j := 0; j < lines; j++ {
		line.WriteString(pad(uint32(clone.pos), 4))
		line.WriteRune(':')
		line.WriteRune(' ')
		p := clone.pos
		for i := 0; i < count; i++ {
			line.WriteString(clone.Hex(clone.Bytes(1)))
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
			line.WriteString(clone.Hex(clone.Bytes(2)))
			line.WriteString("   ")
		}
		line.WriteString("\n")
	}
	return line.String()
}

func (p *walker) Pos(n uint32) *walker {
	if n >= p.len {
		panic(fmt.Sprintf("Pos was set past end: Max=%d Requested=%d", p.len-1, n))
	}
	p.pos = n
	return p
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
	walker := NewWalker(&byteArray, uint32(len(byteArray)))
	image := &image{
		name:       path,
		walker:     walker,
		sof:        walker.Pos(OfsSOI).Hex(walker.Bytes(2)),
		app1Marker: walker.Pos(OfsAPP1Marker).Hex(walker.Bytes(2)),
		app1Size:   uint32(walker.Pos(OfsAPP1Size).bytesToUintBE(walker.Bytes(2))), // Always Big Endian
		exif:       walker.Pos(OfsExifHeader).ZstringEquals("Exif"),
	}
	image.walker.littleE = (walker.Pos(OfsTiffEndian).ZstringEquals("II*"))
	/*
		The rest of the image data needs to know the littleE setting to work
	*/
	image.IFDOffset = OfsTiffHeader + uint32(walker.Pos(OfsTiffHeader).BytesToUint(walker.Bytes(4))) - 2
	tiffCount := int(walker.Pos(OfsTiffCount).BytesToUint(walker.Bytes(2)))
	if tiffCount <= 0 || tiffCount > 1000 {
		panic(fmt.Sprintf("Image TIFF data count is invalid. Expected[1..200]. Actual=[%d]", tiffCount))
	}
	walker.Pos(image.IFDOffset)
	ents := []*IFDEntry{}
	for i := 0; i < tiffCount; i++ {
		ne := NewIDFEntry(walker)
		if ne.tag == 34665 {
			ofs := walker.BytesToUint(ne.location)
			walker.Pos(image.IFDOffset + uint32((ofs + 4)))
			fmt.Printf("%s\n", walker.LinePrint(walker.pos, 12, 10))
		} else {
			ents = append(ents, ne)
		}
	}
	image.IDFBase = OfsTiffEndian
	image.IFDdata = ents
	return image, nil
}

type TiffFormat uint16

/*
Value 	1 	2 	3 	4 	5 	6
Format 	unsigned byte 	ascii strings 	unsigned short 	unsigned long 	unsigned rational 	signed byte
Bytes/component 	1 	1 	2 	4 	8 	1
Value 	7 	8 	9 	10 	11 	12
Format 	undefined 	signed short 	signed long 	signed rational 	single float 	double float
Bytes/component 	1 	2 	4 	8 	4 	8
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

var MapTiffFormats = map[uint16]*TagFormat{
	1:  newTagFormat(FormatUint8, "Byte Uint8", 1),
	2:  newTagFormat(FormatString, "ASCII String", 1),
	3:  newTagFormat(FormatUint16, "Short Uint16", 2),
	4:  newTagFormat(FormatUint32, "Long Uint32", 4),
	5:  newTagFormat(FormatURational, "n/d URationa", 8),
	6:  newTagFormat(FormatInt8, "Byte Int8", 1),
	7:  newTagFormat(FormatUndefined, "Undefined", 1),
	8:  newTagFormat(FormatInt8, "Short Int8", 1),
	9:  newTagFormat(FormatInt16, "Long Int16", 2),
	10: newTagFormat(FormatRational, "n/d Rational", 8),
	11: newTagFormat(FormatFloat32, "Single Float32", 2),
	12: newTagFormat(FormatFloat64, "Double Float64", 4),
}

var MapTags = map[uint32]*Tag{
	254:   newExifTagDetails(254, "NewSubfileType", "A general indication of the kind of data contained in this subfile."),
	255:   newExifTagDetails(255, "SubfileType", "A general indication of the kind of data contained in this subfile."),
	256:   newExifTagDetails(256, "ImageWidth", "The number of columns in the image, i.e., the number of pixels per row."),
	257:   newExifTagDetails(257, "ImageLength", "The number of rows of pixels in the image."),
	258:   newExifTagDetails(258, "BitsPerSample", "Number of bits per component."),
	259:   newExifTagDetails(259, "Compression", "Compression scheme used on the image data."),
	262:   newExifTagDetails(262, "PhotometricInterpretation", "The color space of the image data."),
	263:   newExifTagDetails(263, "Threshholding", "For black and white TIFF files that represent shades of gray, the technique used to convert from gray to black and white pixels."),
	264:   newExifTagDetails(264, "CellWidth", "The width of the dithering or halftoning matrix used to create a dithered or halftoned bilevel file."),
	265:   newExifTagDetails(265, "CellLength", "The length of the dithering or halftoning matrix used to create a dithered or halftoned bilevel file."),
	266:   newExifTagDetails(266, "FillOrder", "The logical order of bits within a byte."),
	270:   newExifTagDetails(270, "ImageDescription", "A string that describes the subject of the image."),
	271:   newExifTagDetails(271, "Make", "The scanner manufacturer."),
	272:   newExifTagDetails(272, "Model", "The scanner model name or number."),
	273:   newExifTagDetails(273, "StripOffsets", "For each strip, the byte offset of that strip."),
	274:   newExifTagDetails(274, "Orientation", "The orientation of the image with respect to the rows and columns."),
	277:   newExifTagDetails(277, "SamplesPerPixel", "The number of components per pixel."),
	278:   newExifTagDetails(278, "RowsPerStrip", "The number of rows per strip."),
	279:   newExifTagDetails(279, "StripByteCounts", "For each strip, the number of bytes in the strip after compression."),
	280:   newExifTagDetails(280, "MinSampleValue", "The minimum component value used."),
	281:   newExifTagDetails(281, "MaxSampleValue", "The maximum component value used."),
	282:   newExifTagDetails(282, "XResolution", "The number of pixels per ResolutionUnit in the ImageWidth direction."),
	283:   newExifTagDetails(283, "YResolution", "The number of pixels per ResolutionUnit in the ImageLength direction."),
	284:   newExifTagDetails(284, "PlanarConfiguration", "How the components of each pixel are stored."),
	288:   newExifTagDetails(288, "FreeOffsets", "For each string of contiguous unused bytes in a TIFF file, the byte offset of the string."),
	289:   newExifTagDetails(289, "FreeByteCounts", "For each string of contiguous unused bytes in a TIFF file, the number of bytes in the string."),
	290:   newExifTagDetails(290, "GrayResponseUnit", "The precision of the information contained in the GrayResponseCurve."),
	291:   newExifTagDetails(291, "GrayResponseCurve", "For grayscale data, the optical density of each possible pixel value."),
	296:   newExifTagDetails(296, "ResolutionUnit", "The unit of measurement for XResolution and YResolution."),
	305:   newExifTagDetails(305, "Software", "Name and version number of the software package(s) used to create the image."),
	306:   newExifTagDetails(306, "DateTime", "Date and time of image creation."),
	315:   newExifTagDetails(315, "Artist", "Person who created the image."),
	316:   newExifTagDetails(316, "HostComputer", "The computer and/or operating system in use at the time of image creation."),
	320:   newExifTagDetails(320, "ColorMap", "A color map for palette color images."),
	338:   newExifTagDetails(338, "ExtraSamples", "Description of extra components."),
	33432: newExifTagDetails(33432, "Copyright", "Copyright notice."),
	269:   newExifTagDetails(269, "DocumentName", "The name of the document from which this image was scanned."),
	285:   newExifTagDetails(285, "PageName", "The name of the page from which this image was scanned."),
	286:   newExifTagDetails(286, "XPosition", "X position of the image."),
	287:   newExifTagDetails(287, "YPosition", "Y position of the image."),
	292:   newExifTagDetails(292, "T4Options", "Options for Group 3 Fax compression"),
	293:   newExifTagDetails(293, "T6Options", "Options for Group 4 Fax compression"),
	297:   newExifTagDetails(297, "PageNumber", "The page number of the page from which this image was scanned."),
	301:   newExifTagDetails(301, "TransferFunction", "Describes a transfer function for the image in tabular style."),
	317:   newExifTagDetails(317, "Predictor", "A mathematical operator that is applied to the image data before an encoding scheme is applied."),
	318:   newExifTagDetails(318, "WhitePoint", "The chromaticity of the white point of the image."),
	319:   newExifTagDetails(319, "PrimaryChromaticities", "The chromaticities of the primaries of the image."),
	321:   newExifTagDetails(321, "HalftoneHints", "Conveys to the halftone function the range of gray levels within a colorimetrically-specified image that should retain tonal detail."),
	322:   newExifTagDetails(322, "TileWidth", "The tile width in pixels. This is the number of columns in each tile."),
	323:   newExifTagDetails(323, "TileLength", "The tile length (height) in pixels. This is the number of rows in each tile."),
	324:   newExifTagDetails(324, "TileOffsets", "For each tile, the byte offset of that tile, as compressed and stored on disk."),
	325:   newExifTagDetails(325, "TileByteCounts", "For each tile, the number of (compressed) bytes in that tile."),
	326:   newExifTagDetails(326, "BadFaxLines", "Used in the TIFF-F standard, denotes the number of 'bad' scan lines encountered by the facsimile device."),
	327:   newExifTagDetails(327, "CleanFaxData", "Used in the TIFF-F standard, indicates if 'bad' lines encountered during reception are stored in the data, or if 'bad' lines have been replaced by the receiver."),
	328:   newExifTagDetails(328, "ConsecutiveBadFaxLines", "Used in the TIFF-F standard, denotes the maximum number of consecutive 'bad' scanlines received."),
	330:   newExifTagDetails(330, "SubIFDs", "Offset to child IFDs."),
	332:   newExifTagDetails(332, "InkSet", "The set of inks used in a separated (PhotometricInterpretation=5) image."),
	333:   newExifTagDetails(333, "InkNames", "The name of each ink used in a separated image."),
	334:   newExifTagDetails(334, "NumberOfInks", "The number of inks."),
	336:   newExifTagDetails(336, "DotRange", "The component values that correspond to a 0% dot and 100% dot."),
	337:   newExifTagDetails(337, "TargetPrinter", "A description of the printing environment for which this separation is intended."),
	339:   newExifTagDetails(339, "SampleFormat", "Specifies how to interpret each data sample in a pixel."),
	340:   newExifTagDetails(340, "SMinSampleValue", "Specifies the minimum sample value."),
	341:   newExifTagDetails(341, "SMaxSampleValue", "Specifies the maximum sample value."),
	342:   newExifTagDetails(342, "TransferRange", "Expands the range of the TransferFunction."),
	343:   newExifTagDetails(343, "ClipPath", "Mirrors the essentials of PostScript's path creation functionality."),
	344:   newExifTagDetails(344, "XClipPathUnits", "The number of units that span the width of the image, in terms of integer ClipPath coordinates."),
	345:   newExifTagDetails(345, "YClipPathUnits", "The number of units that span the height of the image, in terms of integer ClipPath coordinates."),
	346:   newExifTagDetails(346, "Indexed", "Aims to broaden the support for indexed images to include support for any color space."),
	347:   newExifTagDetails(347, "JPEGTables", "JPEG quantization and/or Huffman tables."),
	351:   newExifTagDetails(351, "OPIProxy", "OPI-related."),
	400:   newExifTagDetails(400, "GlobalParametersIFD", "Used in the TIFF-FX standard to point to an IFD containing tags that are globally applicable to the complete TIFF file."),
	401:   newExifTagDetails(401, "ProfileType", "Used in the TIFF-FX standard, denotes the type of data stored in this file or IFD."),
	402:   newExifTagDetails(402, "FaxProfile", "Used in the TIFF-FX standard, denotes the 'profile' that applies to this file."),
	403:   newExifTagDetails(403, "CodingMethods", "Used in the TIFF-FX standard, indicates which coding methods are used in the file."),
	404:   newExifTagDetails(404, "VersionYear", "Used in the TIFF-FX standard, denotes the year of the standard specified by the FaxProfile field."),
	405:   newExifTagDetails(405, "ModeNumber", "Used in the TIFF-FX standard, denotes the mode of the standard specified by the FaxProfile field."),
	433:   newExifTagDetails(433, "Decode", "Used in the TIFF-F and TIFF-FX standards, holds information about the ITULAB (PhotometricInterpretation = 10) encoding."),
	434:   newExifTagDetails(434, "DefaultImageColor", "Defined in the Mixed Raster Content part of RFC 2301, is the default color needed in areas where no image is available."),
	512:   newExifTagDetails(512, "JPEGProc", "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	513:   newExifTagDetails(513, "JPEGInterchangeFormat", "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	514:   newExifTagDetails(514, "JPEGInterchangeFormatLength", "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	515:   newExifTagDetails(515, "JPEGRestartInterval", "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	517:   newExifTagDetails(517, "JPEGLosslessPredictors", "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	518:   newExifTagDetails(518, "JPEGPointTransforms", "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	519:   newExifTagDetails(519, "JPEGQTables", "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	520:   newExifTagDetails(520, "JPEGDCTables", "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	521:   newExifTagDetails(521, "JPEGACTables", "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	529:   newExifTagDetails(529, "YCbCrCoefficients", "The transformation from RGB to YCbCr image data."),
	530:   newExifTagDetails(530, "YCbCrSubSampling", "Specifies the subsampling factors used for the chrominance components of a YCbCr image."),
	531:   newExifTagDetails(531, "YCbCrPositioning", "Specifies the positioning of subsampled chrominance components relative to luminance samples."),
	532:   newExifTagDetails(532, "ReferenceBlackWhite", "Specifies a pair of headroom and footroom image data values (codes) for each pixel component."),
	559:   newExifTagDetails(559, "StripRowCounts", "Defined in the Mixed Raster Content part of RFC 2301, used to replace RowsPerStrip for IFDs with variable-sized strips."),
	700:   newExifTagDetails(700, "XMP", "XML packet containing XMP metadata"),
	32781: newExifTagDetails(32781, "ImageID", "OPI-related."),
	34732: newExifTagDetails(34732, "ImageLayer", "Defined in the Mixed Raster Content part of RFC 2301, used to denote the particular function of this Image in the mixed raster scheme."),
}
