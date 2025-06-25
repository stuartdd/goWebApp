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
const thumbnailTrimPrefix = 20
const thumbnailTrimSuffix = 4

type PanicMessage struct {
	Status int
	Reason string
	Logged string
}

func NewPanicMessageFromString(message string) *PanicMessage {
	s := strings.SplitN(message, ":", 3)
	switch len(s) {
	case 1:
		return &PanicMessage{Reason: message, Status: 404, Logged: message}
	case 2:
		i, err := strconv.Atoi(s[1])
		if err != nil {
			return &PanicMessage{Reason: s[0], Status: 500, Logged: fmt.Sprintf("%s (Unable to parse status code)", message)}
		}
		return &PanicMessage{Reason: s[0], Status: i, Logged: message}
	}
	i, err := strconv.Atoi(s[1])
	if err != nil {
		return &PanicMessage{Reason: s[0], Status: 500, Logged: fmt.Sprintf("%s (Unable to parse status code '%s')", s[2], s[1])}
	}
	return &PanicMessage{Reason: s[0], Status: i, Logged: s[2]}
}

func NewPanicMessage(reason string, status int, logged string) *PanicMessage {
	r := strings.ReplaceAll(reason, ":", ";")
	return &PanicMessage{Reason: r, Status: status, Logged: logged}
}

func (p *PanicMessage) String() string {
	if strings.Contains(p.Logged, "stat") {
		return fmt.Sprintf("%s:%d", p.Reason, p.Status)
	}
	return fmt.Sprintf("%s:%d:%s", p.Reason, p.Status, p.Logged)
}

func (p *PanicMessage) Error() string {
	return p.String()
}

/*
Template data read from configuration data JSONn file.
*/
type TemplateStaticFiles struct {
	Files    []string
	DataFile string
	data     map[string]string
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
	m := make(map[string]interface{})
	err = json.Unmarshal(content, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template json file:%s. Error:%s", f, err.Error())
	}
	t.data = FlattenMap(m, "")

	return t, nil
}

func (t *TemplateStaticFiles) String() string {
	f, _ := filepath.Abs(t.DataFile)
	return fmt.Sprintf("%s. Templates:%s", f, t.Files)
}

func (t *TemplateStaticFiles) Data(plus map[string]string) map[string]string {
	m := map[string]string{}
	for n, v := range t.data {
		m[n] = v
	}
	for n, v := range plus {
		m[n] = v
	}
	return m
}

func (t *TemplateStaticFiles) ShouldTemplate(file string) bool {
	for _, v := range t.Files {
		if v == file {
			return true
		}
	}
	return false
}

type ExecManager struct {
	Path        string
	File        string
	TestCommand string
}

func (ex *ExecManager) IsSet() bool {
	return ex.Path != ""
}

type LogData struct {
	FileNameMask   string
	Path           string
	MonitorSeconds int
	ConsoleOut     bool
}

type StaticData struct {
	Path string
	Home string
}

func (p *StaticData) HasStaticData() bool {
	return p.Path != ""
}

func (p *StaticData) GetHome() string {
	return filepath.Join(p.Path, p.Home)
}

func (p *StaticData) CheckFileExists(file string) bool {
	file = strings.TrimPrefix(file, "/")
	fileParts := strings.SplitN(file, "&", 2)
	if len(fileParts) < 1 {
		return false
	}

	absFilePath := filepath.Join(p.Path, strings.ReplaceAll(fileParts[0], "/", string(os.PathSeparator)))
	stats, err := os.Stat(absFilePath)
	if err != nil {
		return false
	} else {
		if stats.IsDir() {
			return false
		}
	}
	return true

}

func (p *StaticData) CheckHomeExists() error {
	if p.Home == "" {
		return fmt.Errorf("static Home is undefined in StaticData")
	}
	ok := p.CheckFileExists(p.Home)
	if ok {
		return nil
	} else {
		return fmt.Errorf("static Home file[%s] does not exist", filepath.Join(p.Path, p.Home))
	}
}

