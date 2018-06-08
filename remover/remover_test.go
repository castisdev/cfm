package remover

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/castisdev/cfm/common"
	"github.com/stretchr/testify/assert"
)

func Test_collectRemoteDiskUsage(t *testing.T) {

	vs1 := "127.0.0.1:18881"
	vs2 := "127.0.0.1:18882"
	vs3 := "127.0.0.1:18883"

	// setup dummy http server
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 500,
		FreeSize: 500, UsedPercent: 50,
	}
	h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		b, err := json.Marshal(d1)
		assert.Nil(t, err)
		w.Write(b)
	})

	d2 := common.DiskUsage{
		TotalSize: 2000, UsedSize: 1000,
		FreeSize: 1000, UsedPercent: 50,
	}
	h2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		b, err := json.Marshal(d2)
		assert.Nil(t, err)
		w.Write(b)
	})

	d3 := common.DiskUsage{
		TotalSize: 3000, UsedSize: 1500,
		FreeSize: 1500, UsedPercent: 50,
	}
	h3 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		b, err := json.Marshal(d3)
		assert.Nil(t, err)
		w.Write(b)
	})

	ts1 := httptest.NewUnstartedServer(h1)
	l1, _ := net.Listen("tcp", vs1)
	ts1.Listener.Close()
	ts1.Listener = l1
	ts1.Start()
	defer ts1.Close()

	ts2 := httptest.NewUnstartedServer(h2)
	l2, _ := net.Listen("tcp", vs2)
	ts2.Listener.Close()
	ts2.Listener = l2
	ts2.Start()
	defer ts2.Close()

	ts3 := httptest.NewUnstartedServer(h3)
	l3, _ := net.Listen("tcp", vs3)
	ts3.Listener.Close()
	ts3.Listener = l3
	ts3.Start()
	defer ts3.Close()

	hosts := common.NewHosts()
	hosts.Add(vs1)
	hosts.Add(vs2)
	hosts.Add(vs3)

	dus := make(map[string]*common.DiskUsage)
	collectRemoteDiskUsage(hosts, dus)

	assert.Equal(t, d1.TotalSize, dus[vs1].TotalSize)
	assert.Equal(t, d1.UsedSize, dus[vs1].UsedSize)
	assert.Equal(t, d1.FreeSize, dus[vs1].FreeSize)
	assert.Equal(t, d1.UsedPercent, dus[vs1].UsedPercent)

	assert.Equal(t, d2.TotalSize, dus[vs2].TotalSize)
	assert.Equal(t, d2.UsedSize, dus[vs2].UsedSize)
	assert.Equal(t, d2.FreeSize, dus[vs2].FreeSize)
	assert.Equal(t, d2.UsedPercent, dus[vs2].UsedPercent)

	assert.Equal(t, d3.TotalSize, dus[vs3].TotalSize)
	assert.Equal(t, d3.UsedSize, dus[vs3].UsedSize)
	assert.Equal(t, d3.FreeSize, dus[vs3].FreeSize)
	assert.Equal(t, d3.UsedPercent, dus[vs3].UsedPercent)
}

func TestSetDiskUsageLimitPercent(t *testing.T) {

	assert.NotNil(t, SetDiskUsageLimitPercent(-1))
	assert.NotNil(t, SetDiskUsageLimitPercent(101))
	assert.Nil(t, SetDiskUsageLimitPercent(0))
	assert.Nil(t, SetDiskUsageLimitPercent(50))
}
