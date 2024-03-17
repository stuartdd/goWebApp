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

/*
Users can have Exex actions. Derived from JSON!
*/
type ExecInfo struct {
	Cmd []string
	Dir string
	Log string
}

func (p *ExecInfo) ToString() string {
	return fmt.Sprintf("CMD:%s, Dir:%s, Log:%s", p.Cmd, p.Dir, p.Log)
}

/*
Users Data. Derived from JSON!
*/
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

/*
Used to manage a set of request parameters.

Parameters are used to locate data in the users data set.

typical values:

	user=fred - The name of the user
	loc=pics - The location id within the user data identifies a location in the file system
	exec=execId - Exec id identifies an exec within the user data
*/
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
	return p.configData.UserExec(p.GetUser(), p.GetCmdId())
}

/*
get File Path (with file name) for 'user', loc' and 'name' parameters
Fails is 'user' or 'loc' are not found.
Does not check the file path exists
*/
func (p *Parameters) UserLocFilePath() (string, error) {
	return p.configData.GetUserLocFilePathParams(p)
}

/*
get Directory Path for 'user' and loc' parameters
Fails is 'user' or 'loc' are not found.
Does not check the directory path exists
*/
func (p *Parameters) UserLocPath() (string, error) {
	return p.configData.GetUserLocPathParams(p)
}

/*
 */
func (p *Parameters) FilterFiles() []string {
	return p.configData.internal.FilterFiles
}

func (p *Parameters) SubstituteFromMap(cmd []rune) string {
	return SubstituteFromMap(cmd, p.configData.Environment)
}

func (p *Parameters) Environment() map[string]string {
	return p.configData.Environment
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

func (p *Parameters) GetCmdId() string {
	return p.GetParam("exec")
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
	Environment map[string]string
}

/*
LoadConfigData method loads the config data from a file
*/

func NewConfigData(configFileName string) (*ConfigData, *ConfigErrorData) {
	environ := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		switch len(pair) {
		case 1:
			environ[pair[0]] = ""
		case 2:
			environ[pair[0]] = pair[1]
		}
	}

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
		Environment: environ,
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

func (p *ConfigData) SubstituteFromMap(cmd []rune) string {
	return SubstituteFromMap(cmd, p.Environment)
}

func SubstituteFromMap(cmd []rune, env map[string]string) string {
	var buff bytes.Buffer
	var name bytes.Buffer
	havePC := 0
	recoverFrom := 0
	for i, c := range cmd {
		switch havePC {
		case 0:
			if c == '%' {
				havePC = 1
				recoverFrom = i
			} else {
				buff.WriteRune(c)
			}
		case 1:
			if c == '%' {
				buff.WriteRune('%')
				havePC = 1
				recoverFrom = i
			} else {
				if c == '{' {
					havePC++
				} else {
					buff.WriteRune('%')
					buff.WriteRune(c)
					havePC = 0
					name.Reset()
				}
			}
		default:
			if c == '}' {
				v, ok := env[name.String()]
				if ok {
					buff.WriteString(v)
				} else {
					buff.WriteRune('%')
					buff.WriteRune('{')
					buff.Write(name.Bytes())
					buff.WriteRune('}')
				}
				havePC = 0
				name.Reset()
			} else {
				if c == '%' && havePC == 2 {
					havePC = 1
					recoverFrom = i
					buff.WriteRune('%')
					buff.WriteRune('{')
				} else {
					name.WriteRune(c)
				}
			}
		}
	}
	if name.Len() > 0 {
		for i := recoverFrom; i < len(cmd); i++ {
			buff.WriteRune(cmd[i])
		}
	}
	return buff.String()
}

/*
A list of issues found with the configuration data

If returned with a nil ConfigData then it is fatal.
If returned with a ConfigData then it is warnings.
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
