package tasker

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/heartbeater"
	"github.com/castisdev/cfm/tailer"
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

func createfile(dir string, filename string) {
	fp := filepath.Join(dir, filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func deletefile(dir string, filename string) {
	dir = filepath.Clean(dir)
	if dir == "." || dir == ".." {
		log.Fatal(errors.New("do not delete current or parent folder"))
	}
	fp := filepath.Join(dir, filename)
	err := os.RemoveAll(fp)
	if err != nil {
		log.Fatal(err)
	}
}

// makePresetS1D4 :
//
// (*SrcServers)[0] // 127.0.0.1:8081
//
// (*DstServers)[0] // 127.0.0.5:18085
// (*DstServers)[1] // 127.0.0.4:18084
// (*DstServers)[2] // 127.0.0.3:18083
// (*DstServers)[3] // 127.0.0.2:18082
// (*DstServers)[4] // 127.0.0.1:18081
func makePresetS1D4() {
	SrcServers = NewSrcHosts()
	// sort 되어 들어감
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

// makePresetS1D4 :
//
// (*SrcServers)[0] // 127.0.0.4:8084
// (*SrcServers)[1] // 127.0.0.3:8083
// (*SrcServers)[3] // 127.0.0.2:8082
// (*SrcServers)[2] // 127.0.0.1:8081
//
// (*DstServers)[0] // 127.0.0.5:18085
// (*DstServers)[1] // 127.0.0.4:18084
// (*DstServers)[2] // 127.0.0.3:18083
// (*DstServers)[3] // 127.0.0.2:18082
// (*DstServers)[4] // 127.0.0.1:18081
func makePresetS4D5() {
	SrcServers = NewSrcHosts()
	// sort 되어 들어감
	SrcServers.Add("127.0.0.1:8081")
	SrcServers.Add("127.0.0.2:8082")
	SrcServers.Add("127.0.0.3:8083")
	SrcServers.Add("127.0.0.4:8084")
	(*SrcServers)[0].Status = OK // 127.0.0.4:8084
	(*SrcServers)[1].Status = OK // 127.0.0.3:8083
	(*SrcServers)[2].Status = OK // 127.0.0.2:8082
	(*SrcServers)[3].Status = OK // 127.0.0.1:8081

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

// all meta : A, B, C, D, E, F
// A,B,C,D,E,F 는 이미 서버에 있는 파일
func makeFileMetaMapABCDEF() (FileMetaPtrMap, FileMetaPtrMap) {
	fmm := make(FileMetaPtrMap)

	// put grade, size , and severs
	fmm["A.mpg"] = &common.FileMeta{
		Name:  "A.mpg",
		Grade: 1, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1}}

	fmm["B.mpg"] = &common.FileMeta{
		Name:  "B.mpg",
		Grade: 2, Size: 100, RisingHit: 0,
		ServerCount: 2,
		ServerIPs:   map[string]int{"127.0.0.1": 1, "127.0.0.2": 1}}

	fmm["C.mpg"] = &common.FileMeta{
		Name:  "C.mpg",
		Grade: 3, Size: 100, RisingHit: 0,
		ServerCount: 2,
		ServerIPs:   map[string]int{"127.0.0.1": 1, "127.0.0.2": 1}}

	fmm["D.mpg"] = &common.FileMeta{
		Name:  "D.mpg",
		Grade: 4, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1}}

	fmm["E.mpg"] = &common.FileMeta{
		Name:  "E.mpg",
		Grade: 5, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.2": 1}}

	fmm["F.mpg"] = &common.FileMeta{
		Name:  "F.mpg",
		Grade: 6, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.2": 1}}

	dupfmm := make(FileMetaPtrMap)
	dupfmm["B.mpg"] = fmm["B.mpg"]
	dupfmm["C.mpg"] = fmm["C.mpg"]

	return fmm, dupfmm
}

// A,B,C,D,E,F,G,H,I 는 이미 서버에 있는 파일
// B, C 는 두 개의 서버에 있는 파일
// J,K,L,M,N,O 는 서버에 없는 파일
func makeFileMetaMapABCDEFGHIJKLMNO() (FileMetaPtrMap, FileMetaPtrMap) {
	fmm := make(FileMetaPtrMap)

	// put grade, size , and severs
	fmm["A.mpg"] = &common.FileMeta{
		Name:  "A.mpg",
		Grade: 1, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1}}

	fmm["B.mpg"] = &common.FileMeta{
		Name:  "B.mpg",
		Grade: 2, Size: 100, RisingHit: 0,
		ServerCount: 2,
		ServerIPs:   map[string]int{"127.0.0.1": 1, "127.0.0.2": 1}}

	fmm["C.mpg"] = &common.FileMeta{
		Name:  "C.mpg",
		Grade: 3, Size: 100, RisingHit: 0,
		ServerCount: 2,
		ServerIPs:   map[string]int{"127.0.0.1": 1, "127.0.0.2": 1}}

	fmm["D.mpg"] = &common.FileMeta{
		Name:  "D.mpg",
		Grade: 4, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1}}

	fmm["E.mpg"] = &common.FileMeta{
		Name:  "E.mpg",
		Grade: 5, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.2": 1}}

	fmm["F.mpg"] = &common.FileMeta{
		Name:  "F.mpg",
		Grade: 6, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.2": 1}}

	fmm["G.mpg"] = &common.FileMeta{
		Name:  "G.mpg",
		Grade: 7, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.3": 1}}

	fmm["H.mpg"] = &common.FileMeta{
		Name:  "H.mpg",
		Grade: 8, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.3": 1}}

	fmm["I.mpg"] = &common.FileMeta{
		Name:  "I.mpg",
		Grade: 9, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.5": 1}}

	fmm["J.mpg"] = &common.FileMeta{
		Name:  "J.mpg",
		Grade: 10, Size: 100, RisingHit: 0,
		ServerCount: 0,
		ServerIPs:   map[string]int{}}

	fmm["K.mpg"] = &common.FileMeta{
		Name:  "K.mpg",
		Grade: 11, Size: 100, RisingHit: 0,
		ServerCount: 0,
		ServerIPs:   map[string]int{}}

	fmm["L.mpg"] = &common.FileMeta{
		Name:  "L.mpg",
		Grade: 12, Size: 100, RisingHit: 0,
		ServerCount: 0,
		ServerIPs:   map[string]int{}}

	fmm["M.mpg"] = &common.FileMeta{
		Name:  "M.mpg",
		Grade: 13, Size: 100, RisingHit: 0,
		ServerCount: 0,
		ServerIPs:   map[string]int{}}

	fmm["N.mpg"] = &common.FileMeta{
		Name:  "N.mpg",
		Grade: 14, Size: 100, RisingHit: 0,
		ServerCount: 0,
		ServerIPs:   map[string]int{}}

	fmm["O.mpg"] = &common.FileMeta{
		Name:  "O.mpg",
		Grade: 15, Size: 100, RisingHit: 0,
		ServerCount: 0,
		ServerIPs:   map[string]int{}}

	dupfmm := make(FileMetaPtrMap)
	dupfmm["B.mpg"] = fmm["B.mpg"]
	dupfmm["C.mpg"] = fmm["C.mpg"]

	return fmm, dupfmm
}

// 서버 위치 정보는 없는 파일
func makeFileMetaMap(chs, che byte, n uint) (FileMetaPtrMap, FileMetaPtrMap) {
	fmm := make(FileMetaPtrMap)

	ixl := uint(che - chs + 1)
	for i := uint(0); i < n; i++ {
		ix := i % ixl
		fn := fmt.Sprintf("%s", string(chs+byte(ix)))
		if i < ixl {
			fn = fn + ".mpg"
		} else {
			ix2 := i / ixl
			fn = fn + strconv.FormatUint(uint64(ix2), 10) + ".mpg"
		}
		// put grade, size , and severs
		fmm[fn] = &common.FileMeta{
			Name:  fn,
			Grade: 1, Size: 100, RisingHit: 0,
			ServerCount: 0,
			ServerIPs:   map[string]int{}}
	}
	dupfmm := make(FileMetaPtrMap)

	return fmm, dupfmm
}

func makeRisingHitFileMap(files []string) map[string]int {
	risingHitFileMap := make(map[string]int)
	i := 1
	for _, file := range files {
		risingHitFileMap[file] = i
		i++
	}
	return risingHitFileMap
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
	defer heartbeater.Release()

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
	makePresetS1D4()

	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

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

	cleanTask(ts.GetTaskList())
	// 2개의 DONE task 삭제된 후 task 개수
	// t1, t2 삭제
	assert.Equal(t, 3, len(ts.TaskMap))
	assert.NotContains(t, ts.TaskMap, t1.ID)
	assert.NotContains(t, ts.TaskMap, t2.ID)

	(*DstServers)[2].Status = NOTOK
	cleanTask(ts.GetTaskList())
	// dest server 127.0.0.3:8080 의 상태가 NOTOK 로 바뀌어서 삭제됨
	// 따라서 t3(dest가 127.0.0.3:8080인 task) 삭제됨
	assert.Equal(t, 2, len(ts.TaskMap))
	assert.NotContains(t, ts.TaskMap, t3.ID)

	SetTaskTimeout(time.Second * 1)
	time.Sleep(time.Second * 2)
	cleanTask(ts.GetTaskList())
	// 2개 중 2개의 task 가 timeout으로 삭제됨
	assert.Equal(t, 0, len(ts.TaskMap))

}

func Test_getAllHostStatusAndcleanTask(t *testing.T) {
	makePresetS1D4()

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

	d2 := "127.0.0.2:18082"
	d2files := []string{"B.mpg", "C.mpg", "E.mpg", "F.mpg", "SERVER2.mpg"}
	d2du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	d3 := "127.0.0.3:18083"
	d3files := []string{"SERVER3.mpg", "G.mpg"}
	d3du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	d4 := "127.0.0.4:18084"
	d4files := []string{}
	d4du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	d5 := "127.0.0.5:18085"
	d5files := []string{"SERVER1-5.mpg"}
	d5du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	cfw2 := cfw(d2, d2du, d2files)
	cfw2.Start()
	defer cfw2.Close()
	cfw3 := cfw(d3, d3du, d3files)
	cfw3.Start()
	defer cfw3.Close()
	cfw4 := cfw(d4, d4du, d4files)
	cfw4.Start()
	defer cfw4.Close()
	cfw5 := cfw(d5, d5du, d5files)
	cfw5.Start()
	defer cfw5.Close()

	defer heartbeater.Release()
	heartbeater.Add(s1)

	// heartbeater가 동작하기 전이라, status가 NOTOK
	SrcServers.getAllHostStatus()
	assert.Equal(t, NOTOK, (*SrcServers)[0].Status)

	// heartbeater가 동작하기 전이라, status가 NOTOK
	DstServers.getAllHostStatus()
	assert.Equal(t, NOTOK, (*DstServers)[0].Status)
	assert.Equal(t, NOTOK, (*DstServers)[1].Status)
	assert.Equal(t, NOTOK, (*DstServers)[2].Status)
	assert.Equal(t, NOTOK, (*DstServers)[3].Status)
	assert.Equal(t, NOTOK, (*DstServers)[4].Status)

	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.3:18083"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.4:18084"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.5:18085"})

	// src status 가 NOT OK 여서 모두 삭제
	cleanTask(tasks.GetTaskList())
	assert.Equal(t, 0, len(tasks.TaskMap))

	heartbeater.Heartbeat()
	// heartbeater가 동작하고 나면, s1 status가 OK
	SrcServers.getAllHostStatus()
	assert.Equal(t, OK, (*SrcServers)[0].Status)

	// heartbeater가 동작하기 전이라, status가 NOTOK
	DstServers.getAllHostStatus()
	assert.Equal(t, NOTOK, (*DstServers)[0].Status)

	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.3:18083"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.4:18084"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.5:18085"})

	// dest status 가 NOT OK 여서 모두 삭제
	cleanTask(tasks.GetTaskList())
	assert.Equal(t, 0, len(tasks.TaskMap))

	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.3:18083"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.4:18084"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.5:18085"})

	heartbeater.Add(d1)
	heartbeater.Add(d2)
	heartbeater.Add(d3)
	heartbeater.Add(d4)
	heartbeater.Add(d5)
	heartbeater.Heartbeat()

	SrcServers.getAllHostStatus()
	assert.Equal(t, OK, (*SrcServers)[0].Status)

	// heartbeater가 동작하고 나면, status가 OK
	DstServers.getAllHostStatus()
	assert.Equal(t, OK, (*DstServers)[0].Status)
	assert.Equal(t, OK, (*DstServers)[1].Status)
	assert.Equal(t, OK, (*DstServers)[2].Status)
	assert.Equal(t, OK, (*DstServers)[3].Status)
	assert.Equal(t, OK, (*DstServers)[4].Status)

	// 안지워짐
	cleanTask(ts.GetTaskList())
	assert.Equal(t, 5, len(tasks.TaskMap))

	SetTaskTimeout(time.Second * 1)
	time.Sleep(time.Second * 1)
	cleanTask(ts.GetTaskList())
	// 모든 task 가 timeout으로 삭제됨
	assert.Equal(t, 0, len(ts.TaskMap))

}

func Test_setSelected(t *testing.T) {
	makePresetS1D4()

	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

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
	makePresetS1D4()

	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

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
	makePresetS1D4()

	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

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
	makePresetS1D4()

	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

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
	t2 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
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
	makePresetS1D4()

	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

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
	t2 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
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
	makePresetS1D4()

	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

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
	t2 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
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

	fs := make(FileFreqMap)
	collectRemoteFileList(dsthosts, fs)

	assert.Equal(t, 3, len(fs))
	assert.Equal(t, 3, int(fs["A.mpg"]))
	assert.Equal(t, 3, int(fs["B.mpg"]))
	assert.Equal(t, 3, int(fs["C.mpg"]))
}

func Test_getFilesInTasks(t *testing.T) {
	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

	t1 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t.Log(t1)
	t2 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.2:18082"})
	t.Log(t2)

	files := getFilesInTasks(tasks.GetTaskList())
	t.Log(files)

	assert.Equal(t, 1, len(files))
	assert.Contains(t, files, "A.mpg")
	assert.Equal(t, 2, int(files["A.mpg"]))

	t3 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.3:18083"})
	t.Log(t3)
	t4 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.4:18084"})
	t.Log(t4)
	t5 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.5:18085"})
	t.Log(t5)

	files = getFilesInTasks(tasks.GetTaskList())
	t.Log(files)

	assert.Equal(t, 4, len(files))
	assert.Contains(t, files, "A.mpg")
	assert.Equal(t, 2, int(files["A.mpg"]))
	assert.Contains(t, files, "C.mpg")
	assert.Equal(t, 1, int(files["C.mpg"]))
	assert.Contains(t, files, "D.mpg")
	assert.Equal(t, 1, int(files["D.mpg"]))
	assert.Contains(t, files, "E.mpg")
	assert.Equal(t, 1, int(files["E.mpg"]))

}

func Test_updateFileMetasForRisingHitFiles(t *testing.T) {
	// all meta : A, B, C, D, E, F
	allfmm, _ := makeFileMetaMapABCDEF()
	rhfiles := []string{"E.mpg", "F.mpg", "G.mpg"}
	rhitfmm := makeRisingHitFileMap(rhfiles)

	updateFileMetasForRisingHitsFiles(allfmm, rhitfmm)

	// all file meta 에 없는 rising hit file 은 제외되어,
	// E, F 의 rising hit 값이 update 됨
	assert.Equal(t, 1, allfmm["E.mpg"].RisingHit)
	assert.Equal(t, 2, allfmm["F.mpg"].RisingHit)

	// 다른 file 은 그대로
	assert.Equal(t, 0, allfmm["A.mpg"].RisingHit)
	assert.Equal(t, 0, allfmm["B.mpg"].RisingHit)
	assert.Equal(t, 0, allfmm["C.mpg"].RisingHit)
	assert.Equal(t, 0, allfmm["D.mpg"].RisingHit)

}

// src 정보를 file meta 에 update 해주는 함수 test
func Test_updateFileMetaForSrcFilePath(t *testing.T) {
	base := "testsourcefolder"
	SourcePath.Add(base)
	files := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg",
		"G.mpg", "H.mpg", "I.mpg", "J.mpg", "K.mpg"}
	// source path 에 파일 생성
	for _, f := range files {
		createfile(base, f)
	}
	defer deletefile(base, "")

	// all meta : A, B, C, D, E, F
	allfmm, _ := makeFileMetaMapABCDEF()
	assert.Equal(t, true, updateFileMetaForSrcFilePath(allfmm["A.mpg"]))
	assert.Equal(t, filepath.Join(base, "A.mpg"), allfmm["A.mpg"].SrcFilePath)

	assert.Equal(t, true, updateFileMetaForSrcFilePath(allfmm["B.mpg"]))
	assert.Equal(t, filepath.Join(base, "B.mpg"), allfmm["B.mpg"].SrcFilePath)

	assert.Equal(t, true, updateFileMetaForSrcFilePath(allfmm["C.mpg"]))
	assert.Equal(t, filepath.Join(base, "C.mpg"), allfmm["C.mpg"].SrcFilePath)

	assert.Equal(t, true, updateFileMetaForSrcFilePath(allfmm["D.mpg"]))
	assert.Equal(t, filepath.Join(base, "D.mpg"), allfmm["D.mpg"].SrcFilePath)

	assert.Equal(t, false, updateFileMetaForSrcFilePath(allfmm["E.mpg"]))
	assert.Equal(t, "", allfmm["E.mpg"].SrcFilePath)

	assert.Equal(t, false, updateFileMetaForSrcFilePath(allfmm["F.mpg"]))
	assert.Equal(t, "", allfmm["F.mpg"].SrcFilePath)
}

// 실제 src directory에 있다고 해도
// src 정보를 file meta 에 update 해주지 않으면,
// checkForTask 함수는 실패함
func Test_checkForTask(t *testing.T) {
	base := "testsourcefolder"
	SourcePath.Add(base)
	files := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg", "E.mpg", "F.mpg",
		"G.mpg", "H.mpg", "I.mpg", "J.mpg", "K.mpg"}
	// source path 에 파일 생성
	for _, f := range files {
		createfile(base, f)
	}
	defer deletefile(base, "")

	//
	allfmm, _ := makeFileMetaMapABCDEFGHIJKLMNO()

	// source path 에 없는 파일
	deletefile(base, "C.mpg")

	// 광고 파일
	ignores := []string{"AD1", "H", "AD2", "I", "AD3"}
	SetIgnorePrefixes(ignores)

	// 배포에 사용 중인 파일
	taskfilenames := make(FileFreqMap)
	taskfilenames["DANGLING1.mpg"]++
	taskfilenames["A.mpg"]++
	taskfilenames["DANGLING3.mpg"]++
	taskfilenames["F.mpg"]++
	taskfilenames["G.mpg"]++
	taskfilenames["DANGLING2.mpg"]++

	// 서버에 이미 있는 파일
	serverfiles := make(FileFreqMap)
	serverfiles["STRANGE1.mpg"]++
	serverfiles["B.mpg"]++
	serverfiles["K.mpg"]++
	serverfiles["STRANGE2.mpg"]++
	serverfiles["STRANGE2.mpg"]++

	// 실제 src 에 있다고 해도 src 정보를 update 해주지 않으면,
	// checkForTask 함수는 실패함
	// A, B, C, D, E, F, G, H, I 는 이미 서버에 있는 파일이므로, 배포에서 제외됨

	// 배포에 사용되므로, false
	updateFileMetaForSrcFilePath(allfmm["A.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["A.mpg"], taskfilenames, serverfiles))

	// 서버에 이미 있으므로, false
	updateFileMetaForSrcFilePath(allfmm["B.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["B.mpg"], taskfilenames, serverfiles))

	// source path 에 없으므로 false
	updateFileMetaForSrcFilePath(allfmm["C.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["C.mpg"], taskfilenames, serverfiles))

	// 실제 src 에 있다고 해도 src 정보를 update 해주지 않으면,
	// checkForTask 함수는 실패함
	assert.Equal(t, false, checkForTask(allfmm["D.mpg"], taskfilenames, serverfiles))

	// src path 정보를 update 해주어도
	// D는 이미 서버에 있는 파일이어서 실패함
	updateFileMetaForSrcFilePath(allfmm["D.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["D.mpg"], taskfilenames, serverfiles))

	// src path 정보를 update 해주어도
	// E는 이미 서버에 있는 파일이어서 실패함
	updateFileMetaForSrcFilePath(allfmm["E.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["E.mpg"], taskfilenames, serverfiles))

	// 배포에 사용되므로, false
	// E는 이미 서버에 있는 파일이어서 실패함
	updateFileMetaForSrcFilePath(allfmm["F.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["F.mpg"], taskfilenames, serverfiles))

	// 배포에 사용되므로, false
	// E는 이미 서버에 있는 파일이어서 실패함
	updateFileMetaForSrcFilePath(allfmm["G.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["G.mpg"], taskfilenames, serverfiles))

	// ignore prefix, false
	// E는 이미 서버에 있는 파일이어서 실패함
	updateFileMetaForSrcFilePath(allfmm["H.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["H.mpg"], taskfilenames, serverfiles))

	// ignore prefix, false
	// E는 이미 서버에 있는 파일이어서 실패함
	updateFileMetaForSrcFilePath(allfmm["I.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["I.mpg"], taskfilenames, serverfiles))

	// 배포 대상
	updateFileMetaForSrcFilePath(allfmm["J.mpg"])
	assert.Equal(t, true, checkForTask(allfmm["J.mpg"], taskfilenames, serverfiles))

	// 서버에 이미 있는 파일, false
	updateFileMetaForSrcFilePath(allfmm["K.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["K.mpg"], taskfilenames, serverfiles))

	// source path 에 없으므로 false
	updateFileMetaForSrcFilePath(allfmm["L.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["L.mpg"], taskfilenames, serverfiles))

	// source path 에 없으므로 false
	updateFileMetaForSrcFilePath(allfmm["M.mpg"])
	assert.Equal(t, false, checkForTask(allfmm["M.mpg"], taskfilenames, serverfiles))
}

func Test_getSortedFileMetaListForTask(t *testing.T) {
	allfmm, _ := makeFileMetaMapABCDEFGHIJKLMNO()
	rhfiles := []string{"E.mpg", "F.mpg", "J.mpg", "RH1.mpg", "M.mpg", "H.mpg", "O.mpg", "RH2.mpg"}
	rhitfmm := makeRisingHitFileMap(rhfiles)

	sortedfmm := getSortedFileMetaListForTask(allfmm, rhitfmm)

	for _, fm := range sortedfmm {
		t.Log(*fm)
	}
	assert.Equal(t, 15, len(sortedfmm))
	assert.Equal(t, sortedfmm[0].Name, "O.mpg")
	assert.Equal(t, sortedfmm[1].Name, "H.mpg")
	assert.Equal(t, sortedfmm[2].Name, "M.mpg")
	assert.Equal(t, sortedfmm[3].Name, "J.mpg")
	assert.Equal(t, sortedfmm[4].Name, "F.mpg")
	assert.Equal(t, sortedfmm[5].Name, "E.mpg")
	assert.Equal(t, sortedfmm[6].Name, "A.mpg")
	assert.Equal(t, sortedfmm[7].Name, "B.mpg")
	assert.Equal(t, sortedfmm[8].Name, "C.mpg")
	assert.Equal(t, sortedfmm[9].Name, "D.mpg")
	assert.Equal(t, sortedfmm[10].Name, "G.mpg")
	assert.Equal(t, sortedfmm[11].Name, "I.mpg")
	assert.Equal(t, sortedfmm[12].Name, "K.mpg")
	assert.Equal(t, sortedfmm[13].Name, "L.mpg")
	assert.Equal(t, sortedfmm[14].Name, "N.mpg")

}

func Test_selectSourceServer(t *testing.T) {
	srcs := new(SrcHosts)

	//srcs 는 sort 됨
	srcs.Add("127.0.0.1:18001")
	srcs.Add("127.0.0.2:18001")
	srcs.Add("127.0.0.3:18001")

	// 모든 status 이 NOTOK 이므로, select 되지 않음
	for i := 0; i < 3; i++ {
		_, found := srcs.selectSourceServer()
		assert.Equal(t, false, found)
	}

	// (*srcs)[0] status 값이 OK 이므로,(*srcs)[0]이select 됨
	(*srcs)[0].Status = OK
	for i := 0; i < 3; i++ {
		srcs.selectSourceServer()
	}
	assert.Equal(t, true, (*srcs)[0].selected)
	assert.Equal(t, false, (*srcs)[1].selected)
	assert.Equal(t, false, (*srcs)[2].selected)

	// 1,2 번도 Status 값이 OK 이므로, select 됨
	(*srcs)[1].Status = OK
	(*srcs)[2].Status = OK
	for i := 0; i < 3; i++ {
		srcs.selectSourceServer()
	}

	// sort 된 상태에서 "127.0.0.3:18001" 부터 선택됨
	assert.Equal(t, "127.0.0.3:18001", (*srcs)[0].Addr)
	assert.Equal(t, true, (*srcs)[0].selected)

	assert.Equal(t, "127.0.0.2:18001", (*srcs)[1].Addr)
	assert.Equal(t, true, (*srcs)[1].selected)

	assert.Equal(t, "127.0.0.1:18001", (*srcs)[2].Addr)
	assert.Equal(t, true, (*srcs)[2].selected)

	// 이미 3개의 src 를 모두 사용했으모로 src 가 없어야 한다.
	_, found := srcs.selectSourceServer()
	assert.Equal(t, false, found)

	// 한 번 더 실행해도 같으 결과
	_, found = srcs.selectSourceServer()
	assert.Equal(t, false, found)
}

func Test_runWithInfo(t *testing.T) {

	makePresetS4D5()

	s1 := "127.0.0.1:8081"
	s1files := []string{}
	s1du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	cfws1 := cfw(s1, s1du, s1files)
	cfws1.Start()
	defer cfws1.Close()

	s2 := "127.0.0.2:8082"
	s2files := []string{}
	s2du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	cfws2 := cfw(s2, s2du, s2files)
	cfws2.Start()
	defer cfws2.Close()

	s3 := "127.0.0.3:8083"
	s3files := []string{}
	s3du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	cfws3 := cfw(s3, s3du, s3files)
	cfws3.Start()
	defer cfws3.Close()

	s4 := "127.0.0.4:8084"
	s4files := []string{}
	s4du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	cfws4 := cfw(s4, s4du, s4files)
	cfws4.Start()
	defer cfws4.Close()

	//////////////////////////////////////////////////////////////////////////////
	d1 := "127.0.0.1:18081"
	d1files := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg", "SERVER1-5.mpg"}
	d1du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	d2 := "127.0.0.2:18082"
	d2files := []string{"B.mpg", "C.mpg", "E.mpg", "F.mpg", "SERVER2.mpg"}
	d2du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	d3 := "127.0.0.3:18083"
	d3files := []string{"SERVER3.mpg", "G.mpg"}
	d3du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	d4 := "127.0.0.4:18084"
	d4files := []string{}
	d4du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	d5 := "127.0.0.5:18085"
	d5files := []string{"SERVER1-5.mpg"}
	d5du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	cfw1 := cfw(d1, d1du, d1files)
	cfw1.Start()
	defer cfw1.Close()

	cfw2 := cfw(d2, d2du, d2files)
	cfw2.Start()
	defer cfw2.Close()

	cfw3 := cfw(d3, d3du, d3files)
	cfw3.Start()
	defer cfw3.Close()

	cfw4 := cfw(d4, d4du, d4files)
	cfw4.Start()
	defer cfw4.Close()

	cfw5 := cfw(d5, d5du, d5files)
	cfw5.Start()
	defer cfw5.Close()

	base := "testsourcefolder"
	SourcePath.Add(base)

	// source path 에 파일 생성
	for _, f := range d1files {
		createfile(base, f)
	}
	for _, f := range d2files {
		createfile(base, f)
	}
	for _, f := range d3files {
		createfile(base, f)
	}
	for _, f := range d4files {
		createfile(base, f)
	}
	for _, f := range d5files {
		createfile(base, f)
	}

	createfile(base, "H.mpg")
	createfile(base, "I.mpg")
	createfile(base, "J.mpg")
	//createfile(base, "K.mpg")
	createfile(base, "L.mpg")
	createfile(base, "M.mpg")
	createfile(base, "N.mpg")
	//createfile(base, "O.mpg")
	createfile(base, "P.mpg")

	// C.mpg 는 source path 에서 삭제
	deletefile(base, "C.mpg")

	defer deletefile(base, "")

	serverfs := make(FileFreqMap)
	collectRemoteFileList(DstServers, serverfs)
	assert.Equal(t, 10, len(serverfs))
	assert.Equal(t, 1, int(serverfs["A.mpg"]))
	assert.Equal(t, 2, int(serverfs["B.mpg"]))
	assert.Equal(t, 2, int(serverfs["C.mpg"]))
	assert.Equal(t, 1, int(serverfs["D.mpg"]))
	assert.Equal(t, 1, int(serverfs["E.mpg"]))
	assert.Equal(t, 1, int(serverfs["F.mpg"]))
	assert.Equal(t, 1, int(serverfs["G.mpg"]))
	assert.Equal(t, 2, int(serverfs["SERVER1-5.mpg"]))
	assert.Equal(t, 1, int(serverfs["SERVER2.mpg"]))
	assert.Equal(t, 1, int(serverfs["SERVER3.mpg"]))

	allfmm, _ := makeFileMetaMapABCDEFGHIJKLMNO()

	ignores := []string{"AD1", "H", "I", "AD2"}
	SetIgnorePrefixes(ignores)

	rhfiles := []string{"E.mpg", "F.mpg", "J.mpg", "RH1.mpg", "M.mpg", "H.mpg", "O.mpg", "RH2.mpg"}
	rhitfmm := makeRisingHitFileMap(rhfiles)

	defer heartbeater.Release()

	// heartbeater에 등록
	heartbeater.Add(s1)
	heartbeater.Add(s2)
	heartbeater.Add(s3)
	heartbeater.Add(s4)

	heartbeater.Add(d1)
	heartbeater.Add(d2)
	heartbeater.Add(d3)
	heartbeater.Add(d4)
	heartbeater.Add(d5)

	var t1, t2, t3 Task

	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

	t.Log("tasks 1-------------------------------------")
	t1 = ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t.Log(t1)
	t2 = ts.CreateTask(&Task{SrcIP: "127.0.0.2", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.2:8082", DstAddr: "127.0.0.2:18082"})
	t.Log(t2)
	t3 = ts.CreateTask(&Task{SrcIP: "127.0.0.3", FilePath: "/data2/DANGLING1.mpg",
		FileName: "DANGLING1.mpg", SrcAddr: "127.0.0.3:8083", DstAddr: "127.0.0.2:18082"})
	t.Log(t3)

	runWithInfo(allfmm, rhitfmm)

	t.Log("tasks 2-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	assert.Equal(t, 0, len(tasks.TaskMap))

	t.Log("tasks 3-------------------------------------")
	t1 = ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t.Log(t1)
	t2 = ts.CreateTask(&Task{SrcIP: "127.0.0.2", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.2:8082", DstAddr: "127.0.0.2:18082"})
	t.Log(t2)
	t3 = ts.CreateTask(&Task{SrcIP: "127.0.0.3", FilePath: "/data2/DANGLING1.mpg",
		FileName: "DANGLING1.mpg", SrcAddr: "127.0.0.3:8083", DstAddr: "127.0.0.2:18082"})
	t.Log(t3)

	assert.Equal(t, 3, len(tasks.TaskMap))
	assertTask(t, tasks, "A.mpg", s1, d1)
	assertTask(t, tasks, "B.mpg", s2, d2)
	assertTask(t, tasks, "DANGLING1.mpg", s3, d2)

	heartbeater.Heartbeat()
	// src server와 dst server의 heartbeat 가 살아난 후에는 task 가 그대로 있음

	// 배포 대상 파일 :
	// A, B, C, D, E, F, G 는 이미 서버에 있으므로 제외
	// I, H 는 ignore.prefix 이기 때문에 제외

	// RH2.mpg, RH1.mpg는	all meta 에 없으므로 제외
	// O.mpg 는 source path에 없으므로 제외
	// rising hits file인 M.mpg, J.mpg는 우선순위가 높음

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3,s2,s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "A.mpg", s1, d1
	// 배포 중 : "B.mpg", s2, d2
	// 배포 중 : "DANGLING1.mpg", s3, d2

	// 배포 중이 아닌 서버 : s4
	// 배포 중이 아닌 서버 : d5, d4, d3
	// 배포 중이 아닌 파일 : M, J, L, N

	// M.mpg, s4 -> d5 task 가 하나 만들어져야 함
	runWithInfo(allfmm, rhitfmm)

	t.Log("tasks 4-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3,s2,s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 기존의 task는 그대로 있고
	// M.mpg, s4 -> d5 task 가 하나 만들어져야 함
	assert.Equal(t, 4, len(tasks.TaskMap))

	// file, src, dst 로 간접 확인
	// 기존 task 에
	assert.Equal(t, 4, len(tasks.TaskMap))
	assertTask(t, tasks, "A.mpg", s1, d1)
	assertTask(t, tasks, "B.mpg", s2, d2)
	assertTask(t, tasks, "DANGLING1.mpg", s3, d2)
	// M.mpg, s4 -> d5 task 가 하나 만들어져야 함
	assertTask(t, tasks, "M.mpg", s4, d5)

	//////////////////////////////////////////////////////////////////////////////
	// t1이 DONE이 된 경우
	tasks.UpdateStatus(t1.ID, DONE)

	runWithInfo(allfmm, rhitfmm)

	t.Log("tasks 5-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	assert.Equal(t, 4, len(tasks.TaskMap))

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3,s2,s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "B.mpg", s2, d2
	// 배포 중 : "DANGLING1.mpg", s3, d2
	// 배포 중 : "M.mpg", s4, d5

	// 배포 중이 아닌 서버 : s1
	// 배포 중이 아닌 서버 : d4, d3, d1
	// 배포 중이 아닌 파일 : J, L, N

	// file, src, dst 로 간접 확인
	// 기존의 t1 task 가 없어지고, 새로운 task 생성
	assert.Equal(t, 4, len(tasks.TaskMap))
	assertTask(t, tasks, "B.mpg", s2, d2)
	assertTask(t, tasks, "DANGLING1.mpg", s3, d2)
	assertTask(t, tasks, "M.mpg", s4, d5)
	// J.mpg, s1 -> d4 task 가 하나 만들어져야 함
	assertTask(t, tasks, "J.mpg", s1, d4)

	//////////////////////////////////////////////////////////////////////////////
	// t3 이 DONE이 된 경우
	updateStatus(t, tasks, "DANGLING1.mpg", DONE)

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3, s2, s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// DANGLING1.mpg, s3, d2 가 삭제되고 나면

	// 배포 중 : "B.mpg", s2, d2
	// 배포 중 : "M.mpg", s4, d5
	// 배포 중 : "J.mpg", s1, d4

	// 배포 중이 아닌 서버 : s3
	// 배포 중이 아닌 서버 : d3, d1
	// 배포 중이 아닌 파일 : L, N

	runWithInfo(allfmm, rhitfmm)

	t.Log("tasks 6-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	// L.mpg, s3 -> d3 task 가 하나 만들어져야 함
	assert.Equal(t, 4, len(tasks.TaskMap))
	assertTask(t, tasks, "B.mpg", s2, d2)
	assertTask(t, tasks, "M.mpg", s4, d5)
	assertTask(t, tasks, "J.mpg", s1, d4)
	assertTask(t, tasks, "L.mpg", s3, d3)

	//////////////////////////////////////////////////////////////////////////////
	// B가 DONE 된 경우
	// M이 TIMEOUT된 경우, d5와 통신은 되는 경우
	updateStatus(t, tasks, "B.mpg", DONE)
	updateStatus(t, tasks, "M.mpg", TIMEOUT)

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3, s2, s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "J.mpg", s1, d4
	// 배포 중 : "L.mpg", s3, d3

	// 배포 중이 아닌 서버 : s4, s2
	// 배포 중이 아닌 서버 : d5, d2, d1
	// 배포 중이 아닌 파일 : M, N
	// M은 배포 실패햇다고 생각하고, 다시 배포 task에  넣을 수 있다고 가정함
	runWithInfo(allfmm, rhitfmm)

	t.Log("tasks 7-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	assert.Equal(t, 4, len(tasks.TaskMap))
	assertTask(t, tasks, "J.mpg", s1, d4)
	assertTask(t, tasks, "L.mpg", s3, d3)
	// M, s4, d5 추가
	// N, s2, d2 추가
	assertTask(t, tasks, "M.mpg", s4, d5)
	assertTask(t, tasks, "N.mpg", s2, d2)

	//////////////////////////////////////////////////////////////////////////////
	// d2와 통신 실패
	// d5와 통신 실패
	// d3와 통신 실패
	// 통신에 실패하면 task가 삭제됨

	// heartbeater에서 제거만 되도, heartbeat 결과를 가져올 수 없어서
	// 통신에 실패한 것으로 간주함
	heartbeater.Delete(d2)
	heartbeater.Delete(d5)
	heartbeater.Delete(d3)

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3, s2, s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "J.mpg", s1, d4

	// 배포 중이 아닌 서버 : s4, s3, s2
	// 배포 중이 아닌 서버 : d1
	// 통신 실패 : (d5, d3, d2)
	// 배포 중이 아닌 파일 : M, L, N
	runWithInfo(allfmm, rhitfmm)

	t.Log("tasks 8-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	assert.Equal(t, 4, len(tasks.TaskMap))
	assertTask(t, tasks, "J.mpg", s1, d4)
	// 여러 source 서버에서 하나의 destination 서버로 배포가 가능
	// M, s4, d1 추가
	// L, s3, d1 추가
	// N, s2, d1 추가
	assertTask(t, tasks, "M.mpg", s4, d1)
	assertTask(t, tasks, "L.mpg", s3, d1)
	assertTask(t, tasks, "N.mpg", s2, d1)

	//////////////////////////////////////////////////////////////////////////////
	// s1와 통신 실패
	// s2와 통신 실패
	// s3와 통신 실패
	// d4과 통신 실패
	// 통신에 실패하면 task가 삭제됨

	// heartbeater에서 제거만 되도, heartbeat 결과를 가져올 수 없어서
	// 통신에 실패한 것으로 간주함
	heartbeater.Delete(s1)
	heartbeater.Delete(s2)
	heartbeater.Delete(s3)

	heartbeater.Delete(d4)

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3, s2, s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "M.mpg", s4, d1

	// 배포 중이 아닌 서버 :
	// 배포 중이 아닌 서버 :
	// 통신 실패 : (s3, s2, s1)
	// 통신 실패 : (d5, d4, d3, d2)
	// 배포 중이 아닌 파일 : J. L, N
	runWithInfo(allfmm, rhitfmm)

	t.Log("tasks 9-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}
	// M만 남고, 사용 가능한 src서버가 없어서 새로 만들어지지는 않음
	assert.Equal(t, 1, len(tasks.TaskMap))
	assertTask(t, tasks, "M.mpg", s4, d1)

	//////////////////////////////////////////////////////////////////////////////
	// s1은 통신 성공
	heartbeater.Add(s1)
	heartbeater.Heartbeat()

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3, s2, s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "M.mpg", s4, d1

	// 배포 중이 아닌 서버 : s1
	// 배포 중이 아닌 서버 :
	// 통신 실패 : (s3, s2)
	// 통신 실패 : (d5, d4, d3, d2)
	// 배포 중이 아닌 파일 : J. L, N
	runWithInfo(allfmm, rhitfmm)

	t.Log("tasks 10-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}
	// M만 남고,
	// 사용가능한 src 서버 s1이 있지만,
	// 사용가능한 dst 서버가 없어서 새로 만들어지지는 않음
	assert.Equal(t, 1, len(tasks.TaskMap))
	assertTask(t, tasks, "M.mpg", s4, d1)
}

func Test_run(t *testing.T) {

	makePresetS4D5()

	s1 := "127.0.0.1:8081"
	s1files := []string{}
	s1du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	cfws1 := cfw(s1, s1du, s1files)
	cfws1.Start()
	defer cfws1.Close()

	s2 := "127.0.0.2:8082"
	s2files := []string{}
	s2du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	cfws2 := cfw(s2, s2du, s2files)
	cfws2.Start()
	defer cfws2.Close()

	s3 := "127.0.0.3:8083"
	s3files := []string{}
	s3du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	cfws3 := cfw(s3, s3du, s3files)
	cfws3.Start()
	defer cfws3.Close()

	s4 := "127.0.0.4:8084"
	s4files := []string{}
	s4du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	cfws4 := cfw(s4, s4du, s4files)
	cfws4.Start()
	defer cfws4.Close()

	//////////////////////////////////////////////////////////////////////////////
	d1 := "127.0.0.1:18081"
	d1files := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg", "SERVER1-5.mpg"}
	d1du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	d2 := "127.0.0.2:18082"
	d2files := []string{"B.mpg", "C.mpg", "E.mpg", "F.mpg", "SERVER2.mpg"}
	d2du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	d3 := "127.0.0.3:18083"
	d3files := []string{"SERVER3.mpg", "G.mpg"}
	d3du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	d4 := "127.0.0.4:18084"
	d4files := []string{}
	d4du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	d5 := "127.0.0.5:18085"
	d5files := []string{"SERVER1-5.mpg"}
	d5du := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	cfw1 := cfw(d1, d1du, d1files)
	cfw1.Start()
	defer cfw1.Close()

	cfw2 := cfw(d2, d2du, d2files)
	cfw2.Start()
	defer cfw2.Close()

	cfw3 := cfw(d3, d3du, d3files)
	cfw3.Start()
	defer cfw3.Close()

	cfw4 := cfw(d4, d4du, d4files)
	cfw4.Start()
	defer cfw4.Close()

	cfw5 := cfw(d5, d5du, d5files)
	cfw5.Start()
	defer cfw5.Close()

	///////////////////////////////////////////////////////////////////////////////
	base := "testsourcefolder"
	SourcePath.Add(base)

	// source path 에 파일 생성
	for _, f := range d1files {
		createfile(base, f)
	}
	for _, f := range d2files {
		createfile(base, f)
	}
	for _, f := range d3files {
		createfile(base, f)
	}
	for _, f := range d4files {
		createfile(base, f)
	}
	for _, f := range d5files {
		createfile(base, f)
	}

	createfile(base, "H.mpg")
	createfile(base, "I.mpg")
	createfile(base, "J.mpg")
	//createfile(base, "K.mpg")
	createfile(base, "L.mpg")
	createfile(base, "M.mpg")
	createfile(base, "N.mpg")
	//createfile(base, "O.mpg")
	createfile(base, "P.mpg")

	// C.mpg 는 source path 에서 삭제
	deletefile(base, "C.mpg")

	defer deletefile(base, "")

	serverfs := make(FileFreqMap)
	collectRemoteFileList(DstServers, serverfs)
	assert.Equal(t, 10, len(serverfs))
	assert.Equal(t, 1, int(serverfs["A.mpg"]))
	assert.Equal(t, 2, int(serverfs["B.mpg"]))
	assert.Equal(t, 2, int(serverfs["C.mpg"]))
	assert.Equal(t, 1, int(serverfs["D.mpg"]))
	assert.Equal(t, 1, int(serverfs["E.mpg"]))
	assert.Equal(t, 1, int(serverfs["F.mpg"]))
	assert.Equal(t, 1, int(serverfs["G.mpg"]))
	assert.Equal(t, 2, int(serverfs["SERVER1-5.mpg"]))
	assert.Equal(t, 1, int(serverfs["SERVER2.mpg"]))
	assert.Equal(t, 1, int(serverfs["SERVER3.mpg"]))

	ignores := []string{"AD1", "H", "I", "AD2"}
	SetIgnorePrefixes(ignores)

	rhfiles := []string{"E.mpg", "F.mpg", "J.mpg", "RH1.mpg", "M.mpg", "H.mpg", "O.mpg", "RH2.mpg"}

	gradedir := "gradeinfofolder"
	gradefile := ".grade.info"
	makeGradeInfoFileABCDEFGHIJKLMNO(gradedir, gradefile)
	defer deletefile(gradedir, "")

	SetGradeInfoFile(filepath.Join(gradedir, gradefile))

	hcdir := "hitcounthistoryinfofolder"
	hcfile := ".hitcount.history"
	makeHitcountHistoryFileABCDEFGHIJKLMNO(hcdir, hcfile)

	defer deletefile(hcdir, "")

	SetHitcountHistoryFile(filepath.Join(hcdir, hcfile))

	taildir := "taildir"
	tailip := "255.255.255.255"
	watchmin := 10
	hitbase := 5

	Tail.SetWatchDir(taildir)
	Tail.SetWatchIPString(tailip)
	Tail.SetWatchTermMin(watchmin)
	Tail.SetWatchHitBase(hitbase)
	basetm := time.Now()
	makeRisingHitFiles9(taildir, tailip, basetm, watchmin, rhfiles)
	defer deletefile(taildir, "")

	// heartbeater에 등록
	defer heartbeater.Release()
	heartbeater.Add(s1)
	heartbeater.Add(s2)
	heartbeater.Add(s3)
	heartbeater.Add(s4)

	heartbeater.Add(d1)
	heartbeater.Add(d2)
	heartbeater.Add(d3)
	heartbeater.Add(d4)
	heartbeater.Add(d5)

	var t1, t2, t3 Task

	ts := NewTasks()
	tasks = ts
	defer tasks.Release()

	t.Log("tasks 1-------------------------------------")
	t1 = ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t.Log(t1)
	t2 = ts.CreateTask(&Task{SrcIP: "127.0.0.2", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.2:8082", DstAddr: "127.0.0.2:18082"})
	t.Log(t2)
	t3 = ts.CreateTask(&Task{SrcIP: "127.0.0.3", FilePath: "/data2/DANGLING1.mpg",
		FileName: "DANGLING1.mpg", SrcAddr: "127.0.0.3:8083", DstAddr: "127.0.0.2:18082"})
	t.Log(t3)

	run(basetm)

	t.Log("tasks 2-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	assert.Equal(t, 0, len(tasks.TaskMap))

	t.Log("tasks 3-------------------------------------")
	t1 = ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8081", DstAddr: "127.0.0.1:18081"})
	t.Log(t1)
	t2 = ts.CreateTask(&Task{SrcIP: "127.0.0.2", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.2:8082", DstAddr: "127.0.0.2:18082"})
	t.Log(t2)
	t3 = ts.CreateTask(&Task{SrcIP: "127.0.0.3", FilePath: "/data2/DANGLING1.mpg",
		FileName: "DANGLING1.mpg", SrcAddr: "127.0.0.3:8083", DstAddr: "127.0.0.2:18082"})
	t.Log(t3)

	assert.Equal(t, 3, len(tasks.TaskMap))
	assertTask(t, tasks, "A.mpg", s1, d1)
	assertTask(t, tasks, "B.mpg", s2, d2)
	assertTask(t, tasks, "DANGLING1.mpg", s3, d2)

	heartbeater.Heartbeat()
	// src server와 dst server의 heartbeat 가 살아난 후에는 task 가 그대로 있음

	// 배포 대상 파일 :
	// A, B, C, D, E, F, G 는 이미 서버에 있으므로 제외
	// I, H 는 ignore.prefix 이기 때문에 제외

	// RH2.mpg, RH1.mpg는
	// 	all meta 에 없으므로 제외
	// O.mpg 는 source path에 없으므로 제외
	// rising hits file인 M.mpg, J.mpg는 우선순위가 높음

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3,s2,s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "A.mpg", s1, d1
	// 배포 중 : "B.mpg", s2, d2
	// 배포 중 : "DANGLING1.mpg", s3, d2

	// 배포 중이 아닌 서버 : s4
	// 배포 중이 아닌 서버 : d5, d4, d3
	// 배포 중이 아닌 파일 : M, J, L, N

	// M.mpg, s4 -> d5 task 가 하나 만들어져야 함
	run(basetm)

	t.Log("tasks 4-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3,s2,s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 기존의 task는 그대로 있고
	// M.mpg, s4 -> d5 task 가 하나 만들어져야 함
	assert.Equal(t, 4, len(tasks.TaskMap))

	// file, src, dst 로 간접 확인
	// 기존 task 에
	assert.Equal(t, 4, len(tasks.TaskMap))
	assertTask(t, tasks, "A.mpg", s1, d1)
	assertTask(t, tasks, "B.mpg", s2, d2)
	assertTask(t, tasks, "DANGLING1.mpg", s3, d2)
	// M.mpg, s4 -> d5 task 가 하나 만들어져야 함
	assertTask(t, tasks, "M.mpg", s4, d5)

	//////////////////////////////////////////////////////////////////////////////
	// t1이 DONE이 된 경우
	tasks.UpdateStatus(t1.ID, DONE)

	run(basetm)

	t.Log("tasks 5-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	assert.Equal(t, 4, len(tasks.TaskMap))

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3,s2,s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "B.mpg", s2, d2
	// 배포 중 : "DANGLING1.mpg", s3, d2
	// 배포 중 : "M.mpg", s4, d5

	// 배포 중이 아닌 서버 : s1
	// 배포 중이 아닌 서버 : d4, d3, d1
	// 배포 중이 아닌 파일 : J, L, N

	// file, src, dst 로 간접 확인
	// 기존의 t1 task 가 없어지고, 새로운 task 생성
	assert.Equal(t, 4, len(tasks.TaskMap))
	assertTask(t, tasks, "B.mpg", s2, d2)
	assertTask(t, tasks, "DANGLING1.mpg", s3, d2)
	assertTask(t, tasks, "M.mpg", s4, d5)
	// J.mpg, s1 -> d4 task 가 하나 만들어져야 함
	assertTask(t, tasks, "J.mpg", s1, d4)

	//////////////////////////////////////////////////////////////////////////////
	// t3 이 DONE이 된 경우
	updateStatus(t, tasks, "DANGLING1.mpg", DONE)

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3, s2, s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// DANGLING1.mpg, s3, d2 가 삭제되고 나면

	// 배포 중 : "B.mpg", s2, d2
	// 배포 중 : "M.mpg", s4, d5
	// 배포 중 : "J.mpg", s1, d4

	// 배포 중이 아닌 서버 : s3
	// 배포 중이 아닌 서버 : d3, d1
	// 배포 중이 아닌 파일 : L, N

	run(basetm)

	t.Log("tasks 6-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	// L.mpg, s3 -> d3 task 가 하나 만들어져야 함
	assert.Equal(t, 4, len(tasks.TaskMap))
	assertTask(t, tasks, "B.mpg", s2, d2)
	assertTask(t, tasks, "M.mpg", s4, d5)
	assertTask(t, tasks, "J.mpg", s1, d4)
	assertTask(t, tasks, "L.mpg", s3, d3)

	//////////////////////////////////////////////////////////////////////////////
	// B가 DONE 된 경우
	// M이 TIMEOUT된 경우, d5와 통신은 되는 경우
	updateStatus(t, tasks, "B.mpg", DONE)
	updateStatus(t, tasks, "M.mpg", TIMEOUT)

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3, s2, s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "J.mpg", s1, d4
	// 배포 중 : "L.mpg", s3, d3

	// 배포 중이 아닌 서버 : s4, s2
	// 배포 중이 아닌 서버 : d5, d2, d1
	// 배포 중이 아닌 파일 : M, N
	// M은 배포 실패햇다고 생각하고, 다시 배포 task에  넣을 수 있다고 가정함
	run(basetm)

	t.Log("tasks 7-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	assert.Equal(t, 4, len(tasks.TaskMap))
	assertTask(t, tasks, "J.mpg", s1, d4)
	assertTask(t, tasks, "L.mpg", s3, d3)
	// M, s4, d5 추가
	// N, s2, d2 추가
	assertTask(t, tasks, "M.mpg", s4, d5)
	assertTask(t, tasks, "N.mpg", s2, d2)

	//////////////////////////////////////////////////////////////////////////////
	// d2와 통신 실패
	// d5와 통신 실패
	// d3와 통신 실패
	// 통신에 실패하면 task가 삭제됨

	// heartbeater에서 제거만 되도, heartbeat 결과를 가져올 수 없어서
	// 통신에 실패한 것으로 간주함
	heartbeater.Delete(d2)
	heartbeater.Delete(d5)
	heartbeater.Delete(d3)

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3, s2, s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "J.mpg", s1, d4

	// 배포 중이 아닌 서버 : s4, s3, s2
	// 배포 중이 아닌 서버 : d1
	// 통신 실패 : (d5, d3, d2)
	// 배포 중이 아닌 파일 : M, L, N
	run(basetm)

	t.Log("tasks 8-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}

	assert.Equal(t, 4, len(tasks.TaskMap))
	assertTask(t, tasks, "J.mpg", s1, d4)
	// 여러 source 서버에서 하나의 destination 서버로 배포가 가능
	// M, s4, d1 추가
	// L, s3, d1 추가
	// N, s2, d1 추가
	assertTask(t, tasks, "M.mpg", s4, d1)
	assertTask(t, tasks, "L.mpg", s3, d1)
	assertTask(t, tasks, "N.mpg", s2, d1)

	//////////////////////////////////////////////////////////////////////////////
	// s1와 통신 실패
	// s2와 통신 실패
	// s3와 통신 실패
	// d4과 통신 실패
	// 통신에 실패하면 task가 삭제됨

	// heartbeater에서 제거만 되도, heartbeat 결과를 가져올 수 없어서
	// 통신에 실패한 것으로 간주함
	heartbeater.Delete(s1)
	heartbeater.Delete(s2)
	heartbeater.Delete(s3)

	heartbeater.Delete(d4)

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3, s2, s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "M.mpg", s4, d1

	// 배포 중이 아닌 서버 :
	// 배포 중이 아닌 서버 :
	// 통신 실패 : (s3, s2, s1)
	// 통신 실패 : (d5, d4, d3, d2)
	// 배포 중이 아닌 파일 : J. L, N
	run(basetm)

	t.Log("tasks 9-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}
	// M만 남고, 사용 가능한 src서버가 없어서 새로 만들어지지는 않음
	assert.Equal(t, 1, len(tasks.TaskMap))
	assertTask(t, tasks, "M.mpg", s4, d1)

	//////////////////////////////////////////////////////////////////////////////
	// s1은 통신 성공
	heartbeater.Add(s1)
	heartbeater.Heartbeat()

	// 배포 대상은 M, J, L, N 순으로 배포되어야 함
	// src 서버 선택은 s4, s3, s2, s1 순임
	// dst 서버 선택은 d5, d4, d3, d2, d1 순임

	// 배포 중 : "M.mpg", s4, d1

	// 배포 중이 아닌 서버 : s1
	// 배포 중이 아닌 서버 :
	// 통신 실패 : (s3, s2)
	// 통신 실패 : (d5, d4, d3, d2)
	// 배포 중이 아닌 파일 : J. L, N
	run(basetm)

	t.Log("tasks 10-------------------------------------")
	// src server heartbeat fail 로 모든 task 가 clear 됨
	for _, task := range tasks.GetTaskList() {
		t.Log(task)
	}
	// M만 남고,
	// 사용가능한 src 서버 s1이 있지만,
	// 사용가능한 dst 서버가 없어서 새로 만들어지지는 않음
	assert.Equal(t, 1, len(tasks.TaskMap))
	assertTask(t, tasks, "M.mpg", s4, d1)
}

func makeGradeInfoFileABCDEFGHIJKLMNO(dir string, filename string) {
	makeGradeInfoFile(dir, filename, 'A', 'O', 'O'-'A')
}

func makeGradeInfoFile(dir string, filename string, chs, che byte, n uint) {
	fp := filepath.Join(dir, filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.Create(fp)
	if err != nil {
		f.Close()
		log.Fatal(err)
	}

	fmt.Fprintf(f, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "filename", "weightcount", "bitrate", "grade", "sumHitCount", "historyCount", "TargetCopyCount")

	ixl := uint(che - chs + 1)
	for i := uint(0); i < n; i++ {
		ix := i % ixl
		fn := fmt.Sprintf("%s", string(chs+byte(ix)))
		if i < ixl {
			fn = fn + ".mpg"
		} else {
			ix2 := i / ixl
			fn = fn + strconv.FormatUint(uint64(ix2), 10) + ".mpg"
		}

		fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", fn, 4144, 1000, 1, 1554, 24, 5)
	}

	f.Close()
}

func makeHitcountHistoryFileABCDEFGHIJKLMNO(dir string, filename string) {
	makeHitcountHistoryFile(dir, filename, 'A', 'O', 'O'-'A')
}

// 서버 위치 정보는 없는 파일
func makeHitcountHistoryFile(dir string, filename string, chs, che byte, n uint) {
	fp := filepath.Join(dir, filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.Create(fp)
	if err != nil {
		f.Close()
		log.Fatal(err)
	}

	fmt.Fprintln(f, "historyheader:1524047082")

	ixl := uint(che - chs + 1)
	for i := uint(0); i < n; i++ {
		ix := i % uint(che-chs+1)
		fn := fmt.Sprintf("%s", string(chs+byte(ix)))
		if i < ixl {
			fn = fn + ".mpg"
		} else {
			ix2 := i / uint(che-chs+1)
			fn = fn + strconv.FormatUint(uint64(ix2), 10) + ".mpg"
		}

		fmt.Fprintf(f, "%s,1428460337,1000,100,,2,0,0,0=0 0\n", fn)
	}

	f.Close()
}

func makeRisingHitFiles9(dir string, watchip string,
	basetm time.Time, watchmin int,
	risingfilenames []string) {

	// 현재 시각값을 이용하여 N분 전 시각을 구하기 위해선 음수 값이 필요하다.
	from := basetm.Add(time.Minute * time.Duration(watchmin*-1))
	logFileNames := tailer.GetLogFileName(basetm, dir, watchmin)

	baselogTime := from.Unix()
	logFileName := (*logFileNames)[0]
	fp := filepath.Clean(logFileName)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.Create(fp)
	if err != nil {
		f.Close()
		log.Fatal(err)
	}

	for i, risingfile := range risingfilenames {
		writeRisingHit(f, watchip, risingfile, 9+int64(i), baselogTime, watchmin)
	}

	f.Close()
}

func writeRisingHit(f *os.File,
	watchip string,
	risingfile string,
	n int64,
	baselogTime int64, watchmin int) {

	// ------------------------------------------------- 테스트 기준 시각 - 4
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : 3f004a6e-82af-4dce-85ba-9bbf9c7cb8cb, ClientID : 0, GLB IP : 125.144.96.6's file(MCLE901VSGL1500001_K20140915224744.mpg) Request", baselogTime-4, watchip)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,4,%d,File Not Found, UUID : fffb233a-376a-4c2f-842e-553fb68af9cf, GLB IP : 125.144.161.6, MV6F9001SGL1500001_K20150909214818.mpg", baselogTime-4)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.91.87 Selected for Client StreamID : 360527d4-44b3-4b8f-aef7-dbf8fd230d54, ClientID : 0, GLB IP : 125.144.169.6's file(M33E80DTSGL1500001_K20141022144006.mpg) Request", baselogTime-4)
	fmt.Fprintln(f)
	// ------------------------------------------------- 테스트 기준 시각 - 2
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : c93a7db2-ccaf-4765-af8d-7ddc2d33a812, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime-2, watchip, risingfile)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,4,%d,File Not Found, UUID : f1add5cf-75ac-41ab-a6ff-85d9e0927762, GLB IP : 125.144.169.6, MK4E7BK2SGL0800014_K20120725124707.mpg", baselogTime-2)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.97.67 Selected for Client StreamID : 06fb572e-7602-4231-8670-cb6526603fb0, ClientID : 0, GLB IP : 125.146.8.6's file(M33H90E2SGL1500001_K20171008222635.mpg) Request", baselogTime-2)
	fmt.Fprintln(f)
	// ------------------------------------------------- 테스트 기준 시각 - 1
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : 3c61af91-cd6a-4dd6-bc04-5ec6bc78b94f, ClientID : 0, GLB IP : 125.159.40.5's file(MWGI5006SGL1500001_K20180524203234.mpg) Request", baselogTime-1, watchip)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,4,%d,File Not Found, UUID : c585905f-9980-49b1-89bc-97c7140eaa83, GLB IP : 125.159.40.5, M34G80A3SGL1500001_K20160827230242.mpg", baselogTime-1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.97.74 Selected for Client StreamID : 7cf6b886-edd2-471b-9cfd-12763a160b0b, GLB IP : 125.159.40.5's file(M34F60QHSGL1500001_K20150701232550.mpg) Request", baselogTime-1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.97.77 Selected for Client StreamID : 23dd1489-543b-4051-b07a-e877f8b2e052, GLB IP : 125.147.192.6's file(MW0E6JE3SGL0800014_K20120601193450.mpg) Request", baselogTime-1)
	fmt.Fprintln(f)
	// ------------------------------------------------- 테스트 기준 시각
	//fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : 97096b41-afe1-44d8-b57c-e758a70883d9, GLB IP : 125.159.40.5's file(M33F3MA3SGL0800038_K20130326135640.mpg) Request", baselogTime, watchip)
	//fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.91.83 Selected for Client StreamID : aa7de9a1-7d0d-40d5-9586-31dc275a0634, ClientID : 0, GLB IP : 125.147.36.6's file(MADI4008SGL1500001_K20180506231943.mpg) Request", baselogTime)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.91.84 Selected for Client StreamID : 1926c2ba-c313-48fb-977a-b7f3fd27ea98, ClientID : 0, GLB IP : 125.148.160.6's file(MEQI405ISGL1500001_K20180509034746.mpg) Request", baselogTime)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.97.73 Selected for Client StreamID : f179c61a-d5e0-45b9-b046-a3cd4e3dbbfc, ClientID : 0, GLB IP : 125.147.192.6's file(MIAF51OLSGL1500001_K20150511175323.mpg) Request", baselogTime)
	fmt.Fprintln(f)

	for i := int64(0); i < n; i++ {
		fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : 3894d674-d74b-4eca-a2ea-fafbfa1113a8, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime, watchip, risingfile)
		fmt.Fprintln(f)
	}
}

func updateStatus(t *testing.T, tsks *Tasks, filename string, status Status) {
	tsk, ok := tsks.FindTaskByFileName(filename)
	assert.Equal(t, true, ok)
	if ok {
		err := tsks.UpdateStatus(tsk.ID, status)
		if err != nil {
			t.Error(err)
		}
	}
}

func assertTask(t *testing.T, tsks *Tasks, filename, srcaddr, dstaddr string) {
	tsk, ok := tsks.FindTaskByFileName(filename)
	assert.Equal(t, true, ok)
	assert.Equal(t, srcaddr, tsk.SrcAddr)
	assert.Equal(t, dstaddr, tsk.DstAddr)
}

func Benchmark_MakeAllFileMetas(t *testing.B) {
	t.StopTimer()
	gradedir := "gradeinfofolder"
	gradefile := ".grade.info"

	makeGradeInfoFile(gradedir, gradefile, 'A', 'O', 1000000)
	defer deletefile(gradedir, "")

	gradeInfoFile := filepath.Join(gradedir, gradefile)

	hcdir := "hitcounthistoryinfofolder"
	hcfile := ".hitcount.history"
	makeHitcountHistoryFile(hcdir, hcfile, 'A', 'O', 1000000)
	defer deletefile(hcdir, "")

	hitcountHistoryFile := filepath.Join(hcdir, hcfile)

	fileMetaMap := make(map[string]*common.FileMeta)
	duplicatedFileMap := make(map[string]*common.FileMeta)

	dstIPMap := make(map[string]int)
	dstIPMap["127.0.0.1"]++

	t.StartTimer()
	err := common.MakeAllFileMetas(gradeInfoFile, hitcountHistoryFile,
		fileMetaMap, dstIPMap, duplicatedFileMap)
	if err != nil {
		t.Errorf("error(%s)", err)
	}

	t.Logf("len: %d", len(fileMetaMap))
	assert.Equal(t, 1000000, len(fileMetaMap))
}

func Benchmark_MakeAllFileMetas_simpleGetFileMetaListForTask(t *testing.B) {
	t.StopTimer()
	allfmm, _ := makeFileMetaMap('A', 'O', 1000000)

	t.StartTimer()
	taskfilelist := make([]FileMetaPtr, 0, len(allfmm))
	for _, fmm := range allfmm {
		taskfilelist = append(taskfilelist, fmm)
	}
	sort.Slice(taskfilelist, func(i, j int) bool {
		if taskfilelist[i].RisingHit != taskfilelist[j].RisingHit {
			return taskfilelist[i].RisingHit > taskfilelist[j].RisingHit
		} else {
			return taskfilelist[i].Grade < taskfilelist[j].Grade
		}
	})

	t.Logf("len: %d", len(taskfilelist))
	assert.Equal(t, 1000000, len(taskfilelist))
}
