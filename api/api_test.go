package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/castisdev/cfm/fmfm"
	"github.com/castisdev/cfm/tasker"
	"github.com/stretchr/testify/assert"
)

func TestGetTasks(t *testing.T) {
	tskr := tasker.NewTasker()
	tasks := tskr.Tasks()
	defer tasks.DeleteAllTask()

	t1 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.1:8081"})
	t2 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.2:8081"})
	t3 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.3:8081"})
	t4 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.4:8081"})

	tasks.TaskMap[t1.ID].Status = tasker.DONE
	tasks.TaskMap[t2.ID].Status = tasker.READY
	tasks.TaskMap[t3.ID].Status = tasker.WORKING
	tasks.TaskMap[t4.ID].Status = tasker.TIMEOUT

	tskr.SrcServers.Add("127.0.0.1:8080")
	(*tskr.SrcServers)[0].Status = tasker.OK

	tskr.DstServers.Add("127.0.0.1:8081")
	tskr.DstServers.Add("127.0.0.2:8081")
	tskr.DstServers.Add("127.0.0.3:8081")
	tskr.DstServers.Add("127.0.0.4:8081")

	(*tskr.DstServers)[0].Status = tasker.OK
	(*tskr.DstServers)[1].Status = tasker.OK
	(*tskr.DstServers)[2].Status = tasker.OK
	(*tskr.DstServers)[3].Status = tasker.OK

	serverAddr := "127.0.0.1:18881"
	r := fmfm.NewRunner(0, 0, nil, tskr, nil)
	m := fmfm.NewManager(nil, r)
	h := NewAPIHandler(m)
	router := NewRouter(h)
	s := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	ts1 := httptest.NewUnstartedServer(router)
	l1, _ := net.Listen("tcp", serverAddr)
	ts1.Listener.Close()

	ts1.Listener = l1
	ts1.Config = s
	ts1.Start()
	defer ts1.Close()

	url := fmt.Sprintf("http://%s/tasks", serverAddr)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Errorf("failed to get task list, error(%s)", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to get task list, error(%s)", err.Error())
		return
	}

	clienttaskList := make([]tasker.Task, 0)
	if err := json.NewDecoder(resp.Body).Decode(&clienttaskList); err != nil {
		t.Errorf("failed to get task list, error(%s)", err.Error())
		return
	}

	assert.Equal(t, 4, len(clienttaskList))
	doc, _ := json.MarshalIndent(clienttaskList, "", "  ")
	t.Logf("response: %s", doc)

	// api 로 받은 list 와 내부의 list 가 같은 지 test
	// expected list와 내용이 다르면 error
	expecttl := []tasker.Task{
		*tasker.NewTaskFrom(*tasks.TaskMap[t1.ID]),
		*tasker.NewTaskFrom(*tasks.TaskMap[t2.ID]),
		*tasker.NewTaskFrom(*tasks.TaskMap[t3.ID]),
		*tasker.NewTaskFrom(*tasks.TaskMap[t4.ID]),
	}
	if !reflect.DeepEqual(clienttaskList, expecttl) {
		t.Errorf("got from api list(%s)", clienttaskList)
		t.Errorf("expected     list(%s)", expecttl)
	}

	// api 로 받은 list 와 내부의 list 가 같은 지 test2
	// server 의 task list 와도 같아야 함
	servertl := tasks.GetTaskList()
	if !reflect.DeepEqual(clienttaskList, servertl) {
		t.Errorf("got from api list(%s)", clienttaskList)
		t.Errorf("server     list(%s)", servertl)
	}

}

