package heartbeater

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func cfw(cfwaddr string, success bool) *httptest.Server {
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("HEAD").Path("/hb").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if success {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
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

func TestHearbeat(t *testing.T) {
	a1 := "127.0.0.1:18881"
	cfw1 := cfw(a1, true)
	cfw1.Start()
	defer cfw1.Close()

	a2 := "127.0.0.1:18882"
	cfw2 := cfw(a2, true)
	cfw2.Start()
	defer cfw2.Close()

	a3 := "127.0.0.1:18883"
	cfw3 := cfw(a3, false)
	cfw3.Start()
	defer cfw3.Close()

	Add(a1)
	Add(a2)
	Add(a3)

	Heartbeat()
	hbhosts := GetList()

	hbhost0 := hbhosts[0]
	assert.Equal(t, a3, hbhost0.Addr)
	assert.Equal(t, NOTOK, hbhost0.Status)

	hbhost1 := hbhosts[1]
	assert.Equal(t, a2, hbhost1.Addr)
	assert.Equal(t, OK, hbhost1.Status)

	hbhost2 := hbhosts[2]
	assert.Equal(t, a1, hbhost2.Addr)
	assert.Equal(t, OK, hbhost2.Status)
}

func TestAddGetListGetDelete(t *testing.T) {
	a1 := "127.0.0.1:18881"
	a2 := "127.0.0.1:18882"
	a3 := "127.0.0.1:18883"
	Add(a1)
	Add(a2)
	Add(a3)
	hbhosts := GetList()
	assert.Equal(t, a3, hbhosts[0].Addr)
	assert.Equal(t, a2, hbhosts[1].Addr)
	assert.Equal(t, a1, hbhosts[2].Addr)

	h3, ok := Get(a3)
	assert.Equal(t, a3, h3.Addr)
	assert.Equal(t, true, ok)

	_, ok = Get("127.0.0.1")
	assert.Equal(t, false, ok)

	Delete(a3)
	hbhosts = GetList()
	assert.Equal(t, a2, hbhosts[0].Addr)
	assert.Equal(t, a1, hbhosts[1].Addr)

	h3.Status = OK
	ok = Update(h3)
	assert.Equal(t, false, ok)

	h2, ok := Get(a2)
	assert.Equal(t, a2, h2.Addr)
	assert.Equal(t, true, ok)

	h2.Status = OK
	ok = Update(h2)
	assert.Equal(t, true, ok)

	newh2, newok := Get(a2)
	assert.Equal(t, a2, newh2.Addr)
	assert.Equal(t, OK, newh2.Status)
	assert.Equal(t, true, newok)

	hl := []HBHost{
		{Host: common.Host{IP: "127.0.0.1", Port: 18881, Addr: "127.0.0.1:18881"}, Status: OK, Mtime: HBTime(time.Now().Unix())},
		{Host: common.Host{IP: "127.0.0.1", Port: 18882, Addr: "127.0.0.1:18882"}, Status: OK, Mtime: HBTime(time.Now().Unix())},
	}
	UpdateList(hl)
	hbhosts = GetList()
	assert.Equal(t, a2, hbhosts[0].Addr)
	assert.Equal(t, OK, hbhosts[0].Status)
	assert.Equal(t, a1, hbhosts[1].Addr)
	assert.Equal(t, OK, hbhosts[1].Status)
}
