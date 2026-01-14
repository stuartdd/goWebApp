package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ConfigFileExtension = ".json"
const AbsolutePathPrefix = "***"
const defaultReloadConfigSeconds = 3600
const defaultPort = 8080
const thumbnailTrimPrefix = 20
const thumbnailTrimSuffix = 4
const panicMessageStatus = "status:"
const panicMessageLog = "log:"
const StaticPathName = "static"
const ImagesPathName = "images"

type UserProperties struct {
	mu     sync.Mutex
	path   string
	values map[string]string
}

func NewUserProperties(path string) (*UserProperties, error) {
	if path == "" {
		return &UserProperties{values: make(map[string]string), path: path}, nil
	}
	_, err := os.Stat(path)
	if err != nil {
		err = os.WriteFile(path, []byte("{}"), 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to create user properties file:%s. Error:%s", path, err.Error())
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read user properties file:%s. Error:%s", path, err.Error())
	}
	userProps := make(map[string]string)
	err = json.Unmarshal(content, &userProps)
	if err != nil {
		return nil, fmt.Errorf("failed to understand user properties file:%s. Error:%s", path, err.Error())
	}
	return &UserProperties{values: userProps, path: path}, nil
}

func (up *UserProperties) Details() string {
	if len(up.values) == 0 {
		return fmt.Sprintf("%s (No properties stored)", up.path)
	}
	return fmt.Sprintf("%s (File Read %d properties)", up.path, len(up.values))
}

func (up *UserProperties) writeToFile() {
	body, err := json.Marshal(up.values)
	if err != nil {
		panic(fmt.Sprintf("Failed to serialise user properties: %s", err.Error()))
	}
	err = os.WriteFile(up.path, body, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to save user properties: %s", err.Error()))
	}
}

func (up *UserProperties) MapDataForUser(user string, userData *UserData) map[string]interface{} {
	out := make(map[string]interface{})
	for n, v := range up.values {
		if strings.HasPrefix(n, user+".") {
			out[n[len(user)+1:]] = v
		}
	}
	out["id"] = user
	out["info"] = userData.CanSeeInfo()
	out["name"] = userData.Name
	return out
}

// Cannot run concurrently so use mutex lock
func (up *UserProperties) Update(user, key, value string) string {
	if up.path == "" {
		return value
	}

	up.mu.Lock()
	defer up.mu.Unlock()

	v, found := up.values[key]
	if found {
		if value == "" {
			return v
		}
		if v == value {
			return v
		}
	}
	up.values[key] = value
	up.writeToFile()
	return value
}

type ConfigError struct {
	status  int
	message string
	log     string
}

func (pm *ConfigError) Error() string {
	return fmt.Sprintf("Config Error: Status:%d. %s", pm.status, pm.message)
}

func (ee *ConfigError) Map() map[string]interface{} {
	m := make(map[string]interface{})
	m["error"] = true
	m["status"] = ee.Status()
	m["msg"] = http.StatusText(ee.status)
	m["cause"] = ee.String()
	return m
}

func (ee *ConfigError) Status() int {
	return ee.status
}

func (ee *ConfigError) String() string {
	return ee.message
}

func (pm *ConfigError) LogError() string {
	if pm.log == "" {
		return pm.Error()
	}
	return fmt.Sprintf("%s Log:%s", pm.Error(), pm.log)
}

func NewConfigError(message string, status int, logged string) *ConfigError {
	return &ConfigError{message: strings.TrimSpace(message), status: status, log: strings.TrimSpace(logged)}
}

func NewConfigErrorFromString(message string, fallback int) *ConfigError {
	mLc := strings.ToLower(message)
	sp := strings.Index(mLc, panicMessageStatus)
	lp := strings.Index(mLc, panicMessageLog)

	// Every thing after log: is logged with the message
	lm := ""
	if lp >= 0 {
		lm = message[lp+4:]
	}

	if sp < 0 {
		// No status so message is everything up to log:. Status if fallback
		if lp >= 0 {
			message = message[0:lp]
		}
		return NewConfigError(message, fallback, lm)
	}

	// read the status. Return status and the char at the end of the status number.
	// if fails then status is fallback and end
	status, end := parseInt(message, sp+len(panicMessageStatus)-1, fallback)
	m := ""
	if end > sp {
		m = message[0:sp] + message[end+1:]
	} else {
		m = message
	}
	lp = strings.Index(strings.ToLower(m), "log:")
	if lp >= 0 {
		m = m[0:lp]
	}

	return NewConfigError(m, status, lm)
}

func parseInt(s string, pos int, fallback int) (int, int) {
	b := []byte(s)
	n := -1
	p := -1
	for i := pos; i < len(b); i++ {
		p = i
		c := b[i]
		if c >= '0' && c <= '9' {
			if n == -1 {
				n = int(c) - '0'
			} else {
				n = n*10 + int(c) - '0'
			}
			if n > math.MaxInt16 {
				return math.MaxInt16, p
			}
		} else {
			if c == ' ' {
				if n >= 0 {
					return n, p
				}
			} else {
				if c != '.' && c != ':' && c != ';' && c != '_' {
					break
				}
			}
		}
	}
	if n == -1 {
		return fallback, -1
	}
	return n, p
}

/*
Template data read from configuration data JSONn file.
*/
type TemplateStaticFiles struct {
	Files            []string
	DataFile         string
	flatDataFromFile map[string]string
	isTemplating     bool
}

func (t *TemplateStaticFiles) Init(staticPath string, configErrors *ConfigErrorData, addTemplate func(string)) {
	t.flatDataFromFile = make(map[string]string)
	t.isTemplating = false
	if t.DataFile != "" {
		f := filepath.Join(staticPath, t.DataFile)
		content, err := os.ReadFile(f)
		if err != nil {
			configErrors.AddError(fmt.Sprintf("failed to read template data file. Error:%s", err.Error()))
			return
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(content, &m)
		if err != nil {
			configErrors.AddError(fmt.Sprintf("failed to parse template json file:%s. Error:%s", f, err.Error()))
			return
		}
		t.flatDataFromFile = FlattenMap(m, "")
	}
	if len(t.Files) == 0 {
		configErrors.AddError("No template 'TemplateStaticFiles.Files' have been defined")
	} else {
		for _, tf := range t.Files {
			f := filepath.Join(staticPath, tf)
			stat, err := os.Stat(f)
			if err != nil {
				configErrors.AddError(fmt.Sprintf("failed to find template file:%s. Error:%s", f, err.Error()))
			}
			if stat != nil && stat.IsDir() {
				configErrors.AddError(fmt.Sprintf("template file:%s is a directory", f))
			}
			addTemplate(f)
		}
	}
	t.isTemplating = configErrors.ErrorCount() == 0
}

func (t *TemplateStaticFiles) String() string {
	f, _ := filepath.Abs(t.DataFile)
	return fmt.Sprintf("%s. Templates:%s", f, t.Files)
}

func (t *TemplateStaticFiles) DataPlus(plusFlatMap map[string]string) map[string]string {
	m := maps.Clone(t.flatDataFromFile)
	maps.Copy(m, plusFlatMap)
	return m
}

type LogData struct {
	FileNameMask   string
	Path           string
	MonitorSeconds int
	ConsoleOut     bool
}

type StaticWebData struct {
	Paths               map[string]string
	HomePage            string
	TemplateStaticFiles *TemplateStaticFiles
}

func (p *StaticWebData) IsDefined() bool {
	return len(p.Paths) == 0
}

func (p *StaticWebData) GetHomePage() string {
	return p.HomePage
}

func (p *StaticWebData) GetPathOrStatic(id string) string {
	v, ok := p.Paths[id]
	if ok {
		return v
	}
	return p.Paths[StaticPathName]
}

func (p *StaticWebData) GetStaticFile(name string) string {
	return filepath.Join(p.Paths[StaticPathName], name)
}

/*
Check that the file exists in the static path
*/
func (p *StaticWebData) checkFileExists(id, file string, configErrors *ConfigErrorData) (bool, string) {
	file = strings.TrimPrefix(file, "/")
	fileParts := strings.SplitN(file, "&", 2)
	if len(fileParts) < 1 {
		configErrors.AddError(fmt.Sprintf("StaticWebData file name is invalid '%s'", file))
		return false, ""
	}

	absFilePath := filepath.Join(p.GetPathOrStatic(id), strings.ReplaceAll(fileParts[0], "/", string(os.PathSeparator)))
	stats, err := os.Stat(absFilePath)
	if err != nil {
		configErrors.AddError(fmt.Sprintf("StaticWebData file '%s'. %s", absFilePath, err.Error()))
		return false, ""
	}
	if stats.IsDir() {
		configErrors.AddError(fmt.Sprintf("StaticWebData file '%s'. is a directory", absFilePath))
		return false, ""
	}
	return true, absFilePath
}

func (p *StaticWebData) ValidateStaticWebData(configErrors *ConfigErrorData, addTemplate func(string)) bool {
	if p.HomePage == "" {
		configErrors.AddError("StaticWebData 'Home' page is undefined in 'StaticWebData'")
		return false
	}
	if len(p.Paths) == 0 {
		configErrors.AddError("StaticWebData 'Paths' is empty. Requiries at least 'static")
		return false
	}
	staticPath := ""
	for n, v := range p.Paths {
		absFilePath, err := filepath.Abs(v)
		if err != nil {
			configErrors.AddError(fmt.Sprintf("StaticWebData 'Paths[%s]=%s'. %s", n, v, err.Error()))
		}
		stats, err := os.Stat(absFilePath)
		if err != nil {
			configErrors.AddError(fmt.Sprintf("StaticWebData 'Paths[%s]'. %s", n, err))
		} else {
			if !stats.IsDir() {
				configErrors.AddError(fmt.Sprintf("StaticWebData 'Paths[%s]=%s' is not a directory", n, absFilePath))
			}
		}
		if n == "static" {
			staticPath = absFilePath
		}
		if configErrors.ErrorCount() == 0 {
			p.Paths[n] = absFilePath
		}
	}
	if staticPath == "" {
		configErrors.AddError("StaticWebData 'Paths[static]' was not found")
		return false
	}
	if configErrors.ErrorCount() != 0 {
		return false
	}
	ok, path := p.checkFileExists(StaticPathName, p.HomePage, configErrors)
	if ok {
		p.HomePage = path
	}
	if p.TemplateStaticFiles != nil {
		p.TemplateStaticFiles.Init(staticPath, configErrors, addTemplate)
	}
	return configErrors.ErrorCount() == 0
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
	StdOutType    string
	LogDir        string
	LogOutFile    string
	LogErrFile    string
	StartLTSFile  string
	NzCodeReturns int
	Detached      bool
	CanStop       bool
	Description   string
	id            string
	execPath      string
}

func (p *ExecInfo) Validate(execPathRoot string, name string, addError func(string)) {
	p.execPath = execPathRoot
	if execPathRoot == "" {
		return
	}
	p.id = name
	stats, err := os.Stat(execPathRoot)
	if err != nil {
		addError(fmt.Sprintf("Config Error: ExecPath %s does not exist", execPathRoot))
		return
	}
	if !stats.IsDir() {
		addError(fmt.Sprintf("Config Error: ExecPath %s must be a directory", execPathRoot))
		return
	}

	if p.Detached {
		if p.LogDir != "" {
			addError(fmt.Sprintf("Config Error: Exec [%s] is detached. Cannot have LogDir='%s'", p.id, p.LogDir))
		}
		if p.LogOutFile != "" {
			addError(fmt.Sprintf("Config Error: Exec [%s] is detached. Cannot have LogOut='%s'", p.id, p.LogOutFile))
		}
		if p.LogErrFile != "" {
			addError(fmt.Sprintf("Config Error: Exec [%s] is detached. Cannot have LogErr='%s'", p.id, p.LogErrFile))
		}
		if p.StdOutType != "" {
			addError(fmt.Sprintf("Config Error: Exec [%s] is detached. Cannot have StdOutType='%s'", p.id, p.StdOutType))
		}
		if p.NzCodeReturns != 0 {
			addError(fmt.Sprintf("Config Error: Exec [%s] is detached. Cannot have NzCodeReturns='%d'", p.id, p.NzCodeReturns))
		}
	}
	if p.LogDir != "" {
		if strings.HasPrefix(p.LogDir, "..") {
			addError(fmt.Sprintf("Config Error: Exec [%s] log. %s", p.id, "Log Dir prefix ../ is invalid"))
			p.LogDir = p.LogDir[3:]
		}
		fp := filepath.Join(filepath.Dir(execPathRoot), p.LogDir)
		stats, err := os.Stat(fp)
		if err != nil {
			addError(fmt.Sprintf("Config Error: Exec [%s] log %s", p.id, err.Error()))
		} else {
			if stats != nil && !stats.IsDir() {
				addError(fmt.Sprintf("Config Error: Exec [%s] log %s", p.id, "Must be a directory"))
			}
		}
		if p.HasNoLogFilesDefined() {
			addError(fmt.Sprintf("Config Error: Exec [%s] has a Log entry but no LogOutFile or LogErrFile files are defined", p.id))
		}
		p.LogDir = fp
	}
}

func (p *ExecInfo) GetOutLogFile() string {
	if p.LogDir == "" || p.LogOutFile == "" {
		return ""
	}
	return filepath.Join(p.LogDir, p.LogOutFile)
}

func (p *ExecInfo) GetErrLogFile() string {
	if p.LogDir == "" || p.LogErrFile == "" {
		return ""
	}
	return filepath.Join(p.LogDir, p.LogErrFile)
}

func (p *ExecInfo) GetId() string {
	return p.id
}

func (p *ExecInfo) GetDesc() string {
	return p.Description
}

func (p *ExecInfo) HasNoLogFilesDefined() bool {
	if p.LogErrFile == "" && p.LogOutFile == "" {
		return true
	}
	return false
}

func (p *ExecInfo) String() string {
	return fmt.Sprintf("CMD:%s, LogOut:%s, LogErr:%s", p.Cmd, p.GetOutLogFile(), p.GetErrLogFile())
}

/*
Users Data. Derived from JSON!
*/
type UserData struct {
	Hidden    *bool             // If true the user will not appear in the users list "http://server:port/server/users"
	Name      string            // The name of the user. If the user ID is bob. The name could be Bob.
	Home      string            // All locations are prefixed with this path when resolved
	Locations map[string]string // Name,Value list for locations. The names are public the values are resolved relative to Home
	Env       map[string]string // Name,Value list combined with OS environment for substitutions in resolved locations
	Info      *bool
}

func (p *UserData) CanSeeInfo() bool {
	if p.Info == nil {
		return false
	}
	return *p.Info
}

func (p *UserData) IsHidden() bool {
	if p.Hidden == nil {
		return false
	}
	return *p.Hidden
}

type ConfigDataFromFile struct {
	ReloadConfigSeconds int64
	Port                int
	ThumbnailTrim       []int
	UserDataPath        string
	UserPropertiesFile  string
	Users               map[string]UserData
	ContentTypeCharset  string
	LogData             *LogData
	ServerName          string
	FilterFiles         []string
	ServerDataRoot      string
	StaticWebData       *StaticWebData
	Env                 map[string]string
	Exec                map[string]*ExecInfo
	ExecPath            string
}

func (p *ConfigDataFromFile) String() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type ConfigData struct {
	ConfigFileData           *ConfigDataFromFile
	CurrentPath              string
	ModuleName               string
	ConfigName               string
	Debugging                bool
	Environment              map[string]string
	UserProps                *UserProperties
	NextConfigLoadTimeMillis int64
	LocationsCreated         []string
	UpSince                  time.Time
	IsVerbose                bool
	HasStaticWebData         bool
	IsTemplating             bool
	TemplateFiles            []string
}

/*
LoadConfigData method loads the config data from a file
*/

func NewConfigData(configFileName string, moduleName string, debugging, createDir, verbose bool, configErrors *ConfigErrorData) *ConfigData {
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
	wd, _ := os.Getwd()
	if !strings.HasSuffix(configFileName, ConfigFileExtension) {
		configFileName = fmt.Sprintf("%s%s", configFileName, ConfigFileExtension)
	}

	if verbose {
		fmt.Printf("Config file:'%s'\n", configFileName)
	}

	configDataExternal := &ConfigData{
		ConfigFileData:           nil,
		UserProps:                nil,
		Debugging:                debugging,
		CurrentPath:              wd,
		ModuleName:               moduleName,
		ConfigName:               configFileName,
		Environment:              environ,
		NextConfigLoadTimeMillis: 0,
		LocationsCreated:         []string{},
		IsVerbose:                verbose,
		HasStaticWebData:         false,
		IsTemplating:             false,
		TemplateFiles:            []string{},
	}

	configDataFromFile := &ConfigDataFromFile{
		ReloadConfigSeconds: defaultReloadConfigSeconds,
		Port:                defaultPort,
		UserDataPath:        "",
		Users:               make(map[string]UserData),
		UserPropertiesFile:  "",
		LogData:             NewLogData(),
		ContentTypeCharset:  "utf-8",
		ServerName:          moduleName,
		FilterFiles:         []string{},
		ServerDataRoot:      "",
		StaticWebData:       &StaticWebData{Paths: make(map[string]string), HomePage: "", TemplateStaticFiles: nil},
		ThumbnailTrim:       []int{thumbnailTrimPrefix, thumbnailTrimSuffix},
		Env:                 map[string]string{},
		Exec:                map[string]*ExecInfo{},
		ExecPath:            "",
	}

	/*
		load the config object
	*/
	content, err := os.ReadFile(configDataExternal.ConfigName)
	if err != nil {
		configErrors.AddError(fmt.Sprintf("Failed to read config data file:%s. Error:%s", configDataExternal.ConfigName, err.Error()))
		return nil
	}

	content = SubstituteFromMap(content, environ, nil)

	err = json.Unmarshal(content, &configDataFromFile)
	if err != nil {
		configErrors.AddError(fmt.Sprintf("Failed to understand the config data in the file:%s. Error:%s", configDataExternal.ConfigName, err.Error()))
		return nil
	}

	configDataExternal.ConfigFileData = configDataFromFile

	if len(configDataExternal.ConfigFileData.ThumbnailTrim) < 2 {
		configErrors.AddError("Config data entry ThumbnailTrim data has less than 2 entries")
	}

	SetContentTypeCharset(configDataFromFile.ContentTypeCharset)
	/*
		Add config data Env to the Environment variables
	*/
	maps.Copy(configDataExternal.Environment, configDataFromFile.Env)

	configDataExternal.ResetTimeToReloadConfig()

	for i := 0; i < len(configDataFromFile.FilterFiles); i++ {
		f := strings.ToLower(configDataFromFile.FilterFiles[i])
		if !strings.HasPrefix(f, ".") {
			configDataFromFile.FilterFiles[i] = fmt.Sprintf(".%s", f)
		}
	}

	configDataExternal.UserProps, err = NewUserProperties(configDataFromFile.UserPropertiesFile)
	if err != nil {
		panic(fmt.Sprintf("Config file:%s. NewUserProperties: Failed to initialise user properties: %s", configDataExternal.ConfigName, err.Error()))
	}

	err = configDataExternal.loadUserData()
	if err != nil {
		panic(fmt.Sprintf("Config file:%s. UserDataPath: Failed to load user data: %s", configDataExternal.ConfigName, err.Error()))
	}

	return configDataExternal.resolveLocations(createDir, configErrors)

}

func (p *ConfigData) GetPropertiesMapForUser(data map[string]string) map[string]interface{} {
	user, ok := data["user"]
	if !ok {
		panic(NewConfigError("User not defined", http.StatusNotFound, fmt.Sprintf("User=%s", user)))
	}
	userData := p.GetUserData(user)
	if userData == nil {
		panic(NewConfigError("User not found", http.StatusNotFound, fmt.Sprintf("User=%s", user)))
	}
	return p.UserProps.MapDataForUser(user, userData)
}

// run concurrently so must use lock

func (p *ConfigData) GetSetUserProp(data map[string]string) string {
	user, ok := data["user"]
	if !ok {
		return ""
	}
	if p.GetUserData(user) == nil {
		panic(NewConfigError("User not found", http.StatusNotFound, fmt.Sprintf("User=%s", user)))
	}
	name, ok := data["name"]
	if !ok {
		return ""
	}
	value, ok := data["value"]
	if !ok {
		value = ""
	}
	if p.ConfigFileData.UserPropertiesFile == "" {
		return value
	}
	key := fmt.Sprintf("%s.%s", user, name)
	return p.UserProps.Update(user, key, value)
}

/*
Load user data from external file defined in p.FileData.UserDataPath.

Update default values (Home & Hidden)

For each user.location substitute the environment var and check the resolved location exists.
*/
func (p *ConfigData) loadUserData() error {
	if p.ConfigFileData.UserDataPath == "" {
		return nil
	}
	content, err := os.ReadFile(p.ConfigFileData.UserDataPath)
	if err != nil {
		return fmt.Errorf("failed to read user data file:%s. Error:%s", p.ConfigFileData.UserDataPath, err.Error())
	}
	userData := make(map[string]UserData)
	err = json.Unmarshal(content, &userData)
	if err != nil {
		return fmt.Errorf("failed to understand user data file:%s. Error:%s", p.ConfigFileData.UserDataPath, err.Error())
	}

	for n, v := range userData {
		_, ok := p.ConfigFileData.Users[n]
		if ok {
			return fmt.Errorf("duplicate User '%s' defined in Users and UserDataPath. Config file:%s", n, p.ConfigName)
		}
		if v.Home == "" {
			v.Home = n
		}
		if v.Hidden == nil {
			b := false
			v.Hidden = &b
		}
		p.ConfigFileData.Users[n] = v
	}

	return nil
}

func (p *ConfigData) ShouldTemplateFile(name string) bool {
	if p.IsTemplating {
		if slices.Contains(p.TemplateFiles, name) {
			return true
		}
	}
	return false
}

/*
Construct a path from a relative path and user path.

If the 'relative path' is prefixed with an AbsolutePathPrefix, this is removed and the resultant path returned.

If it is just the 'ueser path', It is joined to the ServerDataRoot.

ServerDataRoot + userHome + path
*/
func (p *ConfigData) resolvePaths(userHome, root, path string) string {
	if strings.Contains(path, "..") {
		path = strings.ReplaceAll(path, "../", "")
		path = strings.ReplaceAll(path, "..", "")
	}
	if path == "" {
		return filepath.Join(root, userHome)
	}
	if strings.HasPrefix(path, AbsolutePathPrefix) {
		return path[len(AbsolutePathPrefix):]
	}
	if strings.HasPrefix(path, root) {
		return path
	}
	if userHome == "" {
		return filepath.Join(root, path)
	}
	return filepath.Join(root, userHome, path)
}

func (p *ConfigData) resolveLocations(createDir bool, configErrors *ConfigErrorData) *ConfigData {
	userConfigEnv := p.GetUserEnv("")
	defer func() {
		if rec := recover(); rec != nil {
			switch x := rec.(type) {
			case *ConfigError:
				configErrors.AddError(x.LogError())
			case string:
				configErrors.AddError(x)
			case error:
				configErrors.AddError(x.Error())
			default:
				// Fallback err (per specs, error strings should be lowercase w/o punctuation
				configErrors.AddError(fmt.Sprintf("%v", rec))
			}
		}
	}()

	f, e := p.checkRootPathExists(p.GetServerDataRoot(), userConfigEnv, true) // Will check GetServerDataRoot
	if e != nil {
		configErrors.AddError(fmt.Sprintf("Failed to find ServerDataRoot:%s. Cause:%s", f, e.Error()))
	} else {
		p.SetServerDataRoot(f)
	}

	p.HasStaticWebData = false
	p.IsTemplating = false
	p.TemplateFiles = []string{}
	if p.ConfigFileData.StaticWebData != nil {
		// Check static data
		p.HasStaticWebData = p.ConfigFileData.StaticWebData.ValidateStaticWebData(configErrors, func(s string) {
			p.TemplateFiles = append(p.TemplateFiles, s)
		})
		p.IsTemplating = p.ConfigFileData.StaticWebData.TemplateStaticFiles.isTemplating
	}

	if p.ConfigFileData.ExecPath != "" {
		f, e = p.checkRootPathExists(p.ConfigFileData.ExecPath, userConfigEnv, true) // Will check ExecPath
		if e != nil {
			configErrors.AddError(fmt.Sprintf("Config Error: ExecManager %s", e))
		} else {
			p.ConfigFileData.ExecPath = f
		}
	}

	if p.ConfigFileData.UserDataPath != "" {
		f, e = p.checkRootPathExists(p.ConfigFileData.UserDataPath, userConfigEnv, false) // Will check UserDataPath
		if e != nil {
			configErrors.AddError(fmt.Sprintf("Config Error: UserDataPath %s. %s", f, e))
		}
		p.ConfigFileData.UserDataPath = f
	}

	f, e = p.checkPathExists("", p.GetLogDataPath(), "", userConfigEnv, false)
	if e != nil {
		configErrors.AddError(fmt.Sprintf("Config Error: LogData.Path %s", e))
	} else {
		p.SetLogDataPath(f)
	}

	for execName, execData := range p.ConfigFileData.Exec {
		if p.GetExecPath() == "" {
			configErrors.AddError("Config Error: Exec entries found. ExecPath cannot be undefined")
		} else {
			execData.Validate(p.ConfigFileData.ExecPath, execName, configErrors.AddError)
		}

		for i, v := range execData.Cmd {
			execData.Cmd[i] = p.SubstituteFromMap([]byte(v), userConfigEnv)
		}
		execData.LogOutFile = p.SubstituteFromMap([]byte(execData.LogOutFile), userConfigEnv)
		execData.LogErrFile = p.SubstituteFromMap([]byte(execData.LogErrFile), userConfigEnv)
		if execData.StdOutType != "" && !HasContentType(execData.StdOutType) {
			configErrors.AddError(fmt.Sprintf("Config Error: Exec [%s] StdOutType [%s] not recognised", execName, execData.StdOutType))
		}
		execData.id = execName
	}

	for userId, userData := range p.ConfigFileData.Users {
		if userData.Home == "" {
			userData.Home = userId
		}
		if userData.Hidden == nil {
			b := false
			userData.Hidden = &b
		}
		userConfigEnv = p.GetUserEnv(userId)
		for locName := range userData.Locations {
			location := p.GetUserLocPath(userId, locName)
			f, e := p.checkPathExists(userData.Home, location, userId, userConfigEnv, createDir)
			if e != nil {
				configErrors.AddError(fmt.Sprintf("Config Error: User [%s] Location [%s] %s", userId, locName, e.Error()))
			}
			userData.Locations[locName] = f
		}
	}

	return p
}

func (p *ConfigData) checkRootPathExists(rootPath string, userEnv map[string]string, mustBeDir bool) (string, error) {
	if rootPath == "" {
		return "", fmt.Errorf("path is empty")
	}
	if p.IsVerbose {
		fmt.Printf("checkRootPathExists: %s\n", rootPath)
	}
	absPathSub := p.SubstituteFromMap([]byte(rootPath), userEnv)
	if p.IsVerbose {
		fmt.Printf("checkRootPathExists:SubstituteFromMap: %s\n", absPathSub)
	}
	absPathPath, err := filepath.Abs(absPathSub)
	if err != nil {
		return absPathPath, fmt.Errorf("path [%s] is invalid", absPathPath)
	}
	if p.IsVerbose {
		fmt.Printf("checkRootPathExists:Abs: %s\n", absPathPath)
	}
	stats, err := os.Stat(absPathPath)
	if err != nil {
		return absPathPath, fmt.Errorf("path [%s] Not found", absPathPath)
	}
	if !stats.IsDir() {
		if mustBeDir {
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

func (p *ConfigData) checkPathExists(userHome string, relPath string, userId string, userEnv map[string]string, createDir bool) (string, error) {
	absPath := p.resolvePaths(userHome, p.GetServerDataRoot(), relPath)
	absPathSub := p.SubstituteFromMap([]byte(absPath), userEnv)
	absPathPath, err := filepath.Abs(absPathSub)
	if err != nil {
		return absPathPath, fmt.Errorf("path [%s] is invalid", absPathSub)
	}
	stats, err := os.Stat(absPathPath)
	if err != nil {
		if createDir {
			if p.IsVerbose {
				fmt.Printf("createFullDirectory:%s\n", absPathPath)
			}
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

func (p *ConfigData) IsTimeToReloadConfig(mowMillis int64) bool {
	return p.NextConfigLoadTimeMillis < mowMillis
}

func (p *ConfigData) GetTimeToReloadSeconds() int64 {
	return (p.NextConfigLoadTimeMillis - time.Now().UnixMilli()) / 1000
}

func (p *ConfigData) ResetTimeToReloadConfig() {
	p.NextConfigLoadTimeMillis = time.Now().UnixMilli() + (p.ConfigFileData.ReloadConfigSeconds * 1000)
}

func (p *ConfigData) GetServerName() string {
	return p.ConfigFileData.ServerName
}

func (p *ConfigData) GetExecPath() string {
	return p.ConfigFileData.ExecPath
}

func (p *ConfigData) GetUserData(user string) *UserData {
	ud, ok := p.ConfigFileData.Users[user]
	if ok {
		return &ud
	}
	return nil
}

func (p *ConfigData) GetUsers() *map[string]UserData {
	return &p.ConfigFileData.Users
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
	p.ConfigFileData.Users[user] = ud
	return nil
}

func (p *ConfigData) HasUser(user string) bool {
	ulc := strings.ToLower(user)
	for na := range p.ConfigFileData.Users {
		if strings.ToLower(na) == ulc {
			return true
		}
	}
	return false
}

func (p *ConfigData) GetUserRoot(user string) string {
	return p.resolvePaths(p.GetUserData(user).Home, p.GetServerDataRoot(), "")
}

func (p *ConfigData) GetUserNamesList() []string {
	unl := []string{}
	for na, u := range p.ConfigFileData.Users {
		if !u.IsHidden() {
			unl = append(unl, na)
		}
	}
	return unl
}

func (p *ConfigData) GetFilesFilter() []string {
	return p.ConfigFileData.FilterFiles
}

func (p *ConfigData) ConvertToThumbnail(name string, convert bool) (resp string) {
	if !convert {
		return name
	}
	defer func() {
		if r := recover(); r != nil {
			resp = name
		}
	}()
	return name[p.ConfigFileData.ThumbnailTrim[0] : len(name)-p.ConfigFileData.ThumbnailTrim[1]]
}

func (p *ConfigData) GetServerDataRoot() string {
	return p.ConfigFileData.ServerDataRoot
}

func (p *ConfigData) SetServerDataRoot(f string) {
	p.ConfigFileData.ServerDataRoot = f
}

func (p *ConfigData) GetStaticWebData() *StaticWebData {
	if p.HasStaticWebData {
		return p.ConfigFileData.StaticWebData
	}
	return nil
}

func (p *ConfigData) GetContentTypeCharset() string {
	return p.ConfigFileData.ContentTypeCharset
}

func (p *ConfigData) GetLogDataPath() string {
	return p.ConfigFileData.LogData.Path
}

func (p *ConfigData) GetPathForDisplay(path string) string {
	spl := strings.Split(path, "/")
	l := len(spl)
	if l < 3 {
		return spl[l-1]
	}
	return strings.Join(spl[3:], "/")
}

func (p *ConfigData) SetLogDataPath(f string) {
	p.ConfigFileData.LogData.Path = f
}

func (p *ConfigData) GetLogData() *LogData {
	return p.ConfigFileData.LogData
}

func (p *ConfigData) GetUserEnv(user string) map[string]string {
	m := make(map[string]string)
	if user != "" {
		userData, ok := p.ConfigFileData.Users[user]
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
	return fmt.Sprintf(":%d", p.ConfigFileData.Port)
}

func (p *ConfigData) SetPortString(ps string) {
	if strings.TrimSpace(ps) == "" {
		return // If value is empty then do nothing
	}
	port, err := strconv.Atoi(ps)
	if err != nil {
		panic(fmt.Sprintf("Invalid port override. App parameter: port=%s", ps))
	}
	p.ConfigFileData.Port = port
}

// PANIC
func (p *ConfigData) GetUserLocPath(user string, loc string) string {
	userData, ok := p.ConfigFileData.Users[user]
	if !ok {
		panic(NewConfigError("User not found", http.StatusNotFound, fmt.Sprintf("User=%s", user)))
	}
	locData, ok := userData.Locations[loc]
	if !ok {
		panic(NewConfigError("Location not found", http.StatusNotFound, fmt.Sprintf("User=%s Location=%s", user, loc)))
	}
	return locData
}

// PANIC
func (p *ConfigData) GetExecInfo(execid string) *ExecInfo {
	exec, ok := p.ConfigFileData.Exec[execid]
	if !ok {
		panic(NewConfigError("Exec ID not found", http.StatusNotFound, fmt.Sprintf("exec-id=%s", execid)))
	}
	return exec
}

func (p *ConfigData) GetExecData() map[string]*ExecInfo {
	return p.ConfigFileData.Exec
}

func (p *ConfigData) String() (string, error) {
	data, err := p.ConfigFileData.String()
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
	return string(SubstituteFromMap(cmd, p.Environment, userEnv))
}

func SubstituteFromMap(cmd []byte, env1 map[string]string, env2 map[string]string) []byte {
	if len(cmd) < 4 {
		return cmd
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
	return buff.Bytes()
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

func (p *ConfigErrorData) AddError(s string) {
	p.errors = append(p.errors, s)
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