func NewLogData() *LogData {
	return &LogData{
		FileNameMask:   "",
		Path:           "",
		MonitorSeconds: -1,
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
	LogDir        string
	LogOut        string
	LogErr        string
	NzCodeReturns int
	Detached      bool
	CanStop       bool
}

func (p *ExecInfo) GetOutLogFile() string {
	if p.LogDir == "" || p.LogOut == "" {
		return ""
	}
	return filepath.Join(p.LogDir, p.LogOut)
}

func (p *ExecInfo) GetErrLogFile() string {
	if p.LogDir == "" || p.LogErr == "" {
		return ""
	}
	return filepath.Join(p.LogDir, p.LogErr)
}

func (p *ExecInfo) String() string {
	return fmt.Sprintf("CMD:%s, Dir:%s, LogOut:%s, LogErr:%s", p.Cmd, p.Dir, p.GetOutLogFile(), p.GetErrLogFile())
}

/*
Users Data. Derived from JSON!
*/
type UserData struct {
	Hidden    *bool
	Name      string ""
	Home      string ""
	Locations map[string]string
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
	StaticData          *StaticData
	TemplateStaticFiles *TemplateStaticFiles
	FaviconIcoPath      string
	Env                 map[string]string
	Exec                map[string]*ExecInfo
	ExecManager         *ExecManager
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
	IsVerbose        bool
}

/*
LoadConfigData method loads the config data from a file
*/

func NewConfigData(configFileName string, createDir bool, dontResolve bool, verbose bool) (*ConfigData, *ConfigErrorData) {
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

	if verbose {
		fmt.Printf("Config file name is %s\n", fn)
	}

	configDataExternal := &ConfigData{
		Debugging:        debugging,
		CurrentPath:      wd,
		ModuleName:       moduleName,
		ConfigName:       fn,
		Environment:      environ,
		NextLoadTime:     0,
		LocationsCreated: []string{},
		IsVerbose:        verbose,
	}

	configDataInternal := &ConfigDataInternal{
		ReloadConfigSeconds: defaultConfigReloadTime,
		Port:                8080,
		Users:               make(map[string]UserData),
		LogData:             &LogData{},
		ContentTypeCharset:  "utf-8",
		ServerName:          moduleName,
		FilterFiles:         []string{},
		PanicResponseCode:   500,
		ServerDataRoot:      "",
		StaticData:          &StaticData{Path: "", Home: ""},
		TemplateStaticFiles: nil,
		FaviconIcoPath:      "",
		ThumbnailTrim:       []int{thumbnailTrimPrefix, thumbnailTrimSuffix},
		Env:                 map[string]string{},
		Exec:                map[string]*ExecInfo{},
		ExecManager:         &ExecManager{Path: "", File: "", TestCommand: ""},
	}

	/*
		load the config object
	*/
	content, err := os.ReadFile(configDataExternal.ConfigName)
	if err != nil {
		return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to read config data file:%s. Error:%s", configDataExternal.ConfigName, err.Error()))
	}

	err = json.Unmarshal(content, &configDataInternal)
	if err != nil {
		return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to understand the config data in the file:%s. Error:%s", configDataExternal.ConfigName, err.Error()))
	}

	configDataExternal.internal = configDataInternal

	if len(configDataExternal.internal.ThumbnailTrim) < 2 {
		return nil, NewConfigErrorData().AddError("Config data entry ThumbnailTrim data has less than 2 entries")
	}

	SetContentTypeCharset(configDataInternal.ContentTypeCharset)
	/*
		Add config data Env to the Environment variables
	*/
	for n, v := range configDataInternal.Env {
		configDataExternal.Environment[n] = v
	}

	configDataExternal.NextLoadTime = configDataExternal.getNextReloadConfigMillis()
	if dontResolve {
		return configDataExternal, NewConfigErrorData()
	}
	for i := 0; i < len(configDataInternal.FilterFiles); i++ {
		f := strings.ToLower(configDataInternal.FilterFiles[i])
		if !strings.HasPrefix(f, ".") {
			configDataInternal.FilterFiles[i] = fmt.Sprintf(".%s", f)
		}
	}

	if verbose {
		ret, cfgErr := configDataExternal.resolveLocations(createDir)
		if ret != nil {
			fmt.Println("Final Config Data -------")
			s, err := ret.String()
			if err != nil {
				fmt.Printf("Config data String() returned this error: %s", err.Error())
			} else {
				fmt.Println(s)
			}
			fmt.Println("Final Config Data -------")
		}
		return ret, cfgErr
	} else {
		return configDataExternal.resolveLocations(createDir)
	}
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
	if strings.HasPrefix(userHome, AbsolutePathPrefix) {
		userHome = userHome[len(AbsolutePathPrefix):]
		return filepath.Join(userHome, location)
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
		return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to find ServerDataRoot:%s. Cause:%s", f, e.Error()))
	} else {
		p.SetServerDataRoot(f)
	}

	if p.HasStaticData() {
		f, e = p.checkRootPathExists(p.GetServerStaticRoot(), userConfigEnv) // Will check ServerStaticRoot
		if e != nil {
			return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to find StaticData.Path in config file:%s. Cause:%s", p.ConfigName, e.Error()))
		} else {
			p.SetServerStaticRoot(f)
		}

		if p.internal.StaticData.Home == "" {
			return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to find StaticData.Home in config file:%s", p.ConfigName))
		}

		e = p.GetStaticData().CheckHomeExists()
		if e != nil {
			return nil, NewConfigErrorData().AddError(fmt.Sprintf("Failed to find StaticData.Home in config file:%s. Cause:%s", p.ConfigName, e.Error()))
		}
	}

	errorList := NewConfigErrorData()
	defer func() {
		if r := recover(); r != nil {
			pm := r.(*PanicMessage)
			if pm == nil {
				err := r.(error)
				pm = NewPanicMessageFromString(err.Error())
			}
			errorList.AddError(fmt.Sprintf("Config Error: %s", pm))
		}
	}()

	if p.GetExecManager().Path != "" {
		f, e = p.checkRootPathExists(p.GetExecManager().Path, userConfigEnv) // Will check ServerStaticRoot
		if e != nil {
			errorList.AddError(fmt.Sprintf("Config Error: ExecManager.Path %s", e))
		} else {
			p.GetExecManager().Path = f
		}
	}

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

	for execName, execData := range p.internal.Exec {
		if execData.Detached {
			if execData.LogDir != "" {
				errorList.AddError(fmt.Sprintf("Config Error: Exec [%s] is detached. Cannot have LogDir='%s'", execName, execData.LogDir))
			}
			if execData.LogOut != "" {
				errorList.AddError(fmt.Sprintf("Config Error: Exec [%s] is detached. Cannot have LogOut='%s'", execName, execData.LogOut))
			}
			if execData.LogErr != "" {
				errorList.AddError(fmt.Sprintf("Config Error: Exec [%s] is detached. Cannot have LogErr='%s'", execName, execData.LogErr))
			}
		}
		if execData.LogDir != "" {
			f, e := p.checkPathExists("", execData.LogDir, "", userConfigEnv, createDir)
			if e != nil {
				errorList.AddError(fmt.Sprintf("Config Error: Exec [%s] log %s", execName, e.Error()))
			} else {
				execData.LogDir = f
			}
		}

		if execData.Dir == "" && p.internal.ExecManager != nil && p.internal.ExecManager.Path != "" {
			execData.Dir = p.internal.ExecManager.Path
		} else {
			if execData.Dir == "" {
				execData.Dir = "exec"
			}
			f, e := p.checkPathExists("", execData.Dir, "", userConfigEnv, createDir)
			if e != nil {
				errorList.AddError(fmt.Sprintf("Config Error: Exec [%s] directory %s", execName, e.Error()))
			} else {
				execData.Dir = f
			}
		}

		for i, v := range execData.Cmd {
			execData.Cmd[i] = p.SubstituteFromMap([]byte(v), userConfigEnv)
		}
		execData.LogOut = p.SubstituteFromMap([]byte(execData.LogOut), userConfigEnv)
		execData.LogErr = p.SubstituteFromMap([]byte(execData.LogErr), userConfigEnv)
		if execData.StdOutType != "" && !HasContentType(execData.StdOutType) {
			errorList.AddError(fmt.Sprintf("Config Error: Exec [%s] StdOutType [%s] not recognised", execName, execData.StdOutType))
		}
	}

	for userId, userData := range p.internal.Users {
		if userData.Home == "" {
			userData.Home = userId
		}
		userHome := userData.Home

		userConfigEnv = p.GetUserEnv(userId)
		for locName := range userData.Locations {
			location := p.GetUserLocPath(userId, locName)
			f, e := p.checkPathExists(userHome, location, userId, userConfigEnv, createDir)
			if e != nil {
				errorList.AddError(fmt.Sprintf("Config Error: User [%s] Location [%s] %s", userId, locName, e.Error()))
			}
			userData.Locations[locName] = f
		}

	}
	return p, errorList
}

func (p *ConfigData) checkRootPathExists(rootPath string, userEnv map[string]string) (string, error) {
	if rootPath == "" {
		return "", fmt.Errorf("path is empty")
	}
	p.Verbose(fmt.Sprintf("checkRootPathExists: %s", rootPath))
	absPathSub := p.SubstituteFromMap([]byte(rootPath), userEnv)
	p.Verbose(fmt.Sprintf("checkRootPathExists:SubstituteFromMap: %s", absPathSub))
	absPathPath, err := filepath.Abs(absPathSub)
	if err != nil {
		return absPathPath, fmt.Errorf("path [%s] is invalid", absPathPath)
	}
	p.Verbose(fmt.Sprintf("checkRootPathExists:Abs: %s", absPathPath))
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
			p.Verbose(fmt.Sprintf("createFullDirectory:%s", absPathPath))
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

func (l *ConfigData) Verbose(s string) {
	if l.IsVerbose {
		fmt.Println(s)
	}
}

func (p *ConfigData) HasStaticData() bool {
	return p.internal.StaticData.HasStaticData()
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

func (p *ConfigData) GetExecManager() *ExecManager {
	return p.internal.ExecManager
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
		Home:      "",
		Locations: map[string]string{"data": "stateData"},
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

// PANIC
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

func (p *ConfigData) ConvertToThumbnail(name string) (resp string) {
	defer func() {
		if r := recover(); r != nil {
			resp = name
		}
	}()
	return name[p.internal.ThumbnailTrim[0] : len(name)-p.internal.ThumbnailTrim[1]]
}

func (p *ConfigData) GetServerDataRoot() string {
	return p.internal.ServerDataRoot
}

func (p *ConfigData) SetServerDataRoot(f string) {
	p.internal.ServerDataRoot = f
}

func (p *ConfigData) GetServerStaticRoot() string {
	return p.internal.StaticData.Path
}

func (p *ConfigData) GetStaticData() *StaticData {
	return p.internal.StaticData
}

func (p *ConfigData) GetTemplateData() *TemplateStaticFiles {
	return p.internal.TemplateStaticFiles
}

func (p *ConfigData) IsTemplating() bool {
	return p.internal.TemplateStaticFiles != nil
}

func (p *ConfigData) SetServerStaticRoot(path string) {
	p.internal.StaticData.Path = path
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
	m["doy"] = padTimeDate(t.YearDay())
	m["ms"] = padTimeDate(int(t.UnixMilli()))
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

// PANIC
func (p *ConfigData) GetUserLocPath(user string, loc string) string {
	userData, ok := p.internal.Users[user]
	if !ok {
		panic(&PanicMessage{Reason: "user not found", Status: 404, Logged: fmt.Sprintf("User=%s", user)})
	}
	locData, ok := userData.Locations[loc]
	if !ok {
		panic(&PanicMessage{Reason: "location not found", Status: 404, Logged: fmt.Sprintf("User=%s Location=%s", user, loc)})
	}
	return locData
}

// PANIC
func (p *ConfigData) GetExecInfo(execid string) *ExecInfo {
	exec, ok := p.internal.Exec[execid]
	if !ok {
		panic(&PanicMessage{Reason: "exec ID not found", Status: 404, Logged: fmt.Sprintf("exec-id=%s", execid)})
	}
	return exec
}

func (p *ConfigData) String() (string, error) {
	data, err := p.internal.String()
	if err != nil {
		return "", fmt.Errorf("failed to present data as Json:%s. Error:%s", p.ConfigName, err.Error())
	}
	return string(data), nil
}

func FlattenMap(m map[string]interface{}, prefix string) map[string]string {
	out := make(map[string]string)
	flattenRec(out, []string{}, prefix, m)
	return out
}

func flattenRec(m map[string]string, path []string, n string, v interface{}) {
	switch x := v.(type) {
	case map[string]interface{}:
		for nn, vv := range x {
			flattenRec(m, appendStrF(path, n), nn, vv)
		}
	case []interface{}:
		for nn, vv := range x {
			flattenRec(m, appendStrF(path, n), strconv.Itoa(nn), vv)
		}
	default:
		m[strings.Join(appendStrF(path, n), ".")] = fmt.Sprintf("%v", x)
	}
}

func appendStrF(path []string, p string) []string {
	if p == "" {
		return path
	}
	pp := make([]string, len(path)+1)
	copy(pp, path)
	pp[len(path)] = p
	return pp
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
