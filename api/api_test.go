package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/fmfm"
	"github.com/castisdev/cfm/heartbeater"
	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cfm/tasker"
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

func makeGradeInfoFile(dir string, filename string) {
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
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "A.mpg", 4144, 1000, 1, 1554, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "B.mpg", 4042, 1000, 1, 1516, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "C.mpg", 3861, 1000, 1, 1448, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "D.mpg", 3493, 1000, 1, 1310, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "E.mpg", 3306, 1000, 1, 1240, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "F.mpg", 3285, 1000, 1, 1232, 24, 5)

	f.Close()
}

func makeHitcourntHistoryFile(dir string, filename string) {
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
	fmt.Fprintln(f, "A.mpg,1428460337,1000,100,127.0.0.1,2,0,0,0=0 0")
	fmt.Fprintln(f, "B.mpg,1508301143,1000,100,127.0.0.1 127.0.0.2,2,0,0,1=1 2")
	fmt.Fprintln(f, "C.mpg,1508301143,1000,100,127.0.0.1 127.0.0.2,2,0,0,1=1 2")
	fmt.Fprintln(f, "D.mpg,1428460337,1000,100,127.0.0.1,2,0,0,0=0 0")
	fmt.Fprintln(f, "E.mpg,1428460337,1000,100,127.0.0.2,2,0,0,0=0 0")
	fmt.Fprintln(f, "F.mpg,1428460337,1000,100,127.0.0.2,2,0,0,0=0 0")

	f.Close()
}

// basetime : ex) 1527951611
// watchip : ex) 125.159.40.3
// risingfile : ex) F.mpg
func makeRisingHitFile(dir string, watchip string, risingfile string,
	basetm time.Time, watchmin int) {

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
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : 97096b41-afe1-44d8-b57c-e758a70883d9, GLB IP : 125.159.40.5's file(M33F3MA3SGL0800038_K20130326135640.mpg) Request", baselogTime, watchip)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.91.83 Selected for Client StreamID : aa7de9a1-7d0d-40d5-9586-31dc275a0634, ClientID : 0, GLB IP : 125.147.36.6's file(MADI4008SGL1500001_K20180506231943.mpg) Request", baselogTime)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.91.84 Selected for Client StreamID : 1926c2ba-c313-48fb-977a-b7f3fd27ea98, ClientID : 0, GLB IP : 125.148.160.6's file(MEQI405ISGL1500001_K20180509034746.mpg) Request", baselogTime)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.97.73 Selected for Client StreamID : f179c61a-d5e0-45b9-b046-a3cd4e3dbbfc, ClientID : 0, GLB IP : 125.147.192.6's file(MIAF51OLSGL1500001_K20150511175323.mpg) Request", baselogTime)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : 3894d674-d74b-4eca-a2ea-fafbfa1113a8, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime, watchip, risingfile)
	fmt.Fprintln(f)
	// ------------------------------------------------- 테스트 기준 시각 + 1
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : 8b2e7e4a-270d-4586-85a1-e4284551176d, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime+1, watchip, risingfile)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : f1893245-802b-45b1-b0fa-377bc1415b35, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime+1, watchip, risingfile)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.93.100 Selected for Client StreamID : 3c68d2a5-354e-4a4b-b181-c724d16cf406, GLB IP : 125.147.36.6's file(MVHF201MSGL1500001_K20150216200556.mpg) Request", baselogTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : d41dc03a-91b7-4c14-bb3a-a73823f333e0, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime+1, watchip, risingfile)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,4,%d,File Not Found, UUID : d9672fe8-1b39-491f-a8a4-23bf7a6f096c, GLB IP : 125.144.96.6, M0200000SGL1065016_K20100826000000.MPG", baselogTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : f8a21815-a75b-4fe4-8f6a-dba984ee7c6e, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime+1, watchip, risingfile)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : 7148fadd-edab-4a48-8d27-4bf8c8b74cbd, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime+1, watchip, risingfile)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : b121ae29-954c-4990-8e58-e102959d0239, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime+1, watchip, risingfile)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.93.97 Selected for Client StreamID : 86fd80d3-2691-4274-9172-315d50e90801, ClientID : 0, GLB IP : 125.159.40.5's file(M34I502CSGL1500001_K20180512022857.mpg) Request", baselogTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : 598131b3-8fb2-4415-b0f8-472d52ef054c, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime+1, watchip, risingfile)
	fmt.Fprintln(f)
	// ------------------------------------------------- 테스트 기준 시각 + 2
	fmt.Fprintf(f, "0x40ffff,1,%d,Server %s Selected for Client StreamID : 0590710c-bd2e-4863-941e-041877328d78, ClientID : 0, GLB IP : 125.159.40.5's file(%s) Request", baselogTime+2, watchip, risingfile)
	fmt.Fprintln(f)

	f.Close()
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

