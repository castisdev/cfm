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
	watcher := NewWatcher("grade", "hicount", true, 0, 0)
	assert.Equal(t, POLL, watcher.mode)

	TestInotifyFunc = func() bool { return true }
	watcher = NewWatcher("grade", "hicount", true, 0, 0)
	assert.Equal(t, NOTIFY, watcher.mode)

	TestInotifyFunc = func() bool { return true }
	NewWatcherFunc = func() (*myinotify.Watcher, error) { return nil, errors.New("test:failed to create") }
	watcher = NewWatcher("grade", "hicount", true, 0, 0)
	assert.Equal(t, POLL, watcher.mode)
}

// 파일 둘 다 없는 경우 timeout이 남
func TestWatchPollTimout(t *testing.T) {
	to := uint32(1)
	watcher := NewWatcher("grade", "hicount", true, to, 1)
	watcher.mode = POLL

	timeout := FileMetaFilesEvent{Err: ErrTimeout}
	go watcher.Watch()

	s := common.Start()
	noti := <-watcher.NotiCh
	assert.Equal(t, timeout.Err, noti.Err)
	log.Println("diff:", common.Elapsed(s)-time.Second*time.Duration(to))
	assert.True(t, common.Elapsed(s)-time.Second*time.Duration(to) < time.Duration(1)*time.Millisecond)
}

// 파일 둘 중에 하나만 있는 경우에도 timeout이 남
func TestWatchPollTimoutOneExistTheOtherNotExist(t *testing.T) {
	createfile("testwatcher", "grade")
	to := uint32(1)
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hicount", true, to, 1)
	watcher.mode = POLL

	timeout := FileMetaFilesEvent{Err: ErrTimeout}
	go watcher.Watch()

	s := common.Start()
	noti := <-watcher.NotiCh
	assert.Equal(t, timeout.Err, noti.Err)
	log.Println("diff:", common.Elapsed(s)-time.Second*time.Duration(to))
	assert.True(t, common.Elapsed(s)-time.Second*time.Duration(to) < time.Duration(1)*time.Millisecond)
}

// poll 모드는
// 초기, 파일이 있는 지 여부와
// 이후, modify time 으로 변경했는지에 따라,
// event 발생
// poll 주기 사이에 modify time이 변했다고 해도
// timeout event 가 발생할 수 있음
func TestWatchPollTOEventAndFileEventWtithModTimeChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hicount")
	defer deletefile("testwatcher", "")

	to := uint32(3)
	poll := uint32(to*2 + 1)
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hicount", true, to, poll)
	watcher.mode = POLL

	timeout := FileMetaFilesEvent{Err: ErrTimeout}

	go watcher.Watch()
	polls := common.Start()

	noti := <-watcher.NotiCh
	// 초기 event
	assert.Equal(t, nil, noti.Err)
	diff := common.Elapsed(polls) - time.Second*time.Duration(poll)
	log.Println("poll:", diff)
	assert.True(t, diff < time.Duration(1)*time.Millisecond)
	// 초기 event 후에 polling 주기가 시작됨
	polls = common.Start()

	// 초기 event 후에 timeout 주기 시작
	tos := common.Start()
	noti = <-watcher.NotiCh
	assert.Equal(t, timeout.Err, noti.Err)
	todiff := common.Elapsed(tos) - time.Second*time.Duration(to)
	log.Println("timeout:", todiff)
	assert.True(t, todiff < time.Duration(1)*time.Millisecond)
	// timeout 후에 timeout 주기 시작
	tos = common.Start()

	// 시간 변경으로 event 발생 시킴
	// 다음 poll 주기에 발견됨
	// 다음 poll 주기 전에 timeout evnet 가 발생함
	chtimesfile("testwatcher", "grade")

	// 다음 poll 주기 전에 timeout event 발생
	noti = <-watcher.NotiCh
	assert.Equal(t, timeout.Err, noti.Err)
	todiff = common.Elapsed(tos) - time.Second*time.Duration(to)
	log.Println("timeout:", todiff)
	assert.True(t, todiff < time.Duration(1)*time.Millisecond)

	// 드디어, modify time 이 변경된 것을 발견함
	noti = <-watcher.NotiCh
	assert.Equal(t, nil, noti.Err)
	diff = common.Elapsed(polls) - time.Second*time.Duration(poll)
	log.Println("poll:", diff)
	assert.True(t, diff < time.Duration(1)*time.Millisecond)
}

