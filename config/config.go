package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const fallbackModuleName = "goWebApp"
const configFileExtension = ".json"

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
	return ""
}

func (p *Parameters) UserDataFile() (string, error) {
	return p.configData.UserDataFile(p)
}

func (p *Parameters) UserDataPath() (string, error) {
	return p.configData.UserDataPath(p)
}

func (p *Parameters) FilterFiles() []string {
	return p.configData.FilterFiles
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

type ConfigData struct {
	Port               int
	Users              map[string]UserData
	UserDataRoot       string
	ContentTypeCharset string
	LogData            *LogData
	ServerName         string
	FaviconIcoPath     string
	PanicResponseCode  int
	FilterFiles        []string
	CurrentPath        string `json:"-"`
	ModuleName         string `json:"-"`
	ConfigName         string `json:"-"`
	Debugging          bool   `json:"-"`
}

func checkExists(p string, f string, isFile bool) string {
	var fr string
	var err error

	if !strings.HasPrefix(f, "***") {
		pf := fmt.Sprintf("%s%c%s", p, os.PathSeparator, f)
		fr, err = filepath.Abs(pf)
		if err != nil {
			return fmt.Sprintf("[%s] not resolved to path", f)
		}
	} else {
		fr = f[3:]
	}
	stats, err := os.Stat(fr)
	if err != nil {
		return fmt.Sprintf("Path [%s] Not found", fr)
	} else {
		if isFile {
			if stats.IsDir() {
				return fmt.Sprintf("Path [%s] Not a File", fr)
			}
		} else {
			if !stats.IsDir() {
				return fmt.Sprintf("Path[%s] Not a Directory", fr)
			}
		}
	}
	return ""
}

func (p *ConfigData) resolveLocations() (*ConfigData, []string) {
	errorList := make([]string, 0)

	e := checkExists(p.UserDataRoot, "", false)
	if e != "" {
		errorList = append(errorList, fmt.Sprintf("Config Error: User Data Root %s", e))
	}

	e = checkExists(p.UserDataRoot, p.FaviconIcoPath, true)
	if e != "" {
		errorList = append(errorList, fmt.Sprintf("Config Error: faviconIcoPath not found %s", e))
	}

	e = checkExists(p.UserDataRoot, p.LogData.Path, false)
	if e != "" {
		errorList = append(errorList, fmt.Sprintf("Config Error: LogData.Path %s", e))
	}

	for userName, userData := range p.Users {
		for locName, location := range userData.Locations {
			e := checkExists(p.UserDataRoot, location, false)
			if e != "" {
				errorList = append(errorList, fmt.Sprintf("Config Error: User [%s] Location [%s] %s", userName, locName, e))
			}
		}
		for execName, execData := range userData.Exec {
			if execData.Log != "" {
				s := strings.SplitN(execData.Log, "/", 2)
				if len(s) == 2 {
					loc, ok := userData.Locations[s[0]]
					if !ok {
						errorList = append(errorList, fmt.Sprintf("Config Error: User [%s] Exec [%s] Invalid Log[%s]. Loc [%s] Not found", userName, execName, execData.Log, s[0]))
					} else {
						execData.Log = fmt.Sprintf("%s%c%s%c%s", p.UserDataRoot, os.PathSeparator, loc, os.PathSeparator, s[1])
					}
				} else {
					errorList = append(errorList, fmt.Sprintf("Config Error: User [%s] Exec [%s] Invalid Log[%s]. Use loc/filename", userName, execName, execData.Log))
				}
			}
		}
	}
	return p, errorList
}

/*
LoadConfigData method loads the config data from a file
*/

func NewConfigData(configFileName string) (*ConfigData, []string) {

	moduleName, debugging := getApplicationModuleName()
	if configFileName == "" {
		configFileName = moduleName
	} else {
		if strings.HasSuffix(strings.ToLower(configFileName), configFileExtension) {
			configFileName = configFileName[0 : len(configFileName)-5]
		}
	}

	wd, _ := os.Getwd()
	configDataInstance := &ConfigData{
		Port:               8080,
		Users:              make(map[string]UserData),
		UserDataRoot:       "~/",
		LogData:            nil,
		ContentTypeCharset: "utf-8",
		ServerName:         moduleName,
		FilterFiles:        []string{},
		FaviconIcoPath:     "",
		PanicResponseCode:  500,
		Debugging:          debugging,
		CurrentPath:        wd,
		ModuleName:         moduleName,
		ConfigName:         configFileName + configFileExtension,
	}

	/*
		load the config object
	*/
	content, err := os.ReadFile(configDataInstance.ConfigName)
	if err != nil {
		return nil, []string{fmt.Sprintf("Failed to read config data file:%s. Error:%s", configDataInstance.ConfigName, err.Error())}
	}

	err = json.Unmarshal(content, &configDataInstance)
	if err != nil {
		return nil, []string{fmt.Sprintf("FSailed to understand the config data in the file:%s. Error:%s", configDataInstance.ConfigName, err.Error())}
	}

	for i := 0; i < len(configDataInstance.FilterFiles); i++ {
		configDataInstance.FilterFiles[i] = fmt.Sprintf(".%s", strings.ToLower(configDataInstance.FilterFiles[i]))
	}

	return configDataInstance.resolveLocations()
}

func (p *ConfigData) PortString() string {
	return fmt.Sprintf(":%d", p.Port)
}

func (p *ConfigData) UserDataPath(parameters *Parameters) (string, error) {
	user, ok := p.Users[parameters.GetUser()]
	if !ok {
		return "", fmt.Errorf("user not found")
	}
	loc, ok := user.Locations[parameters.GetLocation()]
	if !ok {
		return "", fmt.Errorf("user location not found")
	}
	if strings.HasPrefix(loc, "***") {
		return loc[3:], nil
	}
	return fmt.Sprintf("%s%c%s", p.UserDataRoot, os.PathSeparator, loc), nil
}

func (p *ConfigData) UserExec(user, execid string) (*ExecInfo, error) {
	userData, ok := p.Users[user]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	exec, ok := userData.Exec[execid]
	if !ok {
		return nil, fmt.Errorf("exec: id not found")
	}
	return exec, nil
}

func (p *ConfigData) UserDataFile(parameters *Parameters) (string, error) {
	path, e := p.UserDataPath(parameters)
	if e != nil {
		return "", e
	}
	fileName := parameters.GetName()
	if fileName == "" {
		return "", fmt.Errorf("filename not found")
	}
	return fmt.Sprintf("%s%c%s", path, os.PathSeparator, fileName), nil
}

func (p *ConfigData) ToString() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
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
