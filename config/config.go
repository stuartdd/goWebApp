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
Data - Read configuration data from the JSON configuration file.
Note any undefined values are defaulted to constants defined below
*/
type Data struct {
	Port               int
	DefaultLogFileName string
	ContentTypeCharset string
	LoggerLevels       map[string]string
	PanicResponseCode  int
	ModuleName         string `json:"-"`
	ConfigName         string `json:"-"`
	Debugging          bool   `json:"-"`
}

func (p *Data) PortString() string {
	return fmt.Sprintf(":%d", p.Port)
}

func (p *Data) ToString() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to present data as Json:%s. Error:%s", p.ConfigName, err.Error())
	}
	return string(data), nil
}

/*
ScriptData - For a goven OS define the scriptpath and the script data
*/
type MuxData struct {
	Mux  *MuxData
	Data []string
}

/*
There should only ever be ONE of these
*/
var configDataInstance *Data

/*
GetConfigDataInstance get the confg data singleton
*/
func GetConfigDataInstance() *Data {
	return configDataInstance
}

/*
LoadConfigData method loads the config data from a file
*/
func LoadConfigData(configFileName string) error {

	moduleName, debugging := getApplicationModuleName()
	if configFileName == "" {
		configFileName = moduleName
	} else {
		if strings.HasSuffix(strings.ToLower(configFileName), configFileExtension) {
			configFileName = configFileName[0 : len(configFileName)-5]
		}
	}

	configDataInstance = &Data{
		Port:               8080,
		DefaultLogFileName: "",
		ContentTypeCharset: "utf-8",
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
		return fmt.Errorf("failed to read config data file:%s. Error:%s", configDataInstance.ConfigName, err.Error())
	}

	err = json.Unmarshal(content, &configDataInstance)
	if err != nil {
		return fmt.Errorf("failed to understand the config data in the file:%s. Error:%s", configDataInstance.ConfigName, err.Error())
	}

	return nil
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
