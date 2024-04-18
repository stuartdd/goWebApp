package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const fallbackModuleName = "goWebApp"
const configFileExtension = ".json"
const AbsolutePathPrefix = "***"
const defaultConfigReloadTime = 3600

/*
ConfigData - Read configuration data from the JSON configuration file.
Note any undefined values are defaulted to constants defined below
*/
type TemplateStaticFiles struct {
	Files        []string
	DataFile     string
	Data         map[string]string
	FullFileName string
}

func (t *TemplateStaticFiles) Init() (*TemplateStaticFiles, error) {
	f, err := filepath.Abs(t.DataFile)
	if err != nil {
		return nil, fmt.Errorf("invalid file name:%s. Error:%s", t.DataFile, err.Error())
	}
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read template data file. Error:%s", err.Error())
	}
	m := make(map[string]string)
	err = json.Unmarshal(content, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template json file:%s. Error:%s", f, err.Error())
	}
	t.Data = m
	t.FullFileName = f
	return t, nil
}
func (t *TemplateStaticFiles) String() string {
	return fmt.Sprintf("%s. Templates:%s", t.FullFileName, t.Files)
}

func (t *TemplateStaticFiles) ShouldTemplate(file string) bool {
	for _, v := range t.Files {
		if v == file {
			return true
		}
	}
	return false
}

type LogData struct {
	FileNameMask   string
	Path           string
	MonitorSeconds int
	LogLevel       string
	ConsoleOut     bool
}

func NewLogData() *LogData {
	return &LogData{
		FileNameMask:   "",
		Path:           "",
		MonitorSeconds: -1,
		LogLevel:       "quiet",
		ConsoleOut:     false,
	}
}

/*
Users can have Exex actions. Derived from JSON!
*/
type ExecInfo struct {
	Cmd           []string
	Dir           string
	StdOutType    string
	Log           string
	LogOut        string
	LogErr        string
	NzCodeReturns int
	Detached      bool
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
	Hidden    *bool
	Name      string
	Home      string
	Locations map[string]string
	Exec      map[string]*ExecInfo
	Env       map[string]string
}

func (p *UserData) IsHidden() bool {
	if p.Hidden == nil {
		return false
	}
	return *p.Hidden
}

type ConfigDataInternal struct {
	ReloadConfigSeconds int64
	Port                int
	ThumbnailTrim       []int
	Users               map[string]UserData
	ContentTypeCharset  string
	LogData             *LogData
	ServerName          string
	PanicResponseCode   int
	FilterFiles         []string
	ServerDataRoot      string
	ServerStaticRoot    string
	TemplateStaticFiles *TemplateStaticFiles
	FaviconIcoPath      string
	Env                 map[string]string
}

func (p *ConfigDataInternal) String() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type ConfigData struct {
	internal         *ConfigDataInternal
	CurrentPath      string
	ModuleName       string
	ConfigName       string
	Debugging        bool
	Templating       bool
	Environment      map[string]string
	NextLoadTime     int64
	LocationsCreated []string
	UpSince          time.Time
}

/*
LoadConfigData method loads the config data from a file
*/

func NewConfigData(configFileName string, createDir bool, dontResolve bool) (*ConfigData, *ConfigErrorData) {
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
	fn, _ := filepath.Abs(configFileName + configFileExtension)

	configDataExtternal := &ConfigData{
		Debugging:        debugging,
		CurrentPath:      wd,
		ModuleName:       moduleName,
		ConfigName:       fn,
		Environment:      environ,
		NextLoadTime:     0,
		LocationsCreated: []string{},
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
		TemplateStaticFiles: nil,
		FaviconIcoPath:      "",
		ThumbnailTrim:       []int{0, 0},
		Env:                 map[string]string{},
	}

	/*
		load the config object
	*/
	content, err := os.ReadFile(configDataExtternal.ConfigName)
	if err != nil {
		return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to read config data file:%s. Error:%s", configDataExtternal.ConfigName, err.Error()))
	}

	err = json.Unmarshal(content, &configDataInternal)
	if err != nil {
		return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to understand the config data in the file:%s. Error:%s", configDataExtternal.ConfigName, err.Error()))
	}

	configDataExtternal.internal = configDataInternal

	SetContentTypeCharset(configDataInternal.ContentTypeCharset)
	/*
		Add config data Env to the Environment variables
	*/
	for n, v := range configDataInternal.Env {
		configDataExtternal.Environment[n] = v
	}

	configDataExtternal.NextLoadTime = configDataExtternal.getNextReloadConfigMillis()
	if dontResolve {
		return configDataExtternal, NewConfigErrorData()
	}
	for i := 0; i < len(configDataInternal.FilterFiles); i++ {
		f := strings.ToLower(configDataInternal.FilterFiles[i])
		if !strings.HasPrefix(f, ".") {
			configDataInternal.FilterFiles[i] = fmt.Sprintf(".%s", f)
		}
	}
	return configDataExtternal.resolveLocations(createDir)
}

