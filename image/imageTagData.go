package image

import (
	"fmt"
)

const TagExifVersion uint32 = 36864

const TagExifSubIFD uint32 = 34665
const TagGPSIFD uint32 = 34853
const TagInteroperabilityIFD uint32 = 40965

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

type Tag struct {
	IsDir          bool
	TagNum         uint32
	Name           string
	validFormats   []TiffFormat
	countIndicator string
	count          int32
	LongDesc       string
}

func newExifTagData(isDir bool, tag uint32, name string, formats []TiffFormat, countInd string, count int32, longD string) *Tag {
	return &Tag{
		IsDir:          isDir,
		TagNum:         tag,
		Name:           name,
		LongDesc:       longD,
		validFormats:   formats,
		countIndicator: countInd,
		count:          count,
	}
}

func (p *Tag) String() string {
	return fmt.Sprintf("%s: %s", p.Name, p.LongDesc)
}

func (p *Tag) GetItemCount(count uint16) uint16 {
	if p.count < 1 {
		return count
	}
	return uint16(p.count)
}

func (p *Tag) IsFormatValid(tf TiffFormat) bool {
	for _, f := range p.validFormats {
		if f == tf {
			return true
		}
	}
	return false
}

func lookUpTagData(tag uint32) *Tag {
	ta, ok := mapTags[tag]
	if !ok {
		ta = &Tag{
			TagNum:         tag,
			Name:           fmt.Sprintf("Undefined. Tag 0x%4x", tag),
			validFormats:   []TiffFormat{FormatUndefined},
			countIndicator: "N",
			count:          1,
			LongDesc:       "",
		}
	}
	return ta
}

type TagFormat struct {
	tiffFormat TiffFormat
	formatName string
	byteLen    uint32
	desc       string
}

func (p *TagFormat) String() string {
	return fmt.Sprintf("id:%d bytes:%d: type:%s", p.tiffFormat, p.byteLen, p.desc)
}

func newTagFormat(format TiffFormat, formatName string, desc string, byteLen uint32) *TagFormat {
	if byteLen < 1 {
		byteLen = 1
	}
	return &TagFormat{
		tiffFormat: format,
		formatName: formatName,
		byteLen:    byteLen,
		desc:       desc,
	}
}

func lookUpTagFormat(formatId uint16) *TagFormat {
	ta, ok := mapTiffFormats[formatId]
	if !ok {
		return mapTiffFormats[uint16(FormatUndefined)]
	}
	return ta
}

type TiffFormat uint16

/*
Format type to type enum, name and bytes per entry

From the image file data we get a uint16 number which we need to convert to a TagFormat! Ref: lookUpTagFormat..
*/
var mapTiffFormats = map[uint16]*TagFormat{
	1:  newTagFormat(FormatUint8, "FormatUint8", "Byte Uint8", 1),
	2:  newTagFormat(FormatString, "FormatString", "ASCII String", 1),
	3:  newTagFormat(FormatUint16, "FormatUint16", "Short Uint16", 2),
	4:  newTagFormat(FormatUint32, "FormatUint32", "Long Uint32", 4),
	5:  newTagFormat(FormatURational, "FormatURational", "n/d URational", 8),
	6:  newTagFormat(FormatInt8, "FormatInt8", "Byte Int8", 1),
	7:  newTagFormat(FormatUndefined, "FormatUndefined", "Undefined", 1),
	8:  newTagFormat(FormatInt8, "FormatInt8", "Short Int8", 1),
	9:  newTagFormat(FormatInt16, "FormatInt16", "Long Int16", 2),
	10: newTagFormat(FormatRational, "FormatRational", "n/d Rational", 8),
	11: newTagFormat(FormatFloat32, "FormatFloat32", "Single Float32", 2),
	12: newTagFormat(FormatFloat64, "FormatFloat64", "Double Float64", 4),
}

