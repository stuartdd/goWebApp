package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const fallbackModuleName = "goWebApp"
const configFileExtension = ".json"
const AbsolutePathPrefix = "***"
const defaultConfigReloadTime = 3600

var emptyMap = map[string]string{}

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
	Cmd    []string
	Dir    string
	Log    string
	LogOut string
	LogErr string
}

func (p *ExecInfo) GetOutLogFile() string {
	if p.Log == "" || p.LogOut == "" {
		return ""
	}
	return filepath.Join(p.Log, p.LogOut)
}

func (p *ExecInfo) GetErrLogFile() string {
	if p.Log == "" || p.LogErr == "" {
		return ""
	}
	return filepath.Join(p.Log, p.LogErr)
}

func (p *ExecInfo) String() string {
	return fmt.Sprintf("CMD:%s, Dir:%s, LogOut:%s, LogErr:%s", p.Cmd, p.Dir, p.GetOutLogFile(), p.GetErrLogFile())
}

/*
Users Data. Derived from JSON!
*/
type UserData struct {
	Name      string
	Home      string
	Locations map[string]string
	Exec      map[string]*ExecInfo
	Env       map[string]string
}

type ConfigDataInternal struct {
	ReloadConfigSeconds int64
	Port                int
	Users               map[string]UserData
	ContentTypeCharset  string
	LogData             *LogData
	ServerName          string
	PanicResponseCode   int
	FilterFiles         []string
	ServerDataRoot      string
	FaviconIcoPath      string
}

func (p *ConfigDataInternal) String() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type ConfigData struct {
	internal     *ConfigDataInternal
	CurrentPath  string
	ModuleName   string
	ConfigName   string
	Debugging    bool
	Environment  map[string]string
	NextLoadTime int64
	UpSince      time.Time
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
		Debugging:    debugging,
		CurrentPath:  wd,
		ModuleName:   moduleName,
		ConfigName:   configFileName + configFileExtension,
		Environment:  environ,
		NextLoadTime: 0,
	}

	configDataInternal := &ConfigDataInternal{
		ReloadConfigSeconds: defaultConfigReloadTime,
		Port:                8080,
		Users:               make(map[string]UserData),
		LogData:             nil,
		ContentTypeCharset:  "utf-8",
		ServerName:          moduleName,
		FilterFiles:         []string{},
		PanicResponseCode:   500,
		ServerDataRoot:      "~/",
		FaviconIcoPath:      "",
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

	SetContentTypeCharset(configDataInternal.ContentTypeCharset)
	configDataExtternal.NextLoadTime = configDataExtternal.getNextReloadConfigMillis()
	return configDataExtternal.resolveLocations()
}

func (p *ConfigData) checkPathExists(relPath string, userPath string, userEnv map[string]string) (string, error) {
	absPath := p.prefixRelativePaths(relPath, userPath)
	absPathSub := p.SubstituteFromMap([]rune(absPath), userEnv)
	absPathPath, err := filepath.Abs(absPathSub)
	if err != nil {
		return "", fmt.Errorf("path [%s] is invalid", absPathSub)
	}
	stats, err := os.Stat(absPathPath)
	if err != nil {
		return "", fmt.Errorf("path [%s] Not found", absPathPath)
	} else {
		if !stats.IsDir() {
			return "", fmt.Errorf("path[%s] Not a Directory", absPathPath)
		}
	}
	return absPathPath, nil
}

func (p *ConfigData) checkFileExists(relPath string, userPath string, file string, userEnv map[string]string) (string, error) {
	absPath := p.prefixRelativePaths(relPath, userPath)
	absPathSub := p.SubstituteFromMap([]rune(absPath), userEnv)
	absFilePath, err := filepath.Abs(absPathSub)
	if err != nil {
		return "", fmt.Errorf("path [%s] is invalid", absPathSub)
	}
	if file == "" {
		return "", fmt.Errorf("file is undefined. Path [%s]", absFilePath)
	}
	absFilePath = filepath.Join(absFilePath, file)
	stats, err := os.Stat(absFilePath)
	if err != nil {
		return "", fmt.Errorf("file [%s] Not found", absFilePath)
	} else {
		if stats.IsDir() {
			return "", fmt.Errorf("file[%s] is a Directory", absFilePath)
		}
	}
	return absFilePath, nil
}

func (p *ConfigData) prefixRelativePaths(relPath string, userPath string) string {
	if strings.HasPrefix(relPath, AbsolutePathPrefix) {
		return relPath[len(AbsolutePathPrefix):]
	}
	if relPath == "" {
		return filepath.Join(p.GetServerDataRoot(), userPath)
	}
	return filepath.Join(filepath.Join(p.GetServerDataRoot(), userPath), strings.TrimPrefix(relPath, ".."))
}