/*
Construct a path from a relative path and user path.

If the 'relative path' is prefixed with an AbsolutePathPrefix, this is removed and the resultant path returned.

If it is just the 'ueser path', It is joined to the ServerDataRoot.

ServerDataRoot + userHome + path
*/
func (p *ConfigData) resolvePaths(userHome string, location string) string {
	if strings.HasPrefix(location, AbsolutePathPrefix) {
		return location[len(AbsolutePathPrefix):]
	}
	if location == "" {
		return filepath.Join(p.GetServerDataRoot(), userHome)
	}
	return filepath.Join(filepath.Join(p.GetServerDataRoot(), userHome), strings.TrimPrefix(location, ".."))
}

func (p *ConfigData) resolveLocations(createDir bool) (*ConfigData, *ConfigErrorData) {
	userConfigEnv := p.GetUserEnv("")

	f, e := p.checkRootPathExists(p.GetServerDataRoot(), userConfigEnv) // Will check GetServerDataRoot
	if e != nil {
		return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to find UserDataRoot:%s. Cause:%s", f, e.Error()))
	} else {
		p.SetServerDataRoot(f)
	}

	f, e = p.checkRootPathExists(p.GetServerStaticRoot(), userConfigEnv) // Will check ServerStaticRoot
	if e != nil {
		return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to find ServerStaticRoot:%s. Cause:%s", f, e.Error()))
	} else {
		p.SetServerStaticRoot(f)
	}

	errorList := NewConfigErrorData()

	if p.IsTemplating() {
		templ := p.GetTemplateData()
		templ.DataFile = p.SubstituteFromMap([]byte(templ.DataFile), userConfigEnv)
		_, err := templ.Init()
		if err != nil {
			return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to initialiase templating:%s", err.Error()))
		}
		errorList.AddLog(fmt.Sprintf("Config template   :%s", templ))
	}

	f, e = p.checkPathExists("", p.GetLogDataPath(), "", userConfigEnv, false)
	if e != nil {
		errorList.AddError(fmt.Sprintf("Config Error: LogData.Path %s", e))
	} else {
		p.SetLogDataPath(f)
	}

	f, e = p.checkFileExists("", "", p.GetFaviconIcoPath(), userConfigEnv)
	if e != nil {
		errorList.AddError(fmt.Sprintf("Config Error: faviconIcoPath not found %s", e.Error()))
	} else {
		p.SetFaviconIcoPath(f)
	}

	switch len(p.internal.ThumbnailTrim) {
	case 0:
		p.internal.ThumbnailTrim = []int{0, 0}
	case 1:
		p.internal.ThumbnailTrim = append(p.internal.ThumbnailTrim, 0)
	}

	for userId, userData := range p.internal.Users {
		if userData.Home == "" {
			userData.Home = userId
		}
		userHome := userData.Home

		userConfigEnv = p.GetUserEnv(userId)
		for locName := range userData.Locations {
			location, err := p.GetUserLocPath(userId, locName)
			if err != nil {
				errorList.AddError(fmt.Sprintf("Config Error: User [%s] Location [%s] Not found", userId, locName))
			}
			f, e := p.checkPathExists(userHome, location, userId, userConfigEnv, createDir)
			if e != nil {
				errorList.AddError(fmt.Sprintf("Config Error: User [%s] Location [%s] %s", userId, locName, e.Error()))
			}
			userData.Locations[locName] = f
		}

		for execName, execData := range userData.Exec {
			if execData.Log != "" {
				path := execData.Log
				f, e := p.checkPathExists(userHome, path, userId, userConfigEnv, createDir)
				if e != nil {
					errorList.AddError(fmt.Sprintf("Config Error: User [%s] Exec [%s] log %s", userId, execName, e.Error()))
				} else {
					execData.Log = f
				}
			}

			path := execData.Dir
			f, e := p.checkPathExists(userHome, path, userId, userConfigEnv, createDir)
			if e != nil {
				errorList.AddError(fmt.Sprintf("Config Error: User [%s] Exec [%s] directory %s", userId, execName, e.Error()))
			} else {
				execData.Dir = f
			}

			for i, v := range execData.Cmd {
				execData.Cmd[i] = p.SubstituteFromMap([]byte(v), userConfigEnv)
			}
			execData.LogOut = p.SubstituteFromMap([]byte(execData.LogOut), userConfigEnv)
			execData.LogErr = p.SubstituteFromMap([]byte(execData.LogErr), userConfigEnv)
			if execData.StdOutType != "" && !HasContentType(execData.StdOutType) {
				errorList.AddError(fmt.Sprintf("Config Error: Exec [%s] StdOutType [%s] no recognised", execName, execData.StdOutType))
			}
		}
	}
	return p, errorList
}