func writefile(dir, filename, text string) {
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
	fmt.Fprintf(f, "%s\n", text)
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func doSomething(sec int) {
	time.Sleep(time.Duration(sec) * time.Second)
}

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
	if resp != nil {
		defer resp.Body.Close()
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
	if resp != nil {
		defer resp.Body.Close()
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
	if resp != nil {
		defer resp.Body.Close()
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
	if resp != nil {
		defer resp.Body.Close()
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

//Mtime 제외
func TestGetHostStateDashBoard(t *testing.T) {
	writefile("dashboard", "hoststate.html",
		"{{range .}}{{.Addr}},{{.Status}}\n{{end}}")
	defer deletefile("dashboard", "")

	s1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	s2 := "127.0.0.2:18882"
	files2 := []string{"B.mpg", "C.mpg", "E.mpg", "F.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	cfw1 := cfw(s1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()
	cfw2 := cfw(s2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	heartbeater.Add(s1)
	heartbeater.Add(s2)
	heartbeater.Heartbeat()
	defer heartbeater.Release()

	serverAddr := "127.0.0.1:28881"
	r := fmfm.NewRunner(0, 0, nil, nil, nil)
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

	url := fmt.Sprintf("http://%s/dashboard/hb", serverAddr)
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
	if resp != nil {
		defer resp.Body.Close()
	}
	body, _ := ioutil.ReadAll(resp.Body)
	scanner := bufio.NewScanner(bytes.NewReader(body))
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		log.Printf("%s", scanner.Text())
	}

	h1desc := fmt.Sprintf("127.0.0.2:18882,ok")
	h2desc := fmt.Sprintf("127.0.0.1:18881,ok")

	assert.Equal(t, h1desc, lines[0])
	assert.Equal(t, h2desc, lines[1])
}

// Ctime, Mtime 제외
func TestGetDashboard(t *testing.T) {
	writefile("dashboard", "layout.html",
		"{{range .}}{{.ID}},{{.Grade}},{{.FilePath}},{{.Status}},{{.SrcAddr}},{{.DstAddr}},{{.Grade}}\n{{end}}")
	defer deletefile("dashboard", "")

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

	serverAddr := "127.0.0.1:28881"
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

	url := fmt.Sprintf("http://%s/dashboard", serverAddr)
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
	if resp != nil {
		defer resp.Body.Close()
	}
	body, _ := ioutil.ReadAll(resp.Body)
	scanner := bufio.NewScanner(bytes.NewReader(body))
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		log.Printf("%s", scanner.Text())
	}

	t1desc := fmt.Sprintf("%d,0,/data2/A.mpg,done,127.0.0.1:8080,127.0.0.1:8081,0", t1.ID)
	t2desc := fmt.Sprintf("%d,0,/data2/C.mpg,ready,127.0.0.1:8080,127.0.0.2:8081,0", t2.ID)
	t3desc := fmt.Sprintf("%d,0,/data2/D.mpg,working,127.0.0.1:8080,127.0.0.3:8081,0", t3.ID)
	t4desc := fmt.Sprintf("%d,0,/data2/E.mpg,timeout,127.0.0.1:8080,127.0.0.4:8081,0", t4.ID)

	assert.Equal(t, t1desc, lines[0])
	assert.Equal(t, t2desc, lines[1])
	assert.Equal(t, t3desc, lines[2])
	assert.Equal(t, t4desc, lines[3])
}

// 간단한 test만 수행
func TestGetFilemetasSimple(t *testing.T) {
	writefile("dashboard", "filemetas.html",
		"{{range .Fmms}}{{.Name}},{{.Grade}},{{.Size}},{{.RisingHit}},{{.ServerIPList}},{{.SrcFilePath}}\n{{end}}")
	defer deletefile("dashboard", "")

	dir := "testwatcher"
	gradefile := "grade"
	makeGradeInfoFile(dir, gradefile)
	gradepath := filepath.Join(dir, gradefile)
	hcfile := "hitcount"
	makeHitcourntHistoryFile(dir, hcfile)
	hitcountpath := filepath.Join(dir, hcfile)
	defer deletefile(dir, "")

	// test를 위해서 poll 모드 제일처음 event 만 발생시키고, 다른 event는 막음
	fmfm.TestInotifyFunc = func() bool { return false }
	watcher := fmfm.NewWatcher(gradepath, hitcountpath, true, 0, 0)

	rmr := remover.NewRemover()
	s1 := "127.0.0.1:18881"
	s2 := "127.0.0.2:18882"
	rmr.Servers.Add(s1)
	rmr.Servers.Add(s2)
	rmr.SetGradeInfoFile(gradepath)
	rmr.SetHitcountHistoryFile(hitcountpath)

	tskr := tasker.NewTasker()
	tskr.SrcServers.Add(s1)
	tskr.DstServers.Add(s1)
	tskr.DstServers.Add(s2)

	taildir := "taildir"
	tailip := "255.255.255.255"
	watchmin := 10
	hitbase := 5
	tlr := tailer.NewTailer()
	tlr.SetWatchDir(taildir)
	tlr.SetWatchIPString(tailip)
	tlr.SetWatchTermMin(watchmin)
	tlr.SetWatchHitBase(hitbase)

	basetm := time.Now()
	makeRisingHitFile(taildir, tailip, "F.mpg", basetm, watchmin)
	defer deletefile(taildir, "")

	serverAddr := "127.0.0.1:28881"

	//  다른 event run 은 막고, 최초 event run만 fmm, rising hit 만드는 것으로 설정
	fmfm.DefaultEventRuns = []fmfm.RUN{fmfm.MakeFMM, fmfm.MakeRisingHit}
	fmfm.DefaultEventTimeoutRuns = []fmfm.RUN{}
	fmfm.DefaultBetweenEventsRuns = []fmfm.RUN{}
	fmfm.DefaultPeriodicRuns = []fmfm.RUN{}
	r := fmfm.NewRunner(0, 0, rmr, tskr, tlr)
	m := fmfm.NewManager(watcher, r)
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

	go m.Manage()

	url := fmt.Sprintf("http://%s/dashboard/filemetas", serverAddr)
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
	if resp != nil {
		defer resp.Body.Close()
	}
	body, _ := ioutil.ReadAll(resp.Body)
	scanner := bufio.NewScanner(bytes.NewReader(body))
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		log.Printf("%s", scanner.Text())
	}
	assert.Equal(t, "F.mpg,6,100,9,127.0.0.2,", lines[0])
	assert.Equal(t, "A.mpg,1,100,0,127.0.0.1,", lines[1])
	assert.Equal(t, "B.mpg,2,100,0,127.0.0.1, 127.0.0.2,", lines[2])
	assert.Equal(t, "C.mpg,3,100,0,127.0.0.1, 127.0.0.2,", lines[3])
	assert.Equal(t, "D.mpg,4,100,0,127.0.0.1,", lines[4])
	assert.Equal(t, "E.mpg,5,100,0,127.0.0.2,", lines[5])

	select {
	case m.CMDCh <- fmfm.STOP:
		<-m.ErrCh
		_, open := <-m.CMDCh
		assert.Equal(t, false, open)
	}
}

// cfw
// heartbeat
// tasker, remover, tailer 실행 후 test
// 초기 event 발생 시 MakeFMM, MakeRisingHit, RunRemover, RunTasker 실행 후 test
func TestGetFilemetasWithCFW(t *testing.T) {
	writefile("dashboard", "filemetas.html",
		"{{range .Fmms}}{{.Name}},{{.Grade}},{{.Size}},{{.RisingHit}},{{.ServerIPList}},{{.SrcFilePath}}\n{{end}}")
	defer deletefile("dashboard", "")

	dir := "testwatcher"
	gradefile := "grade"
	makeGradeInfoFile(dir, gradefile)
	gradepath := filepath.Join(dir, gradefile)
	hcfile := "hitcount"
	makeHitcourntHistoryFile(dir, hcfile)
	hitcountpath := filepath.Join(dir, hcfile)
	defer deletefile(dir, "")

	// test를 위해서 poll 모드 제일처음 event 만 발생시키고, 다른 event는 막음
	fmfm.TestInotifyFunc = func() bool { return false }
	watcher := fmfm.NewWatcher(gradepath, hitcountpath, true, 0, 0)

	rmr := remover.NewRemover()
	s1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	s2 := "127.0.0.2:18882"
	files2 := []string{"B.mpg", "C.mpg", "E.mpg", "F.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	base := "testsourcefolder"
	rmr.SourcePath.Add(base)
	for _, f1 := range files1 {
		createfile(base, f1)
	}
	for _, f2 := range files2 {
		createfile(base, f2)
	}
	deletefile(base, "C.mpg")
	defer deletefile(base, "")

	rmr.Servers.Add(s1)
	rmr.Servers.Add(s2)
	rmr.SetGradeInfoFile(gradepath)
	rmr.SetHitcountHistoryFile(hitcountpath)

	cfw1 := cfw(s1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()
	cfw2 := cfw(s2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	rmr.SetDiskUsageLimitPercent(100)
	ignores := []string{"E"}
	rmr.SetIgnorePrefixes(ignores)

	tskr := tasker.NewTasker()
	tskr.SourcePath.Add(base)
	tskr.SrcServers.Add(s1)
	tskr.DstServers.Add(s1)
	tskr.DstServers.Add(s2)

	// heartbeat ok 처리
	heartbeater.Add(s1)
	heartbeater.Add(s2)
	heartbeater.Heartbeat()
	defer heartbeater.Release()

	taildir := "taildir"
	tailip := "255.255.255.255"
	watchmin := 10
	hitbase := 5
	tlr := tailer.NewTailer()
	tlr.SetWatchDir(taildir)
	tlr.SetWatchIPString(tailip)
	tlr.SetWatchTermMin(watchmin)
	tlr.SetWatchHitBase(hitbase)

	basetm := time.Now()
	makeRisingHitFile(taildir, tailip, "F.mpg", basetm, watchmin)
	defer deletefile(taildir, "")

	serverAddr := "127.0.0.1:28881"

	fmfm.DefaultEventRuns = []fmfm.RUN{fmfm.MakeFMM, fmfm.MakeRisingHit,
		fmfm.RunRemover, fmfm.RunTasker}
	fmfm.DefaultEventTimeoutRuns = []fmfm.RUN{}
	fmfm.DefaultBetweenEventsRuns = []fmfm.RUN{}
	fmfm.DefaultPeriodicRuns = []fmfm.RUN{}
	r := fmfm.NewRunner(0, 0, rmr, tskr, tlr)
	m := fmfm.NewManager(watcher, r)
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

	go m.Manage()

	url := fmt.Sprintf("http://%s/dashboard/filemetas", serverAddr)
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
	if resp != nil {
		defer resp.Body.Close()
	}
	body, _ := ioutil.ReadAll(resp.Body)
	scanner := bufio.NewScanner(bytes.NewReader(body))
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		log.Printf("%s", scanner.Text())
	}
	assert.Equal(t, "F.mpg,6,100,9,127.0.0.2,testsourcefolder/F.mpg", lines[0])
	assert.Equal(t, "A.mpg,1,100,0,127.0.0.1,testsourcefolder/A.mpg", lines[1])
	assert.Equal(t, "B.mpg,2,100,0,127.0.0.1, 127.0.0.2,testsourcefolder/B.mpg", lines[2])
	assert.Equal(t, "C.mpg,3,100,0,127.0.0.1, 127.0.0.2,", lines[3])
	assert.Equal(t, "D.mpg,4,100,0,127.0.0.1,testsourcefolder/D.mpg", lines[4])
	assert.Equal(t, "E.mpg,5,100,0,127.0.0.2,testsourcefolder/E.mpg", lines[5])

	select {
	case m.CMDCh <- fmfm.STOP:
		<-m.ErrCh
		_, open := <-m.CMDCh
		assert.Equal(t, false, open)
	}
}
