package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"github.com/castisdev/cfm/heartbeater"
	"github.com/castisdev/cfm/tasker"

	"github.com/gorilla/mux"
)

func HostStateDashBoard(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles("dashboard/hoststate.html"))
	tpl.Execute(w, heartbeater.GetList())
}

// DashBoard :
func DashBoard(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles("dashboard/layout.html"))
	tpl.Execute(w, tasks.GetTaskList())
}

// TaskIndex is http handler for GET /tasks route
func TaskIndex(w http.ResponseWriter, r *http.Request) {
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

	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		api.Errorf("fail to update task status, decode json fail : %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	t, exists := tasks.FindTaskByID(ID)
	if !exists {
		api.Warningf("fail to update task status(%s) request for invalid task,ID(%d)",
			s.Status, ID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := tasks.UpdateStatus(ID, s.Status); err != nil {
		api.Errorf("fail to update task status(%s), task(%s), error(%s)",
			s.Status, t, err.Error())
		w.WriteHeader(http.StatusConflict)
		return
	}

	api.Infof("update task status(%s),task(%s)",
		s.Status, t)
	w.WriteHeader(http.StatusOK)
}
