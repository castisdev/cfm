package fmfm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/castisdev/cfm/myinotify"
)

type FileMonitor struct {
	myinotify.Event
	FilePath string
	Dir      string
	Name     string
	Dirs     []string
	Exist    bool
	Updated  bool
	Mtime    time.Time
}

func NewFileMonitor(path string) *FileMonitor {
	dir, file := filepath.Split(path)
	dir = filepath.Clean(dir)
	return &FileMonitor{
		myinotify.Event{Name: path, Op: 0},
		path, dir, file, newDirs(path),
		false, false, time.Now(),
	}
}

// 에러처리 안됨
// .../dir/file.txt 같은 path를 에러처리 하지 않음
func newDirs(path string) []string {
	dirs := make([]string, 0)
	dir := path
	for dir != "." && dir != "/" && dir != ".." {
		dir, _ = filepath.Split(dir)
		dir = filepath.Clean(dir)
		dirs = append(dirs, dir)
	}

	return dirs
}

func (f *FileMonitor) String() string {
	return fmt.Sprintf("filePath:%s, dir:%s, name:%s, exist:%v, updated:%v"+
		", mtime:%s, event:%s", f.FilePath, f.Dir, f.Name, f.Exist, f.Updated,
		f.Mtime, f.Event)
}

func (f *FileMonitor) Update(op myinotify.Op) {
	f.Event.Op = op
	f.Mtime = time.Now()
	if op&myinotify.Create == myinotify.Create {
		f.Exist = true
		f.Updated = false
		return
	} else if op&myinotify.Write == myinotify.Write ||
		op&myinotify.Chmod == myinotify.Chmod {
		f.Exist = true
		f.Updated = true
		return
	} else if op&myinotify.Remove == myinotify.Remove ||
		op&myinotify.Rename == myinotify.Rename {
		f.Exist = false
		f.Updated = false
		return
	}
	f.Updated = false
	return
}

func (f *FileMonitor) UpdateExist() bool {
	if _, err := os.Stat(f.FilePath); !os.IsNotExist(err) {
		f.Exist = true
	} else {
		f.Exist = false
		f.Updated = false
	}
	return f.Exist
}

func (f *FileMonitor) UpdateMtime() bool {
	if fi, err := os.Stat(f.FilePath); !os.IsNotExist(err) {
		f.Exist = true
		if f.Mtime != fi.ModTime() {
			f.Event.Op = myinotify.Chmod
			f.Mtime = fi.ModTime()
			f.Updated = true
		} else {
			f.Updated = false
		}
	} else {
		f.Exist = false
		f.Updated = false
	}
	return f.Exist
}

func (f *FileMonitor) ResetUpdate() {
	f.Event.Op = myinotifyUnknown
	f.Updated = false
}

type Watcher struct {
	*myinotify.Watcher
	mode        WatchMode
	grade       *FileMonitor
	hitCount    *FileMonitor
	initialNoti bool
	timeoutSec  uint32
	pollingSec  uint32
	NotiCh      chan FileMetaFilesEvent
	ErrCh       chan error
	doneCh      chan struct{}
}

type WatchMode int

const (
	_              = iota
	POLL WatchMode = iota
	NOTIFY
)

func (m WatchMode) String() string {
	mm := map[WatchMode]string{
		POLL:   "poll",
		NOTIFY: "notify",
	}
	return mm[m]
}

type FileMetaFilesEvent struct {
	Grade    FileMonitor
	HitCount FileMonitor
	Err      error
}

var (
	ErrTimeout       = errors.New("timeout")
	ErrEventClosed   = errors.New("event channel closed")
	ErrEventOverflow = myinotify.ErrEventOverflow
	ErrNotExist      = errors.New("watching directory or file does not exist")
	ErrDirUnmounted  = errors.New("watching directory might be unmounted")
)

const myinotifyUnknown myinotify.Op = 0

func (e FileMetaFilesEvent) String() string {
	if e.Err != nil {
		return fmt.Sprintf("err(%s), grade(%s), hitCount(%s)", e.Err.Error(), &e.Grade, &e.HitCount)
	}
	return fmt.Sprintf("grade(%s), hitCount(%s)", &e.Grade, &e.HitCount)
}

var (
	TestInotifyFunc func() bool                        = TestNotify
	NewWatcherFunc  func() (*myinotify.Watcher, error) = myinotify.NewWatcher
)

