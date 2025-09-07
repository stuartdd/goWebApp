package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/stuartdd/goWebApp/config"
)

// "{\"Alloc\":\"2 MiB (2309672 B)\",\"Sys\":\"12 MiB (12672016 B)\",\"TotalAlloc\":\"2 MiB (2309672 B)\",\"configName\":\"goWebApp.json\",\"error\":false,\"reloadConfig\":3080.27,\"upSince\":\"Fri Apr  5 12:48:19 2024\",\"upTime\":\"00:08:39\"}"
// "[{\"error\":false,}{\"Alloc\":\"1 MiB (1368424 B)\"}]"
func GetServerStatusAsJson(configData *config.ConfigData, logFileName string, upSince time.Time, longRunningJson string) []byte {
	var b bytes.Buffer
	var st runtime.MemStats
	runtime.ReadMemStats(&st)
	b.WriteRune('{')
	writeParamAsJsonString("error", "false", false, false, true, &b)
	b.WriteString("\"status\": {")
	writeParamAsJsonString("ConfigName", filepath.Base(configData.ConfigName), true, false, true, &b)
	writeParamAsJsonString("UpSince", upSince.Format(time.ANSIC), true, false, true, &b)
	writeParamAsJsonString("UpTime", fmtDuration(time.Since(upSince)), true, false, true, &b)
	writeParamAsJsonString("Reload Config in", fmt.Sprintf("%.0f seconds", configData.GetTimeToReloadConfig()), true, false, true, &b)
	writeParamAsJsonString("Alloc", fmtAlloc(st.Alloc), true, false, true, &b)
	writeParamAsJsonString("TotalAlloc", fmtAlloc(st.TotalAlloc), true, false, true, &b)
	writeParamAsJsonString("Sys", fmtAlloc(st.Sys), true, false, true, &b)
	writeParamAsJsonString("Processes", longRunningJson, false, false, true, &b)
	writeParamAsJsonString("OS", GetOSFreeData(configData), false, false, true, &b)
	writeParamAsJsonString("Log_Dir", configData.GetLogDataPathForStatus(), true, false, true, &b)
	writeParamAsJsonString("Log_File", logFileName, true, false, false, &b)
	b.WriteRune('}')
	b.WriteRune('}')
	return b.Bytes()
}

func execDataAsJson(execId string, rc int, stdOut []byte, stdErr []byte) []byte {
	status := http.StatusOK
	m := make(map[string]interface{})
	if rc != 0 {
		m["error"] = true
		status = http.StatusPartialContent
	} else {
		m["error"] = false
	}
	m["status"] = status
	m["msg"] = http.StatusText(status)
	m["rc"] = rc
	m["id"] = execId
	m["stdOut"] = string(stdOut)
	m["stdErr"] = string(stdErr)

	v, err := json.Marshal(m)
	if err != nil {
		panic(NewControllerError("controllers:execDataAsJson:Marshal", http.StatusInternalServerError, fmt.Sprintf("JSON Marshal:Error:%s", err.Error())))
	}
	return v
}

func statusAsJson(status int, cause string, error bool) []byte {
	m := make(map[string]interface{})
	m["error"] = error
	m["status"] = status
	m["msg"] = http.StatusText(status)
	m["cause"] = cause
	v, err := json.Marshal(m)
	if err != nil {
		panic(NewControllerError("controllers:statusAsJson:Marshal", http.StatusInternalServerError, fmt.Sprintf("JSON Marshal:Error:%s", err.Error())))
	}
	return v
}

func treeAsJson(root *TreeDirNode, params *UrlRequestParts) []byte {
	var buffer bytes.Buffer
	buffer.WriteRune('{')
	writeJsonHeader(params, &buffer)
	buffer.WriteString(",\"tree\":")
	buffer.WriteString(string(root.ToJson(false)))
	buffer.WriteString("}")
	return buffer.Bytes()
}

func listFilesAsJson(ents []fs.DirEntry, params *UrlRequestParts, verbose func(string), path string, index int) []byte {
	var buffer bytes.Buffer
	entLen := len(ents)
	buffer.WriteRune('{')
	writeJsonHeader(params, &buffer)
	buffer.WriteRune(',')
	writePathToJson(params.GetOptionalParam(PathParam, ""), PathParam, &buffer)
	buffer.WriteString("\"files\":[")
	count := 0
	if index != -1 {
		if index >= entLen {
			index = entLen - 1
		}
		e := ents[index]
		if filterDirNames(e, params.GetConfigFileFilter()) {
			writeSingleFileNameToJson(e, &buffer)
			count++
		}
	} else {
		bufLen := buffer.Len()
		for i := len(ents) - 1; i >= 0; i-- {
			e := ents[i]
			if filterDirNames(e, params.GetConfigFileFilter()) {
				writeSingleFileNameToJson(e, &buffer)
				bufLen = buffer.Len()
				buffer.WriteRune(',')
				count++
			}
		}
		buffer.Truncate(bufLen)
	}

	buffer.WriteString("]}")
	if verbose != nil {
		verbose(fmt.Sprintf("ListFilesAsJson:%s Returned[%d]", path, count))
	}

	return buffer.Bytes()
}

