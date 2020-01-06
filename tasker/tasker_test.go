package tasker

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCleanTask(t *testing.T) {

	ts := NewTasks()

	t1 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg",
		FileName: "A.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.1:8080"})
	t2 := ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg",
		FileName: "B.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.2:8080"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg",
		FileName: "C.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.3:8080"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg",
		FileName: "D.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.4:8080"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg",
		FileName: "E.mpg", SrcAddr: "127.0.0.1:8080", DstAddr: "127.0.0.5:8080"})

	ts.TaskMap[t1.ID].Status = DONE
	ts.TaskMap[t2.ID].Status = DONE

	SrcServers.Add("127.0.0.1:8080")
	(*SrcServers)[0].Status = OK

	DstServers.Add("127.0.0.1:8080")
	DstServers.Add("127.0.0.2:8080")
	DstServers.Add("127.0.0.3:8080")
	DstServers.Add("127.0.0.4:8080")
	DstServers.Add("127.0.0.5:8080")

	(*DstServers)[0].Status = OK
	(*DstServers)[1].Status = OK
	(*DstServers)[2].Status = OK
	(*DstServers)[3].Status = OK
	(*DstServers)[4].Status = OK

	CleanTask(ts)
	// 2개의 DONE task 삭제된 후 task 개수
	// task의 SrcAddr 와 task 의 DstAddr 의 Status를 구할 수 없어도 삭제됨
	assert.Equal(t, 3, len(ts.TaskMap))

	(*DstServers)[2].Status = NOTOK
	CleanTask(ts)
	// dest server 127.0.0.3:8080 의 상태가 NOTOK 로 바뀌어서 삭제됨
	assert.Equal(t, 2, len(ts.TaskMap))

	SetTaskTimeout(time.Second * 1)
	time.Sleep(time.Second * 2)
	CleanTask(ts)

	// 3개의 task 가 timeout 으로 삭제된 후 task 개수
	assert.Equal(t, 0, len(ts.TaskMap))

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

func TestSrcHosts_Add(t *testing.T) {
	srcs := new(SrcHosts)

	srcs.Add("127.0.0.1:18001")
	srcs.Add("127.0.0.2:18001")
	srcs.Add("127.0.0.3:18001")
	assert.Equal(t, 3, len(*srcs))

	srcs.Add("127.0.0.3:18001")
	assert.Equal(t, 4, len(*srcs))

}

func TestSetAdvPrefix(t *testing.T) {

	prefixes := []string{"M64", "MN1"}
	SetAdvPrefix(prefixes)

	assert.Equal(t, true, reflect.DeepEqual(advPrefixes, prefixes))
}
