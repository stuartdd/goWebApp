package config

import (
	"fmt"
	"strings"
)

var contentTypesMap = makeContentTypesMap()
var contentTypesCharset = "utf-8"

const DefaultContentType = "application/json"

/*
from : https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Complete_list_of_MIME_types
*/
func makeContentTypesMap() map[string]string {
	mime := make(map[string]string)
	mime["aac"] = "audio/aac"
	mime["abw"] = "application/x-abiword"
	mime["arc"] = "application/x-freearc"
	mime["avi"] = "video/x-msvideo"
	mime["azw"] = "application/vnd.amazon.ebook"
	mime["bin"] = "application/octet-stream"
	mime["bmp"] = "image/bmp"
	mime["bz"] = "application/x-bzip"
	mime["bz2"] = "application/x-bzip2"
	mime["csh"] = "application/x-csh"
	mime["css"] = "text/css%0"
	mime["csv"] = "text/csv%0"
	mime["doc"] = "application/msword"
	mime["docx"] = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	mime["eot"] = "application/vnd.ms-fontobject"
	mime["epub"] = "application/epub+zip"
	mime["gif"] = "image/gif"
	mime["htm"] = "text/html%0"
	mime["html"] = "text/html%0"
	mime["ico"] = "image/vnd.microsoft.icon" // Some browsers use image/x-icon. Add to config data to override!
	mime["ics"] = "text/calendar%0"
	mime["jar"] = "application/java-archive"
	mime["jpeg"] = "image/jpeg"
	mime["jpg"] = "image/jpeg"
	mime["js"] = "text/javascript%0"
	mime["json"] = "application/json%0"
	mime["jsonld"] = "application/ld+json%0"
	mime["mid"] = "audio/midi audio/x-midi"
	mime["midi"] = "audio/midi audio/x-midi"
	mime["mjs"] = "text/javascript%0"
	mime["mp3"] = "audio/mpeg"
	mime["mpeg"] = "video/mpeg"
	mime["mpkg"] = "application/vnd.apple.installer+xml"
	mime["odp"] = "application/vnd.oasis.opendocument.presentation"
	mime["ods"] = "application/vnd.oasis.opendocument.spreadsheet"
	mime["odt"] = "application/vnd.oasis.opendocument.text"
	mime["oga"] = "audio/ogg"
	mime["ogv"] = "video/ogg"
	mime["ogx"] = "application/ogg"
	mime["otf"] = "font/otf"
	mime["png"] = "image/png"
	mime["pdf"] = "application/pdf"
	mime["ppt"] = "application/vnd.ms-powerpoint"
	mime["pptx"] = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	mime["rar"] = "application/x-rar-compressed"
	mime["rtf"] = "application/rtf%0"
	mime["sh"] = "application/x-sh"
	mime["svg"] = "image/svg+xml%0"
	mime["swf"] = "application/x-shockwave-flash"
	mime["tar"] = "application/x-tar"
	mime["tif"] = "image/tiff"
	mime["tiff"] = "image/tiff"
	mime["ts"] = "video/mp2t"
	mime["ttf"] = "font/ttf"
	mime["txt"] = "text/plain%0"
	mime["vsd"] = "application/vnd.visio"
	mime["wav"] = "audio/wav"
	mime["wasm"] = "application/wasm"
	mime["weba"] = "audio/webm"
	mime["webm"] = "video/webm"
	mime["webp"] = "image/webp"
	mime["woff"] = "font/woff"
	mime["woff2"] = "font/woff2"
	mime["xhtml"] = "application/xhtml+xml%0"
	mime["xls"] = "application/vnd.ms-excel"
	mime["xlsx"] = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	mime["xml"] = "application/xml%0"
	mime["xul"] = "application/vnd.mozilla.xul+xml"
	mime["zip"] = "application/zip"
	mime["7z"] = "application/x-7z-compressed"

	mime["log"] = "text/plain%0"

	return mime
}

/*
AddNewContentTypeToMap for a given url return the content type based on the .ext
*/

func SetContentTypeCharset(charset string) {
	contentTypesCharset = charset
}

/*
LookupContentType for a given url return the content type based on the .ext

	    A mapping ends with %0 it will have '; charset=x' added where x is defined
		in config 'ContentTypeCharset'. If ContentTypeCharset is empty then no
		charset is added.
*/
func LookupContentType(cType string) string {
	ext := cType
	pos := strings.LastIndex(cType, ".")
	if pos > 0 {
		ext = cType[pos+1:]
	}
	mapping, found := contentTypesMap[ext]
	if found {
		if strings.HasSuffix(mapping, "%0") {
			if contentTypesCharset == "" {
				return mapping
			}
			return strings.ReplaceAll(mapping, "%0", fmt.Sprintf("; charset=%s", contentTypesCharset))
		}
		return mapping
	}
	return DefaultContentType
}

func HasContentType(cType string) bool {
	ext := cType
	pos := strings.LastIndex(cType, ".")
	if pos > 0 {
		ext = cType[pos+1:]
	}
	_, found := contentTypesMap[ext]
	return found
}