func TestGetTasksEmptyList(t *testing.T) {
	tskr := tasker.NewTasker()
	tasks := tskr.Tasks()
	defer tasks.DeleteAllTask()

	tskr.SrcServers.Add("127.0.0.1:8080")
	(*tskr.SrcServers)[0].Status = tasker.OK

	tskr.DstServers.Add("127.0.0.1:8081")
	tskr.DstServers.Add("127.0.0.2:8081")
	tskr.DstServers.Add("127.0.0.3:8081")
	tskr.DstServers.Add("127.0.0.4:8081")

	serverAddr := "127.0.0.1:18881"
	r := fmfm.NewRunner(0, 0, nil, tskr, nil)
	m := fmfm.NewManager(nil, r)
	h := NewAPIHandler(m)
	router := NewRouter(h)
	s := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	ts1 := httptest.NewUnstartedServer(router)
	l1, _ := net.Listen("tcp", serverAddr)
	ts1.Listener.Close()

	ts1.Listener = l1
	ts1.Config = s
	ts1.Start()
	defer ts1.Close()

	url := fmt.Sprintf("http://%s/tasks", serverAddr)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Errorf("failed to get task list, error(%s)", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to get task list, error(%s)", err.Error())
		return
	}

	clienttaskList := make([]tasker.Task, 0)
	if err := json.NewDecoder(resp.Body).Decode(&clienttaskList); err != nil {
		t.Errorf("failed to get task list, error(%s)", err.Error())
		return
	}

	assert.Equal(t, 0, len(clienttaskList))
	doc, _ := json.MarshalIndent(clienttaskList, "", "  ")
	t.Logf("response: %s", doc)

	// api 로 받은 list 와 내부의 list 가 같은 지 test
	// expected list와 내용이 다르면 error
	expecttl := []tasker.Task{}
	if !reflect.DeepEqual(clienttaskList, expecttl) {
		t.Errorf("got from api list(%s)", clienttaskList)
		t.Errorf("expected     list(%s)", expecttl)
	}

	// api 로 받은 list 와 내부의 list 가 같은 지 test2
	// server 의 task list 와도 같아야 함
	servertl := tasks.GetTaskList()
	if !reflect.DeepEqual(clienttaskList, servertl) {
		t.Errorf("got from api list(%s)", clienttaskList)
		t.Errorf("server     list(%s)", servertl)
	}
}

