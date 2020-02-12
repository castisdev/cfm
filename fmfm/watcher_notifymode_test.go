package fmfm

//
// inotfy 모듈이 없으면 모든 notify 모드 test 는 의미없음
//
import (
	"testing"

	"github.com/castisdev/cfm/common"
)

// notify 모드에서는 두 피일 중 하나라도 없을 때 return 됨
// return 되면서, notiCh은 닫히고
// errCh 에는 error를 넘기고 error를 받으면 errCh은 닫힘
func TestWatchNotifyErrorAnyFileNotExist(t *testing.T) {
	createfile("testwatcher", "grade")
	defer deletefile("testwatcher", "")
	to := uint32(1)
	TestInotifyFunc = func() bool { return true }
	watcher := NewWatcher("testwatcher/grade", "hitcount", true, to, 0)
	watcher.mode = NOTIFY

	go watcher.Watch()

	waitWatcherCloseErrNotExist(t, watcher)
	waitWatcherClose(watcher)
}

// 둘 중 하나만 없어도 그냥 끝남
func TestWatchNotifyErrorTwoFileNotExist(t *testing.T) {
	to := uint32(1)
	TestInotifyFunc = func() bool { return true }
	watcher := NewWatcher("grade", "hicount", true, to, 0)
	watcher.mode = NOTIFY

	go watcher.Watch()

	waitWatcherCloseErrNotExist(t, watcher)
	waitWatcherClose(watcher)
}

// 두 파일 중 하나라도 변경이 있을 때, notiCh에 event를 발생
func TestWatchNotifyTOEventAndFileEventWithFileChanged(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	to := uint32(2)
	notifydu := uint32(1) // notify 주기는 없지만, 1초 이내에 noti할 거라고 가정하고 정함
	TestInotifyFunc = func() bool { return true }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, to, 0)
	watcher.mode = NOTIFY

	go watcher.Watch()
	notistart := common.Start()

	// 초기 event
	// 초기 event 후에 다시 noti 시작
	notistart = waitWatcherEvent(t, watcher, notistart, notifydu)
	// 초기 event 후에 timeout 주기 시작
	tostart := common.Start()

	// 변경이 없을 때 timeout event 발생
	// timeout 후에 timeout 주기 시작
	tostart = waitWatcherTimeout(t, watcher, tostart, to)

	// 변경이 없을 때 timeout event 발생
	// timeout 후에 timeout 주기 시작
	tostart = waitWatcherTimeout(t, watcher, tostart, to)

	// 시간 변경으로 event 발생 시킴
	chtimesfile("testwatcher", "grade")

	// modify time 이 변경된 것을 발견함
	notistart = waitWatcherEvent(t, watcher, notistart, to+to+notifydu)
	// 초기 event 후에 timeout 주기 시작
	tostart = common.Start()

	// write 로 event 발생 시킴
	writefile("testwatcher", "hitcount")

	// modify time 이 변경된 것을 발견함
	notistart = waitWatcherEvent(t, watcher, notistart, notifydu)
	// 초기 event 후에 timeout 주기 시작
	tostart = common.Start()

	waitWatcherClose(watcher)
}

// 두 파일 중 하나라도 삭제되거나, 이름 변경 시 error넘기면서 끝남
func TestWatchNotifyTOEventAndErrorWithFileRenamed(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	to := uint32(2)
	notifydu := uint32(1) // notify 주기는 없지만, 1초 이내에 noti할 거라고 가정하고 정함
	TestInotifyFunc = func() bool { return true }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, to, 0)
	watcher.mode = NOTIFY

	go watcher.Watch()
	notistart := common.Start()

	// 초기 event
	// 초기 event 후에 다시 noti 시작
	notistart = waitWatcherEvent(t, watcher, notistart, notifydu)
	// 초기 event 후에 timeout 주기 시작
	tostart := common.Start()

	// 변경이 없을 때 timeout event 발생
	// timeout 후에 timeout 주기 시작
	tostart = waitWatcherTimeout(t, watcher, tostart, to)

	// 이름 변경으로 event 발생 시킴
	renamefile("testwatcher", "grade", "newgrade")

	// modify time 이 변경된 것을 발견함
	// rename 된 경우, errCh에 error 내보내고 끝남
	waitWatcherCloseErrNotExist(t, watcher)
	waitWatcherClose(watcher)
}

