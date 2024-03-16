package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const fallbackModuleName = "goWebApp"
const configFileExtension = ".json"
const absFilePrefix = "***/"

/*
ConfigData - Read configuration data from the JSON configuration file.
Note any undefined values are defaulted to constants defined below
*/

type ConfigErrorData struct {
	errors []string
}

func NewConfigErrorData(m string) *ConfigErrorData {
	ed := &ConfigErrorData{
		errors: make([]string, 0),
	}
	if m != "" {
		ed.Add(m)
	}
	return ed
}

func (p *ConfigErrorData) Len() int {
	return len(p.errors)
}

func (p *ConfigErrorData) Add(s string) {
	p.errors = append(p.errors, s)
}

func (p *ConfigErrorData) ToString() string {
	var buffer bytes.Buffer
	for _, err := range p.errors {
		buffer.WriteString(err)
		buffer.WriteString("\n")
	}
	return buffer.String()
}

type LogData struct {
	FileNameMask   string
	Path           string
	MonitorSeconds int
	LogLevel       string
}

func NewLogData() *LogData {
	return &LogData{
		FileNameMask:   "",
		Path:           "",
		MonitorSeconds: -1,
		LogLevel:       "quiet",
	}
}

type ExecInfo struct {
	Cmd []string
	Dir string
	Log string
}

func (p *ExecInfo) ToString() string {
	return fmt.Sprintf("CMD:%s, Dir:%s, Log:%s", p.Cmd, p.Dir, p.Log)
}

type UserData struct {
	Name      string
	Locations map[string]string
	Exec      map[string]*ExecInfo
}

func NewUserData(name string, locations map[string]string) UserData {
	return UserData{
		Name:      name,
		Locations: locations}
}

type Parameters struct {
	configData *ConfigData
	params     map[string]string
}

func NewParameters(p map[string]string, configData *ConfigData) *Parameters {
	return &Parameters{
		params:     p,
		configData: configData,
	}
}

func (p *Parameters) GetParam(key string) string {
	v, ok := p.params[key]
	if ok {
		return v
	}
	panic(fmt.Errorf("url parameter '%s' is missing", key))
}

func (p *Parameters) UserExec() (exi *ExecInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			exi = nil
		}
	}()
	return p.configData.UserExec(p.GetParam("user"), p.GetParam("sync"))
}

func (p *Parameters) UserLocFilePath() (string, error) {
	return p.configData.GetUserLocFilePathParams(p)
}

func (p *Parameters) UserLocPath() (string, error) {
	return p.configData.GetUserLocPathParams(p)
}

func (p *Parameters) FilterFiles() []string {
	return p.configData.internal.FilterFiles
}

func (p *Parameters) GetUser() string {
	return p.GetParam("user")
}

func (p *Parameters) GetLocation() string {
	return p.GetParam("loc")
}

func (p *Parameters) GetName() string {
	return p.GetParam("name")
}

type ConfigDataInternal struct {
	Port               int
	Users              map[string]UserData
	ContentTypeCharset string
	LogData            *LogData
	ServerName         string
	PanicResponseCode  int
	FilterFiles        []string
	UserDataRoot       string
	FaviconIcoPath     string
}

func (p *ConfigDataInternal) toString() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type ConfigData struct {
	internal    *ConfigDataInternal
	CurrentPath string
	ModuleName  string
	ConfigName  string
	Debugging   bool
}

/*
LoadConfigData method loads the config data from a file
*/