/*
Map of tags to tag,name and a long desc.
*/
var mapTags = map[uint32]*Tag{
	TagExifSubIFD:          newExifTagData(true, TagExifSubIFD, "Exif IFD", []TiffFormat{FormatUndefined}, "4", 4, ""),
	TagInteroperabilityIFD: newExifTagData(true, TagInteroperabilityIFD, "Interoperability IFD", []TiffFormat{FormatUndefined}, "4", 4, "*AltName(ExifInteroperabilityOffset) Extension of 'ExifR98', detail is unknown. This value is offset to IFD format data. Currently there are 2 directory entries, first one is Tag0x0001, value is 'R98', next is Tag0x0002, value is '0100'. "),
	TagGPSIFD:              newExifTagData(true, TagGPSIFD, "GPS IFD", []TiffFormat{FormatUndefined}, "4", 4, ""),

	50829: newExifTagData(false, 50829, "ActiveArea", []TiffFormat{FormatUint32, FormatUint16}, "4", 4, ""),
	50784: newExifTagData(false, 50784, "Alias Layer Metadata", []TiffFormat{FormatString}, "N", -1, ""),
	50727: newExifTagData(false, 50727, "AnalogBalance", []TiffFormat{FormatURational}, "ColorPlanes", -1, ""),
	50738: newExifTagData(false, 50738, "AntiAliasStrength", []TiffFormat{FormatURational}, "1", 1, ""),
	37378: newExifTagData(false, 37378, "ApertureValue", []TiffFormat{FormatURational}, "1", 1, "The actual aperture value of lens when the image was taken. To convert this value to ordinary F-number(F-stop), calculate this value's power of root 2 (=1.4142). For example, if value is '5', F-number is 1.4142^5 = F5.6. "),
	315:   newExifTagData(false, 315, "Artist", []TiffFormat{FormatString}, "N", -1, "Person who created the image."),
	50831: newExifTagData(false, 50831, "AsShotICCProfile", []TiffFormat{FormatUndefined}, "N", -1, ""),
	50728: newExifTagData(false, 50728, "AsShotNeutral", []TiffFormat{FormatUint16, FormatURational}, "ColorPlanes", -1, ""),
	50832: newExifTagData(false, 50832, "AsShotPreProfileMatrix", []TiffFormat{FormatURational, FormatRational}, "3 * ColorPlanes or ColorPlanes * ColorPlanes", -1, ""),
	50934: newExifTagData(false, 50934, "AsShotProfileName", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	50729: newExifTagData(false, 50729, "AsShotWhiteXY", []TiffFormat{FormatURational}, "2", 2, ""),
	326:   newExifTagData(false, 326, "BadFaxLines", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "Used in the TIFF-F standard, denotes the number of 'bad' scan lines encountered by the facsimile device."),
	50730: newExifTagData(false, 50730, "BaselineExposure", []TiffFormat{FormatRational, FormatURational}, "1", 1, ""),
	51109: newExifTagData(false, 51109, "BaselineExposureOffset", []TiffFormat{FormatURational}, "1", 1, ""),
	50731: newExifTagData(false, 50731, "BaselineNoise", []TiffFormat{FormatURational}, "1", 1, ""),
	50732: newExifTagData(false, 50732, "BaselineSharpness", []TiffFormat{FormatURational}, "1", 1, ""),
	50733: newExifTagData(false, 50733, "BayerGreenSplit", []TiffFormat{FormatUint32}, "1", 1, ""),
	50780: newExifTagData(false, 50780, "BestQualityScale", []TiffFormat{FormatURational}, "1", 1, ""),
	258:   newExifTagData(false, 258, "BitsPerSample", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "Number of bits per component."),
	50714: newExifTagData(false, 50714, "BlackLevel", []TiffFormat{FormatUint16, FormatUint32, FormatURational}, "BlackLevelRepeatRows * BlackLevelRepeatCols * SamplesPerPixel", -1, ""),
	50715: newExifTagData(false, 50715, "BlackLevelDeltaH", []TiffFormat{FormatURational, FormatRational}, "ImageWidth", -1, ""),
	50716: newExifTagData(false, 50716, "BlackLevelDeltaV", []TiffFormat{FormatURational, FormatRational}, "ImageLength", -1, ""),
	50713: newExifTagData(false, 50713, "BlackLevelRepeatDim", []TiffFormat{FormatUint16}, "2", 2, ""),
	37379: newExifTagData(false, 37379, "BrightnessValue", []TiffFormat{FormatRational, FormatURational}, "1", 1, "Brightness of taken subject, unit is EV. "),
	50711: newExifTagData(false, 50711, "CFALayout", []TiffFormat{FormatUint16}, "1", 1, ""),
	41730: newExifTagData(false, 41730, "CFAPattern", []TiffFormat{FormatUndefined}, "N", -1, ""),
	50710: newExifTagData(false, 50710, "CFAPlaneColor", []TiffFormat{FormatUint8}, "ColorPlanes", -1, ""),
	50778: newExifTagData(false, 50778, "CalibrationIlluminant1", []TiffFormat{FormatUint16}, "1", 1, ""),
	50779: newExifTagData(false, 50779, "CalibrationIlluminant2", []TiffFormat{FormatUint16}, "1", 1, ""),
	50723: newExifTagData(false, 50723, "CameraCalibration1", []TiffFormat{FormatURational, FormatRational}, "ColorPlanes * ColorPlanes", -1, ""),
	50724: newExifTagData(false, 50724, "CameraCalibration2", []TiffFormat{FormatURational, FormatRational}, "ColorPlanes * ColorPlanes", -1, ""),
	50931: newExifTagData(false, 50931, "CameraCalibrationSignature", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	50735: newExifTagData(false, 50735, "CameraSerialNumber", []TiffFormat{FormatString}, "N", -1, ""),
	265:   newExifTagData(false, 265, "CellLength", []TiffFormat{FormatUint16}, "1", 1, "The length of the dithering or halftoning matrix used to create a dithered or halftoned bilevel file."),
	264:   newExifTagData(false, 264, "CellWidth", []TiffFormat{FormatUint16}, "1", 1, "The width of the dithering or halftoning matrix used to create a dithered or halftoned bilevel file."),
	50737: newExifTagData(false, 50737, "ChromaBlurRadius", []TiffFormat{FormatURational}, "1", 1, ""),
	327:   newExifTagData(false, 327, "CleanFaxData", []TiffFormat{FormatUint16}, "1", 1, "Used in the TIFF-F standard, indicates if 'bad' lines encountered during reception are stored in the data, or if 'bad' lines have been replaced by the receiver."),
	343:   newExifTagData(false, 343, "ClipPath", []TiffFormat{FormatUint8}, "N", -1, "Mirrors the essentials of PostScript's path creation functionality."),
	403:   newExifTagData(false, 403, "CodingMethods", []TiffFormat{FormatUint32}, "1", 1, "Used in the TIFF-FX standard, indicates which coding methods are used in the file."),
	320:   newExifTagData(false, 320, "ColorMap", []TiffFormat{FormatUint16}, "3 * (2**BitsPerSample)", -1, "A color map for palette color images."),
	50721: newExifTagData(false, 50721, "ColorMatrix1", []TiffFormat{FormatURational, FormatRational}, "ColorPlanes * 3", -1, ""),
	50722: newExifTagData(false, 50722, "ColorMatrix2", []TiffFormat{FormatURational, FormatRational}, "ColorPlanes * 3", -1, ""),
	40961: newExifTagData(false, 40961, "ColorSpace", []TiffFormat{FormatUint16}, "1", 1, "Value is '1'. "),
	50879: newExifTagData(false, 50879, "ColorimetricReference", []TiffFormat{FormatUint16}, "1", 1, ""),
	37121: newExifTagData(false, 37121, "ComponentsConfiguration", []TiffFormat{FormatUndefined}, "4", 4, "*AltName(ComponentConfiguration) It seems value 0x00,0x01,0x02,0x03 always. "),
	37122: newExifTagData(false, 37122, "CompressedBitsPerPixel", []TiffFormat{FormatURational}, "1", 1, "The average compression ratio of JPEG. "),
	259:   newExifTagData(false, 259, "Compression", []TiffFormat{FormatUint16}, "1", 1, "Compression scheme used on the image data."),
	328:   newExifTagData(false, 328, "ConsecutiveBadFaxLines", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "Used in the TIFF-F standard, denotes the maximum number of consecutive 'bad' scanlines received."),
	41992: newExifTagData(false, 41992, "Contrast", []TiffFormat{FormatUint16}, "1", 1, ""),
	33432: newExifTagData(false, 33432, "Copyright", []TiffFormat{FormatString}, "N", -1, "Copyright notice."),
	50833: newExifTagData(false, 50833, "CurrentICCProfile", []TiffFormat{FormatUndefined}, "N", -1, ""),
	50834: newExifTagData(false, 50834, "CurrentPreProfileMatrix", []TiffFormat{FormatURational, FormatRational}, "3 * ColorPlanes or ColorPlanes * ColorPlanes", -1, ""),
	41985: newExifTagData(false, 41985, "CustomRendered", []TiffFormat{FormatUint16}, "1", 1, ""),
	50707: newExifTagData(false, 50707, "DNGBackwardVersion", []TiffFormat{FormatUint8}, "4", 4, ""),
	50740: newExifTagData(false, 50740, "DNGPrivateData", []TiffFormat{FormatUint8}, "N", -1, ""),
	50706: newExifTagData(false, 50706, "DNGVersion", []TiffFormat{FormatUint8}, "4", 4, ""),
	306:   newExifTagData(false, 306, "DateTime", []TiffFormat{FormatString}, "20", 20, "Date and time of image creation."),
	36868: newExifTagData(false, 36868, "DateTimeDigitized", []TiffFormat{FormatString}, "20", 20, "Date/Time of image digitized. Usually, it contains the same value of DateTimeOriginal(0x9003). "),
	36867: newExifTagData(false, 36867, "DateTimeOriginal", []TiffFormat{FormatString}, "20", 20, "Date/Time of original image taken. This value should not be modified by user program. "),
	433:   newExifTagData(false, 433, "Decode", []TiffFormat{FormatURational, FormatRational}, "2 * SamplesPerPixel (= 6, for ITULAB)", -1, "Used in the TIFF-F and TIFF-FX standards, holds information about the ITULAB (PhotometricInterpretation = 10) encoding."),
	51110: newExifTagData(false, 51110, "DefaultBlackRender", []TiffFormat{FormatUint32}, "1", 1, ""),
	50719: newExifTagData(false, 50719, "DefaultCropOrigin", []TiffFormat{FormatUint32, FormatURational, FormatUint16}, "2", 2, ""),
	50720: newExifTagData(false, 50720, "DefaultCropSize", []TiffFormat{FormatUint16, FormatUint32, FormatURational}, "2", 2, ""),
	434:   newExifTagData(false, 434, "DefaultImageColor", []TiffFormat{FormatUint16}, "SamplesPerPixel", -1, "Defined in the Mixed Raster Content part of RFC 2301, is the default color needed in areas where no image is available."),
	50718: newExifTagData(false, 50718, "DefaultScale", []TiffFormat{FormatURational}, "2", 2, ""),
	51125: newExifTagData(false, 51125, "DefaultUserCrop", []TiffFormat{FormatURational}, "4", 4, ""),
	41995: newExifTagData(false, 41995, "DeviceSettingDescription", []TiffFormat{FormatUndefined}, "N", -1, ""),
	41988: newExifTagData(false, 41988, "DigitalZoomRatio", []TiffFormat{FormatURational}, "1", 1, "Indicates the digital zoom ratio when the image was shot."),
	269:   newExifTagData(false, 269, "DocumentName", []TiffFormat{FormatString}, "N", -1, "The name of the document from which this image was scanned."),
	336:   newExifTagData(false, 336, "DotRange", []TiffFormat{FormatUint8, FormatUint16}, "2, or 2*SamplesPerPixel", -1, "The component values that correspond to a 0% dot and 100% dot."),
	36864: newExifTagData(false, 36864, "ExifVersion", []TiffFormat{FormatUndefined}, "4", 4, "Exif version number. Stored as 4bytes of ASCII character (like 0210) "),
	37380: newExifTagData(false, 37380, "ExposureBiasValue", []TiffFormat{FormatURational, FormatRational}, "1", 1, "Exposure bias value of taking picture. Unit is EV. "),
	41493: newExifTagData(false, 41493, "ExposureIndex", []TiffFormat{FormatURational}, "1", 1, ""),
	41986: newExifTagData(false, 41986, "ExposureMode", []TiffFormat{FormatUint16}, "1", 1, "Indicates the exposure mode set when the image was shot."),
	34850: newExifTagData(false, 34850, "ExposureProgram", []TiffFormat{FormatUint16}, "1", 1, "Exposure program that the camera used when image was taken. '1' means manual control, '2' program normal, '3' aperture priority, '4' shutter priority, '5' program creative (slow program), '6' program action(high-speed program), '7' portrait mode, '8' landscape mode. "),
	33434: newExifTagData(false, 33434, "ExposureTime", []TiffFormat{FormatURational}, "1", 1, "Exposure time (reciprocal of shutter speed). Unit is second. "),
	50933: newExifTagData(false, 50933, "ExtraCameraProfiles", []TiffFormat{FormatUint32}, "Number of extra camera profiles", -1, ""),
	338:   newExifTagData(false, 338, "ExtraSamples", []TiffFormat{FormatUint16}, "N", -1, "Description of extra components."),
	33437: newExifTagData(false, 33437, "FNumber", []TiffFormat{FormatURational}, "1", 1, "The actual F-number(F-stop) of lens when the image was taken. "),
	402:   newExifTagData(false, 402, "FaxProfile", []TiffFormat{FormatUint8}, "1", 1, "Used in the TIFF-FX standard, denotes the 'profile' that applies to this file."),
	41728: newExifTagData(false, 41728, "FileSource", []TiffFormat{FormatUndefined}, "1", 1, "Unknown but value is '3'. "),
	266:   newExifTagData(false, 266, "FillOrder", []TiffFormat{FormatUint16}, "1", 1, "The logical order of bits within a byte."),
	37385: newExifTagData(false, 37385, "Flash", []TiffFormat{FormatUint16}, "1", 1, "'1' means flash was used, '0' means not used. "),
	41483: newExifTagData(false, 41483, "FlashEnergy", []TiffFormat{FormatURational}, "1", 1, ""),
	40960: newExifTagData(false, 40960, "FlashpixVersion", []TiffFormat{FormatUndefined}, "4", 4, "*AltName(FlashPixVersion) Stores FlashPix version. Unknown but 4bytes of ASCII characters '0100' exists. "),
	37386: newExifTagData(false, 37386, "FocalLength", []TiffFormat{FormatURational}, "1", 1, "Focal length of lens used to take image. Unit is millimeter. "),
	41989: newExifTagData(false, 41989, "FocalLengthIn35mmFilm", []TiffFormat{FormatUint16}, "1", 1, "Indicates the equivalent focal length assuming a 35mm film camera, in mm."),
	41488: newExifTagData(false, 41488, "FocalPlaneResolutionUnit", []TiffFormat{FormatUint16}, "1", 1, "Unit of FocalPlaneXResoluton/FocalPlaneYResolution. '1' means no-unit, '2' inch, '3' centimeter. "),
	41486: newExifTagData(false, 41486, "FocalPlaneXResolution", []TiffFormat{FormatURational}, "1", 1, "CCD's pixel density. "),
	41487: newExifTagData(false, 41487, "FocalPlaneYResolution", []TiffFormat{FormatURational}, "1", 1, "FocalPlaneYResolution "),
	50964: newExifTagData(false, 50964, "ForwardMatrix1", []TiffFormat{FormatURational, FormatRational}, "3 * ColorPlanes", -1, ""),
	50965: newExifTagData(false, 50965, "ForwardMatrix2", []TiffFormat{FormatURational, FormatRational}, "3 * ColorPlanes", -1, ""),
	289:   newExifTagData(false, 289, "FreeByteCounts", []TiffFormat{FormatUint32}, "N", -1, "For each string of contiguous unused bytes in a TIFF file, the number of bytes in the string."),
	288:   newExifTagData(false, 288, "FreeOffsets", []TiffFormat{FormatUint32}, "N", -1, "For each string of contiguous unused bytes in a TIFF file, the byte offset of the string."),
	42112: newExifTagData(false, 42112, "GDAL_METADATA", []TiffFormat{FormatString}, "N", -1, ""),
	42113: newExifTagData(false, 42113, "GDAL_NODATA", []TiffFormat{FormatString}, "N", -1, ""),
	6:     newExifTagData(false, 6, "GPSAltitude", []TiffFormat{FormatURational}, "1", 1, "Indicates the altitude based on the reference in GPSAltitudeRef."),
	5:     newExifTagData(false, 5, "GPSAltitudeRef", []TiffFormat{FormatUint8}, "1", 1, "Indicates the altitude used as the reference altitude."),
	28:    newExifTagData(false, 28, "GPSAreaInformation", []TiffFormat{FormatUndefined}, "N", -1, ""),
	11:    newExifTagData(false, 11, "GPSDOP", []TiffFormat{FormatURational}, "1", 1, ""),
	29:    newExifTagData(false, 29, "GPSDateStamp", []TiffFormat{FormatString}, "11", 11, ""),
	24:    newExifTagData(false, 24, "GPSDestBearing", []TiffFormat{FormatURational}, "1", 1, ""),
	23:    newExifTagData(false, 23, "GPSDestBearingRef", []TiffFormat{FormatString}, "2", 2, ""),
	26:    newExifTagData(false, 26, "GPSDestDistance", []TiffFormat{FormatURational}, "1", 1, ""),
	25:    newExifTagData(false, 25, "GPSDestDistanceRef", []TiffFormat{FormatString}, "2", 2, ""),
	20:    newExifTagData(false, 20, "GPSDestLatitude", []TiffFormat{FormatURational}, "3", 3, ""),
	19:    newExifTagData(false, 19, "GPSDestLatitudeRef", []TiffFormat{FormatString}, "2", 2, ""),
	22:    newExifTagData(false, 22, "GPSDestLongitude", []TiffFormat{FormatURational}, "3", 3, ""),
	21:    newExifTagData(false, 21, "GPSDestLongitudeRef", []TiffFormat{FormatString}, "2", 2, ""),
	30:    newExifTagData(false, 30, "GPSDifferential", []TiffFormat{FormatUint16}, "1", 1, ""),
	17:    newExifTagData(false, 17, "GPSImgDirection", []TiffFormat{FormatURational}, "1", 1, ""),
	16:    newExifTagData(false, 16, "GPSImgDirectionRef", []TiffFormat{FormatString}, "2", 2, ""),
	2:     newExifTagData(false, 2, "GPSLatitude", []TiffFormat{FormatURational}, "3", 3, "Indicates the latitude"),
	1:     newExifTagData(false, 1, "GPSLatitudeRef", []TiffFormat{FormatString}, "2", 2, "Indicates whether the latitude is north or south latitude"),
	4:     newExifTagData(false, 4, "GPSLongitude", []TiffFormat{FormatURational}, "3", 3, "Indicates the longitude."),
	3:     newExifTagData(false, 3, "GPSLongitudeRef", []TiffFormat{FormatString}, "2", 2, "Indicates whether the longitude is east or west longitude."),
	18:    newExifTagData(false, 18, "GPSMapDatum", []TiffFormat{FormatString}, "N", -1, ""),
	10:    newExifTagData(false, 10, "GPSMeasureMode", []TiffFormat{FormatString}, "2", 2, ""),
	27:    newExifTagData(false, 27, "GPSProcessingMethod", []TiffFormat{FormatUndefined}, "N", -1, ""),
	8:     newExifTagData(false, 8, "GPSSatellites", []TiffFormat{FormatString}, "N", -1, ""),
	13:    newExifTagData(false, 13, "GPSSpeed", []TiffFormat{FormatURational}, "1", 1, ""),
	12:    newExifTagData(false, 12, "GPSSpeedRef", []TiffFormat{FormatString}, "2", 2, ""),
	9:     newExifTagData(false, 9, "GPSStatus", []TiffFormat{FormatString}, "2", 2, ""),
	7:     newExifTagData(false, 7, "GPSTimeStamp", []TiffFormat{FormatURational}, "3", 3, ""),
	15:    newExifTagData(false, 15, "GPSTrack", []TiffFormat{FormatURational}, "1", 1, ""),
	14:    newExifTagData(false, 14, "GPSTrackRef", []TiffFormat{FormatString}, "2", 2, ""),
	0:     newExifTagData(false, 0, "GPSVersionID", []TiffFormat{FormatUint8}, "4", 4, "Indicates the version of GPSInfoIFD."),
	41991: newExifTagData(false, 41991, "GainControl", []TiffFormat{FormatUint16}, "1", 1, ""),
	34737: newExifTagData(false, 34737, "GeoAsciiParamsTag", []TiffFormat{FormatString}, "N", -1, ""),
	34736: newExifTagData(false, 34736, "GeoDoubleParamsTag", []TiffFormat{FormatFloat64}, "N", -1, ""),
	34735: newExifTagData(false, 34735, "GeoKeyDirectoryTag", []TiffFormat{FormatUint16}, "N &gt;= 4", -1, ""),
	400:   newExifTagData(false, 400, "GlobalParametersIFD", []TiffFormat{FormatUint32, FormatString}, "1", 1, "Used in the TIFF-FX standard to point to an IFD containing tags that are globally applicable to the complete TIFF file."),
	291:   newExifTagData(false, 291, "GrayResponseCurve", []TiffFormat{FormatUint16}, "2**BitsPerSample", -1, "For grayscale data, the optical density of each possible pixel value."),
	290:   newExifTagData(false, 290, "GrayResponseUnit", []TiffFormat{FormatUint16}, "1", 1, "The precision of the information contained in the GrayResponseCurve."),
	321:   newExifTagData(false, 321, "HalftoneHints", []TiffFormat{FormatUint16}, "2", 2, "Conveys to the halftone function the range of gray levels within a colorimetrically-specified image that should retain tonal detail."),
	316:   newExifTagData(false, 316, "HostComputer", []TiffFormat{FormatString}, "N", -1, "The computer and/or operating system in use at the time of image creation."),
	34908: newExifTagData(false, 34908, "HylaFAX FaxRecvParams", []TiffFormat{FormatUint32}, "1", 1, ""),
	34910: newExifTagData(false, 34910, "HylaFAX FaxRecvTime", []TiffFormat{FormatUint32}, "1", 1, ""),
	34909: newExifTagData(false, 34909, "HylaFAX FaxSubAddress", []TiffFormat{FormatString}, "N", -1, ""),
	33919: newExifTagData(false, 33919, "INGR Flag Registers", []TiffFormat{FormatUint32}, "16", 16, ""),
	33918: newExifTagData(false, 33918, "INGR Packet Data Tag", []TiffFormat{FormatUint16}, "N", -1, ""),
	34855: newExifTagData(false, 34855, "ISOSpeedRatings", []TiffFormat{FormatUint16}, "N", -1, "CCD sensitivity equivalent to Ag-Hr film speedrate. "),
	270:   newExifTagData(false, 270, "ImageDescription", []TiffFormat{FormatString}, "N", -1, "A string that describes the subject of the image."),
	32781: newExifTagData(false, 32781, "ImageID", []TiffFormat{FormatString}, "N", -1, "OPI-related."),
	34732: newExifTagData(false, 34732, "ImageLayer", []TiffFormat{FormatUint16, FormatUint32}, "2", 2, "Defined in the Mixed Raster Content part of RFC 2301, used to denote the particular function of this Image in the mixed raster scheme."),
	257:   newExifTagData(false, 257, "ImageLength", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "The number of rows of pixels in the image."),
	37724: newExifTagData(false, 37724, "ImageSourceData", []TiffFormat{FormatUndefined}, "N", -1, ""),
	42016: newExifTagData(false, 42016, "ImageUniqueID", []TiffFormat{FormatString}, "33", 33, "Indicates an identifier assigned uniquely to each image"),
	256:   newExifTagData(false, 256, "ImageWidth", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "The number of columns in the image, i.e., the number of pixels per row."),
	346:   newExifTagData(false, 346, "Indexed", []TiffFormat{FormatUint16}, "1", 1, "Aims to broaden the support for indexed images to include support for any color space."),
	333:   newExifTagData(false, 333, "InkNames", []TiffFormat{FormatString}, "N = total number of characters in all the ink name strings, including the NULs", -1, "The name of each ink used in a separated image."),
	332:   newExifTagData(false, 332, "InkSet", []TiffFormat{FormatUint16}, "1", 1, "The set of inks used in a separated (PhotometricInterpretation=5) image."),
	33920: newExifTagData(false, 33920, "IrasB Transformation Matrix", []TiffFormat{FormatFloat64}, "17 (possibly 16, but unlikely)", -1, ""),
	521:   newExifTagData(false, 521, "JPEGACTables", []TiffFormat{FormatUint32}, "N = SamplesPerPixel", -1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	520:   newExifTagData(false, 520, "JPEGDCTables", []TiffFormat{FormatUint32}, "N = SamplesPerPixel", -1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	513:   newExifTagData(false, 513, "JPEGInterchangeFormat", []TiffFormat{FormatUint32}, "1", 1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	514:   newExifTagData(false, 514, "JPEGInterchangeFormatLength", []TiffFormat{FormatUint32}, "1", 1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	517:   newExifTagData(false, 517, "JPEGLosslessPredictors", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	518:   newExifTagData(false, 518, "JPEGPointTransforms", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	512:   newExifTagData(false, 512, "JPEGProc", []TiffFormat{FormatUint16}, "1", 1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	519:   newExifTagData(false, 519, "JPEGQTables", []TiffFormat{FormatUint32}, "N = SamplesPerPixel", -1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	515:   newExifTagData(false, 515, "JPEGRestartInterval", []TiffFormat{FormatUint16}, "1", 1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	347:   newExifTagData(false, 347, "JPEGTables", []TiffFormat{FormatUndefined}, "N = number of bytes in tables datastream", -1, "JPEG quantization and/or Huffman tables."),
	50736: newExifTagData(false, 50736, "LensInfo", []TiffFormat{FormatURational}, "4", 4, ""),
	37384: newExifTagData(false, 37384, "LightSource", []TiffFormat{FormatUint16}, "1", 1, "Light source, actually this means white balance setting. '0' means auto, '1' daylight, '2' fluorescent, '3' tungsten, '10' flash. "),
	50734: newExifTagData(false, 50734, "LinearResponseLimit", []TiffFormat{FormatURational}, "1", 1, ""),
	50712: newExifTagData(false, 50712, "LinearizationTable", []TiffFormat{FormatUint16}, "N", -1, ""),
	50709: newExifTagData(false, 50709, "LocalizedCameraModel", []TiffFormat{FormatString, FormatUint8}, "N", -1, ""),
	33447: newExifTagData(false, 33447, "MD ColorTable", []TiffFormat{FormatUint16}, "n", -1, ""),
	33445: newExifTagData(false, 33445, "MD FileTag", []TiffFormat{FormatUint32}, "1", 1, ""),
	33452: newExifTagData(false, 33452, "MD FileUnits", []TiffFormat{FormatString}, "N", -1, ""),
	33448: newExifTagData(false, 33448, "MD LabName", []TiffFormat{FormatString}, "n", -1, ""),
	33450: newExifTagData(false, 33450, "MD PrepDate", []TiffFormat{FormatString}, "n", -1, ""),
	33451: newExifTagData(false, 33451, "MD PrepTime", []TiffFormat{FormatString}, "N", -1, ""),
	33449: newExifTagData(false, 33449, "MD SampleInfo", []TiffFormat{FormatString}, "N", -1, ""),
	33446: newExifTagData(false, 33446, "MD ScalePixel", []TiffFormat{FormatURational}, "1", 1, ""),
	271:   newExifTagData(false, 271, "Make", []TiffFormat{FormatString}, "N", -1, "The scanner manufacturer."),
	37500: newExifTagData(false, 37500, "MakerNote", []TiffFormat{FormatUndefined}, "N", -1, "Maker dependent internal data. Some of maker such as Olympus/Nikon/Sanyo etc. uses IFD format for this area. "),
	50741: newExifTagData(false, 50741, "MakerNoteSafety", []TiffFormat{FormatUint16}, "1", 1, ""),
	50830: newExifTagData(false, 50830, "MaskedAreas", []TiffFormat{FormatUint16, FormatUint32}, "4 * number of rectangles", -1, ""),
	37381: newExifTagData(false, 37381, "MaxApertureValue", []TiffFormat{FormatURational}, "1", 1, "Maximum aperture value of lens. You can convert to F-number by calculating power of root 2 (same process of ApertureValue(0x9202). "),
	281:   newExifTagData(false, 281, "MaxSampleValue", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "The maximum component value used."),
	37383: newExifTagData(false, 37383, "MeteringMode", []TiffFormat{FormatUint16}, "1", 1, "Exposure metering method. '1' means average, '2' center weighted average, '3' spot, '4' multi-spot, '5' multi-segment. "),
	280:   newExifTagData(false, 280, "MinSampleValue", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "The minimum component value used."),
	405:   newExifTagData(false, 405, "ModeNumber", []TiffFormat{FormatUint8}, "1", 1, "Used in the TIFF-FX standard, denotes the mode of the standard specified by the FaxProfile field."),
	272:   newExifTagData(false, 272, "Model", []TiffFormat{FormatString}, "N", -1, "The scanner model name or number."),
	33550: newExifTagData(false, 33550, "ModelPixelScaleTag", []TiffFormat{FormatFloat64}, "3", 3, ""),
	33922: newExifTagData(false, 33922, "ModelTiepointTag", []TiffFormat{FormatFloat64}, "N = 6*K, with K = number of tiepoints", -1, ""),
	34264: newExifTagData(false, 34264, "ModelTransformationTag", []TiffFormat{FormatFloat64}, "16", 16, ""),
	51111: newExifTagData(false, 51111, "NewRawImageDigest", []TiffFormat{FormatUint8}, "16", 16, ""),
	254:   newExifTagData(false, 254, "NewSubfileType", []TiffFormat{FormatUint32}, "1", 1, "A general indication of the kind of data contained in this subfile."),
	51041: newExifTagData(false, 51041, "NoiseProfile", []TiffFormat{FormatFloat64}, "2 or 2 * ColorPlanes", -1, ""),
	50935: newExifTagData(false, 50935, "NoiseReductionApplied", []TiffFormat{FormatURational}, "1", 1, ""),
	334:   newExifTagData(false, 334, "NumberOfInks", []TiffFormat{FormatUint16}, "1", 1, "The number of inks."),
	34856: newExifTagData(false, 34856, "OECF", []TiffFormat{FormatUndefined}, "N", -1, ""),
	351:   newExifTagData(false, 351, "OPIProxy", []TiffFormat{FormatUint16}, "1", 1, "OPI-related."),
	50216: newExifTagData(false, 50216, "Oce Application Selector", []TiffFormat{FormatString}, "N", -1, ""),
	50217: newExifTagData(false, 50217, "Oce Identification Number", []TiffFormat{FormatString}, "N", -1, ""),
	50218: newExifTagData(false, 50218, "Oce ImageLogic Characteristics", []TiffFormat{FormatString}, "N", -1, ""),
	50215: newExifTagData(false, 50215, "Oce Scanjob Description", []TiffFormat{FormatString}, "N", -1, ""),
	51008: newExifTagData(false, 51008, "OpcodeList1", []TiffFormat{FormatUndefined}, "N", -1, ""),
	51009: newExifTagData(false, 51009, "OpcodeList2", []TiffFormat{FormatUndefined}, "N", -1, ""),
	51022: newExifTagData(false, 51022, "OpcodeList3", []TiffFormat{FormatUndefined}, "N", -1, ""),
	274:   newExifTagData(false, 274, "Orientation", []TiffFormat{FormatUint16}, "1", 1, "The orientation of the image with respect to the rows and columns."),
	51090: newExifTagData(false, 51090, "OriginalBestQualityFinalSize", []TiffFormat{FormatUint16, FormatUint32}, "2", 2, ""),
	51091: newExifTagData(false, 51091, "OriginalDefaultCropSize", []TiffFormat{FormatURational, FormatUint16, FormatUint32}, "2", 2, ""),
	51089: newExifTagData(false, 51089, "OriginalDefaultFinalSize", []TiffFormat{FormatUint16, FormatUint32}, "2", 2, ""),
	50828: newExifTagData(false, 50828, "OriginalRawFileData", []TiffFormat{FormatUndefined}, "N", -1, ""),
	50973: newExifTagData(false, 50973, "OriginalRawFileDigest", []TiffFormat{FormatUint8}, "16", 16, ""),
	50827: newExifTagData(false, 50827, "OriginalRawFileName", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	285:   newExifTagData(false, 285, "PageName", []TiffFormat{FormatString}, "N", -1, "The name of the page from which this image was scanned."),
	297:   newExifTagData(false, 297, "PageNumber", []TiffFormat{FormatUint16}, "2", 2, "The page number of the page from which this image was scanned."),
	262:   newExifTagData(false, 262, "PhotometricInterpretation", []TiffFormat{FormatUint16}, "1", 1, "The color space of the image data."),
	34377: newExifTagData(false, 34377, "Photoshop", []TiffFormat{FormatUint8}, "N", -1, ""),
	40962: newExifTagData(false, 40962, "PixelXDimension", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "*AltName(ExifImageWidth) Size of main image. "),
	40963: newExifTagData(false, 40963, "PixelYDimension", []TiffFormat{FormatUint32, FormatUint16}, "1", 1, "*AltName(ExifImageHeight) ExifImageHeight "),
	284:   newExifTagData(false, 284, "PlanarConfiguration", []TiffFormat{FormatUint16}, "1", 1, "How the components of each pixel are stored."),
	317:   newExifTagData(false, 317, "Predictor", []TiffFormat{FormatUint16}, "1", 1, "A mathematical operator that is applied to the image data before an encoding scheme is applied."),
	50966: newExifTagData(false, 50966, "PreviewApplicationName", []TiffFormat{FormatString, FormatUint8}, "N", -1, ""),
	50967: newExifTagData(false, 50967, "PreviewApplicationVersion", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	50970: newExifTagData(false, 50970, "PreviewColorSpace", []TiffFormat{FormatUint32}, "1", 1, ""),
	50971: newExifTagData(false, 50971, "PreviewDateTime", []TiffFormat{FormatString}, "N", -1, ""),
	50969: newExifTagData(false, 50969, "PreviewSettingsDigest", []TiffFormat{FormatUint8}, "16", 16, ""),
	50968: newExifTagData(false, 50968, "PreviewSettingsName", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	319:   newExifTagData(false, 319, "PrimaryChromaticities", []TiffFormat{FormatURational}, "6", 6, "The chromaticities of the primaries of the image."),
	50932: newExifTagData(false, 50932, "ProfileCalibrationSignature", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	50942: newExifTagData(false, 50942, "ProfileCopyright", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	50941: newExifTagData(false, 50941, "ProfileEmbedPolicy", []TiffFormat{FormatUint32}, "1", 1, ""),
	50938: newExifTagData(false, 50938, "ProfileHueSatMapData1", []TiffFormat{FormatFloat32}, "HueDivisions * SaturationDivisions * ValueDivisions * 3", -1, ""),
	50939: newExifTagData(false, 50939, "ProfileHueSatMapData2", []TiffFormat{FormatFloat32}, "HueDivisions * SaturationDivisions * ValueDivisions * 3", -1, ""),
	50937: newExifTagData(false, 50937, "ProfileHueSatMapDims", []TiffFormat{FormatUint32}, "3", 3, ""),
	51107: newExifTagData(false, 51107, "ProfileHueSatMapEncoding", []TiffFormat{FormatUint32}, "1", 1, ""),
	50982: newExifTagData(false, 50982, "ProfileLookTableData", []TiffFormat{FormatFloat32}, "HueDivisions * SaturationDivisions * ValueDivisions * 3", -1, ""),
	50981: newExifTagData(false, 50981, "ProfileLookTableDims", []TiffFormat{FormatUint32}, "3", 3, ""),
	51108: newExifTagData(false, 51108, "ProfileLookTableEncoding", []TiffFormat{FormatUint32}, "1", 1, ""),
	50936: newExifTagData(false, 50936, "ProfileName", []TiffFormat{FormatString, FormatUint8}, "N", -1, ""),
	50940: newExifTagData(false, 50940, "ProfileToneCurve", []TiffFormat{FormatFloat32}, "Samples * 2", -1, ""),
	401:   newExifTagData(false, 401, "ProfileType", []TiffFormat{FormatUint32}, "1", 1, "Used in the TIFF-FX standard, denotes the type of data stored in this file or IFD."),
	50781: newExifTagData(false, 50781, "RawDataUniqueID", []TiffFormat{FormatUint8}, "16", 16, ""),
	50972: newExifTagData(false, 50972, "RawImageDigest", []TiffFormat{FormatUint8}, "16", 16, ""),
	51112: newExifTagData(false, 51112, "RawToPreviewGain", []TiffFormat{FormatFloat64}, "1", 1, ""),
	50725: newExifTagData(false, 50725, "ReductionMatrix1", []TiffFormat{FormatURational, FormatRational}, "3 * ColorPlanes", -1, ""),
	50726: newExifTagData(false, 50726, "ReductionMatrix2", []TiffFormat{FormatRational, FormatURational}, "3 * ColorPlanes", -1, ""),
	532:   newExifTagData(false, 532, "ReferenceBlackWhite", []TiffFormat{FormatURational}, "6", 6, "Specifies a pair of headroom and footroom image data values (codes) for each pixel component."),
	40964: newExifTagData(false, 40964, "RelatedSoundFile", []TiffFormat{FormatString}, "13", 13, "If this digicam can record audio data with image, shows name of audio data. "),
	296:   newExifTagData(false, 296, "ResolutionUnit", []TiffFormat{FormatUint16}, "1", 1, "The unit of measurement for XResolution and YResolution."),
	50975: newExifTagData(false, 50975, "RowInterleaveFactor", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, ""),
	278:   newExifTagData(false, 278, "RowsPerStrip", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "The number of rows per strip."),
	341:   newExifTagData(false, 341, "SMaxSampleValue", []TiffFormat{FormatUint16, FormatUint32, FormatURational, FormatUint8, FormatFloat64}, "N = SamplesPerPixel", -1, "Specifies the maximum sample value."),
	340:   newExifTagData(false, 340, "SMinSampleValue", []TiffFormat{FormatFloat64, FormatUint16, FormatUint32, FormatURational, FormatUint8}, "N = SamplesPerPixel", -1, "Specifies the minimum sample value."),
	339:   newExifTagData(false, 339, "SampleFormat", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "Specifies how to interpret each data sample in a pixel."),
	277:   newExifTagData(false, 277, "SamplesPerPixel", []TiffFormat{FormatUint16}, "1", 1, "The number of components per pixel."),
	41993: newExifTagData(false, 41993, "Saturation", []TiffFormat{FormatUint16}, "1", 1, ""),
	41990: newExifTagData(false, 41990, "SceneCaptureType", []TiffFormat{FormatUint16}, "1", 1, "Indicates the type of scene that was shot."),
	41729: newExifTagData(false, 41729, "SceneType", []TiffFormat{FormatUndefined}, "1", 1, "Unknown but value is '1'. "),
	41495: newExifTagData(false, 41495, "SensingMethod", []TiffFormat{FormatUint16}, "1", 1, "Shows type of image sensor unit. '2' means 1 chip color area sensor, most of all digicam use this type. "),
	50739: newExifTagData(false, 50739, "ShadowScale", []TiffFormat{FormatURational}, "1", 1, ""),
	41994: newExifTagData(false, 41994, "Sharpness", []TiffFormat{FormatUint16}, "1", 1, ""),
	37377: newExifTagData(false, 37377, "ShutterSpeedValue", []TiffFormat{FormatURational, FormatRational}, "1", 1, "Shutter speed. To convert this value to ordinary 'Shutter Speed'; calculate this value's power of 2, then reciprocal. For example, if value is '4', shutter speed is 1/(2^4)=1/16 second. "),
	305:   newExifTagData(false, 305, "Software", []TiffFormat{FormatString}, "N", -1, "Name and version number of the software package(s) used to create the image."),
	41484: newExifTagData(false, 41484, "SpatialFrequencyResponse", []TiffFormat{FormatUndefined}, "N", -1, ""),
	34852: newExifTagData(false, 34852, "SpectralSensitivity", []TiffFormat{FormatString}, "N", -1, ""),
	279:   newExifTagData(false, 279, "StripByteCounts", []TiffFormat{FormatUint16, FormatUint32}, "N = StripsPerImage for PlanarConfiguration equal to 1; N = SamplesPerPixel * StripsPerImage for PlanarConfiguration equal to 2", -1, "For each strip, the number of bytes in the strip after compression."),
	273:   newExifTagData(false, 273, "StripOffsets", []TiffFormat{FormatUint32, FormatUint16}, "N = StripsPerImage for PlanarConfiguration equal to 1; N = SamplesPerPixel * StripsPerImage for PlanarConfiguration equal to 2", -1, "For each strip, the byte offset of that strip."),
	559:   newExifTagData(false, 559, "StripRowCounts", []TiffFormat{FormatUint32}, "number of strips", -1, "Defined in the Mixed Raster Content part of RFC 2301, used to replace RowsPerStrip for IFDs with variable-sized strips."),
	330:   newExifTagData(false, 330, "SubIFDs", []TiffFormat{FormatUint32, FormatString}, "N = number of child IFDs", -1, "Offset to child IFDs."),
	50974: newExifTagData(false, 50974, "SubTileBlockSize", []TiffFormat{FormatUint16, FormatUint32}, "2", 2, ""),
	255:   newExifTagData(false, 255, "SubfileType", []TiffFormat{FormatUint16}, "1", 1, "A general indication of the kind of data contained in this subfile."),
	37382: newExifTagData(false, 37382, "SubjectDistance", []TiffFormat{FormatURational}, "1", 1, "Distance to focus point, unit is meter. "),
	41996: newExifTagData(false, 41996, "SubjectDistanceRange", []TiffFormat{FormatUint16}, "1", 1, ""),
	41492: newExifTagData(false, 41492, "SubjectLocation", []TiffFormat{FormatUint16}, "2", 2, ""),
	37520: newExifTagData(false, 37520, "SubsecTime", []TiffFormat{FormatString}, "N", -1, "Used to record fractions of seconds for the DateTime tag"),
	37522: newExifTagData(false, 37522, "SubsecTimeDigitized", []TiffFormat{FormatString}, "N", -1, "Used to record fractions of seconds for the DateTimeDigitized tag."),
	37521: newExifTagData(false, 37521, "SubsecTimeOriginal", []TiffFormat{FormatString}, "N", -1, "Used to record fractions of seconds for the DateTimeOriginal tag."),
	292:   newExifTagData(false, 292, "T4Options", []TiffFormat{FormatUint32}, "1", 1, "Options for Group 3 Fax compression"),
	293:   newExifTagData(false, 293, "T6Options", []TiffFormat{FormatUint32}, "1", 1, "Options for Group 4 Fax compression"),
	337:   newExifTagData(false, 337, "TargetPrinter", []TiffFormat{FormatString}, "N", -1, "A description of the printing environment for which this separation is intended."),
	263:   newExifTagData(false, 263, "Threshholding", []TiffFormat{FormatUint16}, "1", 1, "For black and white TIFF files that represent shades of gray, the technique used to convert from gray to black and white pixels."),
	325:   newExifTagData(false, 325, "TileByteCounts", []TiffFormat{FormatUint16, FormatUint32}, "N = TilesPerImage for PlanarConfiguration = 1; N = SamplesPerPixel * TilesPerImage for PlanarConfiguration = 2", -1, "For each tile, the number of (compressed) bytes in that tile."),
	323:   newExifTagData(false, 323, "TileLength", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "The tile length (height) in pixels. This is the number of rows in each tile."),
	324:   newExifTagData(false, 324, "TileOffsets", []TiffFormat{FormatUint32}, "N = TilesPerImage for PlanarConfiguration = 1; N = SamplesPerPixel * TilesPerImage for PlanarConfiguration = 2", -1, "For each tile, the byte offset of that tile, as compressed and stored on disk."),
	322:   newExifTagData(false, 322, "TileWidth", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "The tile width in pixels. This is the number of columns in each tile."),
	301:   newExifTagData(false, 301, "TransferFunction", []TiffFormat{FormatUint16}, "(1 or 3) * (1 &lt;&lt; BitsPerSample)", -1, "Describes a transfer function for the image in tabular style."),
	342:   newExifTagData(false, 342, "TransferRange", []TiffFormat{FormatUint16}, "6", 6, "Expands the range of the TransferFunction."),
	50708: newExifTagData(false, 50708, "UniqueCameraModel", []TiffFormat{FormatString}, "N", -1, ""),
	37510: newExifTagData(false, 37510, "UserComment", []TiffFormat{FormatUndefined}, "N", -1, "Stores user comment. "),
	404:   newExifTagData(false, 404, "VersionYear", []TiffFormat{FormatUint8}, "4", 4, "Used in the TIFF-FX standard, denotes the year of the standard specified by the FaxProfile field."),
	32932: newExifTagData(false, 32932, "Wang Annotation", []TiffFormat{FormatUint8}, "N", -1, ""),
	41987: newExifTagData(false, 41987, "WhiteBalance", []TiffFormat{FormatUint16}, "1", 1, "Indicates the white balance mode set when the image was shot."),
	50717: newExifTagData(false, 50717, "WhiteLevel", []TiffFormat{FormatUint16, FormatUint32, FormatURational}, "SamplesPerPixel", -1, ""),
	318:   newExifTagData(false, 318, "WhitePoint", []TiffFormat{FormatURational}, "2", 2, "The chromaticity of the white point of the image."),
	344:   newExifTagData(false, 344, "XClipPathUnits", []TiffFormat{FormatUint32}, "1", 1, "The number of units that span the width of the image, in terms of integer ClipPath coordinates."),
	700:   newExifTagData(false, 700, "XMP", []TiffFormat{FormatUint8}, "N", -1, "XML packet containing XMP metadata"),
	286:   newExifTagData(false, 286, "XPosition", []TiffFormat{FormatURational}, "1", 1, "X position of the image."),
	282:   newExifTagData(false, 282, "XResolution", []TiffFormat{FormatURational}, "1", 1, "The number of pixels per ResolutionUnit in the ImageWidth direction."),
	529:   newExifTagData(false, 529, "YCbCrCoefficients", []TiffFormat{FormatURational}, "3", 3, "The transformation from RGB to YCbCr image data."),
	531:   newExifTagData(false, 531, "YCbCrPositioning", []TiffFormat{FormatUint16}, "1", 1, "Specifies the positioning of subsampled chrominance components relative to luminance samples."),
	530:   newExifTagData(false, 530, "YCbCrSubSampling", []TiffFormat{FormatUint16}, "2", 2, "Specifies the subsampling factors used for the chrominance components of a YCbCr image."),
	345:   newExifTagData(false, 345, "YClipPathUnits", []TiffFormat{FormatUint32}, "1", 1, "The number of units that span the height of the image, in terms of integer ClipPath coordinates."),
	287:   newExifTagData(false, 287, "YPosition", []TiffFormat{FormatURational}, "1", 1, "Y position of the image."),
	283:   newExifTagData(false, 283, "YResolution", []TiffFormat{FormatURational}, "1", 1, "The number of pixels per ResolutionUnit in the ImageLength direction."),
}
