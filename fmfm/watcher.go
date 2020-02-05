package fmfm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileMonitor struct {
	fsnotify.Event
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
		fsnotify.Event{Name: path, Op: 0},
		path, dir, file, newDirs(path),
		false, false, time.Now(),
	}
}

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

func (f *FileMonitor) Update(op fsnotify.Op) {
	f.Event.Op = op
	f.Mtime = time.Now()
	if op&fsnotify.Create == fsnotify.Create {
		f.Exist = true
		f.Updated = false
		return
	} else if op&fsnotify.Write == fsnotify.Write ||
		op&fsnotify.Chmod == fsnotify.Chmod {
		f.Exist = true
		f.Updated = true
		return
	} else if op&fsnotify.Remove == fsnotify.Remove ||
		op&fsnotify.Rename == fsnotify.Rename {
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

func (f *FileMonitor) UpdateMTime() bool {
	if fi, err := os.Stat(f.FilePath); !os.IsNotExist(err) {
		f.Exist = true
		if f.Mtime != fi.ModTime() {
			f.Event.Op = fsnotify.Chmod
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
	f.Event.Op = fsnotifyUnknown
	f.Updated = false
}

type FMFWatcher struct {
	*fsnotify.Watcher
	mode        WatchMode
	grade       *FileMonitor
	hitCount    *FileMonitor
	initialNoti bool
	timeoutSec  uint32
	pollingSec  uint32
	NotiCh      chan FileMetaFilesEvent
	ErrCh       chan error
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
	ErrTimeout               = errors.New("timeout")
	ErrFsNotifyChannelClosed = errors.New("fsnotify channel closed")
	ErrNotExist              = errors.New("watching directory or file does not exist")
	ErrDirUnmounted          = errors.New("watching directory might be unmounted")
)

const fsnotifyUnknown fsnotify.Op = 0

func (e FileMetaFilesEvent) String() string {
	return fmt.Sprintf("grade(%s), hitCount(%s), Err(%s)", &e.Grade, &e.HitCount, e.Err.Error())
}

func NewFMFWatcher(gradeFilePath, hitcountFilePath string,
	initialNoti bool, eventTimeoutSec, pollingSec uint32) *FMFWatcher {
	var m WatchMode
	m = NOTIFY
	if !TestFsNotify() {
		watcherlogger.Errorf("failed to run fsnotify module")
		m = POLL
		watcherlogger.Infof("changed watchmode:%s", m)
	}
	var watcher *fsnotify.Watcher
	var err error
	if m == NOTIFY {
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			watcherlogger.Errorf("failed to run fsnotify module, error(%s)", err.Error())
			m = POLL
			watcherlogger.Infof("changed watchmode:%s", m)
		}
	}
	return &FMFWatcher{
		Watcher:     watcher,
		mode:        m,
		grade:       NewFileMonitor(gradeFilePath),
		hitCount:    NewFileMonitor(hitcountFilePath),
		initialNoti: initialNoti,
		timeoutSec:  eventTimeoutSec,
		pollingSec:  pollingSec,
		NotiCh:      make(chan FileMetaFilesEvent),
		ErrCh:       make(chan error, 1),
	}
}

func (fw *FMFWatcher) Watch() (err error) {
	watcherlogger.Infof("started watcher process, mode:%s", fw.mode)
	defer close(fw.NotiCh)
	defer close(fw.ErrCh)
	switch fw.mode {
	case NOTIFY:
		err = fw.WatchNotify()
	case POLL:
		err = fw.WatchPoll()
	}
	return err
}

func (fw *FMFWatcher) WatchPoll() error {
	pollingtm := fw.newPollingTimer()
	timeouttm := fw.newTimeoutTimer()
	fw.grade.UpdateExist()
	fw.hitCount.UpdateExist()
	if fw.initialNoti {
		if fw.grade.Exist && fw.hitCount.Exist {
			fw.grade.Update(fsnotify.Write)
			fw.hitCount.Update(fsnotify.Write)
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
			fw.grade.UpdateMTime()
			fw.hitCount.UpdateMTime()
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
		}
	}
}

func (fw *FMFWatcher) WatchNotify() error {
	if err := fw.AddWatchingDirsAndFiles(); err != nil {
		fw.ErrCh <- err
		return err
	}
	timeouttm := fw.newTimeoutTimer()
	if fw.initialNoti {
		if fw.grade.Exist && fw.hitCount.Exist {
			fw.grade.Update(fsnotify.Write)
			fw.hitCount.Update(fsnotify.Write)
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
				fw.ErrCh <- ErrFsNotifyChannelClosed
				return ErrFsNotifyChannelClosed
			}
			watcherlogger.Debugf("[EVENT] %s", event)
			// https://ddcode.net/2019/05/11/some-questions-about-file-monitoring-fsnotify-in-golang/
			if fw.isWatchingDirOrFile(event.Name) &&
				(event.Op&fsnotify.Remove == fsnotify.Remove ||
					event.Op&fsnotify.Rename == fsnotify.Rename) {
				fw.grade.UpdateExist()
				fw.hitCount.UpdateExist()
				//log.Printf("[Error:%s]  \n", ErrNotExist)
				fw.ErrCh <- ErrNotExist
				return ErrNotExist
				// unmount하면 fsnotify 모듈에서 알 수 없는 event가 발생함
			} else if fw.isWatchingDir(event.Name) && event.Op == fsnotifyUnknown {
				//log.Printf("[unknown event: unmount?:%s]  \n", ErrDirUnmounted)
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
				fw.ErrCh <- ErrFsNotifyChannelClosed
				return ErrFsNotifyChannelClosed
			}
			fme := FileMetaFilesEvent{Grade: *fw.grade, HitCount: *fw.hitCount, Err: err}
			fw.NotiCh <- fme
		case <-timeouttm:
			fme := FileMetaFilesEvent{Grade: *fw.grade, HitCount: *fw.hitCount, Err: ErrTimeout}
			fw.NotiCh <- fme
			timeouttm = fw.newTimeoutTimer()
		}
	}
}

func (fw *FMFWatcher) AddWatchingDirsAndFiles() (err error) {
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

func (fw *FMFWatcher) isWatchingDirOrFile(name string) bool {
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

func (fw *FMFWatcher) isWatchingDir(name string) bool {
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

func (fw *FMFWatcher) newTimeoutTimer() <-chan time.Time {
	if fw.timeoutSec != 0 {
		return time.After(time.Duration(fw.timeoutSec) * time.Second)
	}
	return nil
}

func (fw *FMFWatcher) newPollingTimer() <-chan time.Time {
	if fw.pollingSec != 0 {
		return time.After(time.Duration(fw.pollingSec) * time.Second)
	}
	return nil
}
