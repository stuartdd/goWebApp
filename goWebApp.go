package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"stuartdd.com/config"
	"stuartdd.com/logging"
	"stuartdd.com/pictures"
	"stuartdd.com/server"
)

func main() {
	help, _ := getArg("help")
	if help != "" {
		h, err := os.ReadFile("helptext.md")
		if err != nil {
			osExitWithMessage(1, "Help file 'helptext.md' not found")
		}
		osExitWithMessage(0, string(h))
	}

	dontResolveConfig := false
	configFileName, _ := getArg("config=")
	s, _ := getArg("create")
	createLocationsFlag := s != ""
	s, addPos := getArg("add")
	addUserFlag := s != ""
	s, scanPos := getArg("scan")
	scanDirFlag := s != ""

	if addUserFlag && addPos == 0 {
		// add was the last parameter!
		osExitWithMessage(1, "Add user. Name not found")
	}

	var addUserName string
	if addUserFlag {
		if createLocationsFlag {
			osExitWithMessage(1, "Cannot use Add and Create at the same time")
		}
		dontResolveConfig = true
		addUserName = os.Args[addPos]
	}

	if createLocationsFlag {
		c := osReader("Create missing USER locations:", "y/n")
		if c != "y" {
			osExitWithMessage(1, "Create missing USER locations: ABORTED")
		}
	}

	cfg, errorList := config.NewConfigData(configFileName, createLocationsFlag, dontResolveConfig)
	if errorList.ErrorCount() > 0 {
		os.Stdout.WriteString(errorList.String())
		osExitWithMessage(1, "Config Errors: Cannot continue")
	}
	if cfg == nil {
		osExitWithMessage(1, "Config not loaded. Cannot continue")
	}

	if scanDirFlag {
		if scanPos == 0 {
			osExitWithMessage(1, "Scan: requires a user name.")
		}
		user := os.Args[scanPos]
		output := scanUserOriginals(user, cfg)
		os.Stdout.WriteString(output)
		os.Exit(0)
	}

	if createLocationsFlag {
		if len(cfg.LocationsCreated) == 0 {
			osExitWithMessage(0, "No user Locations could be crreated:"+s)
		} else {
			for _, s := range cfg.LocationsCreated {
				osExitWithMessage(-1, s)
			}
		}
		os.Exit(0)
	}

	if addUserFlag {
		c := osReader(fmt.Sprintf("Add User with userid: '%s'", addUserName), "y/n")
		if c == "y" {
			err := cfg.AddUser(addUserName)
			if err != nil {
				osExitWithMessage(1, fmt.Sprintf("Add User: '%s'.", err.Error()))
			}
			err = cfg.SaveMe()
			if err != nil {
				osExitWithMessage(1, fmt.Sprintf("Add User: '%s'.", err.Error()))
			}
			osExitWithMessage(0, fmt.Sprintf("Add User: '%s' Added and saved", addUserName))
		} else {
			osExitWithMessage(1, fmt.Sprintf("Add User: '%s' ABORTED", addUserName))
		}
		os.Exit(0)
	}
	/*
		Starting the server...
	*/
	actionQueue := make(chan *server.ActionEvent, 10)
	defer close(actionQueue)

	ld := cfg.GetLogData()

	logger, err := logging.NewLogger(ld.Path, ld.FileNameMask, ld.MonitorSeconds, ld.LogLevel, ld.ConsoleOut)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	lrm := server.NewLongRunningManagerDisabled()
	em := cfg.GetExecManager()
	if em != nil {
		lrm, err = server.NewLongRunningManager(em.Path, em.File, em.TestCommand, logger.Log)
		if err != nil {
			osExitWithMessage(1, fmt.Sprintf("LongRunningManager: failed to initialise. '%s'. ABORTED", err.Error()))
		}
	}

	logger.Log("Application Start :-----------------------------------------------------------------")
	for _, l := range errorList.Logs() {
		logger.Log(l)
	}
	go func() {
		for {
			a := <-actionQueue
			switch a.Id {
			case server.Exit:
				logger.Log(fmt.Sprintf("Server Terminated : %s", a.String()))
				time.Sleep(200 * time.Millisecond)
				logger.Close()
				time.Sleep(200 * time.Millisecond)
				os.Exit(a.Rc)
			case server.Ignore:
				logger.Log("Server: Action Ignore")
			}
		}
	}()

	webAppServer := server.NewWebAppServer(cfg, actionQueue, lrm, logger)
	os.Exit(webAppServer.Start())
}

func getArg(name string) (string, int) {
	nl := strings.ToLower(name)
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		al := strings.ToLower(a)
		if al == nl {
			if i < (len(os.Args) - 1) {
				return nl, i + 1
			}
			return nl, 0
		}
		if strings.HasPrefix(al, nl) {
			if i < (len(os.Args) - 1) {
				return a[len(name):], i + 1
			}
			return a[len(name):], 0
		}
	}
	return "", 0
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

func scanUserOriginals(user string, cfg *config.ConfigData) string {
	path, err := cfg.GetUserLocPath(user, "original")
	if err != nil {
		osExitWithMessage(1, fmt.Sprintf("Scan: User '%s' Location 'original' not found.", user))
	}
	sd, err := pictures.ScanDirectory(path, []string{"jpg", "jpeg"}, pictures.DirDataScanFileName)
	if err != nil {
		osExitWithMessage(1, fmt.Sprintf("Scan: '%s'.", err.Error()))
	}
	var buff bytes.Buffer
	if sd.NewState == nil && sd.OldState != nil && sd.OldStateCount > 0 {
		sd.OldState.VisitEachFile(func(pp *pictures.PicPath, s string) bool {
			buff.WriteString(fmt.Sprintf("NEW:%s", filepath.Join(pp.String(), s)))
			buff.WriteString("\n")
			return true
		})
	} else {
		if sd.FilesAdded != nil {
			sd.FilesAdded.VisitEachFile(func(pp *pictures.PicPath, s string) bool {
				buff.WriteString(fmt.Sprintf("ADD:%s", filepath.Join(pp.String(), s)))
				buff.WriteString("\n")
				return true
			})
		}
		if sd.FilesDeleted != nil {
			sd.FilesDeleted.VisitEachFile(func(pp *pictures.PicPath, s string) bool {
				buff.WriteString(fmt.Sprintf("DEL:%s", filepath.Join(pp.String(), s)))
				buff.WriteString("\n")
				return true
			})
		}
	}
	err = sd.Commit(true)
	if err != nil {
		osExitWithMessage(1, fmt.Sprintf("Scan: '%s'.", err.Error()))
	}
	return buff.String()
}
