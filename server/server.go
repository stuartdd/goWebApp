package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/controllers"
	"github.com/stuartdd/goWebApp/logging"
	"github.com/stuartdd/goWebApp/runCommand"
)

const shouldLogYes = true
const shouldLogNo = false
const ServerExitUrl = "/server/exit"

type ActionId int

const (
	Exit ActionId = iota
	Ignore
)

type LoggableError interface {
	Error() string               // Display to the user, such as cause. NO Sensitive data!
	LogError() string            // Append to the logs. Contains diagnostic data for admin.
	Status() int                 // The status code returned to the browser
	Map() map[string]interface{} // A map that constructs the JSON respo
}

type ActionEvent struct {
	Id  ActionId
	Rc  int
	Msg string
}

func (p *ActionEvent) String() string {
	return fmt.Sprintf("[%d] %s", p.Rc, p.Msg)
}

func NewActionEvent(id ActionId, rc string, fallback int, m string) *ActionEvent {
	i, err := strconv.Atoi(rc)
	if err != nil {
		i = fallback
	}
	return &ActionEvent{Id: id, Rc: i, Msg: m}
}

var rootUrlList = NewRootUrlList()

var getPingMatch = rootUrlList.AddUrlRequestMatcher("/ping", "GET", shouldLogNo)
var getIsUpMatch = rootUrlList.AddUrlRequestMatcher("/isup", "GET", shouldLogNo)

var getServerStatusMatch = rootUrlList.AddUrlRequestMatcher("/server/status", "GET", shouldLogYes)
var getReloadConfigMatch = rootUrlList.AddUrlRequestMatcher("/server/config", "GET", shouldLogYes)
var getServerTimeMatch = rootUrlList.AddUrlRequestMatcher("/server/time", "GET", shouldLogNo)
var getServerUsersMatch = rootUrlList.AddUrlRequestMatcher("/server/users", "GET", shouldLogYes)
var getServerRestartMatch = rootUrlList.AddUrlRequestMatcher("/server/restart", "GET", shouldLogYes)
var getServerExitMatch = rootUrlList.AddUrlRequestMatcher(ServerExitUrl, "GET", shouldLogYes)
var getServerLogMatch = rootUrlList.AddUrlRequestMatcher("/server/log", "GET", shouldLogNo)
var delServerLogMatch = rootUrlList.AddUrlRequestMatcher("/server/log/*", "DELETE", shouldLogYes)

// Exec a script via an ID in config:"Exec" section.
// Script must be in  config:"ExecPath":
// User will be "admin"
var getExecMatch = rootUrlList.AddUrlRequestMatcher("/exec/*", "GET", shouldLogYes)
var getPropUserNameValueMatch = rootUrlList.AddUrlRequestMatcher("/prop/user/*/name/*/value/*", "GET", shouldLogYes)
var getPropUserNameMatch = rootUrlList.AddUrlRequestMatcher("/prop/user/*/name/*", "GET", shouldLogYes)
var getPropUserMatch = rootUrlList.AddUrlRequestMatcher("/prop/user/*", "GET", shouldLogYes)

// Get File asAdmin user. Location (loc) must be defined in admin user.
// var getFileLocNameMatch = rootUrlList.AddUrlRequestMatcher("/files/loc/*/name/*", "GET", shouldLogYes)
var getFileUserLocPathMatch = rootUrlList.AddUrlRequestMatcher("/files/user/*/loc/*/path/*", "GET", shouldLogYes)
var getFileUserLocMatch = rootUrlList.AddUrlRequestMatcher("/files/user/*/loc/*", "GET", shouldLogYes)
var getFileUserLocTreeMatch = rootUrlList.AddUrlRequestMatcher("/files/user/*/loc/*/tree", "GET", shouldLogYes)

// Specific File GET matchers Sub for FastFile!
var getFileUserLocNameMatch = rootUrlList.AddUrlRequestMatcher("/files/user/*/loc/*/name/*", "GET", shouldLogYes)
var getFileUserLocPathNameMatch = rootUrlList.AddUrlRequestMatcher("/files/user/*/loc/*/path/*/name/*", "GET", shouldLogYes)

// File NON GET matchers
var delFileUserLocNameMatch = rootUrlList.AddUrlRequestMatcher("/files/user/*/loc/*/name/*", "DELETE", shouldLogYes)
var postFileUserLocNameMatch = rootUrlList.AddUrlRequestMatcher("/files/user/*/loc/*/name/*", "POST", shouldLogYes)
var postFileUserLocPathNameMatch = rootUrlList.AddUrlRequestMatcher("/files/user/*/loc/*/path/*/name/*", "POST", shouldLogYes)

