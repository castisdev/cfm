package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"github.com/castisdev/cfm/tasker"
	"github.com/castisdev/cilog"

	"github.com/gorilla/mux"
)

// DashBoard :
func DashBoard(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles("dashboard/layout.html"))
	tpl.Execute(w, tasks.GetTaskList())
}

// TaskIndex is http handler for GET /tasks route
func TaskIndex(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(tasks.TaskMap); err != nil {
		panic(err)
	}
}

// TaskDelete is http handler for DELETE /tasks/<taskID> route
func TaskDelete(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	taskID := vars["taskId"]

	if id, err := strconv.ParseInt(taskID, 10, 64); err != nil {
		w.WriteHeader(404)
	} else {
		tasks.DeleteTask(id)
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
	}

}

// TaskUpdate is http handler for PATCH /tasks/<taskID> route
func TaskUpdate(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	taskID := vars["taskId"]

	ID, err := strconv.ParseInt(taskID, 10, 64)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var s struct {
		Status tasker.Status `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		cilog.Errorf("decode json fail : %s", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	defer r.Body.Close()

	t, exists := tasks.FindTaskByID(ID)
	if exists {

		switch s.Status {
		case tasker.WORKING:
			cilog.Infof("start task,ID(%d),Grade(%d),FilePath(%s),SrcIP(%s),DstIP(%s),CopySpeed(%s),Ctime(%d),Mtime(%d)",
				t.ID, t.Grade, t.FilePath, t.SrcIP, t.DstIP, t.CopySpeed, t.Ctime, t.Mtime)
		case tasker.DONE:
			cilog.Successf("finish task,ID(%d),Grade(%d),FilePath(%s),SrcIP(%s),DstIP(%s),CopySpeed(%s),Ctime(%d),Mtime(%d)",
				t.ID, t.Grade, t.FilePath, t.SrcIP, t.DstIP, t.CopySpeed, t.Ctime, t.Mtime)
		default:
			cilog.Warningf("unexpected status(%s)", s.Status.String())
		}

		if err := tasks.UpdateStatus(ID, s.Status); err != nil {
			w.WriteHeader(http.StatusConflict)
			return
		}

		w.WriteHeader(http.StatusOK)

	} else {

		cilog.Warningf("received request to update status for invalid task,ID(%d),Status(%s)", ID, s.Status)
		w.WriteHeader(http.StatusNotFound)
		return
	}
}
