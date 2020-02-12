package fmfm

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/myinotify"
	"github.com/stretchr/testify/assert"
)

func TestNewFileMonitor(t *testing.T) {
	now := time.Now()
	tctbl := []struct {
		path  string
		expfm FileMonitor
	}{
		{path: "./d1/d2/file",
			expfm: FileMonitor{
				Event:    myinotify.Event{Name: "./d1/d2/file", Op: 0},
				FilePath: "./d1/d2/file",
				Dir:      "d1/d2",
				Name:     "file",
				Dirs:     []string{"d1/d2", "d1", "."},
				Exist:    false,
				Mtime:    now,
			}},
		{path: "d1/d2/file",
			expfm: FileMonitor{
				Event:    myinotify.Event{Name: "d1/d2/file", Op: 0},
				FilePath: "d1/d2/file",
				Dir:      "d1/d2",
				Name:     "file",
				Dirs:     []string{"d1/d2", "d1", "."},
				Exist:    false,
				Mtime:    now,
			}},
		{path: "/d1/d2/file",
			expfm: FileMonitor{
				Event:    myinotify.Event{Name: "/d1/d2/file", Op: 0},
				FilePath: "/d1/d2/file",
				Dir:      "/d1/d2",
				Name:     "file",
				Dirs:     []string{"/d1/d2", "/d1", "/"},
				Exist:    false,
				Mtime:    now,
			}},
	}
	for _, tc := range tctbl {
		fm := NewFileMonitor(tc.path)
		fm.Mtime = now
		assert.Equal(t, tc.expfm, *fm)
	}
}

func TestNewDirs(t *testing.T) {
	tctbl := []struct {
		path    string
		expdirs []string
	}{
		{path: "d1/d2/file", expdirs: []string{"d1/d2", "d1", "."}},
		{path: "./d1/d2/file", expdirs: []string{"d1/d2", "d1", "."}},
		{path: "../d1/d2/file", expdirs: []string{"../d1/d2", "../d1", ".."}},
		{path: "/d1/d2/file", expdirs: []string{"/d1/d2", "/d1", "/"}},
		// 잘못된 path지만, error 처리가 안됨
		{path: ".../d1/d2/file", expdirs: []string{".../d1/d2", ".../d1", "...", "."}},
		// file 이 없어도 directory parsing이 됨
		{path: "d1/d2/", expdirs: []string{"d1/d2", "d1", "."}},
		{path: "/d1/d2/", expdirs: []string{"/d1/d2", "/d1", "/"}},
	}
	for _, tc := range tctbl {
		dirs := newDirs(tc.path)
		assert.Equal(t, tc.expdirs, dirs)
	}
}

func TestFileMonitorUpdateAndReset(t *testing.T) {
	tctbl := []struct {
		event   myinotify.Op
		exist   bool
		updated bool
	}{
		{event: myinotify.Create, exist: true, updated: false},
		{event: myinotify.Write, exist: true, updated: true},
		{event: myinotify.Remove, exist: false, updated: false},
		{event: myinotify.Chmod, exist: true, updated: true},
		{event: 0, updated: false},
	}

	f := NewFileMonitor("d1/file")
	for _, tc := range tctbl {
		f.Update(tc.event)
		if tc.event != 0 {
			assert.Equal(t, tc.exist, f.Exist)
			assert.Equal(t, tc.updated, f.Updated)
		} else {
			assert.Equal(t, tc.updated, f.Updated)
		}

		f.ResetUpdate()
		assert.Equal(t, false, f.Updated)
		assert.Equal(t, myinotifyUnknown, f.Event.Op)
	}
}

func TestFileMonitorUpdateMtimeAndResetUpadte(t *testing.T) {
	createfile("testwatcher", "exist")
	defer deletefile("testwatcher", "")
	fi, _ := os.Stat("testwatcher/exist")
	fimtime := fi.ModTime()

	tctbl := []struct {
		path    string
		mtime   time.Time
		exist   bool
		updated bool
	}{
		{path: "testwatcher/exist", mtime: fimtime, exist: true, updated: true},
		{path: "testwatcher/notexist", exist: false, updated: false},
	}
	for _, tc := range tctbl {
		f := NewFileMonitor(tc.path)
		ok := f.UpdateMtime()
		if ok {
			assert.Equal(t, tc.exist, f.Exist)
			assert.Equal(t, tc.updated, f.Updated)
			assert.Equal(t, tc.mtime, f.Mtime)
		} else {
			assert.Equal(t, tc.exist, f.Exist)
			assert.Equal(t, tc.updated, f.Updated)
		}

		f.ResetUpdate()
		assert.Equal(t, false, f.Updated)
		assert.Equal(t, myinotifyUnknown, f.Event.Op)
	}
}

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
	if !w.isClosed() {
		close(w.doneCh)
	}
	<-w.doneCh
}