var getPathsUserLocMatch = rootUrlList.AddUrlRequestMatcher("/paths/user/*/loc/*", "GET", shouldLogYes)

var getTestUserLocNameMatch = rootUrlList.AddUrlRequestMatcher("/test/user/*/loc/*/name/*", "GET", shouldLogNo)

type ServerHandler struct {
	config      *config.ConfigData
	actionQueue chan *ActionEvent
	logger      logging.Logger
	upSince     time.Time
	longRunning *runCommand.LongRunningManager
}

func NewServerHandler(configData *config.ConfigData, actionQueue chan *ActionEvent, lrm *runCommand.LongRunningManager, logger logging.Logger, upSince time.Time) *ServerHandler {
	if lrm == nil {
		lrm = runCommand.NewLongRunningManagerDisabled()
	}

	return &ServerHandler{
		config:      configData,
		actionQueue: actionQueue,
		logger:      logger,
		longRunning: lrm,
		upSince:     upSince,
	}
}

func (p *ServerHandler) GetUpSince() time.Time {
	return p.upSince
}

func (h *ServerHandler) Log(s string) {
	h.logger.Log(s)
}

func (h *ServerHandler) close() {
	h.logger.Close()
}

func (h *ServerHandler) serveFile(w http.ResponseWriter, r *http.Request, name string, verboseFunc func(string), shouldLog bool) {
	stat, err := os.Stat(name)
	if err != nil {
		panic(config.NewServerError("File not found", http.StatusNotFound, fmt.Sprintf("File not found. :%s", h.config.GetPathForDisplay(name))))
	}
	if stat.IsDir() {
		panic(config.NewServerError("Is a Directory", http.StatusBadRequest, fmt.Sprintf("File %s is a Directory.", h.config.GetPathForDisplay(name))))
	}
	if verboseFunc != nil {
		verboseFunc(fmt.Sprintf("FastFile: %s", h.config.GetPathForDisplay(name)))
	}
	w.Header().Set("Server", h.config.GetServerName())
	w.Header().Set("Content-Type", config.LookupContentType(name))
	http.ServeFile(w, r, name)
}