func (p *ConfigData) resolveLocations() (*ConfigData, *ConfigErrorData) {
	f, e := p.checkPathExists("", "", emptyMap) // Will check GetServerDataRoot
	if e != nil {
		return nil, NewConfigErrorData(fmt.Sprintf("Failed to find UserDataRoot:%s.", p.internal.ServerDataRoot))
	} else {
		p.SetServerDataRoot(f)
	}

	errorList := NewConfigErrorData("")

	f, e = p.checkPathExists(p.GetLogDataPath(), "", emptyMap)
	if e != nil {
		errorList.Add(fmt.Sprintf("Config Error: LogData.Path %s", e))
	} else {
		p.SetLogDataPath(f)
	}

	f, e = p.checkFileExists("", "", p.GetFaviconIcoPath(), emptyMap)
	if e != nil {
		errorList.Add(fmt.Sprintf("Config Error: faviconIcoPath not found %s", e.Error()))
	} else {
		p.SetFaviconIcoPath(f)
	}

	for userName, userData := range p.internal.Users {
		userPathPrefix := userData.Home
		for locName, _ := range userData.Locations {
			path, err := p.GetUserLocPath(userName, locName)
			if err != nil {
				errorList.Add(fmt.Sprintf("Config Error: User [%s] Location [%s] Not found", userName, locName))
			}

			f, e := p.checkPathExists(path, userPathPrefix, emptyMap)
			if e != nil {
				errorList.Add(fmt.Sprintf("Config Error: User [%s] Location [%s] path %s", userName, locName, e.Error()))
			}
			userData.Locations[locName] = f
		}

		userConfigEnv := p.GetUserEnv(userName, false)
		for execName, execData := range userData.Exec {
			if execData.Log != "" {
				path := execData.Log
				f, e := p.checkPathExists(path, userPathPrefix, userConfigEnv)
				if e != nil {
					errorList.Add(fmt.Sprintf("Config Error: User [%s] Exec [%s] log path %s", userName, execName, e.Error()))
				} else {
					execData.Log = f
				}
			}
			if execData.Dir != "" {
				path := execData.Dir
				f, e := p.checkPathExists(path, userPathPrefix, userConfigEnv)
				if e != nil {
					errorList.Add(fmt.Sprintf("Config Error: User [%s] Exec [%s] directory %s", userName, execName, e.Error()))
				} else {
					execData.Dir = f
				}
			}

			for i, v := range execData.Cmd {
				execData.Cmd[i] = p.SubstituteFromMap([]rune(v), userConfigEnv)
			}
			execData.LogOut = p.SubstituteFromMap([]rune(execData.LogOut), userConfigEnv)
			execData.LogErr = p.SubstituteFromMap([]rune(execData.LogErr), userConfigEnv)

		}
	}
	return p, errorList
}

func (p *ConfigData) getNextReloadConfigMillis() int64 {
	return time.Now().UnixMilli() + (p.internal.ReloadConfigSeconds * 1000)
}

func (p *ConfigData) IsTimeToReloadConfig() bool {
	return p.NextLoadTime < time.Now().UnixMilli()
}

func (p *ConfigData) ResetTimeToReloadConfig() {
	p.NextLoadTime = p.getNextReloadConfigMillis()
}

func (p *ConfigData) GetTimeToReloadConfig() float64 {
	t := float64(p.NextLoadTime-time.Now().UnixMilli()) / float64(1000)
	return math.Trunc(t*100) / 100

}

func (p *ConfigData) GetServerName() string {
	return p.internal.ServerName
}

func (p *ConfigData) GetUserData(user string) *UserData {
	ud, ok := p.internal.Users[user]
	if ok {
		return &ud
	}
	return nil
}

func (p *ConfigData) GetUserRoot(user string) string {
	return p.prefixRelativePaths("", p.GetUserData(user).Home)
}

func (p *ConfigData) GetUserNamesList() []string {
	unl := []string{}
	for na, _ := range p.internal.Users {
		unl = append(unl, na)
	}
	return unl
}

func (p *ConfigData) GetFilesFilter() []string {
	return p.internal.FilterFiles
}

func (p *ConfigData) GetServerDataRoot() string {
	return p.internal.ServerDataRoot
}

func (p *ConfigData) SetServerDataRoot(f string) {
	p.internal.ServerDataRoot = f
}

func (p *ConfigData) GetContentTypeCharset() string {
	return p.internal.ContentTypeCharset
}

func (p *ConfigData) GetFaviconIcoPath() string {
	return p.internal.FaviconIcoPath
}

func (p *ConfigData) SetFaviconIcoPath(f string) {
	p.internal.FaviconIcoPath = f
}

func (p *ConfigData) GetLogDataPath() string {
	return p.internal.LogData.Path
}

func (p *ConfigData) SetLogDataPath(f string) {
	p.internal.LogData.Path = f
}

func (p *ConfigData) GetLogData() *LogData {
	return p.internal.LogData
}

func (p *ConfigData) GetUserEnv(user string, includeLocations bool) map[string]string {
	m := make(map[string]string)
	userData, ok := p.internal.Users[user]
	if ok {
		for n, v := range userData.Env {
			m["env."+n] = v
		}
		if includeLocations {
			for n, v := range userData.Locations {
				m["loc."+n] = v
			}
		}
	}
	m["user.name"] = user
	return m
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
	return locData, nil
}

func (p *ConfigData) GetUserLocFilePath(user string, loc string, fileName string) (string, error) {
	userData, err := p.GetUserLocPath(user, loc)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}
	return filepath.Join(userData, fileName), nil
}

func (p *ConfigData) GetUserExecInfo(user, execid string) (*ExecInfo, error) {
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

func (p *ConfigData) String() (string, error) {
	data, err := p.internal.String()
	if err != nil {
		return "", fmt.Errorf("failed to present data as Json:%s. Error:%s", p.ConfigName, err.Error())
	}
	return string(data), nil
}

func (p *ConfigData) SubstituteFromMap(cmd []rune, userEnv map[string]string) string {
	return SubstituteFromMap(cmd, p.Environment, userEnv)
}

func SubstituteFromMap(cmd []rune, env1 map[string]string, env2 map[string]string) string {
	if len(cmd) < 4 {
		return string(cmd)
	}
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
				v, ok := env2[name.String()]
				if ok {
					buff.WriteString(v)
				} else {
					v, ok = env1[name.String()]
					if ok {
						buff.WriteString(v)
					} else {
						buff.WriteRune('%')
						buff.WriteRune('{')
						buff.Write(name.Bytes())
						buff.WriteRune('}')
					}
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

func (p *ConfigErrorData) String() string {
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
