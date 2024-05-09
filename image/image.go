package image

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const OfsSOI = 0
const OfsAPP1Marker = 2
const OfsAPP1Size = 4
const OfsExifHeader = 6
const OfsEndianDef = 12
const OfsMainImageOffset = 16

const TiffRecordSize = 12

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
	tagNumber := uint32(walker.BytesToUint(walker.Bytes(2)))
	formatId := uint16(walker.BytesToUint(walker.Bytes(2)))
	itemCount := uint32(walker.BytesToUint(walker.Bytes(4)))
	locationOfDataIsAt := walker.posit
	loc := walker.Bytes(4)

	// Get the tag data from the tagNumber
	tagData := lookUpTagData(tagNumber)
	tagFmt := lookUpTagFormat(formatId)

	if tagFmt.tiffFormat == FormatUndefined {
		tagFmt = lookUpTagFormat(uint16(tagData.validFormats[0]))
	}
	byteCount := itemCount * tagFmt.byteLen
	if byteCount > 4 {
		locationOfDataIsAt = uint32(walker.BytesToUint(loc) + TiffRecordSize)
	}

	return &IFDEntry{
		itemCount:   itemCount,
		location:    loc,
		Value:       "",
		TagData:     tagData,
		tagFormat:   tagFmt,
		DataAddress: locationOfDataIsAt,
		ByteCount:   byteCount,
	}

}

func (p *IFDEntry) Diagnostics(m string) string {
	return fmt.Sprintf("IFD:%s TAG[%d:%s] FORMAT[%s] ITEM_COUNT[%d] LOCATION[%s] VALUE[%s] TAG_DESC[%s]", m, p.TagData.TagNum, p.TagData.Name, p.tagFormat, p.itemCount, bytesToHex(p.location), p.Value, p.TagData.LongDesc)
}

func (p *IFDEntry) Output() string {
	return fmt.Sprintf("%s=%s", p.TagData.Name, p.Value)
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

func NewImage(path string, debug bool, echo bool, sort bool, sel func(*IFDEntry, *Walker) bool) (img *image, err error) {
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
	fil, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer fil.Close()

	walker, err := NewWalker(bufio.NewReader(fil), 1024)
	if err != nil {
		panic(fmt.Sprintf("Failed to read file: %v", err))
	}
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
	return fmt.Sprintf("DEBUG:%s SOI[%s]  APP1 Mark[%s] APP1 Size[%d] FileLen[%d]Name[%s] LittleE[%t] EXIF[%t] OffsetToMainDir[%d] Entries[%d]", m, p.soi, p.app1Marker, p.app1Size, p.walker.data.length, p.name, p.walker.littleE, p.IsExif(), p.mainDirOffset, len(p.IFDdata))
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

func pad(i uint32, n int) string {
	s := fmt.Sprintf("%d", i)
	if len(s) >= n {
		return s
	}
	return "00000000000"[0:n-len(s)] + s
}
