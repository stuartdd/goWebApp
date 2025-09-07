package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/controllers"
	"github.com/stuartdd/goWebApp/logging"
	"github.com/stuartdd/goWebApp/runCommand"
)

const shouldLogTrue = true
const shouldNotLogFalse = false

type ActionId int

const (
	Exit ActionId = iota
	Ignore
)

type LoggableError interface {
	Error() string
	LogError() string
	Status() int
	Map() map[string]interface{}
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

var getFaviconMatch = NewUrlRequestMatcher("/favicon.ico", "GET", shouldLogTrue)
var getPingMatch = NewUrlRequestMatcher("/ping", "GET", shouldNotLogFalse)
var getIsUpMatch = NewUrlRequestMatcher("/isup", "GET", shouldNotLogFalse)

var getServerStatusMatch = NewUrlRequestMatcher("/server/status", "GET", shouldLogTrue)
var getReloadConfigMatch = NewUrlRequestMatcher("/server/config", "GET", shouldLogTrue)
var getServerTimeMatch = NewUrlRequestMatcher("/server/time", "GET", shouldNotLogFalse)
var getServerUsersMatch = NewUrlRequestMatcher("/server/users", "GET", shouldLogTrue)
var getServerRestartMatch = NewUrlRequestMatcher("/server/restart", "GET", shouldLogTrue)
var getServerExitMatch = NewUrlRequestMatcher("/server/exit", "GET", shouldLogTrue)
var getServerLogMatch = NewUrlRequestMatcher("/server/log", "GET", shouldNotLogFalse)

// Get File asAdmin user. Location defined in admin user.
var getFileLocNameMatch = NewUrlRequestMatcher("/files/loc/*/name/*", "GET", shouldLogTrue)

// Exec a script via an ID in config:"Exec" section.
// Script must be in  config:"ExecPath":
// User will be "admin"
var getExecMatch = NewUrlRequestMatcher("/exec/*", "GET", shouldLogTrue)

var getFileUserLocPathMatch = NewUrlRequestMatcher("/files/user/*/loc/*/path/*", "GET", shouldLogTrue)
var getFileUserLocPathNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/path/*/name/*", "GET", shouldLogTrue)
var getFileUserLocMatch = NewUrlRequestMatcher("/files/user/*/loc/*", "GET", shouldLogTrue)
var getFileUserLocTreeMatch = NewUrlRequestMatcher("/files/user/*/loc/*/tree", "GET", shouldLogTrue)
var getFileUserLocNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/name/*", "GET", shouldLogTrue)
var getPathsUserLocMatch = NewUrlRequestMatcher("/paths/user/*/loc/*", "GET", shouldLogTrue)

var delFileUserLocNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/name/*", "DELETE", shouldLogTrue)
var postFileUserLocNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/name/*", "POST", shouldLogTrue)
var postFileUserLocPathNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/path/*/name/*", "POST", shouldLogTrue)

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

func (p *ServerHandler) HasStaticData() bool {
	return p.config.HasStaticData()
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

func (h *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.config.IsTimeToReloadConfig() {
		ts := time.Now().UnixMicro()
		configErrors := config.NewConfigErrorData()
		cfg := config.NewConfigData(h.config.ConfigName, h.config.ModuleName, h.config.Debugging, false, h.config.IsVerbose, configErrors)
		if configErrors.ErrorCount() == 0 {
			h.config = cfg
			h.Log(fmt.Sprintf("Config: %s file reload OK! (%d micro seconds)", h.config.ConfigName, (time.Now().UnixMicro() - ts)))
		} else {
			h.config.ResetTimeToReloadConfig()
			h.Log(fmt.Sprintf("Config: %s Failed to load\n%s", h.config.ConfigName, configErrors))
		}
	}
	urlPath := strings.TrimSpace(r.URL.Path)
	logFunc := h.logger.Log
	verboseFunc := h.logger.VerboseFunction()
	requestInfo := NewRequestInfo(r.Method, urlPath, r.URL.RawQuery, logFunc, verboseFunc)

	defer func() {
		var le LoggableError
		if rec := recover(); rec != nil {
			switch x := rec.(type) {
			case LoggableError:
				le = x
			case string:
				le = config.NewConfigErrorFromString(x, 404)
			case error:
				le = config.NewConfigErrorFromString(x.Error(), 404)
			default:
				// Fallback err (per specs, error strings should be lowercase w/o punctuation
				le = config.NewConfigErrorFromString(fmt.Sprintf("%v", rec), 404)
			}
			logFunc(le.LogError())
			h.writeResponse(w, controllers.NewResponseData(le.Status()).SetHasErrors(true).WithContentMapAsJson(le.Map()), true)
		}
	}()

	staticFileData := h.config.GetStaticData()
	requestData := controllers.NewUrlRequestParts(h.config).WithQuery(r.URL.Query()).WithHeader(r.Header)

	var isAbsolutePath bool
	var requestUrlparts []string
	if urlPath == "/" {
		if staticFileData.HasStaticData() {
			requestUrlparts = []string{"static", staticFileData.HomePage}
		}
	} else {
		requestUrlparts = strings.Split(urlPath, "/")
		if requestUrlparts[0] == "" {
			isAbsolutePath = true
			requestUrlparts = requestUrlparts[1:]
		} else {
			isAbsolutePath = false
		}
	}

	if requestUrlparts[0] == "static" {
		// Panic Check Done
		requestInfo.Log(shouldNotLogFalse)
		h.writeResponse(w, controllers.NewStaticFileHandler(requestUrlparts[1:], requestData, verboseFunc).Submit(), shouldLogTrue)
		return
	}

	_, ok, shouldLog := getPingMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentWithCauseAsJson("Ping"), shouldLog)
		return
	}
	_, ok, shouldLog = getIsUpMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentWithCauseAsJson("ServerIsUp"), shouldLog)
		return
	}
	p, ok, shouldLog := getFileUserLocPathMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		if shouldLog {
			h.logger.Log(fmt.Sprintf("Req:  %s:%s %s", r.Method, urlPath, r.URL.RawQuery))
		}
		h.writeResponse(w, controllers.NewDirHandler(requestData.WithParameters(p), h.config, true, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = getFileUserLocMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		if shouldLog {
			h.logger.Log(fmt.Sprintf("Req:  %s:%s %s", r.Method, urlPath, r.URL.RawQuery))
		}
		h.writeResponse(w, controllers.NewDirHandler(requestData.WithParameters(p), h.config, true, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = getPathsUserLocMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewDirHandler(requestData.WithParameters(p), h.config, false, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = getFileUserLocTreeMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewTreeHandler(requestData.WithParameters(p), h.config).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = delFileUserLocNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewDeleteFileHandler(requestData.WithParameters(p), h.config, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = getFileUserLocPathNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewReadFileHandler(requestData.WithParameters(p), h.config, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = getFileUserLocNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewReadFileHandler(requestData.WithParameters(p), h.config, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = getFileLocNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewReadFileHandler(requestData.WithParameters(p).AsAdmin(), h.config, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = getExecMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check ????
		h.writeResponse(w, controllers.NewExecHandler(requestData.WithParameters(p).AsAdmin(), h.config, nil, logFunc, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = postFileUserLocPathNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewPostFileHandler(requestData.WithParameters(p), h.config, r, false, verboseFunc).Submit(), shouldLog)
		return
	}
	p, ok, shouldLog = postFileUserLocNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewPostFileHandler(requestData.WithParameters(p), h.config, r, false, verboseFunc).Submit(), shouldLog)
		return
	}
	_, ok, shouldLog = getServerRestartMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		a := NewActionEvent(Exit, requestData.GetOptionalQuery("rc", "23"), 23, "Restart Requested")
		h.actionQueue <- a
		h.writeResponse(w, controllers.NewResponseData(http.StatusAccepted).WithContentMapAsJson(map[string]interface{}{"Status": "RESTARTED"}), shouldLog)
		return
	}
	_, ok, shouldLog = getServerExitMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		a := NewActionEvent(Exit, requestData.GetOptionalQuery("rc", "11"), 11, "Exit Requested")
		h.actionQueue <- a
		h.writeResponse(w, controllers.NewResponseData(http.StatusAccepted).WithContentWithCauseAsJson(a.String()), shouldLog)
		return
	}
	// Panic Check Done
	_, ok, shouldLog = getServerStatusMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		if h.longRunning.IsEnabled() {
			h.longRunning.Update()
		}
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentBytes(controllers.GetServerStatusAsJson(h.config, h.logger.LogFileName(), h.GetUpSince(), h.longRunning.ToJson())), shouldLog)
		return
	}

	_, ok, shouldLog = getServerTimeMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentMapAsJson(controllers.GetTimeAsMap()), shouldLog)
		return
	}

	_, ok, shouldLog = getServerUsersMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentMapAsJson(controllers.GetUsersAsMap(h.config.GetUsers())), shouldLog)
		return
	}

	_, ok, shouldLog = getServerLogMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check ????
		ofs := requestData.AsAdmin().GetOptionalQuery("offset", "0")
		h.writeResponse(w, controllers.GetLog(h.config, ofs), shouldLog)
		return
	}
	_, ok, shouldLog = getFaviconMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		// Panic Check Done
		h.writeResponse(w, controllers.GetFaveIcon(h.config), shouldLog)
		return
	}

	_, ok, shouldLog = getReloadConfigMatch.Match(requestUrlparts, isAbsolutePath, r.Method, requestInfo)
	if ok {
		configErrors := config.NewConfigErrorData()
		cfg := config.NewConfigData(h.config.ConfigName, h.config.ModuleName, h.config.Debugging, false, h.config.IsVerbose, configErrors)
		if configErrors.ErrorCount() == 0 {
			h.config = cfg
			h.Log(fmt.Sprintf("Config: %s file reload on demand!", h.config.ConfigName))
			h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentWithCauseAsJson("Config Reloaded"), shouldLog)
		} else {
			panic(config.NewConfigError("Config: Failed to re-load", http.StatusInternalServerError, fmt.Sprintf("Config Reload Failed with %d errors", configErrors.ErrorCount())))
		}
		return
	}
	if staticFileData.HasStaticData() && h.config.GetStaticData().CheckFileExists(urlPath) {
		requestInfo.Log(shouldLogTrue)
		h.writeResponse(w, controllers.NewStaticFileHandler(requestUrlparts, requestData, verboseFunc).Submit(), shouldLogTrue)
		return
	}
	panic(config.NewConfigError("Resource not found", http.StatusNotFound, fmt.Sprintf("Req:  %s:%s%s", r.Method, urlPath, requestData.QueryAsString())))
}

func (p *ServerHandler) writeResponse(w http.ResponseWriter, resp *controllers.ResponseData, shouldLog bool) {
	contentType := config.LookupContentType(resp.MimeType)
	if resp.GetHasErrors() {
		p.Log(fmt.Sprintf("Resp: Error: Status:%d: '%s'", resp.Status, resp.ContentLimit(200)))
	} else {
		if shouldLog {
			p.Log(fmt.Sprintf("Resp: Status:%d Len:%d Type:%s", resp.Status, resp.ContentLength(), contentType))
		}
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	bufrw.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\n", resp.Status, http.StatusText(resp.Status)))
	bufrw.WriteString(fmt.Sprintf("Content-Length: %d\n", resp.ContentLength()))
	bufrw.WriteString(fmt.Sprintf("Content-Type: %s\n", contentType))
	bufrw.WriteString(fmt.Sprintf("Date: %s\n", timeAsString()))
	bufrw.WriteString(fmt.Sprintf("Server: %s\n", p.config.GetServerName()))
	bufrw.WriteString("\n")
	bufrw.Write(resp.Content())
	bufrw.Flush()
}

func timeAsString() string {
	t := time.Now().UTC()
	return t.Format("Mon, 02 Jan 2006 15:04:05 GMT")
}

type WebAppServer struct {
	Handler     *ServerHandler
	LongRunning int
}

func NewWebAppServer(configData *config.ConfigData, actionQueue chan *ActionEvent, lrm *runCommand.LongRunningManager, logger logging.Logger) (*WebAppServer, error) {
	return &WebAppServer{
		Handler: NewServerHandler(configData, actionQueue, lrm, logger, time.Now()),
	}, nil
}

func (p *WebAppServer) Log(s string) {
	p.Handler.Log(s)
}

func (p *WebAppServer) Close(rc int) int {
	p.Handler.close()
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
		p.Log(fmt.Sprintf("Server Log        :%s.", p.Handler.config.GetLogDataPath()))
	} else {
		p.Log("Server Log        :Is not Open. All logging is to the console")
	}
	p.Log(fmt.Sprintf("Server Port       %s.", p.Handler.config.GetPortString()))
	p.Log(fmt.Sprintf("Server Path (wd)  :%s.", p.Handler.config.CurrentPath))
	p.Log(fmt.Sprintf("Server Data Root  :%s.", p.Handler.config.GetServerDataRoot()))
	if p.Handler.config.HasStaticData() {
		p.Log(fmt.Sprintf("Static Data Root  :%s.", p.Handler.config.GetServerStaticRoot()))
	} else {
		p.Log("Static Data       :Undefined. Add StaticData.Home to config")
	}
	if p.Handler.config.IsTemplating() {
		p.Log(fmt.Sprintf("Server Templating :%s.", p.Handler.config.GetTemplateData().String()))
	} else {
		p.Log("Server Templating :OFF.")
	}
	p.Log(fmt.Sprintf("Exec Manager      :%s", p.Handler.longRunning.String()))
	for _, un := range p.Handler.config.GetUserNamesList() {
		p.Log(fmt.Sprintf("Server User       :%s --> %s", un, p.Handler.config.GetUserRoot(un)))
	}

	err := http.ListenAndServe(p.Handler.config.GetPortString(), p.Handler)
	if err != nil {
		p.Log(fmt.Sprintf("Server Error      :%s.", err.Error()))
		if strings.Contains(err.Error(), "address already in use") {
			return p.Close(10)
		}
		return p.Close(1)
	}
	return p.Close(0)
}

func (p *WebAppServer) String() string {
	cAsString, err := p.Handler.config.String()
	if err != nil {
		return fmt.Sprintf("Server Error: In 'Handler.config.String()': %s", err.Error())
	}
	return cAsString
}