func (h *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logFunc := h.logger.Log
	verboseFunc := h.logger.VerboseFunction()
	defer func() {
		var le LoggableError
		if rec := recover(); rec != nil {
			switch x := rec.(type) {
			case LoggableError:
				le = x
			case string:
				le = config.NewPanicError(x, 404)
			case error:
				le = config.NewPanicError(x.Error(), 404)
			default:
				// Fallback err (per specs, error strings should be lowercase w/o punctuation
				le = config.NewPanicError(fmt.Sprintf("%v", rec), 404)
			}
			logFunc(le.LogError())
			h.writeResponse(w, controllers.NewResponseData(le.Status()).SetHasErrors(true).WithContentMapAsJson(le.Map(), r.URL.Query()), true)
		}
	}()

	urlPath := strings.TrimSpace(r.URL.Path)
	if urlPath == "/" {
		if h.config.HasStaticWebData {
			homePage := h.config.GetStaticWebData().GetHomePage()
			if h.config.ShouldTemplateFile(homePage) {
				h.writeResponse(w, controllers.StaticFileTemplate(homePage, controllers.NewUrlRequestParts(h.config).WithQuery(r.URL.Query()).WithHeader(r.Header), logFunc), true)
			} else {
				h.serveFile(w, r, homePage, verboseFunc, shouldLogYes)
			}
			return
		}
		panic(config.NewServerError("Resource not found", http.StatusNotFound, fmt.Sprintf("Req:  %s:%s", r.Method, urlPath)))
	}

	requestUrlparts := strings.Split(urlPath, "/")
	if requestUrlparts[0] == "" {
		requestUrlparts = requestUrlparts[1:]
	}
	requestUrlpartsLen := len(requestUrlparts)
	if requestUrlpartsLen == 0 {
		h.writeErrorResponse(w, "Resource not found", http.StatusNotFound, fmt.Sprintf("Req:  %s: is empty", r.Method))
		return
	}

	// Is the root of the url cached in matcherRequestIds
	requestMatchesRoot := rootUrlList.HasRoot(requestUrlparts[0])

	// If there is Web Data there may be static files.. html , json, icons, jpg etc...
	if h.config.HasStaticWebData {
		staticFileName := ""
		// If url is singular and not a pre defined root url it may be a file name like 'favicon.ico'
		// Root files are not allowed so they are re-mapped to StaticWebData.Paths['static'] path
		if requestUrlpartsLen == 1 {
			// requestMatchesRoot is true for /ping, /isup, /server and other  'non-static' web data
			// so must be excluded.
			if !requestMatchesRoot {
				staticFileName = h.config.GetStaticWebData().GetStaticFile(requestUrlparts[0])
			}
		} else {
			// If url is multiple and the first is found in StaticWebData.Paths[requestUrlparts[0]]
			// This allows a mapping to 'images' of other paths defined in StaticWebData.Paths
			if requestUrlpartsLen > 1 {
				path, ok := h.config.GetStaticWebData().Paths[requestUrlparts[0]]
				if ok {
					// if root url is found then build a file path and name from the url parts (exclude [0])
					l := append([]string{path}, requestUrlparts[1:]...)
					staticFileName = filepath.Join(l...)
				}
			}
		}
		if staticFileName != "" {
			// if we derived a static file name then return the file ASAP
			if h.config.ShouldTemplateFile(staticFileName) {
				h.writeResponse(w, controllers.StaticFileTemplate(staticFileName, controllers.NewUrlRequestParts(h.config).WithQuery(r.URL.Query()).WithHeader(r.Header), verboseFunc), true)
			} else {
				h.serveFile(w, r, staticFileName, verboseFunc, shouldLogYes)
			}
			return
		}
		// Url is not a static file (yet!) so carry on..
	}

	urlRequestParts := controllers.NewUrlRequestParts(h.config).WithQuery(r.URL.Query()).WithHeader(r.Header)
	if !requestMatchesRoot {
		// The root of the url does not match any Matcher so 404!
		h.writeErrorResponse(w, "Resource not found", http.StatusNotFound, fmt.Sprintf("Req:  %s:%s%s", r.Method, urlPath, urlRequestParts.QueryAsString()))
		return
	}
	requestInfo := NewRequestInfo(r.Method, urlPath, r.URL.RawQuery, logFunc, verboseFunc)

	_, ok, shouldLog := getPingMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentWithCauseAsJson("Ping", nil), shouldLog)
		return
	}
	_, ok, shouldLog = getIsUpMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentWithCauseAsJson("ServerIsUp", nil), shouldLog)
		return
	}

	_, ok, shouldLog = getServerTimeMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentMapAsJson(controllers.GetTimeAsMap(), nil), shouldLog)
		return
	}
	_, ok, shouldLog = getFileUserLocNameMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		tn := r.URL.Query().Get("thumbnail")
		name := controllers.GetFastFileName(h.config, requestUrlparts, urlPath, (tn == "true"))
		h.serveFile(w, r, name, verboseFunc, shouldLog)
		return
	}
	_, ok, shouldLog = getFileUserLocPathNameMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		tn := r.URL.Query().Get("thumbnail")
		name := controllers.GetFastFileName(h.config, requestUrlparts, urlPath, (tn == "true"))
		h.serveFile(w, r, name, verboseFunc, shouldLog)
		return
	}
	_, ok, shouldLog = getTestUserLocNameMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		tn := r.URL.Query().Get("thumbnail")
		name := controllers.GetFastFileName(h.config, requestUrlparts, urlPath, (tn == "true"))
		h.serveFile(w, r, name, verboseFunc, shouldLog)
		return
	}
	p, ok, shouldLog := getFileUserLocPathMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		h.writeResponse(w, controllers.NewDirHandler(urlRequestParts.WithParameters(p), h.config, true, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = getFileUserLocMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewDirHandler(urlRequestParts.WithParameters(p), h.config, true, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = getPathsUserLocMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewDirHandler(urlRequestParts.WithParameters(p), h.config, false, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = getFileUserLocTreeMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewTreeHandler(urlRequestParts.WithParameters(p), h.config).Submit(), shouldLog)
		return
	}
	//  Service using FastFiles
	p, ok, shouldLog = delFileUserLocNameMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewDeleteFileHandler(urlRequestParts.WithParameters(p), h.config, verboseFunc).Submit(), shouldLog)
		return
	}

	p, ok, shouldLog = getPropUserNameValueMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewResponseData(200).WithContentBytes([]byte(h.config.GetSetUserProp(p))).WithMimeType("txt").AndLogContent(true), shouldLog)
		return
	}
	p, ok, shouldLog = getPropUserNameMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		//
		h.writeResponse(w, controllers.NewResponseData(200).WithContentBytes([]byte(h.config.GetSetUserProp(p))).WithMimeType("txt").AndLogContent(true), shouldLog)
		return
	}
	p, ok, shouldLog = getPropUserMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.GetPropertiesForUser(urlRequestParts.WithParameters(p), h.config), shouldLog)
		return
	}
	p, ok, shouldLog = postFileUserLocPathNameMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewPostFileHandler(urlRequestParts.WithParameters(p), h.config, r, false, verboseFunc).Submit(), shouldLog)
		return
	}

	p, ok, shouldLog = getExecMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check ????
		h.writeResponse(w, controllers.NewExecHandler(urlRequestParts.WithParameters(p).AsAdmin(), h.config.GetExecPath(), nil, logFunc, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = postFileUserLocNameMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewPostFileHandler(urlRequestParts.WithParameters(p), h.config, r, false, verboseFunc).Submit(), shouldLog)
		return
	}
	_, ok, shouldLog = getServerRestartMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		a := NewActionEvent(Exit, urlRequestParts.GetOptionalQuery("rc", "23"), 23, "Restart Requested")
		h.actionQueue <- a
		h.writeResponse(w, controllers.NewResponseData(http.StatusAccepted).WithContentMapAsJson(map[string]interface{}{"Status": "RESTARTED"}, nil), shouldLog)
		return
	}
	_, ok, shouldLog = getServerExitMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		a := NewActionEvent(Exit, urlRequestParts.GetOptionalQuery("rc", "11"), 11, "Exit Requested")
		h.actionQueue <- a
		h.writeResponse(w, controllers.NewResponseData(http.StatusAccepted).WithContentWithCauseAsJson(a.String(), nil), shouldLog)
		return
	}
	// Panic Check Done
	_, ok, shouldLog = getServerStatusMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		if h.longRunning.IsEnabled() {
			h.longRunning.Update()
		}
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentBytes(controllers.GetServerStatusAsJson(h.config, h.logger.LogFileName(), h.GetUpSince(), h.longRunning.ToJson())), shouldLog)
		return
	}

	_, ok, shouldLog = getServerUsersMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentMapAsJson(controllers.GetUsersAsMap(h.config.GetUsers()), nil), shouldLog)
		return
	}

	p, ok, shouldLog = delServerLogMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		h.writeResponse(w, controllers.DelLog(h.config, p["log"], h.logger.LogFileName(), urlRequestParts.Query), shouldLog)
		return
	}
	_, ok, shouldLog = getServerLogMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		// Panic Check ????
		ofs := urlRequestParts.AsAdmin().GetOptionalQuery("offset", "0")
		h.writeResponse(w, controllers.GetLog(h.config, h.logger.LogFileName(), ofs), shouldLog)
		return
	}
	_, ok, shouldLog = getReloadConfigMatch.Match(requestUrlparts, r.Method, requestInfo)
	if ok {
		configErrors := config.NewConfigErrorData()
		cfg := config.NewConfigData(h.config.ConfigName, h.config.ModuleName, h.config.Debugging, false, h.config.IsVerbose, configErrors)
		if configErrors.ErrorCount() == 0 {
			h.config = cfg
			h.Log(fmt.Sprintf("Config: %s file reload on demand!", h.config.ConfigName))
			h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentWithCauseAsJson("Config Reloaded", nil), shouldLog)
		} else {
			h.writeErrorResponse(w, "Config: Failed to re-load", http.StatusInternalServerError, fmt.Sprintf("Config Reload Failed with %d errors", configErrors.ErrorCount()))
		}
		return
	}
	h.writeErrorResponse(w, "Resource not found", http.StatusNotFound, fmt.Sprintf("Req:  %s:%s%s", r.Method, urlPath, urlRequestParts.QueryAsString()))
}

