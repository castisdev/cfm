package heartbeater

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cilog"
)

type HBStatus int
type HBTime int64

const (
	NOTOK HBStatus = iota
	OK
)

func (s HBStatus) String() string {
	m := map[HBStatus]string{
		NOTOK: "notok",
		OK:    "ok",
	}
	return m[s]
}

type HBHost struct {
	common.Host
	Status HBStatus
	Mtime  HBTime
}

func (t HBTime) String() string {
	return time.Unix(int64(t), 0).Format(time.RFC3339)
}

func (h HBHost) String() string {
	s := fmt.Sprintf(
		"Host(%s), Status(%s), Mtime(%s)",
		h.Host, h.Status, h.Mtime)

	return s
}

var hosts map[string]*HBHost
var timeoutSec uint
var sleepSec uint
var rwlock *sync.RWMutex
var hber common.MLogger

func init() {
	hosts = make(map[string]*HBHost, 0)
	timeoutSec = uint(5)
	sleepSec = uint(10)
	rwlock = &sync.RWMutex{}

	hber = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "heartbeater"}
}

func Release() {
	rwlock.Lock()
	defer rwlock.Unlock()

	hosts = make(map[string]*HBHost, 0)
}

func SetTimoutSec(s uint) {
	hber.Debugf("set timeoutSec(%d)", s)
	timeoutSec = s
}

func SetSleepSec(s uint) {
	hber.Debugf("set sleepSec(%d)", s)
	sleepSec = s
}

// Add :
//
// host 정보 추가
//
// 이미 Add했던 host 를 add 하면 error 리턴
func Add(s string) error {
	rwlock.Lock()
	defer rwlock.Unlock()

	host, err := common.SplitHostPort(s)
	if err != nil {
		return err
	}
	if _, ok := hosts[host.Addr]; !ok {
		hosts[host.Addr] = &HBHost{host, NOTOK, HBTime(time.Now().Unix())}
	} else {
		s := fmt.Sprintf("already added host(%s)", host)
		return errors.New(s)
	}
	return nil
}

// Delete :
// host 정보 제거
func Delete(s string) {
	rwlock.Lock()
	defer rwlock.Unlock()

	delete(hosts, s)
}

// GetList :
// sort된 host heartbeat status list를 반환
func GetList() (hl []HBHost) {
	rwlock.RLock()
	defer rwlock.RUnlock()

	for _, h := range hosts {
		hl = append(hl, *h)
	}

	sort.Slice(hl, func(i, j int) bool {
		return hl[i].Addr > hl[j].Addr
	})

	return hl
}

// Get :
// host 정보 빈환
func Get(s string) (HBHost, bool) {
	rwlock.RLock()
	defer rwlock.RUnlock()

	h, ok := hosts[s]
	if !ok {
		return HBHost{}, false
	}
	return *h, true
}

// Update :
// host 정보 update
// 없는 host 정보 무시됨
func Update(host HBHost) bool {
	rwlock.Lock()
	defer rwlock.Unlock()

	_, ok := hosts[host.Addr]
	if !ok {
		return false
	}
	hosts[host.Addr].Status = host.Status
	hosts[host.Addr].Mtime = host.Mtime
	return true
}

// UpdateList :
// host list로 부터 host 정보 update
// 없는 host 정보 무시됨
func UpdateList(hl []HBHost) {
	rwlock.Lock()
	defer rwlock.Unlock()

	for _, host := range hl {
		_, ok := hosts[host.Addr]
		if ok {
			hosts[host.Addr].Status = host.Status
			hosts[host.Addr].Mtime = host.Mtime
		}
	}
}

// Heartbeat :
func Heartbeat() {
	hl := GetList()

	newhl := make([]HBHost, len(hl))
	for _, h := range hl {
		rc, err := common.Heartbeat(&h.Host, timeoutSec)
		if rc {
			h.Status = OK
			h.Mtime = HBTime(time.Now().Unix())
			hber.Debugf("[%s] heartbeat, host(%s)", h.Host, h)
		} else {
			h.Status = NOTOK
			h.Mtime = HBTime(time.Now().Unix())
			if err != nil {
				hber.Errorf("[%s] fail to heartbeat, host(%s), timeout(%d), error(%s)",
					h.Host, h, timeoutSec, err.Error())
			} else {
				hber.Errorf("[%s] fail to heartbeat, host(%s), timeout(%d)", h.Host, h, timeoutSec)
			}
		}
		newhl = append(newhl, h)
	}

	UpdateList(newhl)
}

// RunForever : host heartbeat 검사
func RunForever() {
	for {
		Heartbeat()
		time.Sleep(time.Second * time.Duration(sleepSec))
	}
}
