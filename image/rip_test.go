package image

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const secondCut = "</table>"

// const logName1 = "ripTest1.log"
// const logName2 = "ripTest2.log"
const logName3 = "ripTest3.log"
const logName4 = "ripTest4.log"
const logName5 = "ripTest5.log"

func TestReadHtml(t *testing.T) {
	os.Remove(logName4)
	log, err := os.OpenFile(logName4, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		t.Fatalf("Could not open log %s", logName4)
	}
	defer closeLog(t, log)

	entries, err := os.ReadDir("../image/ripHtml")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		name := e.Name()
		tagName := strings.Replace(name, ".html", "", 1)
		con := ReadTestHtml(t, e.Name())
		tblData, err := CutToTable(con, tagName)
		if err != nil {
			log.WriteString(fmt.Sprintf("%3d:CUT-TABLE: File:%s TagName:%s. Error:%s\n", count, name, tagName, err.Error()))
		} else {
			rmap := CutToData(tblData)
			name, ok := rmap["Name"]
			if !ok {
				log.WriteString(fmt.Sprintf("%3d:Data: File:%s Error: Name not found. TagName:%s\n", count, name, tagName))
			} else {
				tc, ok := rmap["Code"]
				if !ok {
					log.WriteString(fmt.Sprintf("%3d:Data: File:%s Error: Code not found. TagName:%s\n", count, name, tagName))
				} else {
					tagCode, err := getNumber(tc)
					if err != nil {
						log.WriteString(fmt.Sprintf("%3d:Data: File:%s Error: Invalid Code. TagName:%s\n", count, name, tagName))
					} else {
						dataType, ok := rmap["Type"]
						if !ok {
							log.WriteString(fmt.Sprintf("%3d:Data: File:%s Error: Type not found. TagName:%s\n", count, name, tagName))
						} else {
							countStr, ok := rmap["Count"]
							if !ok {
								log.WriteString(fmt.Sprintf("%3d:Data: File:%s Error: Count not found. TagName:%s\n", count, name, tagName))
							}
							c, err := strconv.Atoi(countStr)
							if err != nil {
								c = -1
							}
							mtData := "\"\""
							mt, ok := mapTags[tagCode]
							if ok {
								if mt.Name != tagName {
									mtData = fmt.Sprintf("\"*AltName(%s) %s\"", mt.Name, mt.LongDesc)
								} else {
									mtData = fmt.Sprintf("\"%s\"", mt.LongDesc)
								}
							}
							var line bytes.Buffer
							formats := getTiffFormatsFromNames(dataType)
							if len(formats) == 0 {
								line.WriteString("[]string{")
								line.WriteRune('"')
								line.WriteString("ERROR:")
								line.WriteString(dataType)
								line.WriteRune('"')
								line.WriteRune('}')
							} else {
								line.WriteString("[]TiffFormat{")
								for i, v := range formats {
									line.WriteString(v.name)
									if i < (len(formats) - 1) {
										line.WriteRune(',')
									}
								}
								line.WriteRune('}')
							}
							// 301:   newExifTagDetails(301, "TransferFunction", FormatUndefined, "Describes a transfer function for the image in tabular style."),
							log.WriteString(fmt.Sprintf("%d: newExifTagDetailsExt(%d, \"%s\", %s, \"%s\", %d, %s),\n", tagCode, tagCode, tagName, line.String(), countStr, c, mtData))
						}
					}

				}

			}
		}
		count++
	}
}

func getNumber(s string) (uint32, error) {
	var line bytes.Buffer
	for _, r := range s {
		if r >= '0' && r <= '9' {
			line.WriteRune(r)
		} else {
			break
		}
	}
	n, err := strconv.Atoi(line.String())
	if err != nil {
		return 0, err
	}
	return uint32(n), nil
}

func TestReadLog(t *testing.T) {
	log, err := os.OpenFile(logName4, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		t.Fatalf("Could not open log %s", logName4)
	}
	defer closeLog(t, log)

	readFile, err := os.Open(logName3)
	if err != nil {
		t.Fatal(err)
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	count := 1

	for fileScanner.Scan() {
		parts := strings.Split(fileScanner.Text(), ":")
		if parts[2] == "ERROR" {
			res := GetTagPage(parts[3])
			log.WriteString(fmt.Sprintf("%3d:%s:%s\n", count, parts[1], res))
			fmt.Printf("%s\n", res)
			count++
		}
	}

	readFile.Close()
}

func TestGetPage(t *testing.T) {
	log, err := os.OpenFile(logName4, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		t.Fatalf("Could not open log %s", logName4)
	}
	defer closeLog(t, log)

	count := 1
	for n, v := range GpsNames {
		res := GetTagPage(v)
		log.WriteString(fmt.Sprintf("%3d:%5d:%s\n", count, n, res))
		fmt.Printf("%s\n", res)
		count++
	}
}

func GetTagPage(tagName string) string {
	fn := strings.ToLower(tagName)
	fn = strings.ReplaceAll(fn, " ", "")
	requestURL := fmt.Sprintf("https://www.awaresystems.be/imaging/tiff/tifftags/privateifd/gps/%s.html", fn)
	res, err := http.Get(requestURL)
	if err != nil {
		return fmt.Sprintf("ERROR:%s:Making http request for tag. %s", tagName, err)
	}
	if res.StatusCode != 200 {
		return fmt.Sprintf("ERROR:%s:Reading tag. Status %s", tagName, res.Status)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Sprintf("ERROR:%s:Reading body for tag. %s", tagName, err)
	}
	err = os.WriteFile(fmt.Sprintf("ripHtml/%s.html", tagName), resBody, 0644)
	if err != nil {
		return fmt.Sprintf("ERROR:%s:Writing body for tag. %s", tagName, err)
	}
	return fmt.Sprintf("   OK:%s:Done", tagName)
}

func CutToTable(b, tagName string) (string, error) {
	firstCut := fmt.Sprintf("<h1>TIFF Tag %s</h1>", tagName)
	_, after, found := strings.Cut(b, firstCut)
	if found {
		before, _, found := strings.Cut(after, secondCut)
		if found {
			return before + secondCut, nil
		}
		return "", fmt.Errorf("Tag %s: Second cut [%s] not found", tagName, secondCut)
	}
	return "", fmt.Errorf("Tag %s: First cut [%s] not found", tagName, firstCut)
}

const c1 = "<tr><th>"
const c2 = "</th><td>"
const c3 = "</td></tr>"

func CutToData(tableText string) map[string]string {
	out := map[string]string{}
	notDone := true
	pos := 0
	bb := tableText
	for notDone {
		p1 := strings.Index(bb, c1)
		p2 := strings.Index(bb, c2)
		p3 := strings.Index(bb, c3)
		if (p1 > 0) && (p2 > p1) && (p3 > p2) {
			name := bb[p1+len(c1) : p2]
			value := bb[p2+len(c2) : p3]
			pos = p3 + len(c3)
			out[name] = value
			bb = bb[pos:]
		} else {
			notDone = false
		}

	}
	return out
}

func ReadTestHtml(t *testing.T, tagName string) string {
	fileName, err := filepath.Abs(fmt.Sprintf("ripHtml/%s", tagName))
	if err != nil {
		t.Fatalf("Tag File [%s] Is not ABS. %v", tagName, err)
	}

	content, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("Tag File [%s] not found. %v", fileName, err)
	}
	return string(content)
}