func (p *ConfigData) checkRootPathExists(rootPath string, userEnv map[string]string) (string, error) {
	if rootPath == "" {
		return "", fmt.Errorf("path is empty")
	}
	absPathSub := p.SubstituteFromMap([]byte(rootPath), userEnv)
	absPathPath, err := filepath.Abs(absPathSub)
	if err != nil {
		return absPathPath, fmt.Errorf("path [%s] is invalid", rootPath)
	}
	stats, err := os.Stat(absPathPath)
	if err != nil {
		return absPathPath, fmt.Errorf("path [%s] Not found", absPathPath)
	} else {
		if !stats.IsDir() {
			return absPathPath, fmt.Errorf("path[%s] Not a Directory", absPathPath)
		}
	}
	return absPathPath, nil
}

func (p *ConfigData) createFullDirectory(path, userId string) error {
	if !strings.HasPrefix(path, p.GetUserRoot(userId)) {
		return fmt.Errorf("[%s] could NOT be created. It is not in %s", path, p.GetUserRoot(userId))
	}
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return fmt.Errorf("[%s] could NOT be created. %s", path, err.Error())
	}
	p.LocationsCreated = append(p.LocationsCreated, fmt.Sprintf("Created: User[%s] Path[%s]", userId, path))
	return nil
}

func (p *ConfigData) checkPathExists(userPath string, relPath string, userId string, userEnv map[string]string, createDir bool) (string, error) {
	absPath := p.resolvePaths(userPath, relPath)
	absPathSub := p.SubstituteFromMap([]byte(absPath), userEnv)
	absPathPath, err := filepath.Abs(absPathSub)
	if err != nil {
		return absPathPath, fmt.Errorf("path [%s] is invalid", absPathSub)
	}
	stats, err := os.Stat(absPathPath)
	if err != nil {
		if createDir {
			err = p.createFullDirectory(absPathPath, userId)
			if err != nil {
				return absPathPath, err
			}
		} else {
			return absPathPath, fmt.Errorf("path [%s] Not found", absPathPath)
		}
	} else {
		if !stats.IsDir() {
			return absPathPath, fmt.Errorf("path[%s] Not a Directory", absPathPath)
		}
	}
	return absPathPath, nil
}