func listDirectoriesAsJson(dir string, param *UrlRequestParts, verbose func(string), path string) []byte {
	list := &[]string{}
	listDirectoriesRec(dir, dir+string(os.PathSeparator), param.GetConfigFileFilter(), list)
	listLen := len(*list)
	var buffer bytes.Buffer
	buffer.WriteRune('{')
	writeJsonHeader(param, &buffer)
	buffer.WriteString(",\"paths\":[")
	count := 0
	for i, s := range *list {
		buffer.WriteString("{\"name\":\"")
		buffer.WriteString(s)
		buffer.WriteString("\",")
		buffer.WriteString("\"encName\":\"")
		buffer.WriteString(encodeValue(s))
		buffer.WriteString("\"}")
		if i < (listLen - 1) {
			buffer.WriteRune(',')
		}
		count++
	}
	buffer.WriteString("]}")
	if verbose != nil {
		verbose(fmt.Sprintf("listDirectoriesAsJson:%s Returned[%d]", path, count))
	}
	return buffer.Bytes()
}

func listDirectoriesRec(path string, root string, filter []string, l *[]string) {
	entries, _ := os.ReadDir(path)
	dirCount := 0
	fileCount := 0
	for _, ent := range entries {
		n := ent.Name()
		if ent.IsDir() {
			if !strings.HasPrefix(n, ".") && !strings.HasPrefix(n, "_") {
				dirCount++
			}
		} else {
			if filterFileNames(ent.Name(), filter) {
				fileCount++
			}
		}
	}
	if (fileCount > 0) && strings.HasPrefix(path, root) {
		*l = append(*l, path[len(root):])
	}
	for _, ent := range entries {
		if ent.IsDir() {
			p := filepath.Join(path, ent.Name())
			listDirectoriesRec(p, root, filter, l)
		}
	}
}

func writePathToJson(pathUnencoded string, key string, buffer *bytes.Buffer) {
	buffer.WriteRune('"')
	buffer.WriteString(key)
	buffer.WriteString("\":")
	if pathUnencoded != "" {
		buffer.WriteString("{\"name\":\"")
		buffer.WriteString(pathUnencoded)
		buffer.WriteString("\", \"encName\":\"")
		buffer.WriteString(encodeValue(pathUnencoded))
		buffer.WriteString("\"},")
	} else {
		buffer.WriteString("null,")
	}
}

func writeJsonHeader(param *UrlRequestParts, buffer *bytes.Buffer) {
	writeErrorAsJsonString(false, buffer)
	writeParamAsJsonString(UserParam, param.GetOptionalParam(UserParam, ""), true, true, false, buffer)
	writeParamAsJsonString(LocationParam, param.GetOptionalParam(LocationParam, ""), true, true, false, buffer)
}

func writeSingleFileNameToJson(file fs.DirEntry, buffer *bytes.Buffer) {
	inf, err := file.Info()
	if err != nil {
		return
	}
	buffer.WriteString("{\"size\":")
	buffer.WriteString(strconv.Itoa(int(inf.Size())))
	buffer.WriteString(",\"name\":{\"name\":\"")
	buffer.WriteString(file.Name())
	buffer.WriteString("\", \"encName\":\"")
	buffer.WriteString(encodeValue(file.Name()))
	buffer.WriteString("\"}}")
}

func writeParamAsJsonString(key string, value string, quoted bool, commaAtStart, commaAtEnd bool, buffer *bytes.Buffer) {
	if value != "" {
		if commaAtStart {
			buffer.WriteRune(',')
		}
		buffer.WriteRune('"')
		buffer.WriteString(key)
		buffer.WriteString("\":")
		if quoted {
			buffer.WriteRune('"')
		}
		buffer.WriteString(value)
		if quoted {
			buffer.WriteRune('"')
		}
		if commaAtEnd {
			buffer.WriteRune(',')
		}
	}
}

func writeErrorAsJsonString(error bool, buffer *bytes.Buffer) {
	buffer.WriteRune('"')
	buffer.WriteString(ErrorParam)
	buffer.WriteString("\":")
	if error {
		buffer.WriteString("true")
	} else {
		buffer.WriteString("false")
	}
}
