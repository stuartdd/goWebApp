package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"stuartdd.com/config"
	"stuartdd.com/controllers"
	"stuartdd.com/tools"
)

type ActionId int

const (
	Exit ActionId = iota
	Ignore
)

var getFaviconMatch = tools.NewUrlRequestParts("/favicon.ico").WithReqType("GET")
var getExitMatch = tools.NewUrlRequestParts("/exit").WithReqType("GET")
var getPingMatch = tools.NewUrlRequestParts("/ping").WithReqType("GET")

var getFileUserLocTreeMatch = tools.NewUrlRequestParts("/files/user/*/loc/*/tree").WithReqType("GET")
var getFileUserLocMatch = tools.NewUrlRequestParts("/files/user/*/loc/*").WithReqType("GET")
var getFileUserLocNameMatch = tools.NewUrlRequestParts("/files/user/*/loc/*/name/*").WithReqType("GET")
var postFileUserLocNameMatch = tools.NewUrlRequestParts("/files/user/*/loc/*/name/*").WithReqType("POST")
var execUserCmdMatch = tools.NewUrlRequestParts("/exec/user/*/exec/*").WithReqType("GET")

type ServerHandler struct {
	config      *config.ConfigData
	actionQueue chan ActionId
	logger      tools.Logger
}

func NewServerHandler(configData *config.ConfigData, actionQueue chan ActionId, logger tools.Logger) *ServerHandler {
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
	h.Log(fmt.Sprintf("Req:  %s", r.RequestURI))
	urlParts := tools.NewUrlRequestParts(r.RequestURI).WithReqType(r.Method).WithQuery(r.URL.Query()).WithHeader(r.Header)
	if urlParts.Match(getExitMatch) {
		h.actionQueue <- Exit
		h.writeResponse(w, controllers.NewResponseData(http.StatusAccepted).WithContentReasonAsJson("Server Stopped", false))
		return
	}
	if urlParts.Match(getPingMatch) {
		h.writeResponse(w, controllers.NewResponseData(http.StatusOK).WithContentReasonAsJson("Ping", false))
		return
	}
	if urlParts.Match(getFileUserLocNameMatch) {
		h.writeResponse(w, controllers.NewFileHandler(urlParts.UrlParamMap(getFileUserLocNameMatch), h.config).Submit())
		return
	}
	if urlParts.Match(postFileUserLocNameMatch) {
		h.writeResponse(w, controllers.NewFilePostHandler(urlParts.UrlParamMap(postFileUserLocNameMatch), h.config, r).Submit())
		return
	}
	if urlParts.Match(getFileUserLocMatch) {
		h.writeResponse(w, controllers.NewDirHandler(urlParts.UrlParamMap(getFileUserLocMatch), h.config).Submit())
		return
	}
	if urlParts.Match(getFileUserLocTreeMatch) {
		h.writeResponse(w, controllers.NewTreeHandler(urlParts.UrlParamMap(getFileUserLocTreeMatch), h.config).Submit())
		return
	}
	if urlParts.Match(getFaviconMatch) {
		h.writeResponse(w, controllers.GetFaveIcon(h.config))
		return
	}
	if urlParts.Match(execUserCmdMatch) {
		h.writeResponse(w, controllers.NewExecHandler(urlParts.UrlParamMap(execUserCmdMatch), h.config, nil).Submit())
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

func NewWebAppServer(configData *config.ConfigData, actionQueue chan ActionId, logger tools.Logger) *WebAppServer {
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
		p.Log(fmt.Sprintf("Server User     :%s --> %s", un, p.Handler.config.GetUserDataRoot(un)))
	}
	log.Fatal(http.ListenAndServe(p.Handler.config.GetPortString(), p.Handler))
}

func (p *WebAppServer) ToString() string {
	cAsString, err := p.Handler.config.ToString()
	if err != nil {
		return fmt.Sprintf("Server Error: In 'Handler.config.ToString()': %s", err.Error())
	}
	return cAsString
}