func TestAPI_TaskDelete(t *testing.T) {
	tskr := tasker.NewTasker()
	tasks := tskr.Tasks()
	defer tasks.DeleteAllTask()

	t1 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.1:8081"})
	t2 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.2:8081"})
	t3 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.3:8081"})
	t4 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.4:8081"})

	tasks.TaskMap[t1.ID].Status = tasker.DONE
	tasks.TaskMap[t2.ID].Status = tasker.READY
	tasks.TaskMap[t3.ID].Status = tasker.WORKING
	tasks.TaskMap[t4.ID].Status = tasker.TIMEOUT

	tskr.SrcServers.Add("127.0.0.1:8080")
	(*tskr.SrcServers)[0].Status = tasker.OK

	tskr.DstServers.Add("127.0.0.1:8081")
	tskr.DstServers.Add("127.0.0.2:8081")
	tskr.DstServers.Add("127.0.0.3:8081")
	tskr.DstServers.Add("127.0.0.4:8081")

	(*tskr.DstServers)[0].Status = tasker.OK
	(*tskr.DstServers)[1].Status = tasker.OK
	(*tskr.DstServers)[2].Status = tasker.OK
	(*tskr.DstServers)[3].Status = tasker.OK

	serverAddr := "127.0.0.1:18881"
	r := fmfm.NewRunner(0, 0, nil, tskr, nil)
	m := fmfm.NewManager(nil, r)
	h := NewAPIHandler(m)
	router := NewRouter(h)
	s := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	ts1 := httptest.NewUnstartedServer(router)
	l1, _ := net.Listen("tcp", serverAddr)
	ts1.Listener.Close()

	ts1.Listener = l1
	ts1.Config = s
	ts1.Start()
	defer ts1.Close()

	url := fmt.Sprintf("http://%s/tasks/%d", serverAddr, t3.ID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		t.Errorf("failed to delete task, error(%s)", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to delete task, error(%s)", err.Error())
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("failed to delete task, response(%s)", resp.Status)
		return
	}

	// delete 후의 task list
	servertl := tasks.GetTaskList()

	// expect task list
	// 위의 delete api 호출로 인해서
	// t3.ID 의 task 는 지워졌어야 함
	expecttl := []tasker.Task{
		*tasker.NewTaskFrom(*tasks.TaskMap[t1.ID]),
		*tasker.NewTaskFrom(*tasks.TaskMap[t2.ID]),
		*tasker.NewTaskFrom(*tasks.TaskMap[t4.ID]),
	}
	if !reflect.DeepEqual(expecttl, servertl) {
		t.Errorf("after delete, server task list(%s)", servertl)
		t.Errorf("exptected                 list(%s)", expecttl)
	}
}

func TestAPI_TaskUpdate(t *testing.T) {
	tskr := tasker.NewTasker()
	tasks := tskr.Tasks()
	defer tasks.DeleteAllTask()

	t1 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.1:8081"})
	t2 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.2:8081"})
	t3 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.3:8081"})
	t4 := tasks.CreateTask(&tasker.Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.4:8081"})

	tasks.TaskMap[t1.ID].Status = tasker.DONE
	tasks.TaskMap[t2.ID].Status = tasker.READY
	tasks.TaskMap[t3.ID].Status = tasker.WORKING
	tasks.TaskMap[t4.ID].Status = tasker.TIMEOUT

	tskr.SrcServers.Add("127.0.0.1:8080")
	(*tskr.SrcServers)[0].Status = tasker.OK

	tskr.DstServers.Add("127.0.0.1:8081")
	tskr.DstServers.Add("127.0.0.2:8081")
	tskr.DstServers.Add("127.0.0.3:8081")
	tskr.DstServers.Add("127.0.0.4:8081")

	(*tskr.DstServers)[0].Status = tasker.OK
	(*tskr.DstServers)[1].Status = tasker.OK
	(*tskr.DstServers)[2].Status = tasker.OK
	(*tskr.DstServers)[3].Status = tasker.OK

	serverAddr := "127.0.0.1:18881"
	r := fmfm.NewRunner(0, 0, nil, tskr, nil)
	m := fmfm.NewManager(nil, r)
	h := NewAPIHandler(m)
	router := NewRouter(h)
	s := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	ts1 := httptest.NewUnstartedServer(router)
	l1, _ := net.Listen("tcp", serverAddr)
	ts1.Listener.Close()

	ts1.Listener = l1
	ts1.Config = s
	ts1.Start()
	defer ts1.Close()

	st := struct {
		Status tasker.Status `json:"status"`
	}{
		Status: tasker.DONE,
	}
	body, err := json.Marshal(&st)
	if err != nil {
		t.Errorf("failed to update task, error(%s)", err.Error())
		return
	}

	url := fmt.Sprintf("http://%s/tasks/%d", serverAddr, t3.ID)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		t.Errorf("failed to update task, error(%s)", err.Error())
		return
	}
	t.Logf("request body: %s", body)

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to update task, error(%s)", err.Error())
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("failed to update task, response(%s)", resp.Status)
		return
	}

	// update 후의 task list
	servertl := tasks.GetTaskList()

	// expect task list
	// 위의 update api 호출로 인해서
	// t3 의 상태가 done 으로 바뀌어 있어야 함
	updatedt3 := *tasker.NewTaskFrom(*tasks.TaskMap[t3.ID])
	updatedt3.Status = tasker.DONE
	expecttl := []tasker.Task{
		*tasker.NewTaskFrom(*tasks.TaskMap[t1.ID]),
		*tasker.NewTaskFrom(*tasks.TaskMap[t2.ID]),
		updatedt3,
		*tasker.NewTaskFrom(*tasks.TaskMap[t4.ID]),
	}

	serverdoc, _ := json.MarshalIndent(servertl, "", "  ")
	t.Logf("server list: %s", serverdoc)
	expecteddoc, _ := json.MarshalIndent(expecttl, "", "  ")
	t.Logf("expected list: %s", expecteddoc)

	if !reflect.DeepEqual(expecttl, servertl) {
		t.Errorf("after delete, server task list(%s)", servertl)
		t.Errorf("exptected                 list(%s)", expecttl)
	}
}
