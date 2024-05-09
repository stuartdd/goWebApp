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
									line.WriteString(v.formatName)
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

func getTiffFormatsFromNames(formatNames string) []*TagFormat {
	resp := []*TagFormat{}
	for n, v := range mapTiffFormetNames {
		if strings.Contains(formatNames, n) {
			resp = append(resp, getTiffFormats(v))
		}
	}
	return resp
}

func getTiffFormats(tf TiffFormat) *TagFormat {
	for _, v := range mapTiffFormats {
		if v.tiffFormat == tf {
			return v
		}
	}
	return nil
}

var mapTiffFormetNames = map[string]TiffFormat{
	"ASCII":     FormatString,
	"SHORT":     FormatUint16,
	"LONG":      FormatUint32,
	"RATIONAL":  FormatURational,
	"BYTE":      FormatUint8,
	"SRATIONAL": FormatRational,
	"IFD":       FormatString,
	"UNDEFINED": FormatUndefined,
	"FLOAT":     FormatFloat32,
	"DOUBLE":    FormatFloat64,
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
