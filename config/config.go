package config

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
)

const fallbackModuleName = "goWebApp"
const configFileExtension = ".json"

/*
ConfigData - Read configuration data from the JSON configuration file.
Note any undefined values are defaulted to constants defined below
*/

type UserData struct {
	Name      string
	Locations map[string]string
}

func NewUserData(name string, locations map[string]string) UserData {
	return UserData{
		Name:      name,
		Locations: locations}
}

type Parameters struct {
	params map[string]string
}

func NewParameters(p map[string]string) *Parameters {
	return &Parameters{
		params: p,
	}
}

func (p *Parameters) GetParam(key string) string {
	v, ok := p.params[key]
	if ok {
		return v
	}
	return ""
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
	DefaultLogFileName string
	ServerName         string
	PanicResponseCode  int
	LoggerLevels       map[string]string
	ModuleName         string `json:"-"`
	ConfigName         string `json:"-"`
	Debugging          bool   `json:"-"`
}

/*
LoadConfigData method loads the config data from a file
*/

func NewConfigData(configFileName string) (*ConfigData, error) {

	moduleName, debugging := getApplicationModuleName()
	if configFileName == "" {
		configFileName = moduleName
	} else {
		if strings.HasSuffix(strings.ToLower(configFileName), configFileExtension) {
			configFileName = configFileName[0 : len(configFileName)-5]
		}
	}

	configDataInstance := &ConfigData{
		Port:               8080,
		Users:              make(map[string]UserData),
		UserDataRoot:       "~/",
		DefaultLogFileName: "",
		ContentTypeCharset: "utf-8",
		ServerName:         moduleName,
		LoggerLevels:       make(map[string]string),
		PanicResponseCode:  500,
		Debugging:          debugging,
		ModuleName:         moduleName,
		ConfigName:         configFileName + configFileExtension,
	}

	/*
		load the config object
	*/
	content, err := os.ReadFile(configDataInstance.ConfigName)
	if err != nil {
		return nil, fmt.Errorf("failed to read config data file:%s. Error:%s", configDataInstance.ConfigName, err.Error())
	}

	err = json.Unmarshal(content, &configDataInstance)
	if err != nil {
		return nil, fmt.Errorf("failed to understand the config data in the file:%s. Error:%s", configDataInstance.ConfigName, err.Error())
	}

	return configDataInstance, nil
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
	return fmt.Sprintf("%s%c%s", p.UserDataRoot, os.PathSeparator, loc), nil
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
