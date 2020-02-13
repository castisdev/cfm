package fmfm

import (
	"testing"

	"github.com/castisdev/cfm/common"
)

// 초기 event 가 없고,
// file 이 변경되지 않은 경우, timeout만 발생
func TestWatchPollNoInitialEventTOEventWithoutFileChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	to := uint32(1)
	poll := uint32(1)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", false, to, poll)
	watcher.mode = POLL

	go watcher.Watch()
	tos := common.Start()
	// timeout 발생
	// timeout 후에 timeout 주기 시작
	tos = waitWatcherTimeout(t, watcher, tos, to)

	// timeout 발생
	// timeout 후에 timeout 주기 시작
	tos = waitWatcherTimeout(t, watcher, tos, to)
	waitWatcherClose(watcher)
}

// poll 주기가 짧을 때
func TestWatchPollTOEventAndFileEventWithRecreated(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	to := uint32(2)
	poll := uint32(1)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, to, poll)
	watcher.mode = POLL

	go watcher.Watch()
	polls := common.Start()

	// 초기 event
	// 초기 event 후에 polling 주기가 시작됨
	polls = waitWatcherEvent(t, watcher, polls, poll)
	// 초기 event 후에 timeout 주기 시작
	tos := common.Start()

	// timeout 발생
	// timeout 후에 timeout 주기 시작
	tos = waitWatcherTimeout(t, watcher, tos, to)

	// 지웠다가, 다시 create해서 event 발생 시킴
	// poll 주기 후에 발견됨
	polls = common.Start()
	deletefile("testwatcher", "hitcount")
	createfile("testwatcher", "hitcount")

	// 드디어, modify time 이 변경된 것을 발견함
	polls = waitWatcherEvent(t, watcher, polls, poll)
	waitWatcherClose(watcher)
}

func TestWatchPollFileEventOnlyWithModTimeChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	to := uint32(0)
	poll := uint32(1)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, to, poll)
	watcher.mode = POLL

	go watcher.Watch()
	polls := common.Start()

	// 초기 event
	polls = waitWatcherEvent(t, watcher, polls, poll)

	// 시간 변경으로 event 발생 시킴
	chtimesfile("testwatcher", "grade")

	// event 후에 polling 주기가 시작됨
	polls = waitWatcherEvent(t, watcher, polls, poll)

	// 시간 변경으로 event 발생 시킴
	chtimesfile("testwatcher", "grade")
	// event 후에 polling 주기가 시작됨
	polls = waitWatcherEvent(t, watcher, polls, poll)

	waitWatcherClose(watcher)
}