func NewConfigData(configFileName string) (*ConfigData, *ConfigErrorData) {

	moduleName, debugging := getApplicationModuleName()
	if configFileName == "" {
		configFileName = moduleName
	} else {
		if strings.HasSuffix(strings.ToLower(configFileName), configFileExtension) {
			configFileName = configFileName[0 : len(configFileName)-5]
		}
	}

	wd, _ := os.Getwd()

	configDataExtternal := &ConfigData{
		Debugging:   debugging,
		CurrentPath: wd,
		ModuleName:  moduleName,
		ConfigName:  configFileName + configFileExtension,
	}

	configDataInternal := &ConfigDataInternal{
		Port:               8080,
		Users:              make(map[string]UserData),
		LogData:            nil,
		ContentTypeCharset: "utf-8",
		ServerName:         moduleName,
		FilterFiles:        []string{},
		PanicResponseCode:  500,
		UserDataRoot:       "~/",
		FaviconIcoPath:     "",
	}

	/*
		load the config object
	*/
	content, err := os.ReadFile(configDataExtternal.ConfigName)
	if err != nil {
		return nil, NewConfigErrorData(fmt.Sprintf("Failed to read config data file:%s. Error:%s", configDataExtternal.ConfigName, err.Error()))
	}

	err = json.Unmarshal(content, &configDataInternal)
	if err != nil {
		return nil, NewConfigErrorData(fmt.Sprintf("Failed to understand the config data in the file:%s. Error:%s", configDataExtternal.ConfigName, err.Error()))
	}

	configDataExtternal.internal = configDataInternal

	for i := 0; i < len(configDataInternal.FilterFiles); i++ {
		configDataInternal.FilterFiles[i] = fmt.Sprintf(".%s", strings.ToLower(configDataInternal.FilterFiles[i]))
	}

	e := configDataExtternal.checkPathExists(configDataInternal.UserDataRoot) // test UserDataRoot exists!
	if e != "" {
		return nil, NewConfigErrorData(fmt.Sprintf("Faild to find UserDataRoot:%s.", configDataInternal.UserDataRoot))
	}

	SetContentTypeCharset(configDataInternal.ContentTypeCharset)
	return configDataExtternal.resolveLocations()
}

func (p *ConfigData) checkPathExists(path string) string {
	stats, err := os.Stat(path)
	if err != nil {
		return fmt.Sprintf("Path [%s] Not found", path)
	} else {
		if !stats.IsDir() {
			return fmt.Sprintf("Path[%s] Not a Directory", path)
		}
	}
	return ""
}

func (p *ConfigData) checkFileExists(file string) string {
	stats, err := os.Stat(file)
	if err != nil {
		return fmt.Sprintf("File [%s] Not found", file)
	} else {
		if stats.IsDir() {
			return fmt.Sprintf("File[%s] is a Directory", file)
		}
	}
	return ""
}

func (p *ConfigData) toFullFilePath(relative string, name string) string {
	var root string
	if strings.HasPrefix(relative, absFilePrefix) {
		root = relative[3:]
	} else {
		if relative == "" {
			root = p.internal.UserDataRoot
		} else {
			root = fmt.Sprintf("%s%c%s", p.internal.UserDataRoot, os.PathSeparator, relative)
		}
	}
	if name == "" {
		fr, err := filepath.Abs(root)
		if err != nil {
			return root
		}
		return fr
	}
	root = fmt.Sprintf("%s%c%s", root, os.PathSeparator, name)
	fr, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	return fr
}

func (p *ConfigData) resolveLocations() (*ConfigData, *ConfigErrorData) {
	errorList := NewConfigErrorData("")

	e := p.checkFileExists(p.GetFaviconIcoPath())
	if e != "" {
		errorList.Add(fmt.Sprintf("Config Error: faviconIcoPath not found %s", e))
	}

	e = p.checkPathExists(p.GetLogDataPath())
	if e != "" {
		errorList.Add(fmt.Sprintf("Config Error: LogData.Path %s", e))
	}

	for userName, userData := range p.internal.Users {
		for locName, _ := range userData.Locations {
			ulp, err := p.GetUserLocPath(userName, locName)
			if err != nil {
				errorList.Add(fmt.Sprintf("Config Error: User [%s] Location [%s] Not found", userName, locName))
			}
			e := p.checkPathExists(ulp)
			if e != "" {
				errorList.Add(fmt.Sprintf("Config Error: User [%s] Location [%s] path %s", userName, locName, e))
			}
		}
		for execName, execData := range userData.Exec {
			if execData.Log != "" {
				s := strings.SplitN(execData.Log, "/", 2)
				if len(s) == 2 {
					loc, ok := userData.Locations[s[0]]
					if !ok {
						errorList.Add(fmt.Sprintf("Config Error: User [%s] Exec [%s] Invalid Log[%s]. Loc [%s] Not found", userName, execName, execData.Log, s[0]))
					} else {
						execData.Log = fmt.Sprintf("%s%c%s%c%s", p.internal.UserDataRoot, os.PathSeparator, loc, os.PathSeparator, s[1])
					}
				} else {
					errorList.Add(fmt.Sprintf("Config Error: User [%s] Exec [%s] Invalid Log[%s]. Use loc/filename", userName, execName, execData.Log))
				}
			}
			e := p.checkPathExists(p.toFullFilePath(execData.Dir, ""))
			if e != "" {
				errorList.Add(fmt.Sprintf("Config Error: User [%s] Exec [%s] Dir [%s] %s", userName, execName, execData.Dir, e))
			}
			execData.Dir = strings.TrimPrefix(execData.Dir, "***")
		}
	}
	return p, errorList
}

