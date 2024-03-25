package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"stuartdd.com/config"
	"stuartdd.com/controllers"
	"stuartdd.com/logging"
)

type ActionId int

const (
	Exit ActionId = iota
	Ignore
)

var getFaviconMatch = NewUrlRequestMatcher("/favicon.ico", "GET")
var getExitMatch = NewUrlRequestMatcher("/exit", "GET")
var getPingMatch = NewUrlRequestMatcher("/ping", "GET")
var getReloadConfigMatch = NewUrlRequestMatcher("/reload/config", "GET")
var getServerStatusMatch = NewUrlRequestMatcher("/status", "GET")

var getFileUserLocPathMatch = NewUrlRequestMatcher("/files/user/*/loc/*/path/*", "GET")
var getFileUserLocMatch = NewUrlRequestMatcher("/files/user/*/loc/*", "GET")
var getPathsUserLocMatch = NewUrlRequestMatcher("/paths/user/*/loc/*", "GET")

var getFileUserLocTreeMatch = NewUrlRequestMatcher("/files/user/*/loc/*/tree", "GET")
var getFileUserLocNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/name/*", "GET")
var postFileUserLocNameMatch = NewUrlRequestMatcher("/files/user/*/loc/*/name/*", "POST")
var execUserCmdMatch = NewUrlRequestMatcher("/exec/user/*/exec/*", "GET")

type ServerHandler struct {
	config      *config.ConfigData
	actionQueue chan ActionId
	logger      logging.Logger
}

func NewServerHandler(configData *config.ConfigData, actionQueue chan ActionId, logger logging.Logger) *ServerHandler {
	return &ServerHandler{
		config:      configData,
		actionQueue: actionQueue,
		logger:      logger,
	}
}

func (h *ServerHandler) Log(s string) {
	h.logger.Log(s)
}

func (h *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.config.IsTimeToReloadConfig() {
		ts := time.Now().UnixMicro()
		cfg, errorList := config.NewConfigData(h.config.ConfigName)
		if errorList.Len() == 0 {
			h.config = cfg
			h.Log(fmt.Sprintf("Config: %s file reload OK! (%d micro seconds)", h.config.ConfigName, (time.Now().UnixMicro() - ts)))
		} else {
			h.config.ResetTimeToReloadConfig()
			h.Log(fmt.Sprintf("Config: %s Failed to load\n%s", h.config.ConfigName, errorList))
		}
	}

	h.Log(fmt.Sprintf("Req:  %s", r.RequestURI))

	urlParts := controllers.NewUrlRequestParts(h.config).WithQuery(r.URL.Query()).WithHeader(r.Header)

	requestUriparts := strings.Split(strings.TrimSpace(r.URL.Path), "/")
	if requestUriparts[0] == "" {
		requestUriparts = requestUriparts[1:]
	}
	isAbsolutePath := strings.HasPrefix(r.URL.Path, "/")

	p, ok := execUserCmdMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.writeResponse(w, controllers.NewExecHandler(urlParts.WithParameters(p), h.config, nil).Submit())
		return
	}
	p, ok = getFileUserLocPathMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.writeResponse(w, controllers.NewDirHandler(urlParts.WithParameters(p), h.config, true).Submit())
		return
	}
	p, ok = getFileUserLocMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.writeResponse(w, controllers.NewDirHandler(urlParts.WithParameters(p), h.config, true).Submit())
		return
	}
	p, ok = getPathsUserLocMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.writeResponse(w, controllers.NewDirHandler(urlParts.WithParameters(p), h.config, false).Submit())
		return
	}
	p, ok = getFileUserLocTreeMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.writeResponse(w, controllers.NewTreeHandler(urlParts.WithParameters(p), h.config).Submit())
		return
	}

	p, ok = getFileUserLocNameMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.writeResponse(w, controllers.NewReadFileHandler(urlParts.WithParameters(p), h.config).Submit())
		return
	}
	p, ok = postFileUserLocNameMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.writeResponse(w, controllers.NewPostFileHandler(urlParts.WithParameters(p), h.config, r).Submit())
		return
	}
	_, ok = getExitMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.actionQueue <- Exit
		h.writeResponse(w, controllers.NewResponseData(http.StatusAccepted).WithContentReasonAsJson("Server Stopped", false))
		return
	}

	_, ok = getPingMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentReasonAsJson("Ping", false))
		return
	}
	_, ok = getServerStatusMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentReasonAsJson(fmt.Sprintf("Status (%v)", h.config.GetTimeToReloadConfig()), false))
		return
	}

	_, ok = getFaviconMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		h.writeResponse(w, controllers.GetFaveIcon(h.config))
		return
	}
	_, ok = getReloadConfigMatch.Match(requestUriparts, isAbsolutePath, r.Method)
	if ok {
		cfg, errorList := config.NewConfigData(h.config.ConfigName)
		if errorList.Len() == 0 {
			h.config = cfg
			h.Log(fmt.Sprintf("Config: %s file reload on demand!", h.config.ConfigName))
			h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentReasonAsJson("Config Reloaded", false))

		} else {
			h.Log(fmt.Sprintf("Config: %s Failed to load\n%s", h.config.ConfigName, errorList))
			h.writeResponse(w, controllers.NewResponseData(http.StatusExpectationFailed).WithContentReasonAsJson("Config Reload Failed", true))
		}
		return
	}

	h.writeResponse(w, controllers.NewResponseData(http.StatusNotFound).WithContentReasonAsJson("Resource not found", true))
}

func (p *ServerHandler) writeResponse(w http.ResponseWriter, resp *controllers.ResponseData) {
	contentType := config.LookupContentType(resp.MimeType)
	if resp.GetShouldLog() {
		p.Log(fmt.Sprintf("Error: Status:%d: '%s'", resp.Status, resp.ContentLimit(150)))
	} else {
		p.Log(fmt.Sprintf("Resp: Status:%d Len:%d Type:%s", resp.Status, resp.ContentLength(), contentType))
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
	Handler *ServerHandler
}

func NewWebAppServer(configData *config.ConfigData, actionQueue chan ActionId, logger logging.Logger) *WebAppServer {
	return &WebAppServer{
		Handler: NewServerHandler(configData, actionQueue, logger),
	}
}

func (p *WebAppServer) Log(s string) {
	p.Handler.Log(s)
}

func (p *WebAppServer) Start() {
	p.Log("Server running.")
	p.Log(fmt.Sprintf("Server Port     %s.", p.Handler.config.GetPortString()))
	p.Log(fmt.Sprintf("Server Log      :%s.", p.Handler.config.GetLogDataPath()))
	p.Log(fmt.Sprintf("Server Path (wd):%s.", p.Handler.config.CurrentPath))
	p.Log(fmt.Sprintf("Server Data Root:%s.", p.Handler.config.GetServerDataRoot()))
	for _, un := range p.Handler.config.GetUserNamesList() {
		p.Log(fmt.Sprintf("Server User     :%s --> %s", un, p.Handler.config.GetUserRoot(un)))
	}
	log.Fatal(http.ListenAndServe(p.Handler.config.GetPortString(), p.Handler))
}

func (p *WebAppServer) String() string {
	cAsString, err := p.Handler.config.String()
	if err != nil {
		return fmt.Sprintf("Server Error: In 'Handler.config.String()': %s", err.Error())
	}
	return cAsString
}