func TestWatchPollTOEventAndFileEventWtithRecreated(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hicount")
	defer deletefile("testwatcher", "")

	to := uint32(3)
	poll := uint32(to*2 + 1)
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hicount", true, to, poll)
	watcher.mode = POLL

	timeout := FileMetaFilesEvent{Err: ErrTimeout}

	go watcher.Watch()
	polls := common.Start()

	noti := <-watcher.NotiCh
	// 초기 event
	assert.Equal(t, nil, noti.Err)
	diff := common.Elapsed(polls) - time.Second*time.Duration(poll)
	log.Println("poll:", diff)
	assert.True(t, diff < time.Duration(1)*time.Millisecond)
	// 초기 event 후에 polling 주기가 시작됨
	polls = common.Start()

	// 초기 event 후에 timeout 주기 시작
	tos := common.Start()
	noti = <-watcher.NotiCh
	assert.Equal(t, timeout.Err, noti.Err)
	todiff := common.Elapsed(tos) - time.Second*time.Duration(to)
	log.Println("timeout:", todiff)
	assert.True(t, todiff < time.Duration(1)*time.Millisecond)
	// timeout 후에 timeout 주기 시작
	tos = common.Start()

	noti = <-watcher.NotiCh
	assert.Equal(t, timeout.Err, noti.Err)
	todiff = common.Elapsed(tos) - time.Second*time.Duration(to)
	log.Println("timeout:", todiff)
	assert.True(t, todiff < time.Duration(1)*time.Millisecond)

	// 지웠다가, 다시 create해서 event 발생 시킴
	// 이렇게 해도 ModTime이 변하는 효과가 생김
	deletefile("testwatcher", "hicount")
	createfile("testwatcher", "hicount")

	noti = <-watcher.NotiCh
	assert.Equal(t, nil, noti.Err)
	diff = common.Elapsed(polls) - time.Second*time.Duration(poll)
	log.Println("poll:", diff)
	assert.True(t, diff < time.Duration(1)*time.Millisecond)
}

func TestWatchPollFileEventOnlyWtithModTimeChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hicount")
	defer deletefile("testwatcher", "")

	to := uint32(0)
	poll := uint32(2)
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hicount", true, to, poll)
	watcher.mode = POLL

	go watcher.Watch()
	polls := common.Start()

	noti := <-watcher.NotiCh
	// 초기 event
	assert.Equal(t, nil, noti.Err)
	diff := common.Elapsed(polls) - time.Second*time.Duration(poll)
	log.Println("poll:", diff)
	assert.True(t, diff < time.Duration(1)*time.Millisecond)
	// 초기 event 후에 polling 주기가 시작됨
	polls = common.Start()

	// 시간 변경으로 event 발생 시킴
	chtimesfile("testwatcher", "grade")
	noti = <-watcher.NotiCh
	assert.Equal(t, nil, noti.Err)
	diff = common.Elapsed(polls) - time.Second*time.Duration(poll)
	log.Println("poll:", diff)
	assert.True(t, diff < time.Duration(1)*time.Millisecond)
	// event 후에 polling 주기가 시작됨
	polls = common.Start()

	// 시간 변경으로 event 발생 시킴
	chtimesfile("testwatcher", "grade")
	noti = <-watcher.NotiCh
	assert.Equal(t, nil, noti.Err)
	diff = common.Elapsed(polls) - time.Second*time.Duration(poll)
	log.Println("poll:", diff)
	assert.True(t, diff < time.Duration(1)*time.Millisecond)
	// event 후에 polling 주기가 시작됨
	polls = common.Start()

	// 시간 변경으로 event 발생 시킴
	chtimesfile("testwatcher", "grade")
	noti = <-watcher.NotiCh
	assert.Equal(t, nil, noti.Err)
	diff = common.Elapsed(polls) - time.Second*time.Duration(poll)
	log.Println("poll:", diff)
	assert.True(t, diff < time.Duration(1)*time.Millisecond)
}

// notify 모드에서는 두 피일 중 하나라도 없을 때 return 됨
// return 되면서, notiCh은 닫히고
// errCh 에는 error를 넘기고 error를 받으면 errCh은 닫힘
func TestWatchNotifyErrorTwoFileNotExist(t *testing.T) {
	to := uint32(2)
	watcher := NewWatcher("grade", "hicount", true, to, 0)
	watcher.mode = NOTIFY

	go watcher.Watch()

	_, opennotich := <-watcher.NotiCh
	assert.Equal(t, false, opennotich)

	err, openerrch := <-watcher.ErrCh
	assert.Equal(t, true, openerrch)
	assert.NotEqual(t, nil, err)
	assert.Equal(t, "watching directory or file does not exist", err.Error())
	log.Println(err)

	_, openerrch = <-watcher.ErrCh
	assert.Equal(t, false, openerrch)
}
