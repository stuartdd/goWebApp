package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stuartdd/goWebApp/config"
	"github.com/stuartdd/goWebApp/controllers"
	"github.com/stuartdd/goWebApp/logging"
	"github.com/stuartdd/goWebApp/runCommand"
)

const shouldLog = true
const shouldNotLog = false

type ActionId int

const (
	Exit ActionId = iota
	Ignore
)

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

var getFaviconMatch = NewUrlRequestMatcher("/favicon.ico", "GET", shouldLog)
var getPingMatch = NewUrlRequestMatcher("/ping", "GET", shouldNotLog)
var getIsUpMatch = NewUrlRequestMatcher("/isup", "GET", shouldNotLog)

var getServerStatusMatch = NewUrlRequestMatcher("/server/status", "GET", shouldLog)
var getReloadConfigMatch = NewUrlRequestMatcher("/server/config", "GET", shouldLog)
var getServerTimeMatch = NewUrlRequestMatcher("/server/time", "GET", shouldNotLog)
var getServerUsersMatch = NewUrlRequestMatcher("/server/users", "GET", shouldLog)
var getServerRestartMatch = NewUrlRequestMatcher("/server/restart", "GET", shouldLog)
var getServerExitMatch = NewUrlRequestMatcher("/server/exit", "GET", shouldLog)
var getServerLogMatch = NewUrlRequestMatcher("/server/log", "GET", shouldNotLog)

// Get File asAdmin user. Location defined in admin user.
var getFileLocNameMatch = NewUrlRequestMatcher("/files/loc/*/name/*", "GET", shouldLog)

// Exec a script via an ID in config:"Exec" section.
// Script must be in  config:"ExecPath":
// User will be "admin"
var getExecMatch = NewUrlRequestMatcher("/exec/*", "GET", shouldLog)

