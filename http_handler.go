package main

import (
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/castisdev/cfm/heartbeater"
	"github.com/castisdev/cfm/tasker"

	"github.com/gorilla/mux"
)

func HostStateDashBoard(w http.ResponseWriter, r *http.Request) {
	api.Infof("[%s] received stateDashBoard request", r.RemoteAddr)
	defer api.Infof("[%s] responsed stateDashBoard request", r.RemoteAddr)

	tpl := template.Must(template.ParseFiles("dashboard/hoststate.html"))
	tpl.Execute(w, heartbeater.GetList())
}

// DashBoard :
func DashBoard(w http.ResponseWriter, r *http.Request) {
	api.Infof("[%s] received dashBoard request", r.RemoteAddr)
	defer api.Infof("[%s] responsed dashBoard request", r.RemoteAddr)

	tpl := template.Must(template.ParseFiles("dashboard/layout.html"))
	tpl.Execute(w, tasks.GetTaskList())
}

// TaskIndex is http handler for GET /tasks route
func TaskIndex(w http.ResponseWriter, r *http.Request) {
	api.Infof("[%s] received getTasks request", r.RemoteAddr)
	defer api.Infof("[%s] responsed getTasks request", r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	t := tasks.GetTaskList()
	if err := json.NewEncoder(w).Encode(t); err != nil {
		api.Errorf("decode json fail : %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// TaskDelete is http handler for DELETE /tasks/<taskID> route
func TaskDelete(w http.ResponseWriter, r *http.Request) {
	api.Infof("[%s] received deleteTask request", r.RemoteAddr)
	defer api.Infof("[%s] responsed deleteTask request", r.RemoteAddr)

	vars := mux.Vars(r)
	taskID := vars["taskId"]
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if id, err := strconv.ParseInt(taskID, 10, 64); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		if err := tasks.DeleteTask(id); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// TaskUpdate is http handler for PATCH /tasks/<taskID> route
func TaskUpdate(w http.ResponseWriter, r *http.Request) {
	api.Infof("[%s] received updateTask request", r.RemoteAddr)
	defer api.Infof("[%s] responsed updateTask request", r.RemoteAddr)

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
		api.Errorf("failed to update task status, decode json fail : %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// https://stackoverflow.com/questions/33229860/go-http-requests-json-reusing-connections
	if dec.More() {
		// there's more data in the stream, so discard whatever is left
		io.Copy(ioutil.Discard, r.Body)
	}
	defer r.Body.Close()

	t, exists := tasks.FindTaskByID(ID)
	if !exists {
		api.Warningf("failed to update task status(%s) invalid task,ID(%d)",
			s.Status, ID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := tasks.UpdateStatus(ID, s.Status); err != nil {
		api.Errorf("failed to update task status(%s), task(%s), error(%s)",
			s.Status, t, err.Error())
		w.WriteHeader(http.StatusConflict)
		return
	}

	api.Infof("updated task status(%s),task(%s)", s.Status, t)
	w.WriteHeader(http.StatusOK)
}
