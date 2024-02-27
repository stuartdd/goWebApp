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

var getExitMatch = tools.NewUrlRequestParts("/exit").WithReqType("GET")
var getTapMatch = tools.NewUrlRequestParts("/tap").WithReqType("GET")
var getFileUserLocNameMatch = tools.NewUrlRequestParts("/files/user/*/loc/*/name/*").WithReqType("GET")

type ServerHandler struct {
	config      *config.ConfigData
	actionQueue chan ActionId
}

func NewServerHandler(configData *config.ConfigData, actionQueue chan ActionId) *ServerHandler {
	return &ServerHandler{
		config:      configData,
		actionQueue: actionQueue,
	}
}

func (h *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlParts := tools.NewUrlRequestParts(r.RequestURI).WithReqType(r.Method).WithQuery(r.URL.Query()).WithHeader(r.Header)
	if urlParts.Match(getExitMatch) {
		h.actionQueue <- Exit
		h.writeResponse(w, controllers.NewErrorResponse(http.StatusAccepted, http.StatusText(http.StatusAccepted)).Submit())
		return
	}
	if urlParts.Match(getTapMatch) {
		h.writeResponse(w, controllers.NewErrorResponse(http.StatusOK, http.StatusText(http.StatusOK)).Submit())
		return
	}
	if urlParts.Match(getFileUserLocNameMatch) {
		h.writeResponse(w, controllers.NewErrorResponse(202, http.StatusText(202)).Submit())
		return
	}
	h.writeResponse(w, controllers.NewErrorResponse(http.StatusNotFound, http.StatusText(http.StatusNotFound)).Submit())
}

func (p *ServerHandler) writeResponse(w http.ResponseWriter, resp *controllers.ResponseData) {

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
	bufrw.WriteString(fmt.Sprintf("Content-Type: %s; charset=%s\n", config.DefaultContentType, p.config.ContentTypeCharset))
	bufrw.WriteString(fmt.Sprintf("Date: %s\n", timeAsString()))
	bufrw.WriteString(fmt.Sprintf("Server: %s\n", p.config.ServerName))
	bufrw.WriteString("\n")
	bufrw.WriteString(resp.Content())
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

func NewWebAppServer(configData *config.ConfigData, actionQueue chan ActionId) *WebAppServer {
	return &WebAppServer{
		Handler: NewServerHandler(configData, actionQueue),
	}
}

func (p *WebAppServer) Start() {
	fmt.Printf("Server running on port:%d\n", p.Handler.config.Port)
	log.Fatal(http.ListenAndServe(p.Handler.config.PortString(), p.Handler))

}

func (p *WebAppServer) ToString() string {
	cAsString, err := p.Handler.config.ToString()
	if err != nil {
		return fmt.Sprintf("Server Error: In 'Handler.config.ToString()': %s", err.Error())
	}
	return cAsString
}
