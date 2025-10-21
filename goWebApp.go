package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/logging"
	"github.com/stuartdd/goWebApp/runCommand"
	"github.com/stuartdd/goWebApp/server"
)

const fallbackModuleName = "goWebApp"

func main() {
	createLocationsFlag := getArgFlag("c")
	doNotRun := getArgFlag("t")
	verbose := getArgFlag("v")
	killServer := getArgFlag("k")
	help := getArgFlag("h")

	if help {
		h, err := os.ReadFile("helptext.md")
		if err != nil {
			osExitWithMessage(1, "Help file 'helptext.md' not found")
		}
		osExitWithMessage(0, string(h))
	}

	moduleName, debugging := getApplicationModuleName(fallbackModuleName)
	configFileName, ok := getArgValue("config=")
	if !ok {
		configFileName = moduleName
	}

	if createLocationsFlag {
		c := osReader("Create missing USER locations:", "y/n")
		if c != "y" {
			osExitWithMessage(1, "Create missing USER locations: ABORTED")
		}
	}

	if verbose || doNotRun {
		fmt.Printf("Verbose: debugging=%t. killServer(k)=%t. createLocationsFlag(c)=%t.\n", killServer, debugging, createLocationsFlag)
	}

	configErrors := config.NewConfigErrorData()
	cfg := config.NewConfigData(configFileName, moduleName, debugging, createLocationsFlag, verbose, configErrors)
	if configErrors.ErrorCount() > 0 {
		os.Stdout.WriteString(configErrors.String())
		osExitWithMessage(1, "Config Errors: Cannot continue")
	}
	if cfg == nil {
		osExitWithMessage(1, "Config not loaded. Cannot continue")
	}

	if killServer {
		server.SendToHost(cfg.GetPortString(), "server/exit")
		time.Sleep(999 * time.Millisecond)
		osExitWithMessage(0, "Server exit requested")
	}

	if createLocationsFlag {
		if len(cfg.LocationsCreated) == 0 {
			osExitWithMessage(0, "No user Locations could be created:")
		} else {
			for _, s := range cfg.LocationsCreated {
				osExitWithMessage(-1, s)
			}
		}
		os.Exit(0)
	}

	if doNotRun {
		s, _ := cfg.String()
		os.Stdout.WriteString(s)
		os.Stdout.WriteString("\n")
	}

	/*
		Starting the server...
	*/
	actionQueue := make(chan *server.ActionEvent, 10)
	defer close(actionQueue)

	ld := cfg.GetLogData()

	logger, err := logging.NewLogger(ld.Path, ld.FileNameMask, ld.MonitorSeconds, ld.ConsoleOut, verbose)
	if err != nil {
		osExitWithMessage(1, fmt.Sprintf("NewLogger: failed to initialise. '%s'. ABORTED", err.Error()))
	}

	lrm, err := runCommand.NewLongRunningManager(cfg.GetExecPath(), logger.Log)
	if err != nil {
		osExitWithMessage(1, fmt.Sprintf("LongRunningManager: failed to initialise. '%s'. ABORTED", err.Error()))
	}
	if lrm.IsEnabled() {
		for n, v := range cfg.GetExecData() {
			if v.Detached {
				err := lrm.AddLongRunningProcessData(n, v.Description, v.Cmd, v.CanStop)
				if err != nil {
					osExitWithMessage(1, fmt.Sprintf("LongRunningManager: failed to add long running (Detached) process. '%s'. ABORTED", err.Error()))
				}
			}
		}
	}

	logger.Log("Application Start :-----------------------------------------------------------------")
	for _, l := range configErrors.Logs() {
		logger.Log(l)
	}

	if doNotRun {
		fmt.Print("Option -t (test) used. Server aborted")
		os.Exit(0)
	}

	webAppServer, err := server.NewWebAppServer(cfg, actionQueue, lrm, logger)

	if err != nil {
		s := fmt.Sprintf("Server Error : %s\n", err.Error())
		logger.Log(s)
		os.Stderr.WriteString(s)
		os.Exit(1)
	}

	go func() {
		for {
			a := <-actionQueue
			if a != nil {
				switch a.Id {
				case server.Exit:
					logger.Log(fmt.Sprintf("Server Terminated : %s", a.String()))
					time.Sleep(200 * time.Millisecond)
					logger.Close()
					time.Sleep(200 * time.Millisecond)
					webAppServer.Close(a.Rc)
				case server.Ignore:
					logger.Log("Server: Action Ignore")
				}
			}
		}
	}()
	rc := webAppServer.Start()
	os.Exit(rc)
}

func getArgFlag(name string) bool {
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		if strings.HasPrefix(a, "-") {
			if strings.Contains(a, name) {
				return true
			}
		}
	}
	return false
}

func getArgValue(name string) (string, bool) {
	nl := strings.ToLower(name) + "="
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		al := strings.ToLower(a)
		if strings.HasPrefix(al, nl) {
			return a[len(nl):], true
		}
	}
	return "", false
}

func osExitWithMessage(rc int, message string) {
	if rc > 0 {
		os.Stderr.WriteString(message)
		os.Stderr.WriteString("\n")
	} else {
		os.Stdout.WriteString(message)
		os.Stdout.WriteString("\n")
	}
	if rc >= 0 {
		os.Exit(rc)
	}
}

func osReader(message string, chars string) string {
	os.Stdout.WriteString(message)
	os.Stdout.WriteString(" (")
	os.Stdout.WriteString(chars)
	os.Stdout.WriteString(")? :")
	reader := bufio.NewReader(os.Stdin)
	charsLc := strings.ToLower(chars)
	char, _, err := reader.ReadRune()
	if err != nil {
		osExitWithMessage(1, fmt.Sprintf("Input was not understood: Error:%s", err.Error()))
	}
	s := strings.ToLower(string(char))
	if strings.Contains(charsLc, s) {
		return s
	}
	return ""
}

/*
GetApplicationModuleName returns the name of the application. Testing and debugging changes this name so the code
removes debug, test and .exe from the executable name.
*/
func getApplicationModuleName(fallbackModuleName string) (string, bool) {
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
