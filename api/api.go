package api

import (
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/fmfm"
	"github.com/castisdev/cfm/heartbeater"
	"github.com/castisdev/cfm/tasker"
	"github.com/castisdev/cilog"
	"github.com/gorilla/mux"
)

var apilogger common.MLogger

func init() {
	apilogger = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "api"}
}

type APIHandler struct {
	manager *fmfm.Manager
}

func NewAPIHandler(m *fmfm.Manager) *APIHandler {
	return &APIHandler{manager: m}
}

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// NewRouter is constructor of Route struct
func NewRouter(h *APIHandler) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/tasks", h.GetTasks).Methods("GET")
	router.HandleFunc("/tasks/{taskId}", h.DeleteTask).Methods("DELETE")
	router.HandleFunc("/tasks/{taskId}", h.UpdateTask).Methods("PATCH")
	router.HandleFunc("/dashboard", h.GetDashBoard).Methods("GET")
	router.HandleFunc("/dashboard/hb", h.GetHostStateDashBoard).Methods("GET")
	router.HandleFunc("/dashboard/filemetas", h.GetFileMetas).Methods("GET")

	return router
}

// GetHostStateDashBoard
func (h *APIHandler) GetHostStateDashBoard(w http.ResponseWriter, r *http.Request) {
	apilogger.Infof("[%s] received getHostStateDashBoard request", r.RemoteAddr)
	defer apilogger.Infof("[%s] responsed getHostStateDashBoard request", r.RemoteAddr)

	tpl := template.Must(template.ParseFiles("dashboard/hoststate.html"))
	tpl.Execute(w, heartbeater.GetList())
}

// GetDashBoard :
func (h *APIHandler) GetDashBoard(w http.ResponseWriter, r *http.Request) {
	apilogger.Infof("[%s] received getDashBoard request", r.RemoteAddr)
	defer apilogger.Infof("[%s] responsed getDashBoard request", r.RemoteAddr)

	tpl := template.Must(template.ParseFiles("dashboard/layout.html"))
	tpl.Execute(w, h.manager.Tasks().GetTaskList())
}

// GetFileMetas :
func (h *APIHandler) GetFileMetas(w http.ResponseWriter, r *http.Request) {
	apilogger.Infof("[%s] received getFileMetas request", r.RemoteAddr)
	defer apilogger.Infof("[%s] responsed getFileMetas request", r.RemoteAddr)

	req := fmfm.GetFileMetas{RespCh: make(chan fmfm.FileMetas)}
	h.manager.GetFileMetasCh <- req
	res := <-req.RespCh

	tpl := template.Must(template.ParseFiles("dashboard/filemetas.html"))
	tpl.Execute(w, res)
}

// GetTasks is http handler for GET /tasks route
func (h *APIHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	apilogger.Infof("[%s] received getTasks request", r.RemoteAddr)
	defer apilogger.Infof("[%s] responsed getTasks request", r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	t := h.manager.Tasks().GetTaskList()
	if err := json.NewEncoder(w).Encode(t); err != nil {
		apilogger.Errorf("decode json fail : %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// DeleteTask is http handler for DELETE /tasks/<taskID> route
func (h *APIHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	apilogger.Infof("[%s] received deleteTask request", r.RemoteAddr)
	defer apilogger.Infof("[%s] responsed deleteTask request", r.RemoteAddr)

	vars := mux.Vars(r)
	taskID := vars["taskId"]
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if id, err := strconv.ParseInt(taskID, 10, 64); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		if err := h.manager.Tasks().DeleteTask(id); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// UpdateTask is http handler for PATCH /tasks/<taskID> route
func (h *APIHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	apilogger.Infof("[%s] received updateTask request", r.RemoteAddr)
	defer apilogger.Infof("[%s] responsed updateTask request", r.RemoteAddr)

	vars := mux.Vars(r)
	taskID := vars["taskId"]
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	ID, err := strconv.ParseInt(taskID, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var s struct {
		Status tasker.Status `json:"status"`
	}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&s); err != nil {
		apilogger.Errorf("failed to update task status, decode json fail : %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// https://stackoverflow.com/questions/33229860/go-http-requests-json-reusing-connections
	if dec.More() {
		// there's more data in the stream, so discard whatever is left
		io.Copy(ioutil.Discard, r.Body)
	}
	defer r.Body.Close()

	t, exists := h.manager.Tasks().FindTaskByID(ID)
	if !exists {
		apilogger.Warningf("failed to update task status(%s) invalid task,ID(%d)",
			s.Status, ID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := h.manager.Tasks().UpdateStatus(ID, s.Status); err != nil {
		apilogger.Errorf("failed to update task status(%s), task(%s), error(%s)",
			s.Status, t, err.Error())
		w.WriteHeader(http.StatusConflict)
		return
	}

	apilogger.Infof("updated task status(%s),task(%s)", s.Status, t)
	w.WriteHeader(http.StatusOK)
}