func NewWatcher(gradeFilePath, hitcountFilePath string,
	initialNoti bool, eventTimeoutSec, pollingSec uint32) *Watcher {
	var m WatchMode
	m = NOTIFY
	if !TestInotifyFunc() {
		watcherlogger.Infof("failed to run inotify module")
		m = POLL
		watcherlogger.Infof("changed mode:%s", m)
	}
	var watcher *myinotify.Watcher
	var err error
	if m == NOTIFY {
		watcher, err = NewWatcherFunc()
		if err != nil {
			watcherlogger.Infof("failed to run inotify module, error(%s)", err.Error())
			m = POLL
			watcherlogger.Infof("changed mode:%s", m)
		}
	}
	return &Watcher{
		Watcher:     watcher,
		mode:        m,
		grade:       NewFileMonitor(gradeFilePath),
		hitCount:    NewFileMonitor(hitcountFilePath),
		initialNoti: initialNoti,
		timeoutSec:  eventTimeoutSec,
		pollingSec:  pollingSec,
		NotiCh:      make(chan FileMetaFilesEvent),
		ErrCh:       make(chan error, 1),
		doneCh:      make(chan struct{}),
	}
}

func (fw *Watcher) clone() *Watcher {
	return NewWatcher(
		fw.grade.FilePath,
		fw.hitCount.FilePath,
		fw.initialNoti,
		fw.timeoutSec,
		fw.pollingSec,
	)
}

func (fw *Watcher) Close() {
	// poll mode 에서는 Watcher가 nil 임
	if fw.Watcher != nil {
		fw.Watcher.Close()
	}
}

func (fw *Watcher) isCloseDoneCh() bool {
	select {
	case <-fw.doneCh:
		return true
	default:
		return false
	}
}

func (fw *Watcher) Watch() (err error) {
	watcherlogger.Infof("started watcher process, mode:%s", fw.mode)
	defer close(fw.NotiCh)
	defer func() {
		if !fw.isCloseDoneCh() {
			close(fw.doneCh)
		}
	}()
	defer close(fw.ErrCh)
	switch fw.mode {
	case NOTIFY:
		err = fw.WatchNotify()
	case POLL:
		err = fw.WatchPoll()
	}
	return err
}

// WatchPoll :
//
// 초기, 두 파일이 모두 있는 경우 event 발생
//
// 이후, 두 파일 중 하나라도 modify time이 변경된 경우 event 발생
//
// 특정 시간동안 event가 발생하지 않으면 timeout err event 발생
//
// CMDCh 이 닫히면 return
func (fw *Watcher) WatchPoll() error {
	pollingtm := fw.newPollingTimer()
	timeouttm := fw.newTimeoutTimer()
	fw.grade.UpdateExist()
	fw.hitCount.UpdateExist()
	if fw.grade.Exist && fw.hitCount.Exist {
		fw.grade.UpdateMtime()
		fw.hitCount.UpdateMtime()
		if fw.initialNoti {
			fme := FileMetaFilesEvent{Grade: *fw.grade, HitCount: *fw.hitCount}
			fw.NotiCh <- fme
			pollingtm = fw.newPollingTimer()
			timeouttm = fw.newTimeoutTimer()
		}
	}
	for {
		select {
		case <-pollingtm:
			watcherlogger.Debugf("[POLL] %s, %s\n", fw.grade.FilePath, fw.hitCount.FilePath)
			fw.grade.UpdateMtime()
			fw.hitCount.UpdateMtime()
			if fw.grade.Exist && fw.hitCount.Updated ||
				fw.grade.Updated && fw.hitCount.Exist ||
				fw.grade.Updated && fw.hitCount.Updated {
				fme := FileMetaFilesEvent{Grade: *fw.grade, HitCount: *fw.hitCount}
				fw.NotiCh <- fme
				timeouttm = fw.newTimeoutTimer()
			}
			pollingtm = fw.newPollingTimer()
		case <-timeouttm:
			fme := FileMetaFilesEvent{Grade: *fw.grade, HitCount: *fw.hitCount, Err: ErrTimeout}
			fw.NotiCh <- fme
			timeouttm = fw.newTimeoutTimer()
		case _, open := <-fw.doneCh:
			if !open {
				watcherlogger.Debugf("stopped, command channel closed")
				fw.ErrCh <- ErrStopped
				return ErrStopped
			}
		}
	}
}

