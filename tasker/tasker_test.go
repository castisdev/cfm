package tasker

import (
	"cfm/common"
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

	ts := NewTasks(5)

	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg", FileName: "A.mpg"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg", FileName: "B.mpg"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg", FileName: "C.mpg"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/D.mpg", FileName: "D.mpg"})
	ts.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/E.mpg", FileName: "E.mpg"})

	ts.TaskList[0].Status = DONE
	ts.TaskList[1].Status = DONE

	CleanTask(ts)

	// 2개의 DONE task 삭제된 후 task 개수
	assert.Equal(t, 3, len(ts.TaskList))

	SetTaskTimeout(time.Second * 1)
	time.Sleep(time.Second * 2)
	CleanTask(ts)

	// 3개의 task 가 timeout 으로 삭제된 후 task 개수
	assert.Equal(t, 0, len(ts.TaskList))

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

	hosts := common.NewHosts()
	hosts.Add("127.0.0.1:18881")
	hosts.Add("127.0.0.1:18882")
	hosts.Add("127.0.0.1:18883")

	fs := make(map[string]int)
	CollectRemoteFileList(hosts, fs)

	assert.Equal(t, 3, fs["A.mpg"])
	assert.Equal(t, 3, fs["B.mpg"])
	assert.Equal(t, 3, fs["C.mpg"])

}

func Test_selectSourceServer(t *testing.T) {

	srcs := new(SrcHosts)

	srcs.Add("127.0.0.1:18001")
	srcs.Add("127.0.0.2:18001")
	srcs.Add("127.0.0.3:18001")

	for i := 0; i < 3; i++ {
		_, exists := srcs.selectSourceServer()
		assert.Equal(t, true, exists)
	}

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