var getFileUserLocPathMatch = NewUrlRequestMatcher("/files/user/*/loc/*/path/*", "GET", shouldLog)
var getFileUserLocPathNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/path/*/name/*", "GET", shouldLog)
var getFileUserLocMatch = NewUrlRequestMatcher("/files/user/*/loc/*", "GET", shouldLog)
var getFileUserLocTreeMatch = NewUrlRequestMatcher("/files/user/*/loc/*/tree", "GET", shouldLog)
var getFileUserLocNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/name/*", "GET", shouldLog)
var getTestUserLocNameMatch = NewUrlRequestMatcher("/test/user/*/loc/*/name/*", "GET", shouldNotLog)
var getPathsUserLocMatch = NewUrlRequestMatcher("/paths/user/*/loc/*", "GET", shouldLog)

var delFileUserLocNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/name/*", "DELETE", shouldLog)
var postFileUserLocNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/name/*", "POST", shouldLog)
var postFileUserLocPathNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/path/*/name/*", "POST", shouldLog)
var postFileUserLocNameLogMatch = NewUrlRequestMatcher("/log/user/*/name/*/action/*", "POST", shouldNotLog)

type ServerHandler struct {
	config      *config.ConfigData
	actionQueue chan *ActionEvent
	logger      logging.Logger
	upSince     time.Time
	longRunning *LongRunningManager
}

func NewServerHandler(configData *config.ConfigData, actionQueue chan *ActionEvent, lrm *LongRunningManager, logger logging.Logger, upSince time.Time) *ServerHandler {
	if lrm == nil {
		lrm = NewLongRunningManagerDisabled()
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
		cfg, errorList := config.NewConfigData(h.config.ConfigName, h.config.ModuleName, h.config.Debugging, false, false, h.config.IsVerbose)
		if errorList.ErrorCount() == 0 {
			h.config = cfg
			h.Log(fmt.Sprintf("Config: %s file reload OK! (%d micro seconds)", h.config.ConfigName, (time.Now().UnixMicro() - ts)))
		} else {
			h.config.ResetTimeToReloadConfig()
			h.Log(fmt.Sprintf("Config: %s Failed to load\n%s", h.config.ConfigName, errorList))
		}
	}
	logFunc := h.logger.Log
	verboseFunc := h.logger.GetVerbose()

	urlPath := strings.TrimSpace(r.URL.Path)

	defer func() {
		if r := recover(); r != nil {
			pm := config.NewPanicMessageFromRecover(r, 400)
			logFunc("Panic:" + pm.String())
			h.writeResponse(w, controllers.NewResponseData(pm.Status).WithContentReasonAsJson(pm.Reason, true), shouldLog)
		}
	}()

	staticFileData := h.config.GetStaticData()

	requestData := controllers.NewUrlRequestParts(h.config).WithQuery(r.URL.Query()).WithHeader(r.Header)

	var isAbsolutePath bool
	var requestUrlparts []string
	if urlPath == "/" {
		if staticFileData.HasStaticData() {
			requestUrlparts = []string{"static", staticFileData.Home}
		}
	} else {
		requestUrlparts = strings.Split(urlPath, "/")
		if requestUrlparts[0] == "" {
			isAbsolutePath = true
			requestUrlparts = requestUrlparts[1:]
		} else {
			isAbsolutePath = false
		}
		if requestUrlparts[0] == "" {
			requestUrlparts = requestUrlparts[1:]
		}
	}

	if len(requestUrlparts) > 1 {
		if requestUrlparts[0] == "static" {
			h.writeResponse(w, controllers.NewStaticFileHandler(requestUrlparts[1:], requestData, h.config.IsVerbose, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog := getFileUserLocPathMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewDirHandler(requestData.WithParameters(p), h.config, true, h.config.IsVerbose, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = getFileUserLocMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewDirHandler(requestData.WithParameters(p), h.config, true, h.config.IsVerbose, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = getPathsUserLocMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewDirHandler(requestData.WithParameters(p), h.config, false, h.config.IsVerbose, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = getFileUserLocTreeMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewTreeHandler(requestData.WithParameters(p), h.config).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = delFileUserLocNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewDeleteFileHandler(requestData.WithParameters(p), h.config, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = getFileUserLocPathNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewReadFileHandler(requestData.WithParameters(p), h.config, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = getFileUserLocNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewReadFileHandler(requestData.WithParameters(p), h.config, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = getTestUserLocNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewReadFileHandler(requestData.WithParameters(p), h.config, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = getFileLocNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewReadFileHandler(requestData.WithParameters(p).AsAdmin(), h.config, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = getExecMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewExecHandler(requestData.WithParameters(p).AsAdmin(), h.config, nil, logFunc, h.config.IsVerbose, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = postFileUserLocNameLogMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewPostFileHandler(requestData.WithParameters(p).WithParam("loc", "logs"), h.config, r, h.config.IsVerbose, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = postFileUserLocPathNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewPostFileHandler(requestData.WithParameters(p), h.config, r, h.config.IsVerbose, verboseFunc).Submit(), shouldLog)
			return
		}
		p, ok, shouldLog = postFileUserLocNameMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
		if ok {
			h.writeResponse(w, controllers.NewPostFileHandler(requestData.WithParameters(p), h.config, r, h.config.IsVerbose, verboseFunc).Submit(), shouldLog)
			return
		}
	}

	_, ok, shouldLog := getServerRestartMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
	if ok {
		a := NewActionEvent(Exit, requestData.GetOptionalQuery("rc", "23"), 23, "Restart Requested")
		h.actionQueue <- a
		h.writeResponse(w, controllers.NewResponseData(http.StatusAccepted).WithContentMapJson(map[string]interface{}{"Status": "RESTARTED"}), shouldLog)
		return
	}
	_, ok, shouldLog = getServerExitMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
	if ok {
		a := NewActionEvent(Exit, requestData.GetOptionalQuery("rc", "11"), 11, "Exit Requested")
		h.actionQueue <- a
		h.writeResponse(w, controllers.NewResponseData(http.StatusAccepted).WithContentReasonAsJson(a.String(), false), shouldLog)
		return
	}
	_, ok, shouldLog = getPingMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
	if ok {
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentReasonAsJson("Ping", false), shouldLog)
		return
	}

	_, ok, shouldLog = getIsUpMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
	if ok {
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentReasonAsJson("ServerIsUp", false), shouldLog)
		return
	}

	_, ok, shouldLog = getServerStatusMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
	if ok {
		if h.longRunning.enabled {
			for _, v := range h.longRunning.longRunningProcess {
				v.PID = 0
				v.PS = ""
				runCommand.ForEachSystemProcess(func(cmd string, p int) (bool, error) {
					if strings.Contains(cmd, filepath.Join(h.longRunning.path, v.ID)) {
						pos := strings.Index(cmd, h.longRunning.path)
						v.PID = p
						v.PS = fmt.Sprintf("%s %s", strings.TrimSpace(cmd[0:pos]), v.ID)
						return true, nil
					}
					return false, nil
				})
			}
		}
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentBytes(controllers.GetServerStatusAsJson(h.config, h.logger.LogFileName(), h.GetUpSince(), h.longRunning.ToJson(), h.logger.Log)), shouldLog)
		return
	}

	_, ok, shouldLog = getServerTimeMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
	if ok {
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentMapJson(controllers.GetTimeAsMap()), shouldLog)
		return
	}

	_, ok, shouldLog = getServerUsersMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
	if ok {
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentMapJson(controllers.GetUsersAsMap(h.config.GetUsers())), shouldLog)
		return
	}

	_, ok, shouldLog = getServerLogMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
	if ok {
		ofs, err := requestData.AsAdmin().GetOptionalQueryAsInt("offset", 0)
		if err != nil {
			h.writeResponse(w, controllers.NewResponseData(http.StatusBadRequest).WithContentReasonAsJson(err.Error(), true), true)
			return
		}
		h.writeResponse(w, controllers.GetLog(h.config, ofs), shouldLog)
		return
	}
	_, ok, shouldLog = getFaviconMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
	if ok {
		h.writeResponse(w, controllers.GetFaveIcon(h.config), shouldLog)
		return
	}

	_, ok, shouldLog = getReloadConfigMatch.Match(requestUrlparts, isAbsolutePath, r.Method, logFunc)
	if ok {
		cfg, errorList := config.NewConfigData(h.config.ConfigName, h.config.ModuleName, h.config.Debugging, false, false, h.config.IsVerbose)
		if errorList.ErrorCount() == 0 {
			h.config = cfg
			h.Log(fmt.Sprintf("Config: %s file reload on demand!", h.config.ConfigName))
			h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentReasonAsJson("Config Reloaded", false), shouldLog)

		} else {
			h.Log(fmt.Sprintf("Config: %s Failed to load\n%s", h.config.ConfigName, errorList))
			h.writeResponse(w, controllers.NewResponseData(http.StatusExpectationFailed).WithContentReasonAsJson("Config Reload Failed", true), shouldLog)
		}
		return
	}
	if staticFileData.HasStaticData() && h.config.GetStaticData().CheckFileExists(urlPath) {
		h.writeResponse(w, controllers.NewStaticFileHandler(requestUrlparts, requestData, h.config.IsVerbose, verboseFunc).Submit(), shouldLog)
		return
	}
	logFunc(fmt.Sprintf("Req:  %s:%s%s", r.Method, urlPath, requestData.QueryAsString()))
	h.writeResponse(w, controllers.NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Resource not found", true), shouldLog)
}

func (p *ServerHandler) writeResponse(w http.ResponseWriter, resp *controllers.ResponseData, shouldLog bool) {
	contentType := config.LookupContentType(resp.MimeType)
	if resp.GetHasErrors() {
		p.Log(fmt.Sprintf("Error: Status:%d: '%s'", resp.Status, resp.ContentLimit(150)))
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
	formattedDate := t.Format("Mon, 02 Jan 2006 15:04:05 GMT")
	return formattedDate
	// Sun, 25 Feb 2024 13:13:09 GMT
}

type WebAppServer struct {
	Handler     *ServerHandler
	LongRunning int
}

func NewWebAppServer(configData *config.ConfigData, actionQueue chan *ActionEvent, lrm *LongRunningManager, logger logging.Logger) (*WebAppServer, error) {
	if lrm != nil && lrm.enabled {
		for n, v := range configData.GetExecData() {
			if v.Detached {
				err := lrm.AddLongRunningProcessData(n, v.Cmd, v.CanStop)
				if err != nil {
					return nil, err
				}
			}
		}
	}
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