func closeLog(t *testing.T, l *os.File) {
	err := l.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func newExifTagDetailsExt(tag uint32, name string, fmts []TiffFormat, countStr string, count int, desc string) *Tag {
	return nil
}

// tiff ifd field names cribbed from the Library of Congress
// http://www.loc.gov/preservation/digital/formats/content/tiff_tags.shtml
var GpsNames = map[uint16]string{
	0x0000: "GPSVersionID",
	0x0001: "GPSLatitudeRef",
	0x0002: "GPSLatitude",
	0x0003: "GPSLongitudeRef",
	0x0004: "GPSLongitude",
	0x0005: "GPSAltitudeRef",
	0x0006: "GPSAltitude",
	0x0007: "GPSTimeStamp",
	0x0008: "GPSSatellites",
	0x0009: "GPSStatus",
	0x000A: "GPSMeasureMode",
	0x000B: "GPSDOP",
	0x000C: "GPSSpeedRef",
	0x000D: "GPSSpeed",
	0x000E: "GPSTrackRef",
	0x000F: "GPSTrack",
	0x0010: "GPSImgDirectionRef",
	0x0011: "GPSImgDirection",
	0x0012: "GPSMapDatum",
	0x0013: "GPSDestLatitudeRef",
	0x0014: "GPSDestLatitude",
	0x0015: "GPSDestLongitudeRef",
	0x0016: "GPSDestLongitude",
	0x0017: "GPSDestBearingRef",
	0x0018: "GPSDestBearing",
	0x0019: "GPSDestDistanceRef",
	0x001A: "GPSDestDistance",
	0x001B: "GPSProcessingMethod",
	0x001C: "GPSAreaInformation",
	0x001D: "GPSDateStamp",
	0x001E: "GPSDifferential",
}

var FieldNames = map[uint16]string{
	254:   "NewSubfileType",
	255:   "SubfileType",
	256:   "ImageWidth",
	257:   "ImageLength",
	258:   "BitsPerSample",
	259:   "Compression",
	262:   "PhotometricInterpretation",
	263:   "Threshholding",
	264:   "CellWidth",
	265:   "CellLength",
	266:   "FillOrder",
	269:   "DocumentName",
	270:   "ImageDescription",
	271:   "Make",
	272:   "Model",
	273:   "StripOffsets",
	274:   "Orientation",
	277:   "SamplesPerPixel",
	278:   "RowsPerStrip",
	279:   "StripByteCounts",
	280:   "MinSampleValue",
	281:   "MaxSampleValue",
	282:   "XResolution",
	283:   "YResolution",
	284:   "PlanarConfiguration",
	285:   "PageName",
	286:   "XPosition",
	287:   "YPosition",
	288:   "FreeOffsets",
	289:   "FreeByteCounts",
	290:   "GrayResponseUnit",
	291:   "GrayResponseCurve",
	292:   "T4Options",
	293:   "T6Options",
	296:   "ResolutionUnit",
	297:   "PageNumber",
	301:   "TransferFunction",
	305:   "Software",
	306:   "DateTime",
	315:   "Artist",
	316:   "HostComputer",
	317:   "Predictor",
	318:   "WhitePoint",
	319:   "PrimaryChromaticities",
	320:   "ColorMap",
	321:   "HalftoneHints",
	322:   "TileWidth",
	323:   "TileLength",
	324:   "TileOffsets",
	325:   "TileByteCounts",
	326:   "BadFaxLines",
	327:   "CleanFaxData",
	328:   "ConsecutiveBadFaxLines",
	330:   "SubIFDs",
	332:   "InkSet",
	333:   "InkNames",
	334:   "NumberOfInks",
	336:   "DotRange",
	337:   "TargetPrinter",
	338:   "ExtraSamples",
	339:   "SampleFormat",
	340:   "SMinSampleValue",
	341:   "SMaxSampleValue",
	342:   "TransferRange",
	343:   "ClipPath",
	344:   "XClipPathUnits",
	345:   "YClipPathUnits",
	346:   "Indexed",
	347:   "JPEGTables",
	351:   "OPIProxy",
	400:   "GlobalParametersIFD",
	401:   "ProfileType",
	402:   "FaxProfile",
	403:   "CodingMethods",
	404:   "VersionYear",
	405:   "ModeNumber",
	433:   "Decode",
	434:   "DefaultImageColor",
	512:   "JPEGProc",
	513:   "JPEGInterchangeFormat",
	514:   "JPEGInterchangeFormatLength",
	515:   "JPEGRestartInterval",
	517:   "JPEGLosslessPredictors",
	518:   "JPEGPointTransforms",
	519:   "JPEGQTables",
	520:   "JPEGDCTables",
	521:   "JPEGACTables",
	529:   "YCbCrCoefficients",
	530:   "YCbCrSubSampling",
	531:   "YCbCrPositioning",
	532:   "ReferenceBlackWhite",
	559:   "StripRowCounts",
	700:   "XMP",
	18246: "Image.Rating",
	18249: "Image.RatingPercent",
	32781: "ImageID",
	32932: "Wang Annotation",
	33421: "CFARepeatPatternDim",
	33422: "CFAPattern",
	33423: "BatteryLevel",
	33432: "Copyright",
	33434: "ExposureTime",
	33437: "FNumber",
	33445: "MD FileTag",
	33446: "MD ScalePixel",
	33447: "MD ColorTable",
	33448: "MD LabName",
	33449: "MD SampleInfo",
	33450: "MD PrepDate",
	33451: "MD PrepTime",
	33452: "MD FileUnits",
	33550: "ModelPixelScaleTag",
	33723: "IPTC/NAA",
	33918: "INGR Packet Data Tag",
	33919: "INGR Flag Registers",
	33920: "IrasB Transformation Matrix",
	33922: "ModelTiepointTag",
	34016: "Site",
	34017: "ColorSequence",
	34018: "IT8Header",
	34019: "RasterPadding",
	34020: "BitsPerRunLength",
	34021: "BitsPerExtendedRunLength",
	34022: "ColorTable",
	34023: "ImageColorIndicator",
	34024: "BackgroundColorIndicator",
	34025: "ImageColorValue",
	34026: "BackgroundColorValue",
	34027: "PixelIntensityRange",
	34028: "TransparencyIndicator",
	34029: "ColorCharacterization",
	34030: "HCUsage",
	34031: "TrapIndicator",
	34032: "CMYKEquivalent",
	34033: "Reserved",
	34034: "Reserved",
	34035: "Reserved",
	34264: "ModelTransformationTag",
	34377: "Photoshop",
	34665: "Exif IFD",
	34675: "InterColorProfile",
	34732: "ImageLayer",
	34735: "GeoKeyDirectoryTag",
	34736: "GeoDoubleParamsTag",
	34737: "GeoAsciiParamsTag",
	34850: "ExposureProgram",
	34852: "SpectralSensitivity",
	34853: "GPSInfo",
	34855: "ISOSpeedRatings",
	34856: "OECF",
	34857: "Interlace",
	34858: "TimeZoneOffset",
	34859: "SelfTimeMode",
	34864: "SensitivityType",
	34865: "StandardOutputSensitivity",
	34866: "RecommendedExposureIndex",
	34867: "ISOSpeed",
	34868: "ISOSpeedLatitudeyyy",
	34869: "ISOSpeedLatitudezzz",
	34908: "HylaFAX FaxRecvParams",
	34909: "HylaFAX FaxSubAddress",
	34910: "HylaFAX FaxRecvTime",
	36864: "ExifVersion",
	36867: "DateTimeOriginal",
	36868: "DateTimeDigitized",
	37121: "ComponentsConfiguration",
	37122: "CompressedBitsPerPixel",
	37377: "ShutterSpeedValue",
	37378: "ApertureValue",
	37379: "BrightnessValue",
	37380: "ExposureBiasValue",
	37381: "MaxApertureValue",
	37382: "SubjectDistance",
	37383: "MeteringMode",
	37384: "LightSource",
	37385: "Flash",
	37386: "FocalLength",
	37387: "FlashEnergy",
	37388: "SpatialFrequencyResponse",
	37389: "Noise",
	37390: "FocalPlaneXResolution",
	37391: "FocalPlaneYResolution",
	37392: "FocalPlaneResolutionUnit",
	37393: "ImageNumber",
	37394: "SecurityClassification",
	37395: "ImageHistory",
	37396: "SubjectLocation",
	37397: "ExposureIndex",
	37398: "TIFF/EPStandardID",
	37399: "SensingMethod",
	37500: "MakerNote",
	37510: "UserComment",
	37520: "SubsecTime",
	37521: "SubsecTimeOriginal",
	37522: "SubsecTimeDigitized",
	37724: "ImageSourceData",
	40091: "XPTitle",
	40092: "XPComment",
	40093: "XPAuthor",
	40094: "XPKeywords",
	40095: "XPSubject",
	40960: "FlashpixVersion",
	40961: "ColorSpace",
	40962: "PixelXDimension",
	40963: "PixelYDimension",
	40964: "RelatedSoundFile",
	40965: "Interoperability IFD",
	41483: "FlashEnergy",
	41484: "SpatialFrequencyResponse",
	41486: "FocalPlaneXResolution",
	41487: "FocalPlaneYResolution",
	41488: "FocalPlaneResolutionUnit",
	41492: "SubjectLocation",
	41493: "ExposureIndex",
	41495: "SensingMethod",
	41728: "FileSource",
	41729: "SceneType",
	41730: "CFAPattern",
	41985: "CustomRendered",
	41986: "ExposureMode",
	41987: "WhiteBalance",
	41988: "DigitalZoomRatio",
	41989: "FocalLengthIn35mmFilm",
	41990: "SceneCaptureType",
	41991: "GainControl",
	41992: "Contrast",
	41993: "Saturation",
	41994: "Sharpness",
	41995: "DeviceSettingDescription",
	41996: "SubjectDistanceRange",
	42016: "ImageUniqueID",
	42032: "CameraOwnerName",
	42033: "BodySerialNumber",
	42034: "LensSpecification",
	42035: "LensMake",
	42036: "LensModel",
	42037: "LensSerialNumber",
	42112: "GDAL_METADATA",
	42113: "GDAL_NODATA",
	48129: "PixelFormat",
	48130: "Transformation",
	48131: "Uncompressed",
	48132: "ImageType",
	48256: "ImageWidth",
	48257: "ImageHeight",
	48258: "WidthResolution",
	48259: "HeightResolution",
	48320: "ImageOffset",
	48321: "ImageByteCount",
	48322: "AlphaOffset",
	48323: "AlphaByteCount",
	48324: "ImageDataDiscard",
	48325: "AlphaDataDiscard",
	50215: "Oce Scanjob Description",
	50216: "Oce Application Selector",
	50217: "Oce Identification Number",
	50218: "Oce ImageLogic Characteristics",
	50341: "PrintImageMatching",
	50706: "DNGVersion",
	50707: "DNGBackwardVersion",
	50708: "UniqueCameraModel",
	50709: "LocalizedCameraModel",
	50710: "CFAPlaneColor",
	50711: "CFALayout",
	50712: "LinearizationTable",
	50713: "BlackLevelRepeatDim",
	50714: "BlackLevel",
	50715: "BlackLevelDeltaH",
	50716: "BlackLevelDeltaV",
	50717: "WhiteLevel",
	50718: "DefaultScale",
	50719: "DefaultCropOrigin",
	50720: "DefaultCropSize",
	50721: "ColorMatrix1",
	50722: "ColorMatrix2",
	50723: "CameraCalibration1",
	50724: "CameraCalibration2",
	50725: "ReductionMatrix1",
	50726: "ReductionMatrix2",
	50727: "AnalogBalance",
	50728: "AsShotNeutral",
	50729: "AsShotWhiteXY",
	50730: "BaselineExposure",
	50731: "BaselineNoise",
	50732: "BaselineSharpness",
	50733: "BayerGreenSplit",
	50734: "LinearResponseLimit",
	50735: "CameraSerialNumber",
	50736: "LensInfo",
	50737: "ChromaBlurRadius",
	50738: "AntiAliasStrength",
	50739: "ShadowScale",
	50740: "DNGPrivateData",
	50741: "MakerNoteSafety",
	50778: "CalibrationIlluminant1",
	50779: "CalibrationIlluminant2",
	50780: "BestQualityScale",
	50781: "RawDataUniqueID",
	50784: "Alias Layer Metadata",
	50827: "OriginalRawFileName",
	50828: "OriginalRawFileData",
	50829: "ActiveArea",
	50830: "MaskedAreas",
	50831: "AsShotICCProfile",
	50832: "AsShotPreProfileMatrix",
	50833: "CurrentICCProfile",
	50834: "CurrentPreProfileMatrix",
	50879: "ColorimetricReference",
	50931: "CameraCalibrationSignature",
	50932: "ProfileCalibrationSignature",
	50933: "ExtraCameraProfiles",
	50934: "AsShotProfileName",
	50935: "NoiseReductionApplied",
	50936: "ProfileName",
	50937: "ProfileHueSatMapDims",
	50938: "ProfileHueSatMapData1",
	50939: "ProfileHueSatMapData2",
	50940: "ProfileToneCurve",
	50941: "ProfileEmbedPolicy",
	50942: "ProfileCopyright",
	50964: "ForwardMatrix1",
	50965: "ForwardMatrix2",
	50966: "PreviewApplicationName",
	50967: "PreviewApplicationVersion",
	50968: "PreviewSettingsName",
	50969: "PreviewSettingsDigest",
	50970: "PreviewColorSpace",
	50971: "PreviewDateTime",
	50972: "RawImageDigest",
	50973: "OriginalRawFileDigest",
	50974: "SubTileBlockSize",
	50975: "RowInterleaveFactor",
	50981: "ProfileLookTableDims",
	50982: "ProfileLookTableData",
	51008: "OpcodeList1",
	51009: "OpcodeList2",
	51022: "OpcodeList3",
	51041: "NoiseProfile",
	51089: "OriginalDefaultFinalSize",
	51090: "OriginalBestQualityFinalSize",
	51091: "OriginalDefaultCropSize",
	51107: "ProfileHueSatMapEncoding",
	51108: "ProfileLookTableEncoding",
	51109: "BaselineExposureOffset",
	51110: "DefaultBlackRender",
	51111: "NewRawImageDigest",
	51112: "RawToPreviewGain",
	51125: "DefaultUserCrop",
}

var mapNewTags = map[uint32]*Tag{
	50829: newExifTagDetailsExt(50829, "ActiveArea", []TiffFormat{FormatUint16, FormatUint32}, "4", 4, ""),
	50784: newExifTagDetailsExt(50784, "Alias Layer Metadata", []TiffFormat{FormatString}, "N", -1, ""),
	50727: newExifTagDetailsExt(50727, "AnalogBalance", []TiffFormat{FormatURational}, "ColorPlanes", -1, ""),
	50738: newExifTagDetailsExt(50738, "AntiAliasStrength", []TiffFormat{FormatURational}, "1", 1, ""),
	37378: newExifTagDetailsExt(37378, "ApertureValue", []TiffFormat{FormatURational}, "1", 1, "The actual aperture value of lens when the image was taken. To convert this value to ordinary F-number(F-stop), calculate this value's power of root 2 (=1.4142). For example, if value is '5', F-number is 1.4142^5 = F5.6. "),
	315:   newExifTagDetailsExt(315, "Artist", []TiffFormat{FormatString}, "N", -1, "Person who created the image."),
	50831: newExifTagDetailsExt(50831, "AsShotICCProfile", []TiffFormat{FormatUndefined}, "N", -1, ""),
	50728: newExifTagDetailsExt(50728, "AsShotNeutral", []TiffFormat{FormatUint16, FormatURational}, "ColorPlanes", -1, ""),
	50832: newExifTagDetailsExt(50832, "AsShotPreProfileMatrix", []TiffFormat{FormatRational, FormatURational}, "3 * ColorPlanes or ColorPlanes * ColorPlanes", -1, ""),
	50934: newExifTagDetailsExt(50934, "AsShotProfileName", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	50729: newExifTagDetailsExt(50729, "AsShotWhiteXY", []TiffFormat{FormatURational}, "2", 2, ""),
	326:   newExifTagDetailsExt(326, "BadFaxLines", []TiffFormat{FormatUint32, FormatUint16}, "1", 1, "Used in the TIFF-F standard, denotes the number of 'bad' scan lines encountered by the facsimile device."),
	50730: newExifTagDetailsExt(50730, "BaselineExposure", []TiffFormat{FormatRational, FormatURational}, "1", 1, ""),
	51109: newExifTagDetailsExt(51109, "BaselineExposureOffset", []TiffFormat{FormatURational}, "1", 1, ""),
	50731: newExifTagDetailsExt(50731, "BaselineNoise", []TiffFormat{FormatURational}, "1", 1, ""),
	50732: newExifTagDetailsExt(50732, "BaselineSharpness", []TiffFormat{FormatURational}, "1", 1, ""),
	50733: newExifTagDetailsExt(50733, "BayerGreenSplit", []TiffFormat{FormatUint32}, "1", 1, ""),
	50780: newExifTagDetailsExt(50780, "BestQualityScale", []TiffFormat{FormatURational}, "1", 1, ""),
	258:   newExifTagDetailsExt(258, "BitsPerSample", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "Number of bits per component."),
	50714: newExifTagDetailsExt(50714, "BlackLevel", []TiffFormat{FormatURational, FormatUint16, FormatUint32}, "BlackLevelRepeatRows * BlackLevelRepeatCols * SamplesPerPixel", -1, ""),
	50715: newExifTagDetailsExt(50715, "BlackLevelDeltaH", []TiffFormat{FormatURational, FormatRational}, "ImageWidth", -1, ""),
	50716: newExifTagDetailsExt(50716, "BlackLevelDeltaV", []TiffFormat{FormatURational, FormatRational}, "ImageLength", -1, ""),
	50713: newExifTagDetailsExt(50713, "BlackLevelRepeatDim", []TiffFormat{FormatUint16}, "2", 2, ""),
	37379: newExifTagDetailsExt(37379, "BrightnessValue", []TiffFormat{FormatRational, FormatURational}, "1", 1, "Brightness of taken subject, unit is EV. "),
	50711: newExifTagDetailsExt(50711, "CFALayout", []TiffFormat{FormatUint16}, "1", 1, ""),
	41730: newExifTagDetailsExt(41730, "CFAPattern", []TiffFormat{FormatUndefined}, "N", -1, ""),
	50710: newExifTagDetailsExt(50710, "CFAPlaneColor", []TiffFormat{FormatUint8}, "ColorPlanes", -1, ""),
	50778: newExifTagDetailsExt(50778, "CalibrationIlluminant1", []TiffFormat{FormatUint16}, "1", 1, ""),
	50779: newExifTagDetailsExt(50779, "CalibrationIlluminant2", []TiffFormat{FormatUint16}, "1", 1, ""),
	50723: newExifTagDetailsExt(50723, "CameraCalibration1", []TiffFormat{FormatRational, FormatURational}, "ColorPlanes * ColorPlanes", -1, ""),
	50724: newExifTagDetailsExt(50724, "CameraCalibration2", []TiffFormat{FormatRational, FormatURational}, "ColorPlanes * ColorPlanes", -1, ""),
	50931: newExifTagDetailsExt(50931, "CameraCalibrationSignature", []TiffFormat{FormatString, FormatUint8}, "N", -1, ""),
	50735: newExifTagDetailsExt(50735, "CameraSerialNumber", []TiffFormat{FormatString}, "N", -1, ""),
	265:   newExifTagDetailsExt(265, "CellLength", []TiffFormat{FormatUint16}, "1", 1, "The length of the dithering or halftoning matrix used to create a dithered or halftoned bilevel file."),
	264:   newExifTagDetailsExt(264, "CellWidth", []TiffFormat{FormatUint16}, "1", 1, "The width of the dithering or halftoning matrix used to create a dithered or halftoned bilevel file."),
	50737: newExifTagDetailsExt(50737, "ChromaBlurRadius", []TiffFormat{FormatURational}, "1", 1, ""),
	327:   newExifTagDetailsExt(327, "CleanFaxData", []TiffFormat{FormatUint16}, "1", 1, "Used in the TIFF-F standard, indicates if 'bad' lines encountered during reception are stored in the data, or if 'bad' lines have been replaced by the receiver."),
	343:   newExifTagDetailsExt(343, "ClipPath", []TiffFormat{FormatUint8}, "N", -1, "Mirrors the essentials of PostScript's path creation functionality."),
	403:   newExifTagDetailsExt(403, "CodingMethods", []TiffFormat{FormatUint32}, "1", 1, "Used in the TIFF-FX standard, indicates which coding methods are used in the file."),
	320:   newExifTagDetailsExt(320, "ColorMap", []TiffFormat{FormatUint16}, "3 * (2**BitsPerSample)", -1, "A color map for palette color images."),
	50721: newExifTagDetailsExt(50721, "ColorMatrix1", []TiffFormat{FormatRational, FormatURational}, "ColorPlanes * 3", -1, ""),
	50722: newExifTagDetailsExt(50722, "ColorMatrix2", []TiffFormat{FormatURational, FormatRational}, "ColorPlanes * 3", -1, ""),
	40961: newExifTagDetailsExt(40961, "ColorSpace", []TiffFormat{FormatUint16}, "1", 1, "Value is '1'. "),
	50879: newExifTagDetailsExt(50879, "ColorimetricReference", []TiffFormat{FormatUint16}, "1", 1, ""),
	37121: newExifTagDetailsExt(37121, "ComponentsConfiguration", []TiffFormat{FormatUndefined}, "4", 4, "*AltName(ComponentConfiguration) It seems value 0x00,0x01,0x02,0x03 always. "),
	37122: newExifTagDetailsExt(37122, "CompressedBitsPerPixel", []TiffFormat{FormatURational}, "1", 1, "The average compression ratio of JPEG. "),
	259:   newExifTagDetailsExt(259, "Compression", []TiffFormat{FormatUint16}, "1", 1, "Compression scheme used on the image data."),
	328:   newExifTagDetailsExt(328, "ConsecutiveBadFaxLines", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "Used in the TIFF-F standard, denotes the maximum number of consecutive 'bad' scanlines received."),
	41992: newExifTagDetailsExt(41992, "Contrast", []TiffFormat{FormatUint16}, "1", 1, ""),
	33432: newExifTagDetailsExt(33432, "Copyright", []TiffFormat{FormatString}, "N", -1, "Copyright notice."),
	50833: newExifTagDetailsExt(50833, "CurrentICCProfile", []TiffFormat{FormatUndefined}, "N", -1, ""),
	50834: newExifTagDetailsExt(50834, "CurrentPreProfileMatrix", []TiffFormat{FormatRational, FormatURational}, "3 * ColorPlanes or ColorPlanes * ColorPlanes", -1, ""),
	41985: newExifTagDetailsExt(41985, "CustomRendered", []TiffFormat{FormatUint16}, "1", 1, ""),
	50707: newExifTagDetailsExt(50707, "DNGBackwardVersion", []TiffFormat{FormatUint8}, "4", 4, ""),
	50740: newExifTagDetailsExt(50740, "DNGPrivateData", []TiffFormat{FormatUint8}, "N", -1, ""),
	50706: newExifTagDetailsExt(50706, "DNGVersion", []TiffFormat{FormatUint8}, "4", 4, ""),
	306:   newExifTagDetailsExt(306, "DateTime", []TiffFormat{FormatString}, "20", 20, "Date and time of image creation."),
	36868: newExifTagDetailsExt(36868, "DateTimeDigitized", []TiffFormat{FormatString}, "20", 20, "Date/Time of image digitized. Usually, it contains the same value of DateTimeOriginal(0x9003). "),
	36867: newExifTagDetailsExt(36867, "DateTimeOriginal", []TiffFormat{FormatString}, "20", 20, "Date/Time of original image taken. This value should not be modified by user program. "),
	433:   newExifTagDetailsExt(433, "Decode", []TiffFormat{FormatURational, FormatRational}, "2 * SamplesPerPixel (= 6, for ITULAB)", -1, "Used in the TIFF-F and TIFF-FX standards, holds information about the ITULAB (PhotometricInterpretation = 10) encoding."),
	51110: newExifTagDetailsExt(51110, "DefaultBlackRender", []TiffFormat{FormatUint32}, "1", 1, ""),
	50719: newExifTagDetailsExt(50719, "DefaultCropOrigin", []TiffFormat{FormatURational, FormatUint32, FormatUint16}, "2", 2, ""),
	50720: newExifTagDetailsExt(50720, "DefaultCropSize", []TiffFormat{FormatUint16, FormatUint32, FormatURational}, "2", 2, ""),
	434:   newExifTagDetailsExt(434, "DefaultImageColor", []TiffFormat{FormatUint16}, "SamplesPerPixel", -1, "Defined in the Mixed Raster Content part of RFC 2301, is the default color needed in areas where no image is available."),
	50718: newExifTagDetailsExt(50718, "DefaultScale", []TiffFormat{FormatURational}, "2", 2, ""),
	51125: newExifTagDetailsExt(51125, "DefaultUserCrop", []TiffFormat{FormatURational}, "4", 4, ""),
	41995: newExifTagDetailsExt(41995, "DeviceSettingDescription", []TiffFormat{FormatUndefined}, "N", -1, ""),
	41988: newExifTagDetailsExt(41988, "DigitalZoomRatio", []TiffFormat{FormatURational}, "1", 1, "Indicates the digital zoom ratio when the image was shot."),
	269:   newExifTagDetailsExt(269, "DocumentName", []TiffFormat{FormatString}, "N", -1, "The name of the document from which this image was scanned."),
	336:   newExifTagDetailsExt(336, "DotRange", []TiffFormat{FormatUint16, FormatUint8}, "2, or 2*SamplesPerPixel", -1, "The component values that correspond to a 0% dot and 100% dot."),
	34665: newExifTagDetailsExt(34665, "Exif IFD", []TiffFormat{FormatString, FormatUint32}, "1", 1, ""),
	36864: newExifTagDetailsExt(36864, "ExifVersion", []TiffFormat{FormatUndefined}, "4", 4, "Exif version number. Stored as 4bytes of ASCII character (like 0210) "),
	37380: newExifTagDetailsExt(37380, "ExposureBiasValue", []TiffFormat{FormatRational, FormatURational}, "1", 1, "Exposure bias value of taking picture. Unit is EV. "),
	41493: newExifTagDetailsExt(41493, "ExposureIndex", []TiffFormat{FormatURational}, "1", 1, ""),
	41986: newExifTagDetailsExt(41986, "ExposureMode", []TiffFormat{FormatUint16}, "1", 1, "Indicates the exposure mode set when the image was shot."),
	34850: newExifTagDetailsExt(34850, "ExposureProgram", []TiffFormat{FormatUint16}, "1", 1, "Exposure program that the camera used when image was taken. '1' means manual control, '2' program normal, '3' aperture priority, '4' shutter priority, '5' program creative (slow program), '6' program action(high-speed program), '7' portrait mode, '8' landscape mode. "),
	33434: newExifTagDetailsExt(33434, "ExposureTime", []TiffFormat{FormatURational}, "1", 1, "Exposure time (reciprocal of shutter speed). Unit is second. "),
	50933: newExifTagDetailsExt(50933, "ExtraCameraProfiles", []TiffFormat{FormatUint32}, "Number of extra camera profiles", -1, ""),
	338:   newExifTagDetailsExt(338, "ExtraSamples", []TiffFormat{FormatUint16}, "N", -1, "Description of extra components."),
	33437: newExifTagDetailsExt(33437, "FNumber", []TiffFormat{FormatURational}, "1", 1, "The actual F-number(F-stop) of lens when the image was taken. "),
	402:   newExifTagDetailsExt(402, "FaxProfile", []TiffFormat{FormatUint8}, "1", 1, "Used in the TIFF-FX standard, denotes the 'profile' that applies to this file."),
	41728: newExifTagDetailsExt(41728, "FileSource", []TiffFormat{FormatUndefined}, "1", 1, "Unknown but value is '3'. "),
	266:   newExifTagDetailsExt(266, "FillOrder", []TiffFormat{FormatUint16}, "1", 1, "The logical order of bits within a byte."),
	37385: newExifTagDetailsExt(37385, "Flash", []TiffFormat{FormatUint16}, "1", 1, "'1' means flash was used, '0' means not used. "),
	41483: newExifTagDetailsExt(41483, "FlashEnergy", []TiffFormat{FormatURational}, "1", 1, ""),
	40960: newExifTagDetailsExt(40960, "FlashpixVersion", []TiffFormat{FormatUndefined}, "4", 4, "*AltName(FlashPixVersion) Stores FlashPix version. Unknown but 4bytes of ASCII characters '0100' exists. "),
	37386: newExifTagDetailsExt(37386, "FocalLength", []TiffFormat{FormatURational}, "1", 1, "Focal length of lens used to take image. Unit is millimeter. "),
	41989: newExifTagDetailsExt(41989, "FocalLengthIn35mmFilm", []TiffFormat{FormatUint16}, "1", 1, "Indicates the equivalent focal length assuming a 35mm film camera, in mm."),
	41488: newExifTagDetailsExt(41488, "FocalPlaneResolutionUnit", []TiffFormat{FormatUint16}, "1", 1, "Unit of FocalPlaneXResoluton/FocalPlaneYResolution. '1' means no-unit, '2' inch, '3' centimeter. "),
	41486: newExifTagDetailsExt(41486, "FocalPlaneXResolution", []TiffFormat{FormatURational}, "1", 1, "CCD's pixel density. "),
	41487: newExifTagDetailsExt(41487, "FocalPlaneYResolution", []TiffFormat{FormatURational}, "1", 1, "FocalPlaneYResolution "),
	50964: newExifTagDetailsExt(50964, "ForwardMatrix1", []TiffFormat{FormatURational, FormatRational}, "3 * ColorPlanes", -1, ""),
	50965: newExifTagDetailsExt(50965, "ForwardMatrix2", []TiffFormat{FormatRational, FormatURational}, "3 * ColorPlanes", -1, ""),
	289:   newExifTagDetailsExt(289, "FreeByteCounts", []TiffFormat{FormatUint32}, "N", -1, "For each string of contiguous unused bytes in a TIFF file, the number of bytes in the string."),
	288:   newExifTagDetailsExt(288, "FreeOffsets", []TiffFormat{FormatUint32}, "N", -1, "For each string of contiguous unused bytes in a TIFF file, the byte offset of the string."),
	42112: newExifTagDetailsExt(42112, "GDAL_METADATA", []TiffFormat{FormatString}, "N", -1, ""),
	42113: newExifTagDetailsExt(42113, "GDAL_NODATA", []TiffFormat{FormatString}, "N", -1, ""),
	6:     newExifTagDetailsExt(6, "GPSAltitude", []TiffFormat{FormatURational}, "1", 1, "Indicates the altitude based on the reference in GPSAltitudeRef."),
	5:     newExifTagDetailsExt(5, "GPSAltitudeRef", []TiffFormat{FormatUint8}, "1", 1, "Indicates the altitude used as the reference altitude."),
	28:    newExifTagDetailsExt(28, "GPSAreaInformation", []TiffFormat{FormatUndefined}, "N", -1, ""),
	11:    newExifTagDetailsExt(11, "GPSDOP", []TiffFormat{FormatURational}, "1", 1, ""),
	29:    newExifTagDetailsExt(29, "GPSDateStamp", []TiffFormat{FormatString}, "11", 11, ""),
	24:    newExifTagDetailsExt(24, "GPSDestBearing", []TiffFormat{FormatURational}, "1", 1, ""),
	23:    newExifTagDetailsExt(23, "GPSDestBearingRef", []TiffFormat{FormatString}, "2", 2, ""),
	26:    newExifTagDetailsExt(26, "GPSDestDistance", []TiffFormat{FormatURational}, "1", 1, ""),
	25:    newExifTagDetailsExt(25, "GPSDestDistanceRef", []TiffFormat{FormatString}, "2", 2, ""),
	20:    newExifTagDetailsExt(20, "GPSDestLatitude", []TiffFormat{FormatURational}, "3", 3, ""),
	19:    newExifTagDetailsExt(19, "GPSDestLatitudeRef", []TiffFormat{FormatString}, "2", 2, ""),
	22:    newExifTagDetailsExt(22, "GPSDestLongitude", []TiffFormat{FormatURational}, "3", 3, ""),
	21:    newExifTagDetailsExt(21, "GPSDestLongitudeRef", []TiffFormat{FormatString}, "2", 2, ""),
	30:    newExifTagDetailsExt(30, "GPSDifferential", []TiffFormat{FormatUint16}, "1", 1, ""),
	17:    newExifTagDetailsExt(17, "GPSImgDirection", []TiffFormat{FormatURational}, "1", 1, ""),
	16:    newExifTagDetailsExt(16, "GPSImgDirectionRef", []TiffFormat{FormatString}, "2", 2, ""),
	2:     newExifTagDetailsExt(2, "GPSLatitude", []TiffFormat{FormatURational}, "3", 3, "Indicates the latitude"),
	1:     newExifTagDetailsExt(1, "GPSLatitudeRef", []TiffFormat{FormatString}, "2", 2, "Indicates whether the latitude is north or south latitude"),
	4:     newExifTagDetailsExt(4, "GPSLongitude", []TiffFormat{FormatURational}, "3", 3, "Indicates the longitude."),
	3:     newExifTagDetailsExt(3, "GPSLongitudeRef", []TiffFormat{FormatString}, "2", 2, "Indicates whether the longitude is east or west longitude."),
	18:    newExifTagDetailsExt(18, "GPSMapDatum", []TiffFormat{FormatString}, "N", -1, ""),
	10:    newExifTagDetailsExt(10, "GPSMeasureMode", []TiffFormat{FormatString}, "2", 2, ""),
	27:    newExifTagDetailsExt(27, "GPSProcessingMethod", []TiffFormat{FormatUndefined}, "N", -1, ""),
	8:     newExifTagDetailsExt(8, "GPSSatellites", []TiffFormat{FormatString}, "N", -1, ""),
	13:    newExifTagDetailsExt(13, "GPSSpeed", []TiffFormat{FormatURational}, "1", 1, ""),
	12:    newExifTagDetailsExt(12, "GPSSpeedRef", []TiffFormat{FormatString}, "2", 2, ""),
	9:     newExifTagDetailsExt(9, "GPSStatus", []TiffFormat{FormatString}, "2", 2, ""),
	7:     newExifTagDetailsExt(7, "GPSTimeStamp", []TiffFormat{FormatURational}, "3", 3, ""),
	15:    newExifTagDetailsExt(15, "GPSTrack", []TiffFormat{FormatURational}, "1", 1, ""),
	14:    newExifTagDetailsExt(14, "GPSTrackRef", []TiffFormat{FormatString}, "2", 2, ""),
	0:     newExifTagDetailsExt(0, "GPSVersionID", []TiffFormat{FormatUint8}, "4", 4, "Indicates the version of GPSInfoIFD."),
	41991: newExifTagDetailsExt(41991, "GainControl", []TiffFormat{FormatUint16}, "1", 1, ""),
	34737: newExifTagDetailsExt(34737, "GeoAsciiParamsTag", []TiffFormat{FormatString}, "N", -1, ""),
	34736: newExifTagDetailsExt(34736, "GeoDoubleParamsTag", []TiffFormat{FormatFloat64}, "N", -1, ""),
	34735: newExifTagDetailsExt(34735, "GeoKeyDirectoryTag", []TiffFormat{FormatUint16}, "N &gt;= 4", -1, ""),
	400:   newExifTagDetailsExt(400, "GlobalParametersIFD", []TiffFormat{FormatString, FormatUint32}, "1", 1, "Used in the TIFF-FX standard to point to an IFD containing tags that are globally applicable to the complete TIFF file."),
	291:   newExifTagDetailsExt(291, "GrayResponseCurve", []TiffFormat{FormatUint16}, "2**BitsPerSample", -1, "For grayscale data, the optical density of each possible pixel value."),
	290:   newExifTagDetailsExt(290, "GrayResponseUnit", []TiffFormat{FormatUint16}, "1", 1, "The precision of the information contained in the GrayResponseCurve."),
	321:   newExifTagDetailsExt(321, "HalftoneHints", []TiffFormat{FormatUint16}, "2", 2, "Conveys to the halftone function the range of gray levels within a colorimetrically-specified image that should retain tonal detail."),
	316:   newExifTagDetailsExt(316, "HostComputer", []TiffFormat{FormatString}, "N", -1, "The computer and/or operating system in use at the time of image creation."),
	34908: newExifTagDetailsExt(34908, "HylaFAX FaxRecvParams", []TiffFormat{FormatUint32}, "1", 1, ""),
	34910: newExifTagDetailsExt(34910, "HylaFAX FaxRecvTime", []TiffFormat{FormatUint32}, "1", 1, ""),
	34909: newExifTagDetailsExt(34909, "HylaFAX FaxSubAddress", []TiffFormat{FormatString}, "N", -1, ""),
	33919: newExifTagDetailsExt(33919, "INGR Flag Registers", []TiffFormat{FormatUint32}, "16", 16, ""),
	33918: newExifTagDetailsExt(33918, "INGR Packet Data Tag", []TiffFormat{FormatUint16}, "N", -1, ""),
	34855: newExifTagDetailsExt(34855, "ISOSpeedRatings", []TiffFormat{FormatUint16}, "N", -1, "CCD sensitivity equivalent to Ag-Hr film speedrate. "),
	270:   newExifTagDetailsExt(270, "ImageDescription", []TiffFormat{FormatString}, "N", -1, "A string that describes the subject of the image."),
	32781: newExifTagDetailsExt(32781, "ImageID", []TiffFormat{FormatString}, "N", -1, "OPI-related."),
	34732: newExifTagDetailsExt(34732, "ImageLayer", []TiffFormat{FormatUint32, FormatUint16}, "2", 2, "Defined in the Mixed Raster Content part of RFC 2301, used to denote the particular function of this Image in the mixed raster scheme."),
	257:   newExifTagDetailsExt(257, "ImageLength", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "The number of rows of pixels in the image."),
	37724: newExifTagDetailsExt(37724, "ImageSourceData", []TiffFormat{FormatUndefined}, "N", -1, ""),
	42016: newExifTagDetailsExt(42016, "ImageUniqueID", []TiffFormat{FormatString}, "33", 33, "Indicates an identifier assigned uniquely to each image"),
	256:   newExifTagDetailsExt(256, "ImageWidth", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "The number of columns in the image, i.e., the number of pixels per row."),
	346:   newExifTagDetailsExt(346, "Indexed", []TiffFormat{FormatUint16}, "1", 1, "Aims to broaden the support for indexed images to include support for any color space."),
	333:   newExifTagDetailsExt(333, "InkNames", []TiffFormat{FormatString}, "N = total number of characters in all the ink name strings, including the NULs", -1, "The name of each ink used in a separated image."),
	332:   newExifTagDetailsExt(332, "InkSet", []TiffFormat{FormatUint16}, "1", 1, "The set of inks used in a separated (PhotometricInterpretation=5) image."),
	40965: newExifTagDetailsExt(40965, "Interoperability IFD", []TiffFormat{FormatUint32, FormatString}, "1", 1, "*AltName(ExifInteroperabilityOffset) Extension of 'ExifR98', detail is unknown. This value is offset to IFD format data. Currently there are 2 directory entries, first one is Tag0x0001, value is 'R98', next is Tag0x0002, value is '0100'. "),
	33920: newExifTagDetailsExt(33920, "IrasB Transformation Matrix", []TiffFormat{FormatFloat64}, "17 (possibly 16, but unlikely)", -1, ""),
	521:   newExifTagDetailsExt(521, "JPEGACTables", []TiffFormat{FormatUint32}, "N = SamplesPerPixel", -1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	520:   newExifTagDetailsExt(520, "JPEGDCTables", []TiffFormat{FormatUint32}, "N = SamplesPerPixel", -1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	513:   newExifTagDetailsExt(513, "JPEGInterchangeFormat", []TiffFormat{FormatUint32}, "1", 1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	514:   newExifTagDetailsExt(514, "JPEGInterchangeFormatLength", []TiffFormat{FormatUint32}, "1", 1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	517:   newExifTagDetailsExt(517, "JPEGLosslessPredictors", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	518:   newExifTagDetailsExt(518, "JPEGPointTransforms", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	512:   newExifTagDetailsExt(512, "JPEGProc", []TiffFormat{FormatUint16}, "1", 1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	519:   newExifTagDetailsExt(519, "JPEGQTables", []TiffFormat{FormatUint32}, "N = SamplesPerPixel", -1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	515:   newExifTagDetailsExt(515, "JPEGRestartInterval", []TiffFormat{FormatUint16}, "1", 1, "Old-style JPEG compression field. TechNote2 invalidates this part of the specification."),
	347:   newExifTagDetailsExt(347, "JPEGTables", []TiffFormat{FormatUndefined}, "N = number of bytes in tables datastream", -1, "JPEG quantization and/or Huffman tables."),
	50736: newExifTagDetailsExt(50736, "LensInfo", []TiffFormat{FormatURational}, "4", 4, ""),
	37384: newExifTagDetailsExt(37384, "LightSource", []TiffFormat{FormatUint16}, "1", 1, "Light source, actually this means white balance setting. '0' means auto, '1' daylight, '2' fluorescent, '3' tungsten, '10' flash. "),
	50734: newExifTagDetailsExt(50734, "LinearResponseLimit", []TiffFormat{FormatURational}, "1", 1, ""),
	50712: newExifTagDetailsExt(50712, "LinearizationTable", []TiffFormat{FormatUint16}, "N", -1, ""),
	50709: newExifTagDetailsExt(50709, "LocalizedCameraModel", []TiffFormat{FormatString, FormatUint8}, "N", -1, ""),
	33447: newExifTagDetailsExt(33447, "MD ColorTable", []TiffFormat{FormatUint16}, "n", -1, ""),
	33445: newExifTagDetailsExt(33445, "MD FileTag", []TiffFormat{FormatUint32}, "1", 1, ""),
	33452: newExifTagDetailsExt(33452, "MD FileUnits", []TiffFormat{FormatString}, "N", -1, ""),
	33448: newExifTagDetailsExt(33448, "MD LabName", []TiffFormat{FormatString}, "n", -1, ""),
	33450: newExifTagDetailsExt(33450, "MD PrepDate", []TiffFormat{FormatString}, "n", -1, ""),
	33451: newExifTagDetailsExt(33451, "MD PrepTime", []TiffFormat{FormatString}, "N", -1, ""),
	33449: newExifTagDetailsExt(33449, "MD SampleInfo", []TiffFormat{FormatString}, "N", -1, ""),
	33446: newExifTagDetailsExt(33446, "MD ScalePixel", []TiffFormat{FormatURational}, "1", 1, ""),
	271:   newExifTagDetailsExt(271, "Make", []TiffFormat{FormatString}, "N", -1, "The scanner manufacturer."),
	37500: newExifTagDetailsExt(37500, "MakerNote", []TiffFormat{FormatUndefined}, "N", -1, "Maker dependent internal data. Some of maker such as Olympus/Nikon/Sanyo etc. uses IFD format for this area. "),
	50741: newExifTagDetailsExt(50741, "MakerNoteSafety", []TiffFormat{FormatUint16}, "1", 1, ""),
	50830: newExifTagDetailsExt(50830, "MaskedAreas", []TiffFormat{FormatUint32, FormatUint16}, "4 * number of rectangles", -1, ""),
	37381: newExifTagDetailsExt(37381, "MaxApertureValue", []TiffFormat{FormatURational}, "1", 1, "Maximum aperture value of lens. You can convert to F-number by calculating power of root 2 (same process of ApertureValue(0x9202). "),
	281:   newExifTagDetailsExt(281, "MaxSampleValue", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "The maximum component value used."),
	37383: newExifTagDetailsExt(37383, "MeteringMode", []TiffFormat{FormatUint16}, "1", 1, "Exposure metering method. '1' means average, '2' center weighted average, '3' spot, '4' multi-spot, '5' multi-segment. "),
	280:   newExifTagDetailsExt(280, "MinSampleValue", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "The minimum component value used."),
	405:   newExifTagDetailsExt(405, "ModeNumber", []TiffFormat{FormatUint8}, "1", 1, "Used in the TIFF-FX standard, denotes the mode of the standard specified by the FaxProfile field."),
	272:   newExifTagDetailsExt(272, "Model", []TiffFormat{FormatString}, "N", -1, "The scanner model name or number."),
	33550: newExifTagDetailsExt(33550, "ModelPixelScaleTag", []TiffFormat{FormatFloat64}, "3", 3, ""),
	33922: newExifTagDetailsExt(33922, "ModelTiepointTag", []TiffFormat{FormatFloat64}, "N = 6*K, with K = number of tiepoints", -1, ""),
	34264: newExifTagDetailsExt(34264, "ModelTransformationTag", []TiffFormat{FormatFloat64}, "16", 16, ""),
	51111: newExifTagDetailsExt(51111, "NewRawImageDigest", []TiffFormat{FormatUint8}, "16", 16, ""),
	254:   newExifTagDetailsExt(254, "NewSubfileType", []TiffFormat{FormatUint32}, "1", 1, "A general indication of the kind of data contained in this subfile."),
	51041: newExifTagDetailsExt(51041, "NoiseProfile", []TiffFormat{FormatFloat64}, "2 or 2 * ColorPlanes", -1, ""),
	50935: newExifTagDetailsExt(50935, "NoiseReductionApplied", []TiffFormat{FormatURational}, "1", 1, ""),
	334:   newExifTagDetailsExt(334, "NumberOfInks", []TiffFormat{FormatUint16}, "1", 1, "The number of inks."),
	34856: newExifTagDetailsExt(34856, "OECF", []TiffFormat{FormatUndefined}, "N", -1, ""),
	351:   newExifTagDetailsExt(351, "OPIProxy", []TiffFormat{FormatUint16}, "1", 1, "OPI-related."),
	50216: newExifTagDetailsExt(50216, "Oce Application Selector", []TiffFormat{FormatString}, "N", -1, ""),
	50217: newExifTagDetailsExt(50217, "Oce Identification Number", []TiffFormat{FormatString}, "N", -1, ""),
	50218: newExifTagDetailsExt(50218, "Oce ImageLogic Characteristics", []TiffFormat{FormatString}, "N", -1, ""),
	50215: newExifTagDetailsExt(50215, "Oce Scanjob Description", []TiffFormat{FormatString}, "N", -1, ""),
	51008: newExifTagDetailsExt(51008, "OpcodeList1", []TiffFormat{FormatUndefined}, "N", -1, ""),
	51009: newExifTagDetailsExt(51009, "OpcodeList2", []TiffFormat{FormatUndefined}, "N", -1, ""),
	51022: newExifTagDetailsExt(51022, "OpcodeList3", []TiffFormat{FormatUndefined}, "N", -1, ""),
	274:   newExifTagDetailsExt(274, "Orientation", []TiffFormat{FormatUint16}, "1", 1, "The orientation of the image with respect to the rows and columns."),
	51090: newExifTagDetailsExt(51090, "OriginalBestQualityFinalSize", []TiffFormat{FormatUint16, FormatUint32}, "2", 2, ""),
	51091: newExifTagDetailsExt(51091, "OriginalDefaultCropSize", []TiffFormat{FormatURational, FormatUint16, FormatUint32}, "2", 2, ""),
	51089: newExifTagDetailsExt(51089, "OriginalDefaultFinalSize", []TiffFormat{FormatUint16, FormatUint32}, "2", 2, ""),
	50828: newExifTagDetailsExt(50828, "OriginalRawFileData", []TiffFormat{FormatUndefined}, "N", -1, ""),
	50973: newExifTagDetailsExt(50973, "OriginalRawFileDigest", []TiffFormat{FormatUint8}, "16", 16, ""),
	50827: newExifTagDetailsExt(50827, "OriginalRawFileName", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	285:   newExifTagDetailsExt(285, "PageName", []TiffFormat{FormatString}, "N", -1, "The name of the page from which this image was scanned."),
	297:   newExifTagDetailsExt(297, "PageNumber", []TiffFormat{FormatUint16}, "2", 2, "The page number of the page from which this image was scanned."),
	262:   newExifTagDetailsExt(262, "PhotometricInterpretation", []TiffFormat{FormatUint16}, "1", 1, "The color space of the image data."),
	34377: newExifTagDetailsExt(34377, "Photoshop", []TiffFormat{FormatUint8}, "N", -1, ""),
	40962: newExifTagDetailsExt(40962, "PixelXDimension", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "*AltName(ExifImageWidth) Size of main image. "),
	40963: newExifTagDetailsExt(40963, "PixelYDimension", []TiffFormat{FormatUint32, FormatUint16}, "1", 1, "*AltName(ExifImageHeight) ExifImageHeight "),
	284:   newExifTagDetailsExt(284, "PlanarConfiguration", []TiffFormat{FormatUint16}, "1", 1, "How the components of each pixel are stored."),
	317:   newExifTagDetailsExt(317, "Predictor", []TiffFormat{FormatUint16}, "1", 1, "A mathematical operator that is applied to the image data before an encoding scheme is applied."),
	50966: newExifTagDetailsExt(50966, "PreviewApplicationName", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	50967: newExifTagDetailsExt(50967, "PreviewApplicationVersion", []TiffFormat{FormatString, FormatUint8}, "N", -1, ""),
	50970: newExifTagDetailsExt(50970, "PreviewColorSpace", []TiffFormat{FormatUint32}, "1", 1, ""),
	50971: newExifTagDetailsExt(50971, "PreviewDateTime", []TiffFormat{FormatString}, "N", -1, ""),
	50969: newExifTagDetailsExt(50969, "PreviewSettingsDigest", []TiffFormat{FormatUint8}, "16", 16, ""),
	50968: newExifTagDetailsExt(50968, "PreviewSettingsName", []TiffFormat{FormatString, FormatUint8}, "N", -1, ""),
	319:   newExifTagDetailsExt(319, "PrimaryChromaticities", []TiffFormat{FormatURational}, "6", 6, "The chromaticities of the primaries of the image."),
	50932: newExifTagDetailsExt(50932, "ProfileCalibrationSignature", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	50942: newExifTagDetailsExt(50942, "ProfileCopyright", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	50941: newExifTagDetailsExt(50941, "ProfileEmbedPolicy", []TiffFormat{FormatUint32}, "1", 1, ""),
	50938: newExifTagDetailsExt(50938, "ProfileHueSatMapData1", []TiffFormat{FormatFloat32}, "HueDivisions * SaturationDivisions * ValueDivisions * 3", -1, ""),
	50939: newExifTagDetailsExt(50939, "ProfileHueSatMapData2", []TiffFormat{FormatFloat32}, "HueDivisions * SaturationDivisions * ValueDivisions * 3", -1, ""),
	50937: newExifTagDetailsExt(50937, "ProfileHueSatMapDims", []TiffFormat{FormatUint32}, "3", 3, ""),
	51107: newExifTagDetailsExt(51107, "ProfileHueSatMapEncoding", []TiffFormat{FormatUint32}, "1", 1, ""),
	50982: newExifTagDetailsExt(50982, "ProfileLookTableData", []TiffFormat{FormatFloat32}, "HueDivisions * SaturationDivisions * ValueDivisions * 3", -1, ""),
	50981: newExifTagDetailsExt(50981, "ProfileLookTableDims", []TiffFormat{FormatUint32}, "3", 3, ""),
	51108: newExifTagDetailsExt(51108, "ProfileLookTableEncoding", []TiffFormat{FormatUint32}, "1", 1, ""),
	50936: newExifTagDetailsExt(50936, "ProfileName", []TiffFormat{FormatUint8, FormatString}, "N", -1, ""),
	50940: newExifTagDetailsExt(50940, "ProfileToneCurve", []TiffFormat{FormatFloat32}, "Samples * 2", -1, ""),
	401:   newExifTagDetailsExt(401, "ProfileType", []TiffFormat{FormatUint32}, "1", 1, "Used in the TIFF-FX standard, denotes the type of data stored in this file or IFD."),
	50781: newExifTagDetailsExt(50781, "RawDataUniqueID", []TiffFormat{FormatUint8}, "16", 16, ""),
	50972: newExifTagDetailsExt(50972, "RawImageDigest", []TiffFormat{FormatUint8}, "16", 16, ""),
	51112: newExifTagDetailsExt(51112, "RawToPreviewGain", []TiffFormat{FormatFloat64}, "1", 1, ""),
	50725: newExifTagDetailsExt(50725, "ReductionMatrix1", []TiffFormat{FormatURational, FormatRational}, "3 * ColorPlanes", -1, ""),
	50726: newExifTagDetailsExt(50726, "ReductionMatrix2", []TiffFormat{FormatURational, FormatRational}, "3 * ColorPlanes", -1, ""),
	532:   newExifTagDetailsExt(532, "ReferenceBlackWhite", []TiffFormat{FormatURational}, "6", 6, "Specifies a pair of headroom and footroom image data values (codes) for each pixel component."),
	40964: newExifTagDetailsExt(40964, "RelatedSoundFile", []TiffFormat{FormatString}, "13", 13, "If this digicam can record audio data with image, shows name of audio data. "),
	296:   newExifTagDetailsExt(296, "ResolutionUnit", []TiffFormat{FormatUint16}, "1", 1, "The unit of measurement for XResolution and YResolution."),
	50975: newExifTagDetailsExt(50975, "RowInterleaveFactor", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, ""),
	278:   newExifTagDetailsExt(278, "RowsPerStrip", []TiffFormat{FormatUint16, FormatUint32}, "1", 1, "The number of rows per strip."),
	341:   newExifTagDetailsExt(341, "SMaxSampleValue", []TiffFormat{FormatFloat64, FormatURational, FormatUint16, FormatUint32, FormatUint8}, "N = SamplesPerPixel", -1, "Specifies the maximum sample value."),
	340:   newExifTagDetailsExt(340, "SMinSampleValue", []TiffFormat{FormatUint16, FormatUint32, FormatUint8, FormatFloat64, FormatURational}, "N = SamplesPerPixel", -1, "Specifies the minimum sample value."),
	339:   newExifTagDetailsExt(339, "SampleFormat", []TiffFormat{FormatUint16}, "N = SamplesPerPixel", -1, "Specifies how to interpret each data sample in a pixel."),
	277:   newExifTagDetailsExt(277, "SamplesPerPixel", []TiffFormat{FormatUint16}, "1", 1, "The number of components per pixel."),
	41993: newExifTagDetailsExt(41993, "Saturation", []TiffFormat{FormatUint16}, "1", 1, ""),
	41990: newExifTagDetailsExt(41990, "SceneCaptureType", []TiffFormat{FormatUint16}, "1", 1, "Indicates the type of scene that was shot."),
	41729: newExifTagDetailsExt(41729, "SceneType", []TiffFormat{FormatUndefined}, "1", 1, "Unknown but value is '1'. "),
	41495: newExifTagDetailsExt(41495, "SensingMethod", []TiffFormat{FormatUint16}, "1", 1, "Shows type of image sensor unit. '2' means 1 chip color area sensor, most of all digicam use this type. "),
	50739: newExifTagDetailsExt(50739, "ShadowScale", []TiffFormat{FormatURational}, "1", 1, ""),
	41994: newExifTagDetailsExt(41994, "Sharpness", []TiffFormat{FormatUint16}, "1", 1, ""),
	37377: newExifTagDetailsExt(37377, "ShutterSpeedValue", []TiffFormat{FormatRational, FormatURational}, "1", 1, "Shutter speed. To convert this value to ordinary 'Shutter Speed'; calculate this value's power of 2, then reciprocal. For example, if value is '4', shutter speed is 1/(2^4)=1/16 second. "),
	305:   newExifTagDetailsExt(305, "Software", []TiffFormat{FormatString}, "N", -1, "Name and version number of the software package(s) used to create the image."),
	41484: newExifTagDetailsExt(41484, "SpatialFrequencyResponse", []TiffFormat{FormatUndefined}, "N", -1, ""),
	34852: newExifTagDetailsExt(34852, "SpectralSensitivity", []TiffFormat{FormatString}, "N", -1, ""),
	279:   newExifTagDetailsExt(279, "StripByteCounts", []TiffFormat{FormatUint16, FormatUint32}, "N = StripsPerImage for PlanarConfiguration equal to 1; N = SamplesPerPixel * StripsPerImage for PlanarConfiguration equal to 2", -1, "For each strip, the number of bytes in the strip after compression."),
	273:   newExifTagDetailsExt(273, "StripOffsets", []TiffFormat{FormatUint16, FormatUint32}, "N = StripsPerImage for PlanarConfiguration equal to 1; N = SamplesPerPixel * StripsPerImage for PlanarConfiguration equal to 2", -1, "For each strip, the byte offset of that strip."),
	559:   newExifTagDetailsExt(559, "StripRowCounts", []TiffFormat{FormatUint32}, "number of strips", -1, "Defined in the Mixed Raster Content part of RFC 2301, used to replace RowsPerStrip for IFDs with variable-sized strips."),
	330:   newExifTagDetailsExt(330, "SubIFDs", []TiffFormat{FormatUint32, FormatString}, "N = number of child IFDs", -1, "Offset to child IFDs."),
	50974: newExifTagDetailsExt(50974, "SubTileBlockSize", []TiffFormat{FormatUint16, FormatUint32}, "2", 2, ""),
	255:   newExifTagDetailsExt(255, "SubfileType", []TiffFormat{FormatUint16}, "1", 1, "A general indication of the kind of data contained in this subfile."),
	37382: newExifTagDetailsExt(37382, "SubjectDistance", []TiffFormat{FormatURational}, "1", 1, "Distance to focus point, unit is meter. "),
	41996: newExifTagDetailsExt(41996, "SubjectDistanceRange", []TiffFormat{FormatUint16}, "1", 1, ""),
	41492: newExifTagDetailsExt(41492, "SubjectLocation", []TiffFormat{FormatUint16}, "2", 2, ""),
	37520: newExifTagDetailsExt(37520, "SubsecTime", []TiffFormat{FormatString}, "N", -1, "Used to record fractions of seconds for the DateTime tag"),
	37522: newExifTagDetailsExt(37522, "SubsecTimeDigitized", []TiffFormat{FormatString}, "N", -1, "Used to record fractions of seconds for the DateTimeDigitized tag."),
	37521: newExifTagDetailsExt(37521, "SubsecTimeOriginal", []TiffFormat{FormatString}, "N", -1, "Used to record fractions of seconds for the DateTimeOriginal tag."),
	292:   newExifTagDetailsExt(292, "T4Options", []TiffFormat{FormatUint32}, "1", 1, "Options for Group 3 Fax compression"),
	293:   newExifTagDetailsExt(293, "T6Options", []TiffFormat{FormatUint32}, "1", 1, "Options for Group 4 Fax compression"),
	337:   newExifTagDetailsExt(337, "TargetPrinter", []TiffFormat{FormatString}, "N", -1, "A description of the printing environment for which this separation is intended."),
	263:   newExifTagDetailsExt(263, "Threshholding", []TiffFormat{FormatUint16}, "1", 1, "For black and white TIFF files that represent shades of gray, the technique used to convert from gray to black and white pixels."),
	325:   newExifTagDetailsExt(325, "TileByteCounts", []TiffFormat{FormatUint16, FormatUint32}, "N = TilesPerImage for PlanarConfiguration = 1; N = SamplesPerPixel * TilesPerImage for PlanarConfiguration = 2", -1, "For each tile, the number of (compressed) bytes in that tile."),
	323:   newExifTagDetailsExt(323, "TileLength", []TiffFormat{FormatUint32, FormatUint16}, "1", 1, "The tile length (height) in pixels. This is the number of rows in each tile."),
	324:   newExifTagDetailsExt(324, "TileOffsets", []TiffFormat{FormatUint32}, "N = TilesPerImage for PlanarConfiguration = 1; N = SamplesPerPixel * TilesPerImage for PlanarConfiguration = 2", -1, "For each tile, the byte offset of that tile, as compressed and stored on disk."),
	322:   newExifTagDetailsExt(322, "TileWidth", []TiffFormat{FormatUint32, FormatUint16}, "1", 1, "The tile width in pixels. This is the number of columns in each tile."),
	301:   newExifTagDetailsExt(301, "TransferFunction", []TiffFormat{FormatUint16}, "(1 or 3) * (1 &lt;&lt; BitsPerSample)", -1, "Describes a transfer function for the image in tabular style."),
	342:   newExifTagDetailsExt(342, "TransferRange", []TiffFormat{FormatUint16}, "6", 6, "Expands the range of the TransferFunction."),
	50708: newExifTagDetailsExt(50708, "UniqueCameraModel", []TiffFormat{FormatString}, "N", -1, ""),
	37510: newExifTagDetailsExt(37510, "UserComment", []TiffFormat{FormatUndefined}, "N", -1, "Stores user comment. "),
	404:   newExifTagDetailsExt(404, "VersionYear", []TiffFormat{FormatUint8}, "4", 4, "Used in the TIFF-FX standard, denotes the year of the standard specified by the FaxProfile field."),
	32932: newExifTagDetailsExt(32932, "Wang Annotation", []TiffFormat{FormatUint8}, "N", -1, ""),
	41987: newExifTagDetailsExt(41987, "WhiteBalance", []TiffFormat{FormatUint16}, "1", 1, "Indicates the white balance mode set when the image was shot."),
	50717: newExifTagDetailsExt(50717, "WhiteLevel", []TiffFormat{FormatUint16, FormatUint32, FormatURational}, "SamplesPerPixel", -1, ""),
	318:   newExifTagDetailsExt(318, "WhitePoint", []TiffFormat{FormatURational}, "2", 2, "The chromaticity of the white point of the image."),
	344:   newExifTagDetailsExt(344, "XClipPathUnits", []TiffFormat{FormatUint32}, "1", 1, "The number of units that span the width of the image, in terms of integer ClipPath coordinates."),
	700:   newExifTagDetailsExt(700, "XMP", []TiffFormat{FormatUint8}, "N", -1, "XML packet containing XMP metadata"),
	286:   newExifTagDetailsExt(286, "XPosition", []TiffFormat{FormatURational}, "1", 1, "X position of the image."),
	282:   newExifTagDetailsExt(282, "XResolution", []TiffFormat{FormatURational}, "1", 1, "The number of pixels per ResolutionUnit in the ImageWidth direction."),
	529:   newExifTagDetailsExt(529, "YCbCrCoefficients", []TiffFormat{FormatURational}, "3", 3, "The transformation from RGB to YCbCr image data."),
	531:   newExifTagDetailsExt(531, "YCbCrPositioning", []TiffFormat{FormatUint16}, "1", 1, "Specifies the positioning of subsampled chrominance components relative to luminance samples."),
	530:   newExifTagDetailsExt(530, "YCbCrSubSampling", []TiffFormat{FormatUint16}, "2", 2, "Specifies the subsampling factors used for the chrominance components of a YCbCr image."),
	345:   newExifTagDetailsExt(345, "YClipPathUnits", []TiffFormat{FormatUint32}, "1", 1, "The number of units that span the height of the image, in terms of integer ClipPath coordinates."),
	287:   newExifTagDetailsExt(287, "YPosition", []TiffFormat{FormatURational}, "1", 1, "Y position of the image."),
	283:   newExifTagDetailsExt(283, "YResolution", []TiffFormat{FormatURational}, "1", 1, "The number of pixels per ResolutionUnit in the ImageLength direction."),
}
