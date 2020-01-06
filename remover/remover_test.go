package remover_test

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/castisdev/cfm/common"
	"github.com/stretchr/testify/assert"

	"github.com/castisdev/cfm/remover"
)

func Test_collectRemoteDiskUsage(t *testing.T) {

	vs1 := "127.0.0.1:18881"
	vs2 := "127.0.0.1:18882"
	vs3 := "127.0.0.1:18883"

	// setup dummy http server
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		b, err := json.Marshal(d1)
		assert.Nil(t, err)
		w.Write(b)
	})

	d2 := common.DiskUsage{
		TotalSize: 2000, UsedSize: 1000,
		FreeSize: 1000, AvailSize: 1000, UsedPercent: 50,
	}
	h2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		b, err := json.Marshal(d2)
		assert.Nil(t, err)
		w.Write(b)
	})

	d3 := common.DiskUsage{
		TotalSize: 3000, UsedSize: 1500,
		FreeSize: 1500, AvailSize: 1500, UsedPercent: 50,
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

	remover.SetDiskUsageLimitPercent(55)
	dservers := remover.GetServerWithNotEnoughDisk(hosts)

	assert.Equal(t, 1, len(dservers))

	if len(dservers) < 1 {
		t.Fatal("cannot find server that run out of disk space")
	}
	overuseServer := dservers[0]
	assert.Equal(t, uint(60), overuseServer.Du.UsedPercent)

	overUsedSize := overuseServer.Du.GetOverUsedSize(remover.DiskUsageLimitPercent())
	assert.Less(t, uint64(0), uint64(overUsedSize))
	// 600(current used) - 550(limit used : 1000 * 55%)
	assert.Equal(t, uint64(50), uint64(overUsedSize))
}

func TestSetDiskUsageLimitPercent(t *testing.T) {
	assert.NotNil(t, remover.SetDiskUsageLimitPercent(101))
	assert.Nil(t, remover.SetDiskUsageLimitPercent(0))
	assert.Nil(t, remover.SetDiskUsageLimitPercent(50))
}