func (p *ServerHandler) writeErrorResponse(w http.ResponseWriter, cause string, status int, log string) {
	m := config.NewServerError(cause, status, log)
	p.writeResponse(w, controllers.NewResponseData(m.Status()).WithContentMapAsJson(m.Map(), nil), true)
}

func (p *ServerHandler) writeResponse(w http.ResponseWriter, resp *controllers.ResponseData, shouldLog bool) {
	contentType := config.LookupContentType(resp.MimeType)
	if resp.GetHasErrors() {
		p.Log(fmt.Sprintf("Resp: Error: Status:%d: '%s'", resp.Status, resp.ContentLimit(200)))
	} else {
		if shouldLog {
			if resp.LogContent() {
				p.Log(fmt.Sprintf("Resp: Status:%d Len:%d Type:%s Content:%s", resp.Status, resp.ContentLength(), contentType, resp.Content()))
			} else {
				p.Log(fmt.Sprintf("Resp: Status:%d Len:%d Type:%s", resp.Status, resp.ContentLength(), contentType))
			}
		}
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Server", p.config.GetServerName())
	w.WriteHeader(resp.Status)
	w.Write(resp.Content())
}

type WebAppServer struct {
	Handler  *ServerHandler
	Server   *http.Server
	ExitCode int
}

func NewWebAppServer(configData *config.ConfigData, actionQueue chan *ActionEvent, lrm *runCommand.LongRunningManager, logger logging.Logger) (*WebAppServer, error) {
	handler := NewServerHandler(configData, actionQueue, lrm, logger, time.Now())
	return &WebAppServer{
		Handler: handler,
		Server: &http.Server{
			Addr:    configData.GetPortString(),
			Handler: handler,
		},
		ExitCode: 0,
	}, nil
}

func (p *WebAppServer) Log(s string) {
	p.Handler.Log(s)
}

func (p *WebAppServer) Close(rc int) int {
	p.ExitCode = rc
	p.Handler.close()
	p.Server.Shutdown(context.TODO())
	return rc
}

func (p *WebAppServer) Start() int {
	p.Log(fmt.Sprintf("Server Config     :%s.", p.Handler.config.ConfigName))
	if p.Handler.config.IsVerbose {
		s, _ := p.Handler.config.String()
		p.Log(s)
	}
	p.Log(fmt.Sprintf("Server Started    :%s.", p.Handler.GetUpSince().Format(time.ANSIC)))
	if p.Handler.logger.IsOpen() {
		p.Log(fmt.Sprintf("Server Log        :%s.", p.Handler.config.GetPathForDisplay(p.Handler.config.ConfigFileData.LogData.Path)))
	} else {
		p.Log("Server Log        :Is not Open. All logging is to the console")
	}
	p.Log(fmt.Sprintf("Server Port       %s.", p.Handler.config.GetPortString()))
	p.Log(fmt.Sprintf("Server Root       :%s.", p.Handler.config.GetPathForDisplay(p.Handler.config.CurrentPath)))
	p.Log(fmt.Sprintf("Server Data Root  :%s.", p.Handler.config.GetPathForDisplay(p.Handler.config.GetServerDataRoot())))
	if p.Handler.config.HasStaticWebData {
		for n, v := range p.Handler.config.ConfigFileData.StaticWebData.Paths {
			p.Log(fmt.Sprintf("Web Server Path   :%s --> %s.", n, p.Handler.config.GetPathForDisplay(v)))
		}
	} else {
		p.Log("Static Data       :Undefined. Add StaticWebData.Home to config")
	}
	if p.Handler.config.IsTemplating {
		p.Log(fmt.Sprintf("Server Templating :%s.", p.Handler.config.GetStaticWebData().TemplateStaticFiles.DataFile))
	} else {
		p.Log("Server Templating :OFF.")
	}
	p.Log(fmt.Sprintf("Exec Files        :%s", p.Handler.config.GetPathForDisplay(p.Handler.config.GetExecPath())))
	for _, un := range p.Handler.config.GetUserNamesList() {
		p.Log(fmt.Sprintf("Server User Root  :%s --> %s", un, p.Handler.config.GetPathForDisplay(p.Handler.config.GetUserRoot(un))))
	}
	p.Log(fmt.Sprintf("User Properties   :%s.", p.Handler.config.GetPathForDisplay(p.Handler.config.UserProps.Details())))

	err := p.Server.ListenAndServe()
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			p.Log("Server Shutdown Clean")
			return p.Close(p.ExitCode)
		}
		p.Log(fmt.Sprintf("Server Error      :%s.", err.Error()))
		if strings.Contains(err.Error(), "address already in use") {
			return p.Close(10)
		}
	}
	return p.Close(1)
}

func (p *WebAppServer) String() string {
	cAsString, err := p.Handler.config.String()
	if err != nil {
		return fmt.Sprintf("Server Error: In 'Handler.config.String()': %s", err.Error())
	}
	return cAsString
}