func (p *ConfigData) GetServerName() string {
	return p.internal.ServerName
}

func (p *ConfigData) GetUserDataRoot() string {
	return p.toFullFilePath(p.internal.UserDataRoot, "")
}

func (p *ConfigData) GetContentTypeCharset() string {
	return p.internal.ContentTypeCharset
}
func (p *ConfigData) GetFaviconIcoPath() string {
	return p.toFullFilePath("", p.internal.FaviconIcoPath)
}

func (p *ConfigData) GetLogDataPath() string {
	return p.toFullFilePath("", p.internal.LogData.Path)
}

func (p *ConfigData) GetLogData() *LogData {
	return p.internal.LogData
}

func (p *ConfigData) GetPortString() string {
	return fmt.Sprintf(":%d", p.internal.Port)
}

func (p *ConfigData) GetUserLocPath(user string, loc string) (string, error) {
	userData, ok := p.internal.Users[user]
	if !ok {
		return "", fmt.Errorf("user not found")
	}
	locData, ok := userData.Locations[loc]
	if !ok {
		return "", fmt.Errorf("user location not found")
	}
	if strings.HasPrefix(locData, "***") {
		return locData[3:], nil
	}
	return p.toFullFilePath(locData, ""), nil
}

func (p *ConfigData) GetUserLocPathParams(parameters *Parameters) (res string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			res = ""
		}
	}()
	path, err := p.GetUserLocPath(parameters.GetUser(), parameters.GetLocation())
	if err != nil {
		return "", err
	}
	return path, nil
}

func (p *ConfigData) GetUserLocFilePathParams(parameters *Parameters) (res string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			res = ""
		}
	}()
	path, e := p.GetUserLocPath(parameters.GetUser(), parameters.GetLocation())
	if e != nil {
		return "", e
	}
	fileName := parameters.GetName()
	if fileName == "" {
		return "", fmt.Errorf("filename not defined")
	}
	return fmt.Sprintf("%s%c%s", path, os.PathSeparator, fileName), nil
}

func (p *ConfigData) UserExec(user, execid string) (*ExecInfo, error) {
	userData, ok := p.internal.Users[user]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	exec, ok := userData.Exec[execid]
	if !ok {
		return nil, fmt.Errorf("exec id not found")
	}
	return exec, nil
}

func (p *ConfigData) ToString() (string, error) {
	data, err := p.internal.toString()
	if err != nil {
		return "", fmt.Errorf("failed to present data as Json:%s. Error:%s", p.ConfigName, err.Error())
	}
	return string(data), nil
}

/*
GetOS returns the name of the operating system. Used to look up os
specific paths in config data staticPaths and templatePaths.
Use this in error messages to indicate a path is not found for the OS
*/
func GetOS() string {
	return runtime.GOOS
}

/*
GetApplicationModuleName returns the name of the application. Testing and debugging changes this name so the code
removes debug, test and .exe from the executable name.
*/
func getApplicationModuleName() (string, bool) {
	exec, err := os.Executable()
	if err != nil {
		return fallbackModuleName, false
	}
	parts := strings.Split(exec, string(os.PathSeparator))
	exec = parts[len(parts)-1]
	if strings.HasPrefix(exec, "__debug_") {
		return fallbackModuleName, true
	}
	if strings.HasSuffix(strings.ToLower(exec), ".exe") {
		return exec[0 : len(exec)-4], false
	}
	if strings.HasSuffix(strings.ToLower(exec), ".test") {
		return exec[0 : len(exec)-5], false
	}
	return exec, false
}
