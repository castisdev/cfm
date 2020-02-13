package fmfm

import (
	"log"
	"testing"
	"time"

	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cfm/tasker"
	"github.com/stretchr/testify/assert"
)

func TestManagePollModeStop(t *testing.T) {
	poll := uint32(1)
	eto := uint32(0)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, eto, poll)

	DefaultEventRuns = []RUN{NOP}
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	manager := NewManager(watcher, runner)
	go manager.Manage()

	select {
	case manager.CMDCh <- STOP:
		<-manager.ErrCh
		_, open := <-manager.CMDCh
		assert.Equal(t, false, open)
	}
}

func TestManagePollInitialEventWithoutFileChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	poll := uint32(1)
	eto := uint32(0)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, eto, poll)

	DefaultEventRuns = []RUN{NOP}
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	runcount := 0
	runner.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			log.Println("eventrun:NOP")
		},
	}

	m := NewManager(watcher, runner)
	go m.Manage()

	// poll time 이 N번의 시간이 지나도, 파일에 변화가 없으므로,
	// 최초의 event가 발생하지 않게 watcher를 setting했기 때문에,
	// runner는 최초 한 번만 실행된다.
	doSomething(int(poll))
	doSomething(int(poll))
	assert.Equal(t, 1, runcount)

	waitManagerStop(m)
}

func TestManagePollFirstEventWithoutFileChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	poll := uint32(1)
	eto := uint32(0)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", false, eto, poll)

	DefaultEventRuns = []RUN{NOP}
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	runcount := 0
	runner.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			log.Println("eventrun:NOP")
		},
	}

	m := NewManager(watcher, runner)
	go m.Manage()

	// poll time 이 N번의 시간이 지나도, 파일에 변화가 없으므로,
	// 최초의 event가 발생하지 않게 watcher를 setting 했으므로,
	// 파일에 변화가 없으면,
	// runner가 실행되지 않는다.
	doSomething(int(poll))
	doSomething(int(poll))
	assert.Equal(t, 0, runcount)

	waitManagerStop(m)
}

func TestManagePollFirstEventWithFileChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	poll := uint32(1)
	eto := uint32(0)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, eto, poll)

	DefaultEventRuns = []RUN{NOP}
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	runcount := 0
	runner.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			log.Println("eventrun:NOP")
		},
	}

	m := NewManager(watcher, runner)
	go m.Manage()

	// 최초 한 번, 시간 변하고 한 번으로 두 번 실행된다.
	doSomething(int(poll))
	chtimesfile("testwatcher", "grade")
	doSomething(int(poll))
	doSomething(int(poll))
	assert.Equal(t, 2, runcount)

	waitManagerStop(m)
}

func TestManagePollFirstAndTimeoutEventWithoutFileChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	poll := uint32(1)
	eto := uint32(1)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, eto, poll)

	DefaultEventRuns = []RUN{NOP}
	DefaultEventTimeoutRuns = []RUN{PrintFMM}
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	runcount, etoruncount := 0, 0
	runner.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			log.Println("eventrun:NOP")
		},
		PrintFMM: func(r *Runner, e FileMetaFilesEvent) {
			etoruncount++
			log.Println("eventtimoutrun:PrintFMM")
		},
	}

	m := NewManager(watcher, runner)
	go m.Manage()

	// 최초 event 한 번 실행
	// timeoutevent 두 번 또는 세 번 실행
	doSomething(int(poll))
	doSomething(int(poll))
	doSomething(int(poll))
	assert.Equal(t, 1, runcount)
	assert.True(t, 2 == etoruncount || 3 == etoruncount)

	waitManagerStop(m)
}

func TestManagePollFirstAndBetweenEventsWithoutFileChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	poll := uint32(1)
	eto := uint32(0)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, eto, poll)

	DefaultEventRuns = []RUN{NOP}
	DefaultBetweenEventsRuns = []RUN{PrintFMM}
	betweenEventsRunSec := uint32(1)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	runcount, bteruncount := 0, 0
	runner.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			log.Println("eventrun:NOP")
		},
		PrintFMM: func(r *Runner, e FileMetaFilesEvent) {
			bteruncount++
			log.Println("betweeneventsrun:PrintFMM")
		},
	}

	m := NewManager(watcher, runner)
	go m.Manage()

	// 최초 event 한 번 실행
	// 이벤트와 이벤트 사이에 두 번 또는 세 번 실행
	doSomething(int(poll))
	doSomething(int(poll))
	doSomething(int(poll))
	assert.Equal(t, 1, runcount)
	assert.True(t, 2 == bteruncount || 3 == bteruncount)

	waitManagerStop(m)
}

func waitManagerStop(m *Manager) {
	select {
	case m.CMDCh <- STOP:
		<-m.ErrCh
		return
	}
}

func doSomething(sec int) {
	time.Sleep(time.Duration(sec) * time.Second)
}
