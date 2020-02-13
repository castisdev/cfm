package fmfm

import (
	"log"
	"testing"

	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cfm/tasker"
	"github.com/stretchr/testify/assert"
)

func TestManageNotifyFirstAndWithFileDeleted(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	notify := uint32(1)
	poll := uint32(0)
	eto := uint32(0)
	TestInotifyFunc = func() bool { return true }
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
	// 최초 한 번 실행 후
	doSomething(int(notify))
	// 파일이 지워지면,
	// watcher는 내려가고,
	// runner는 event가 없는 동안, betweenEventsRun을 실행한다.
	deletefile("testwatcher", "grade")
	doSomething(int(notify))
	doSomething(int(notify))
	assert.Equal(t, true, testWatcherDown(t, watcher))
	assert.Equal(t, 1, runcount)
	assert.True(t, 2 == bteruncount || 3 == bteruncount)

	waitManagerStop(m)
}

func TestManageNotifyFirstAndWithDirectoryDeleted(t *testing.T) {
	createdir("testparent")
	createfile("testparent/testwatcher", "grade")
	createfile("testparent/testwatcher", "hitcount")
	defer deletefile("testwatcher", "")
	defer deletefile("testparent", "")

	notify := uint32(1)
	poll := uint32(0)
	eto := uint32(0)
	TestInotifyFunc = func() bool { return true }
	watcher := NewWatcher("testparent/testwatcher/grade", "testparent/testwatcher/hitcount", true, eto, poll)

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
	// 최초 한 번 실행 후
	doSomething(int(notify))
	// 상위 directory가 지워지면,
	// 파일이 지워지면서
	// watcher는 내려가고,
	// runner는 event가 없는 동안, betweenEventsRun을 실행한다.
	deletefile("testparent", "")
	doSomething(int(notify))
	doSomething(int(notify))
	assert.Equal(t, true, testWatcherDown(t, watcher))
	assert.Equal(t, 1, runcount)
	assert.True(t, 2 == bteruncount || 3 == bteruncount)

	waitManagerStop(m)
}

func TestManageNotifyFirstAndWithDirectoryRenamed(t *testing.T) {
	createdir("testparent")
	createfile("testparent/testwatcher", "grade")
	createfile("testparent/testwatcher", "hitcount")
	defer deletefile("testwatcher", "")
	defer deletefile("testparent", "")
	defer deletefile("newtestparent", "")

	notify := uint32(1)
	poll := uint32(0)
	eto := uint32(0)
	TestInotifyFunc = func() bool { return true }
	watcher := NewWatcher("testparent/testwatcher/grade", "testparent/testwatcher/hitcount", true, eto, poll)

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
	// 최초 한 번 실행 후
	doSomething(int(notify))
	// 상위 directory가 지워지면,
	// 파일이 지워지면서
	// watcher는 내려가고,
	// runner는 event가 없는 동안, betweenEventsRun을 실행한다.
	renamedir("testparent", "newtestparent")
	doSomething(int(notify))
	doSomething(int(notify))
	assert.Equal(t, true, testWatcherDown(t, watcher))
	assert.Equal(t, 1, runcount)
	assert.True(t, 2 == bteruncount || 3 == bteruncount)

	waitManagerStop(m)
}

func testWatcherDown(t *testing.T, w *Watcher) bool {
	select {
	case <-w.ErrCh:
		return true
	default:
		return false
	}
}