// 두 파일 중 하나라도 삭제되거나, 이름 변경 시 error넘기면서 끝남
func TestWatchNotifyTOEventAndErrorWithFileDeleted(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	to := uint32(1)
	notifydu := uint32(1) // notify 주기는 없지만, 1초 이내에 noti할 거라고 가정하고 정함
	TestInotifyFunc = func() bool { return true }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, to, 0)
	watcher.mode = NOTIFY

	go watcher.Watch()
	notistart := common.Start()

	// 초기 event
	// 초기 event 후에 다시 noti 시작
	notistart = waitWatcherEvent(t, watcher, notistart, notifydu)
	// 초기 event 후에 timeout 주기 시작
	tostart := common.Start()

	// 변경이 없을 때 timeout event 발생
	// timeout 후에 timeout 주기 시작
	tostart = waitWatcherTimeout(t, watcher, tostart, to)

	// 파일 삭제로 event 발생 시킴
	deletefile("testwatcher", "hitcount")

	// 파일 삭제된 경우, errCh에 error 내보내고 끝남
	waitWatcherCloseErrNotExist(t, watcher)
	waitWatcherClose(watcher)
}

// directory 이름 변경 시 error넘기면서 끝남
func TestWatchNotifyTOEventAndErrorWithDirectoryRenamed(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")
	defer deletefile("newtestwatcher", "")

	to := uint32(1)
	notifydu := uint32(1) // notify 주기는 없지만, 1초 이내에 noti할 거라고 가정하고 정함
	TestInotifyFunc = func() bool { return true }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, to, 0)
	watcher.mode = NOTIFY

	go watcher.Watch()
	notistart := common.Start()

	// 초기 event
	// 초기 event 후에 다시 noti 시작
	notistart = waitWatcherEvent(t, watcher, notistart, notifydu)

	// directory이름 변경으로 event 발생 시킴
	renamedir("testwatcher", "newtestwatcher")

	// modify time 이 변경된 것을 발견함
	// rename 된 경우, errCh에 error 내보내고 끝남
	waitWatcherCloseErrNotExist(t, watcher)
	waitWatcherClose(watcher)
}

// 상위 directory 이름 변경 시 error넘기면서 끝남
func TestWatchNotifyTOEventAndErrorWithParentDirectoryRenamed(t *testing.T) {
	createfile("testwatcher", "")
	createfile("testwatcher/d1", "grade")
	createfile("testwatcher/d1", "hitcount")
	defer deletefile("testwatcher", "")
	defer deletefile("testwatcher2", "")

	to := uint32(1)
	notifydu := uint32(1) // notify 주기는 없지만, 1초 이내에 noti할 거라고 가정하고 정함
	TestInotifyFunc = func() bool { return true }
	watcher := NewWatcher("testwatcher/d1/grade", "testwatcher/d1/hitcount", true, to, 0)
	watcher.mode = NOTIFY

	go watcher.Watch()
	notistart := common.Start()

	// 초기 event
	// 초기 event 후에 다시 noti 시작
	notistart = waitWatcherEvent(t, watcher, notistart, notifydu)

	// write로 event 발생 시킴
	writefile("testwatcher/d1", "grade")
	notistart = waitWatcherEvent(t, watcher, notistart, to+notifydu)
	// 초기 event 후에 timeout 주기 시작

	// directory이름 변경으로 event 발생 시킴
	renamedir("testwatcher", "testwatcher2")

	// modify time 이 변경된 것을 발견함
	// rename 된 경우, errCh에 error 내보내고 끝남
	waitWatcherCloseErrNotExist(t, watcher)
	waitWatcherClose(watcher)
}

// directory, 파일 삭제 시 error넘기면서 끝남
func TestWatchNotifyTOEventAndErrorWithDirectoryDeleted(t *testing.T) {
	createfile("testwatcher", "grade")
	createfile("testwatcher", "hitcount")
	defer deletefile("testwatcher", "")

	to := uint32(1)
	notifydu := uint32(1) // notify 주기는 없지만, 1초 이내에 noti할 거라고 가정하고 정함
	TestInotifyFunc = func() bool { return true }
	watcher := NewWatcher("testwatcher/grade", "testwatcher/hitcount", true, to, 0)
	watcher.mode = NOTIFY

	go watcher.Watch()
	notistart := common.Start()

	// 초기 event
	// 초기 event 후에 다시 noti 시작
	notistart = waitWatcherEvent(t, watcher, notistart, notifydu)
	// 초기 event 후에 timeout 주기 시작
	tostart := common.Start()

	// 변경이 없을 때 timeout event 발생
	// timeout 후에 timeout 주기 시작
	tostart = waitWatcherTimeout(t, watcher, tostart, to)

	// directory 삭제로 event 발생 시킴
	deletefile("testwatcher", "")

	// directory delete 된 경우, errCh에 error 내보내고 끝남
	waitWatcherCloseErrNotExist(t, watcher)
	waitWatcherClose(watcher)
}

// directory unmount시 error넘기면서 끝남
// TODO:
func TestWatchNotifyErrorWithDirectoryUnmounted(t *testing.T) {
}