func (p *ConfigData) checkFileExists(userPath string, location string, file string, userEnv map[string]string) (string, error) {
	absPath := p.resolvePaths(userPath, location)
	absPathSub := p.SubstituteFromMap([]byte(absPath), userEnv)
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

func (p *ConfigData) SaveMe() error {
	s, err := p.String()
	if err != nil {
		return err
	}
	err = os.WriteFile(p.ConfigName, []byte(s), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
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

func (p *ConfigData) GetUsers() *map[string]UserData {
	return &p.internal.Users
}

func (p *ConfigData) AddUser(user string) error {
	if p.HasUser(user) {
		return fmt.Errorf("user '%s' already exists", user)
	}
	ud := UserData{
		Hidden:    nil,
		Name:      strings.ToUpper(user[0:1]) + user[1:],
		Home:      user,
		Locations: map[string]string{"home": "", "data": "stateData"},
		Exec:      map[string]*ExecInfo{},
		Env:       map[string]string{},
	}
	p.internal.Users[user] = ud
	return nil
}

func (p *ConfigData) HasUser(user string) bool {
	ulc := strings.ToLower(user)
	for na := range p.internal.Users {
		if strings.ToLower(na) == ulc {
			return true
		}
	}
	return false
}

func (p *ConfigData) GetUserRoot(user string) string {
	return p.resolvePaths(p.GetUserData(user).Home, "")
}

func (p *ConfigData) GetUserNamesList() []string {
	unl := []string{}
	for na, u := range p.internal.Users {
		if !u.IsHidden() {
			unl = append(unl, na)
		}
	}
	return unl
}

func (p *ConfigData) GetFilesFilter() []string {
	return p.internal.FilterFiles
}

func (p *ConfigData) GetThumbnailTrim() []int {
	return p.internal.ThumbnailTrim
}

func (p *ConfigData) GetServerDataRoot() string {
	return p.internal.ServerDataRoot
}

func (p *ConfigData) SetServerDataRoot(f string) {
	p.internal.ServerDataRoot = f
}

func (p *ConfigData) GetServerStaticRoot() string {
	return p.internal.ServerStaticRoot
}

func (p *ConfigData) GetTemplateData() *TemplateStaticFiles {
	return p.internal.TemplateStaticFiles
}

func (p *ConfigData) IsTemplating() bool {
	return p.internal.TemplateStaticFiles != nil
}

func (p *ConfigData) SetServerStaticRoot(path string) {
	p.internal.ServerStaticRoot = path
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

func (p *ConfigData) GetUserEnv(user string) map[string]string {
	m := make(map[string]string)
	if user != "" {
		userData, ok := p.internal.Users[user]
		if ok {
			m["id"] = user
			m["name"] = userData.Name
			m["home"] = userData.Home
			for n, v := range userData.Env {
				m[n] = v
			}
		}
	}
	t := time.Now()
	m["year"] = strconv.Itoa(t.Year())
	m["month"] = padTimeDate(int(t.Month()))
	m["day"] = padTimeDate(t.Day())
	m["hour"] = padTimeDate(t.Hour())
	m["min"] = padTimeDate(int(t.Minute()))
	m["sec"] = padTimeDate(t.Second())
	return m
}

func padTimeDate(v int) string {
	s := strconv.Itoa(v)
	if len(s) == 1 {
		return "0" + s
	}
	return s
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

// func (p *ConfigData) GetUserLocFilePath(user string, loc string, fileName string) (string, error) {
// 	userData, err := p.GetUserLocPath(user, loc)
// 	if err != nil {
// 		return "", fmt.Errorf("user not found")
// 	}
// 	return filepath.Join(userData, fileName), nil
// }

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

func (p *ConfigData) SubstituteFromMap(cmd []byte, userEnv map[string]string) string {
	return SubstituteFromMap(cmd, p.Environment, userEnv)
}

func SubstituteFromMap(cmd []byte, env1 map[string]string, env2 map[string]string) string {
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
				buff.WriteByte(c)
			}
		case 1:
			if c == '%' {
				buff.WriteByte('%')
				havePC = 1
				recoverFrom = i
			} else {
				if c == '{' {
					havePC++
				} else {
					buff.WriteByte('%')
					buff.WriteByte(c)
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
						buff.WriteByte('%')
						buff.WriteByte('{')
						buff.Write(name.Bytes())
						buff.WriteByte('}')
					}
				}
				havePC = 0
				name.Reset()
			} else {
				if c == '%' && havePC == 2 {
					havePC = 1
					recoverFrom = i
					buff.WriteByte('%')
					buff.WriteByte('{')
				} else {
					name.WriteByte(c)
				}
			}
		}
	}
	if name.Len() > 0 {
		for i := recoverFrom; i < len(cmd); i++ {
			buff.WriteByte(cmd[i])
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
	logs   []string
}

func NewConfigErrorData() *ConfigErrorData {
	ed := &ConfigErrorData{
		errors: make([]string, 0),
		logs:   make([]string, 0),
	}
	return ed
}

func (p *ConfigErrorData) ErrorCount() int {
	return len(p.errors)
}

func (p *ConfigErrorData) LogCount() int {
	return len(p.logs)
}

func (p *ConfigErrorData) AddError(s string) *ConfigErrorData {
	p.errors = append(p.errors, s)
	return p
}

func (p *ConfigErrorData) AddLog(s string) *ConfigErrorData {
	p.logs = append(p.logs, s)
	return p
}

func (p *ConfigErrorData) Logs() []string {
	return p.logs
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