// WatchNotify :
//
// 초기, 두 피일 중 하나라도 없을 때 errCh에 error 넘기면서 끝남
//
// 이후에는,
//
// 두 파일 중 하나라도 변경이 있을 때, notiCh에 event를 발생
//
// 특정 시간동안 event가 발생하지 않으면 timeout err event 발생
//
// 두 파일 중 하나라도 삭제되거나, rename이 될 때 errCh에 error 넘기면서 끝남
//
// 두 파일 중 하나의 상위 directory가 삭제되거나 rename, unmount 될 때  errCh에 error 넘기면서 끝남
func (fw *Watcher) WatchNotify() error {
	if err := fw.AddWatchingDirsAndFiles(); err != nil {
		fw.ErrCh <- err
		return err
	}
	timeouttm := fw.newTimeoutTimer()
	if fw.initialNoti {
		if fw.grade.Exist && fw.hitCount.Exist {
			fw.grade.Update(myinotify.Write)
			fw.hitCount.Update(myinotify.Write)
			fme := FileMetaFilesEvent{Grade: *fw.grade, HitCount: *fw.hitCount}
			fw.NotiCh <- fme
			timeouttm = fw.newTimeoutTimer()
			fw.grade.ResetUpdate()
			fw.hitCount.ResetUpdate()
		}
	}
	for {
		select {
		case event, ok := <-fw.Watcher.Events:
			if !ok {
				fw.ErrCh <- ErrEventClosed
				return ErrEventClosed
			}
			watcherlogger.Debugf("[EVENT] %s", event)
			// https://ddcode.net/2019/05/11/some-questions-about-file-monitoring-fsnotify-in-golang/
			if fw.isWatchingDirOrFile(event.Name) &&
				(event.Op&myinotify.Remove == myinotify.Remove ||
					event.Op&myinotify.Rename == myinotify.Rename) {
				fw.grade.UpdateExist()
				fw.hitCount.UpdateExist()
				fw.ErrCh <- ErrNotExist
				return ErrNotExist
			} else if fw.isWatchingDir(event.Name) && event.Op == myinotify.Unmount {
				fw.grade.UpdateExist()
				fw.hitCount.UpdateExist()
				fw.ErrCh <- ErrDirUnmounted
				return ErrDirUnmounted
			}
			if event.Name == fw.grade.FilePath {
				fw.grade.Update(event.Op)
			} else if event.Name == fw.hitCount.FilePath {
				fw.hitCount.Update(event.Op)
			}
			if fw.grade.Exist && fw.hitCount.Updated ||
				fw.grade.Updated && fw.hitCount.Exist ||
				fw.grade.Updated && fw.hitCount.Updated {
				fme := FileMetaFilesEvent{Grade: *fw.grade, HitCount: *fw.hitCount}
				fw.NotiCh <- fme
				timeouttm = fw.newTimeoutTimer()
				fw.grade.ResetUpdate()
				fw.hitCount.ResetUpdate()
			}
		case err, ok := <-fw.Watcher.Errors:
			if !ok {
				fw.ErrCh <- ErrEventClosed
				return ErrEventClosed
			}
			if err == myinotify.ErrEventOverflow {
				fw.ErrCh <- ErrEventOverflow
				return ErrEventOverflow
			}
			fw.ErrCh <- err
			return err
		case <-timeouttm:
			fme := FileMetaFilesEvent{Grade: *fw.grade, HitCount: *fw.hitCount, Err: ErrTimeout}
			fw.NotiCh <- fme
			timeouttm = fw.newTimeoutTimer()
		case _, open := <-fw.doneCh:
			if !open {
				watcherlogger.Debugf("stopped, command channel closed")
				fw.ErrCh <- ErrStopped
				return ErrStopped
			}
		}
	}
}

func (fw *Watcher) AddWatchingDirsAndFiles() (err error) {
	if !fw.grade.UpdateExist() ||
		!fw.hitCount.UpdateExist() {
		return ErrNotExist
	}
	for _, dir := range fw.grade.Dirs {
		err = fw.Watcher.Add(dir)
		if err != nil {
			if err.Error() == "no such file or directory" {
				return ErrNotExist
			}
			return err
		}
	}
	for _, dir := range fw.hitCount.Dirs {
		err = fw.Watcher.Add(dir)
		if err != nil {
			if err.Error() == "no such file or directory" {
				return ErrNotExist
			}
			return err
		}
	}
	return nil
}

func (fw *Watcher) isWatchingDirOrFile(name string) bool {
	for _, dir := range fw.grade.Dirs {
		if name == dir {
			return true
		}
	}
	for _, dir := range fw.hitCount.Dirs {
		if name == dir {
			return true
		}
	}
	if name == fw.grade.FilePath || name == fw.hitCount.FilePath {
		return true
	}
	return false
}

func (fw *Watcher) isWatchingDir(name string) bool {
	for _, dir := range fw.grade.Dirs {
		if name == dir {
			return true
		}
	}
	for _, dir := range fw.hitCount.Dirs {
		if name == dir {
			return true
		}
	}
	return false
}

func (fw *Watcher) newTimeoutTimer() <-chan time.Time {
	if fw.timeoutSec != 0 {
		return time.After(time.Duration(fw.timeoutSec) * time.Second)
	}
	return nil
}

func (fw *Watcher) newPollingTimer() <-chan time.Time {
	if fw.pollingSec != 0 {
		return time.After(time.Duration(fw.pollingSec) * time.Second)
	}
	return nil
}
