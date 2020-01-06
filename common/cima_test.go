package common_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/castisdev/cfm/common"
)

func TestGetRemoteFileList(t *testing.T) {

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/files", r.URL.EscapedPath())

		fmt.Fprintln(w, "A.mpg")
		fmt.Fprintln(w, "B.mpg")
		fmt.Fprintln(w, "C.mpg")
	}))

	l, _ := net.Listen("tcp", "127.0.0.1:18888")
	h := common.Host{IP: "127.0.0.1", Port: 18888, Addr: "127.0.0.1:18888"}

	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	fs := make([]string, 0, 5)
	common.GetRemoteFileList(&h, &fs)

	assert.Equal(t, 3, len(fs))
}

func TestGetRemoteDiskUsage(t *testing.T) {

	duSample := common.DiskUsage{
		TotalSize:   100,
		UsedSize:    80,
		AvailSize:   10,
		FreeSize:    20,
		UsedPercent: 89,
	}

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/df", r.URL.EscapedPath())

		bytes, err := json.Marshal(duSample)
		require.Nil(t, err)

		w.Write(bytes)

	}))

	l, _ := net.Listen("tcp", "127.0.0.1:18888")
	h := common.Host{IP: "127.0.0.1", Port: 18888, Addr: "127.0.0.1:18888"}

	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	du := new(common.DiskUsage)
	common.GetRemoteDiskUsage(&h, du)

	assert.Equal(t, true, reflect.DeepEqual(*du, duSample))

}

func TestDeleteFileOnRemote(t *testing.T) {
	// want/got string, int

	fileNameToDelete := "a.mpg"

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/files/a.mpg", r.URL.EscapedPath())
		w.WriteHeader(http.StatusOK)
	}))

	l, _ := net.Listen("tcp", "127.0.0.1:18888")
	h := common.Host{IP: "127.0.0.1", Port: 18888, Addr: "127.0.0.1:18888"}

	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	assert.Nil(t, common.DeleteFileOnRemote(&h, fileNameToDelete))
}

func TestDeleteNotExistFileOnRemote(t *testing.T) {
	// want/got string, int

	NotExistfileNameToDelete := "a.mpg"

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/files/a.mpg", r.URL.EscapedPath())
		w.WriteHeader(http.StatusNotFound)
	}))

	l, _ := net.Listen("tcp", "127.0.0.1:18888")
	h := common.Host{IP: "127.0.0.1", Port: 18888, Addr: "127.0.0.1:18888"}

	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	err := common.DeleteFileOnRemote(&h, NotExistfileNameToDelete)
	assert.EqualError(t, err, "404 Not Found")
}
