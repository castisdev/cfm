package tasker

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/heartbeater"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func cfw(cfwaddr string, du common.DiskUsage, filenames []string) *httptest.Server {
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("HEAD").Path("/hb").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	router.Methods("GET").Path("/df").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			bd, err := json.Marshal(du)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(bd)
		})
	router.Methods("GET").Path("/files").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			for _, fn := range filenames {
				fmt.Fprintln(w, fn)
			}
		})
	router.Methods("DELETE").Path("/files/{fileName}").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			fileName, exists := vars["fileName"]
			if !exists {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			for _, fn := range filenames {
				if fileName == fn {
					w.WriteHeader(http.StatusOK)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		})

	s := &http.Server{
		Addr:         cfwaddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	cfw := httptest.NewUnstartedServer(router)
	l, _ := net.Listen("tcp", cfwaddr)
	cfw.Listener.Close()
	cfw.Listener = l
	cfw.Config = s

	return cfw
}

// makePresetSrcDestServers :
// (*SrcServers)[0] // 127.0.0.1:8081
//
// (*DstServers)[0] // 127.0.0.5:18085
// (*DstServers)[1] // 127.0.0.4:18084
// (*DstServers)[2] // 127.0.0.3:18083
// (*DstServers)[3] // 127.0.0.2:18082
// (*DstServers)[4] // 127.0.0.1:18081
func makePresetSrcDestServers1() {
	SrcServers = NewSrcHosts()
	SrcServers.Add("127.0.0.1:8081")
	(*SrcServers)[0].Status = OK // 127.0.0.1:8081

	DstServers = NewDstHosts()
	// sort 되어 들어감
	DstServers.Add("127.0.0.1:18081")
	DstServers.Add("127.0.0.2:18082")
	DstServers.Add("127.0.0.3:18083")
	DstServers.Add("127.0.0.4:18084")
	DstServers.Add("127.0.0.5:18085")

	(*DstServers)[0].Status = OK // 127.0.0.5:18085
	(*DstServers)[1].Status = OK // 127.0.0.4:18082
	(*DstServers)[2].Status = OK // 127.0.0.3:18083
	(*DstServers)[3].Status = OK // 127.0.0.2:18082
	(*DstServers)[4].Status = OK // 127.0.0.1:18081
}

func TestSrcHosts_Add(t *testing.T) {
	srcs := new(SrcHosts)

	srcs.Add("127.0.0.1:18001")
	srcs.Add("127.0.0.2:18001")
	srcs.Add("127.0.0.3:18001")
	assert.Equal(t, 3, len(*srcs))

	assert.Equal(t, "127.0.0.3", (*srcs)[0].IP)
	assert.Equal(t, 18001, (*srcs)[0].Port)
	assert.Equal(t, "127.0.0.3:18001", (*srcs)[0].Addr)

	assert.Equal(t, "127.0.0.2", (*srcs)[1].IP)
	assert.Equal(t, 18001, (*srcs)[1].Port)
	assert.Equal(t, "127.0.0.2:18001", (*srcs)[1].Addr)

	assert.Equal(t, "127.0.0.1", (*srcs)[2].IP)
	assert.Equal(t, 18001, (*srcs)[2].Port)
	assert.Equal(t, "127.0.0.1:18001", (*srcs)[2].Addr)

	srcs.Add("127.0.0.3:18001")
	assert.Equal(t, 4, len(*srcs))

	assert.Equal(t, "127.0.0.3", (*srcs)[0].IP)
	assert.Equal(t, 18001, (*srcs)[0].Port)
	assert.Equal(t, "127.0.0.3:18001", (*srcs)[0].Addr)

	assert.Equal(t, "127.0.0.3", (*srcs)[0].IP)
	assert.Equal(t, 18001, (*srcs)[0].Port)
	assert.Equal(t, "127.0.0.3:18001", (*srcs)[0].Addr)

	assert.Equal(t, "127.0.0.2", (*srcs)[2].IP)
	assert.Equal(t, 18001, (*srcs)[2].Port)
	assert.Equal(t, "127.0.0.2:18001", (*srcs)[2].Addr)

	assert.Equal(t, "127.0.0.1", (*srcs)[3].IP)
	assert.Equal(t, 18001, (*srcs)[3].Port)
	assert.Equal(t, "127.0.0.1:18001", (*srcs)[3].Addr)
}

func Test_getAllHostStatus(t *testing.T) {
	s1 := "127.0.0.1:8081"
	s1files := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	s1d := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	cfws1 := cfw(s1, s1d, s1files)
	cfws1.Start()
	defer cfws1.Close()

	d1 := "127.0.0.1:18081"
	d1files := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	d1d := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	cfwd1 := cfw(d1, d1d, d1files)
	cfwd1.Start()
	defer cfwd1.Close()

	SrcServers = NewSrcHosts()
	SrcServers.Add(s1)

	// heartbeater가 동작하기 전이라, status가 NOTOK
	SrcServers.getAllHostStatus()
	assert.Equal(t, NOTOK, (*SrcServers)[0].Status)

	heartbeater.Add(s1)
	heartbeater.Heartbeat()

	// heartbeater가 동작하고 나면, status가 OK
	SrcServers.getAllHostStatus()
	assert.Equal(t, OK, (*SrcServers)[0].Status)

	DstServers = NewDstHosts()
	DstServers.Add(d1)

	// heartbeater가 동작하기 전이라, status가 NOTOK
	DstServers.getAllHostStatus()
	assert.Equal(t, NOTOK, (*DstServers)[0].Status)

	heartbeater.Add(d1)
	heartbeater.Heartbeat()

	// heartbeater가 동작하고 나면, status가 OK
	DstServers.getAllHostStatus()
	assert.Equal(t, OK, (*DstServers)[0].Status)

}

func Test_cleanTask(t *testing.T) {
	makePresetSrcDestServers1()

	ts := NewTasks()

	t1 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t2 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
	t3 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.3:18083"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.4:18084"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.5:18085"})

	ts.TaskMap[t1.ID].Status = DONE
	ts.TaskMap[t2.ID].Status = DONE

	cleanTask(ts)
	// 2개의 DONE task 삭제된 후 task 개수
	// t1, t2 삭제
	assert.Equal(t, 3, len(ts.TaskMap))
	assert.NotContains(t, ts.TaskMap, t1.ID)
	assert.NotContains(t, ts.TaskMap, t2.ID)

	(*DstServers)[2].Status = NOTOK
	cleanTask(ts)
	// dest server 127.0.0.3:8080 의 상태가 NOTOK 로 바뀌어서 삭제됨
	// 따라서 t3(dest가 127.0.0.3:8080인 task) 삭제됨
	assert.Equal(t, 2, len(ts.TaskMap))
	assert.NotContains(t, ts.TaskMap, t3.ID)

	SetTaskTimeout(time.Second * 1)
	time.Sleep(time.Second * 2)
	cleanTask(ts)
	// 2개 중 2개의 task 가 timeout으로 삭제됨
	assert.Equal(t, 0, len(ts.TaskMap))

}

func Test_setSelected(t *testing.T) {
	makePresetSrcDestServers1()

	ts := NewTasks()
	tasks = ts

	// Task에 사용 중이지 않는 src server의 selected 상태는 false로 바뀜
	assert.Equal(t, false, (*SrcServers)[0].selected)

	// Task에 사용 중이지 않는 dest server의 selected 상태는 false로 바뀜
	assert.Equal(t, false, (*DstServers)[0].selected)
	assert.Equal(t, false, (*DstServers)[1].selected)
	assert.Equal(t, false, (*DstServers)[2].selected)
	assert.Equal(t, false, (*DstServers)[3].selected)
	assert.Equal(t, false, (*DstServers)[4].selected)

	// Task에 사용 중이지 않는 src server의 selected 상태는 true 에서도
	(*SrcServers)[0].selected = true
	SrcServers.setSelected(tasks.GetTaskList())
	// Task에 사용 중이지 않는 src server의 selected 상태는 false로 바뀜
	assert.Equal(t, false, (*SrcServers)[0].selected)

	// Task에 사용 중이지 않는 dest server의 selected 상태는 true 에서도
	(*DstServers)[0].selected = true
	(*DstServers)[1].selected = true
	(*DstServers)[2].selected = true
	(*DstServers)[3].selected = true
	(*DstServers)[4].selected = true
	DstServers.setSelected(tasks.GetTaskList())
	// Task에 사용 중이지 않는 dest server의 selected 상태는 false로 바뀜
	assert.Equal(t, false, (*DstServers)[0].selected)
	assert.Equal(t, false, (*DstServers)[1].selected)
	assert.Equal(t, false, (*DstServers)[2].selected)
	assert.Equal(t, false, (*DstServers)[3].selected)
	assert.Equal(t, false, (*DstServers)[4].selected)

	t1 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t2 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
	t3 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.3:18083"})
	t4 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.4:18084"})
	t5 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.5:18085"})
	t.Log(t1)
	t.Log(t2)
	t.Log(t3)
	t.Log(t4)
	t.Log(t5)

	(*SrcServers)[0].selected = false
	SrcServers.setSelected(tasks.GetTaskList())
	// Task에 사용 중이면, src server의 selected 상태가 true 로 바뀜
	assert.Equal(t, true, (*SrcServers)[0].selected)

	(*DstServers)[0].selected = false
	(*DstServers)[1].selected = false
	(*DstServers)[2].selected = false
	(*DstServers)[3].selected = false
	(*DstServers)[4].selected = false
	DstServers.setSelected(tasks.GetTaskList())
	// Task에 사용 중이면 dest server의 selected 상태는 true로 바뀜
	assert.Equal(t, true, (*DstServers)[0].selected)
	assert.Equal(t, true, (*DstServers)[1].selected)
	assert.Equal(t, true, (*DstServers)[2].selected)
	assert.Equal(t, true, (*DstServers)[3].selected)
	assert.Equal(t, true, (*DstServers)[4].selected)

}

func Test_getSelectableCount(t *testing.T) {
	makePresetSrcDestServers1()

	ts := NewTasks()
	tasks = ts

	// task에 사용되지 않고, 상태가 ok인 src server 개수 return
	n := SrcServers.getSelectableCount()
	assert.Equal(t, 1, n)

	// 상태가 notok 이므로, 0 return
	(*SrcServers)[0].Status = NOTOK // 127.0.0.1:8081
	n = SrcServers.getSelectableCount()
	assert.Equal(t, 0, n)

	(*SrcServers)[0].Status = OK     // 127.0.0.1:8081
	(*SrcServers)[0].selected = true // 127.0.0.1:8081
	n = SrcServers.getSelectableCount()
	assert.Equal(t, 0, n)

	(*SrcServers)[0].Status = OK // 127.0.0.1:8081
	t1 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t.Log(t1)
	// 모든 src server가 task 에 사용되므로,
	// src server의 select 상태가 true 바뀜
	SrcServers.setSelected(tasks.GetTaskList())

	// 상태가 ok 이고, task 에 사용되지 않는 src server가 없어서 0 return
	n = SrcServers.getSelectableCount()
	assert.Equal(t, 0, n)
}

func Test_getAvailableSrcServerCount(t *testing.T) {
	makePresetSrcDestServers1()

	ts := NewTasks()
	tasks = ts

	// task에 사용되지 않고, 상태가 ok인 src server 개수 return
	n := getAvailableSrcServerCount(tasks.GetTaskList())
	assert.Equal(t, 1, n)

	// 상태가 notok 이므로, 사용할 수 있는 src server 개수가 0 return
	(*SrcServers)[0].Status = NOTOK // 127.0.0.1:8081
	n = getAvailableSrcServerCount(tasks.GetTaskList())
	assert.Equal(t, 0, n)

	(*SrcServers)[0].Status = OK     // 127.0.0.1:8081
	(*SrcServers)[0].selected = true // 127.0.0.1:8081
	// 상태는 ok 이고, selected 상태가 reset 되고,
	// task 가 만들어지지 않은 상태여서, 사용할 수 있는 src server개수가 1 return
	n = getAvailableSrcServerCount(tasks.GetTaskList())
	assert.Equal(t, 1, n)

	(*SrcServers)[0].Status = OK // 127.0.0.1:8081
	t1 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t.Log(t1)
	// 모든 src server가 task 에 사용되므로,
	// 상태가 ok 이고, task 에 사용되지 않는 src server가 없어서 0 return
	n = getAvailableSrcServerCount(tasks.GetTaskList())
	assert.Equal(t, 0, n)
}

func Test_getSelectableList(t *testing.T) {
	makePresetSrcDestServers1()

	ts := NewTasks()
	tasks = ts

	// task에 사용되지 않고, 상태가 ok인 src server 개수 return
	dests := DstServers.getSelectableList()
	assert.Equal(t, 5, len(dests))

	// 하나의 상태가 notok 이므로, 5개가 아닌 4 return
	(*DstServers)[0].Status = NOTOK
	dests = DstServers.getSelectableList()
	assert.Equal(t, 4, len(dests))

	(*DstServers)[0].Status = OK     // 127.0.0.1:8081
	(*DstServers)[0].selected = true // 127.0.0.1:8081
	// selected 가 하나 있으므로, 4 return
	dests = DstServers.getSelectableList()
	assert.Equal(t, 4, len(dests))

	t1 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t.Log(t1)
	t2 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
	t.Log(t2)

	// dest server중, t1, t2에 사용되는 server의 select 상태가 true 바뀜
	DstServers.setSelected(tasks.GetTaskList())

	// // 127.0.0.5:18085 의 상태가 NOTOK 로 바뀐다면
	(*DstServers)[0].Status = NOTOK // 127.0.0.5:18085

	// 상태가 ok 이고, task 에 사용되지 않는 dest server는 3개 return 해야하지만
	// 127.0.0.5:18085 의 상태가 NOTOK 이므로, 2개 return
	dests = DstServers.getSelectableList()
	assert.Equal(t, 2, len(dests))
	assert.Contains(t, dests, DstHost{
		common.Host{IP: "127.0.0.3", Port: 18083, Addr: "127.0.0.3:18083"},
		false, OK})
	assert.Contains(t, dests, DstHost{
		common.Host{IP: "127.0.0.4", Port: 18084, Addr: "127.0.0.4:18084"},
		false, OK})

	t3 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.3:18083"})
	t.Log(t3)
	t4 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.4:18084"})
	t.Log(t4)
	t5 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.5:18085"})
	t.Log(t5)

	// dest server중, t1, t2에 사용되는 server의 select 상태가 true 바뀜
	DstServers.setSelected(tasks.GetTaskList())
	(*DstServers)[0].Status = OK // 127.0.0.5:18085
	// 127.0.0.5:18085 의 상태가 OK 로 바뀐 것과 상관없이
	// 모든 dst server 가 task 에 사용 중이므로,
	// 상태가 ok 이고, task 에 사용되지 않는 dest server는 없음
	dests = DstServers.getSelectableList()
	t.Log(dests)
	assert.Equal(t, 0, len(dests))
}

func Test_getAvailableDstServerList(t *testing.T) {
	makePresetSrcDestServers1()

	ts := NewTasks()
	tasks = ts

	// task에 사용되지 않고, 상태가 ok인 src server 개수 return
	dests := getAvailableDstServerList(tasks.GetTaskList())
	assert.Equal(t, 5, len(dests))

	// 하나의 상태가 notok 이므로, 5개가 아닌 4 return
	(*DstServers)[0].Status = NOTOK
	dests = getAvailableDstServerList(tasks.GetTaskList())
	assert.Equal(t, 4, len(dests))

	(*DstServers)[0].Status = OK     // 127.0.0.1:8081
	(*DstServers)[0].selected = true // 127.0.0.1:8081
	// selected 가 하나있으나,
	// getAvailableDstServerList 함수를 호출하면, selected 상태가 task list 검사해서 update됨
	// task 가 현재 없으므로, available list는 5개가 됨
	dests = getAvailableDstServerList(tasks.GetTaskList())
	assert.Equal(t, 5, len(dests))

	t1 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t.Log(t1)
	t2 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
	t.Log(t2)

	// // 127.0.0.5:18085 의 상태가 NOTOK 로 바뀐다면
	(*DstServers)[0].Status = NOTOK // 127.0.0.5:18085

	// dest server중, t1, t2에 사용되는 server의 select 상태가 true 바뀜
	dests = getAvailableDstServerList(tasks.GetTaskList())

	// 상태가 ok 이고, task 에 사용되지 않는 dest server는 3개 return 해야하지만
	// 127.0.0.5:18085 의 상태가 NOTOK 이므로, 2개 return
	assert.Equal(t, 2, len(dests))
	assert.Contains(t, dests, DstHost{
		common.Host{IP: "127.0.0.3", Port: 18083, Addr: "127.0.0.3:18083"},
		false, OK})
	assert.Contains(t, dests, DstHost{
		common.Host{IP: "127.0.0.4", Port: 18084, Addr: "127.0.0.4:18084"},
		false, OK})

	t3 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.3:18083"})
	t.Log(t3)
	t4 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.4:18084"})
	t.Log(t4)
	t5 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.5:18085"})
	t.Log(t5)

	(*DstServers)[0].Status = OK // 127.0.0.5:18085
	// 127.0.0.5:18085 의 상태가 OK 로 바뀐 것과 상관없이
	// 모든 dst server 가 task 에 사용 중이므로,
	// 상태가 ok 이고, task 에 사용되지 않는 dest server는 없음
	dests = getAvailableDstServerList(tasks.GetTaskList())
	t.Log(dests)
	assert.Equal(t, 0, len(dests))
}

func Test_getAvailableDstServerRing(t *testing.T) {
	makePresetSrcDestServers1()

	ts := NewTasks()
	tasks = ts

	// task에 사용되지 않고, 상태가 ok인 src server 개수 return
	dests := getAvailableDstServerRing(tasks.GetTaskList())
	assert.Equal(t, 5, dests.Len())

	// 하나의 상태가 notok 이므로, 5개가 아닌 4 return
	(*DstServers)[0].Status = NOTOK
	dests = getAvailableDstServerRing(tasks.GetTaskList())
	assert.Equal(t, 4, dests.Len())

	(*DstServers)[0].Status = OK     // 127.0.0.1:8081
	(*DstServers)[0].selected = true // 127.0.0.1:8081
	// selected 가 하나있으나,
	// getAvailableDstServerList 함수를 호출하면, selected 상태가 task list 검사해서 update됨
	// task 가 현재 없으므로, available list는 5개가 됨
	dests = getAvailableDstServerRing(tasks.GetTaskList())
	assert.Equal(t, 5, dests.Len())

	t1 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t.Log(t1)
	t2 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
	t.Log(t2)

	// // 127.0.0.5:18085 의 상태가 NOTOK 로 바뀐다면
	(*DstServers)[0].Status = NOTOK // 127.0.0.5:18085

	// dest server중, t1, t2에 사용되는 server의 select 상태가 true 바뀜
	dests = getAvailableDstServerRing(tasks.GetTaskList())

	// 상태가 ok 이고, task 에 사용되지 않는 dest server는 3개 return 해야하지만
	// 127.0.0.5:18085 의 상태가 NOTOK 이므로, 2개 return
	assert.Equal(t, 2, dests.Len())
	// sort 되어서, 127.0.0.4:18084, 127.0.0.3:18083 순으로 들어있음
	assert.Equal(t, dests.Value, DstHost{
		common.Host{IP: "127.0.0.4", Port: 18084, Addr: "127.0.0.4:18084"},
		false, OK})
	assert.Equal(t, dests.Next().Value, DstHost{
		common.Host{IP: "127.0.0.3", Port: 18083, Addr: "127.0.0.3:18083"},
		false, OK})

	t3 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.3:18083"})
	t.Log(t3)
	t4 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.4:18084"})
	t.Log(t4)
	t5 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.5:18085"})
	t.Log(t5)

	// dest server중, t1, t2에 사용되는 server의 select 상태가 true 바뀜
	(*DstServers)[0].Status = OK // 127.0.0.5:18085
	// 127.0.0.5:18085 의 상태가 OK 로 바뀐 것과 상관없이
	// 모든 dst server 가 task 에 사용 중이므로,
	// 상태가 ok 이고, task 에 사용되지 않는 dest server는 없음
	dests = getAvailableDstServerRing(tasks.GetTaskList())
	assert.Nil(t, dests)
}

func TestCollectRemoteFileList(t *testing.T) {

	// set dummy http server
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "A.mpg")
		fmt.Fprintln(w, "B.mpg")
		fmt.Fprintln(w, "C.mpg")
	})

	ts1 := httptest.NewUnstartedServer(h)
	l1, _ := net.Listen("tcp", "127.0.0.1:18881")
	ts1.Listener.Close()
	ts1.Listener = l1
	ts1.Start()
	defer ts1.Close()

	ts2 := httptest.NewUnstartedServer(h)
	l2, _ := net.Listen("tcp", "127.0.0.1:18882")
	ts2.Listener.Close()
	ts2.Listener = l2
	ts2.Start()
	defer ts2.Close()

	ts3 := httptest.NewUnstartedServer(h)
	l3, _ := net.Listen("tcp", "127.0.0.1:18883")
	ts3.Listener.Close()
	ts3.Listener = l3
	ts3.Start()
	defer ts3.Close()

	dsthosts := NewDstHosts()
	dsthosts.Add("127.0.0.1:18881")
	dsthosts.Add("127.0.0.1:18882")
	dsthosts.Add("127.0.0.1:18883")

	fs := make(map[string]int)
	CollectRemoteFileList(dsthosts, fs)

	assert.Equal(t, 3, fs["A.mpg"])
	assert.Equal(t, 3, fs["B.mpg"])
	assert.Equal(t, 3, fs["C.mpg"])
}

func Test_selectSourceServer(t *testing.T) {

	srcs := new(SrcHosts)

	srcs.Add("127.0.0.1:18001")
	srcs.Add("127.0.0.2:18001")
	srcs.Add("127.0.0.3:18001")

	// status 이 NOTOK 이므로, select 되지 않음
	for i := 0; i < 3; i++ {
		_, exists := srcs.selectSourceServer()
		assert.Equal(t, false, exists)
	}

	// (*srcs)[0] status 값이 OK 이므로,(*srcs)[0]이select 됨
	(*srcs)[0].Status = OK
	for i := 0; i < 3; i++ {
		srcs.selectSourceServer()
	}
	assert.Equal(t, (*srcs)[0].selected, true)

	// 1,2 번도 Status 값이 OK 이므로, select 됨
	(*srcs)[1].Status = OK
	(*srcs)[2].Status = OK
	for i := 0; i < 3; i++ {
		srcs.selectSourceServer()
	}
	assert.Equal(t, (*srcs)[1].selected, true)
	assert.Equal(t, (*srcs)[2].selected, true)

	// 이미 3개의 src 를 모두 사용했으모로 src 가 없어야 한다.
	_, exists := srcs.selectSourceServer()
	assert.Equal(t, false, exists)
}
