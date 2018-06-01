package common

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
)

func TestGetRemoteFileList(t *testing.T) {

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/cfm/ls", r.URL.EscapedPath())

		fmt.Fprintln(w, "A.mpg")
		fmt.Fprintln(w, "B.mpg")
		fmt.Fprintln(w, "C.mpg")
	}))

	l, _ := net.Listen("tcp", "127.0.0.1:18888")
	h := Host{IP: "127.0.0.1", Port: 18888}

	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	fs := make([]string, 0, 5)
	GetRemoteFileList(&h, &fs)

	assert.Equal(t, 3, len(fs))
}

func TestGetRemoteDiskUsage(t *testing.T) {

	duSample := DiskUsage{
		TotalSize:   100,
		UsedSize:    80,
		FreeSize:    20,
		UsedPercent: 80,
	}

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/cfm/df", r.URL.EscapedPath())

		bytes, err := json.Marshal(duSample)
		require.Nil(t, err)

		w.Write(bytes)

	}))

	l, _ := net.Listen("tcp", "127.0.0.1:18888")
	h := Host{IP: "127.0.0.1", Port: 18888}

	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	du := new(DiskUsage)
	GetRemoteDiskUsage(&h, du)

	assert.Equal(t, true, reflect.DeepEqual(*du, duSample))

}

func TestDeleteFileOnRemote(t *testing.T) {

	// want/got string, int
	fileNameToDelete := "a.mpg"

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		r.ParseForm()

		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/cfm/rm", r.URL.EscapedPath())
		assert.Equal(t, fileNameToDelete, r.Form.Get("file"))

	}))

	l, _ := net.Listen("tcp", "127.0.0.1:18888")
	h := Host{IP: "127.0.0.1", Port: 18888}

	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	assert.Nil(t, DeleteFileOnRemote(&h, fileNameToDelete))
}
