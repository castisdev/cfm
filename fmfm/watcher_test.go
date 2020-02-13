package fmfm

import (
	"errors"
	"log"
	"testing"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/myinotify"
	"github.com/stretchr/testify/assert"
)

func TestNewWatcherMode(t *testing.T) {
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("grade", "hictount", true, 0, 0)
	assert.Equal(t, POLL, watcher.mode)

	TestInotifyFunc = func() bool { return true }
	watcher = NewWatcher("grade", "hitcount", true, 0, 0)
	assert.Equal(t, NOTIFY, watcher.mode)

	TestInotifyFunc = func() bool { return true }
	NewWatcherFunc = func() (*myinotify.Watcher, error) { return nil, errors.New("test:failed to create") }
	watcher = NewWatcher("grade", "hitcount", true, 0, 0)
	assert.Equal(t, POLL, watcher.mode)
}

func TestWatcherPollClone(t *testing.T) {
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("grade", "hictount", true, 0, 0)
	assert.Equal(t, POLL, watcher.mode)

	clone := watcher.clone()
	assert.Equal(t, watcher.mode, clone.mode)
	assert.Equal(t, watcher.grade.FilePath, clone.grade.FilePath)
	assert.Equal(t, watcher.hitCount.FilePath, clone.hitCount.FilePath)
	assert.Equal(t, watcher.initialNoti, clone.initialNoti)
	assert.Equal(t, watcher.timeoutSec, clone.timeoutSec)
	assert.Equal(t, watcher.pollingSec, clone.pollingSec)
}

// 파일 둘 다 없는 경우 timeout이 남
func TestWatchPollTimout(t *testing.T) {
	to := uint32(1)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("grade", "hitcount", true, to, 1)
	watcher.mode = POLL

	go watcher.Watch()
	s := common.Start()

	waitWatcherTimeout(t, watcher, s, to)
	waitWatcherClose(watcher)
}

// 파일 둘 중에 하나만 있는 경우에도 timeout이 남
func TestWatchPollTimoutOneExistTheOtherNotExist(t *testing.T) {
	createfile("testwatcher", "grade")
	defer deletefile("testwatcher", "")
	to := uint32(1)
	TestInotifyFunc = func() bool { return false }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, to, 1)
	watcher.mode = POLL

	go watcher.Watch()
	s := common.Start()

	waitWatcherTimeout(t, watcher, s, to)
	waitWatcherClose(watcher)
}

// poll 모드는
// 초기, 파일이 있는 지 여부와
// 이후, modify time 으로 변경했는지에 따라,
// event 발생
// poll 주기 사이에 modify time이 변했다고 해도
// timeout event 가 발생할 수 있음
func TestWatchPollTOEventAndFileEventWithModTimeChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	to := uint32(1)
	poll := uint32(to*2 + 1)
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

	// 시간 변경으로 event 발생 시킴
	// 다음 poll 주기에 발견됨
	// 다음 poll 주기 전에 timeout evnet 가 발생함
	chtimesfile("testwatcher", "grade")

	// 다음 poll 주기 전에 timeout event 발생
	tos = waitWatcherTimeout(t, watcher, tos, to)

	// 드디어, modify time 이 변경된 것을 발견함
	polls = waitWatcherEvent(t, watcher, polls, poll)
	waitWatcherClose(watcher)
}

func waitWatcherEvent(t *testing.T, w *Watcher, start time.Time, duSec uint32) time.Time {
	noti := <-w.NotiCh
	assert.Equal(t, nil, noti.Err)
	diff := common.Elapsed(start) - time.Second*time.Duration(duSec)
	log.Println("event duration:", diff)
	assert.True(t, diff < time.Duration(1)*time.Millisecond)

	return common.Start()
}

func waitWatcherTimeout(t *testing.T, w *Watcher, start time.Time, duSec uint32) time.Time {
	noti := <-w.NotiCh
	assert.Equal(t, ErrTimeout, noti.Err)
	diff := common.Elapsed(start) - time.Second*time.Duration(duSec)
	log.Println("timeout duration:", diff)
	assert.True(t, diff < time.Duration(1)*time.Millisecond)

	return common.Start()
}

func waitWatcherCloseErrNotExist(t *testing.T, w *Watcher) {
	_, opennotich := <-w.NotiCh
	assert.Equal(t, false, opennotich)

	err, openerrch := <-w.ErrCh
	assert.Equal(t, true, openerrch)
	assert.NotEqual(t, nil, err)
	assert.Equal(t, ErrNotExist.Error(), err.Error())
	log.Println(err)
	_, openerrch = <-w.ErrCh
	assert.Equal(t, false, openerrch)
}

func waitWatcherClose(w *Watcher) {
	if !w.isCloseDoneCh() {
		close(w.doneCh)
	}
	<-w.ErrCh
}
