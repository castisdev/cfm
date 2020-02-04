package fmfm

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cfm/tasker"
	"github.com/castisdev/cilog"
	"github.com/fsnotify/fsnotify"
)

var mgrlogger common.MLogger
var watcherlogger common.MLogger
var runnerlogger common.MLogger

func init() {
	mgrlogger = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "manager"}
	watcherlogger = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "watcher"}
	runnerlogger = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "runner"}
}

// map: file name -> *common.FileMeta
type FileMetaPtrMap map[string]*common.FileMeta

type FileMonitor struct {
	fsnotify.Event
	FilePath string
	Dir      string
	Name     string
	Exist    bool
	Updated  bool
	Mtime    time.Time
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
}

func (f *FileMonitor) Clear() {
	f.Event.Op = 0
	f.Mtime = time.Now()
	f.Updated = false
}

func (f *FileMonitor) UpdateExist() bool {
	if _, err := os.Stat(f.FilePath); !os.IsNotExist(err) {
		f.Exist = true
	} else {
		f.Exist = false
	}
	f.Mtime = time.Now()
	return f.Exist
}

func (f *FileMonitor) IsDirExist() bool {
	if _, err := os.Stat(f.Dir); !os.IsNotExist(err) {
		return true
	}
	return false
}

type FMFManager struct {
	watcher *FMFWatcher
	runner  *FMFRunner
	ErrCh   chan error
}

func NewFMFManager(watcher *FMFWatcher, runner *FMFRunner) *FMFManager {
	return &FMFManager{
		watcher: watcher,
		runner:  runner,
		ErrCh:   make(chan error, 1),
	}
}

func (fm *FMFManager) Tasks() *tasker.Tasks {
	return fm.runner.tasker.Tasks()
}

func (fm *FMFManager) Manage() {
	for {
		go fm.watcher.Watch()
		go fm.runner.Run(fm.watcher.NotiCh)

		rc := fm.waitWatcher()
		if rc == ErrDirNotExist {
			mgrlogger.Infof("[%s] watching directory removed or renamed", ErrDirNotExist)
			fm.waitUntilDirExist()
			fm.closeWatcherAndStopRunner()
			newfw, err := fm.cloneFMFWatcher()
			if err != nil {
				fm.ErrCh <- err
				return
			}
			newrunner := fm.cloneFMFRunner()
			fm.watcher = newfw
			fm.runner = newrunner
		} else {
			fm.closeWatcherAndStopRunner()
			fm.ErrCh <- rc
			return
		}
	}
}

func (fm *FMFManager) waitWatcher() error {
	err := <-fm.watcher.ErrCh
	return err
}

func (fm *FMFManager) waitUntilDirExist() {
	waiting := make(chan bool, 1)
	go func() {
		for {
			time.Sleep(time.Second) //avoid cpu consuming
			if fm.watcher.grade.IsDirExist() && fm.watcher.hitCount.IsDirExist() {
				mgrlogger.Debugf("found watching directory")
				waiting <- false
				return
			}
		}
	}()
	<-waiting
}

func (fm *FMFManager) cloneFMFWatcher() (*FMFWatcher, error) {
	return NewFMFWatcher(
		fm.watcher.grade.FilePath,
		fm.watcher.hitCount.FilePath,
		fm.watcher.initialNoti,
		fm.watcher.timeoutSec,
	)
}

func (fm *FMFManager) cloneFMFRunner() *FMFRunner {
	return NewFMFRunner(fm.runner.grade.FilePath,
		fm.runner.hitCount.FilePath,
		fm.runner.btwEventsPeriodicRunSec,
		fm.runner.periodicRunSec,
		fm.runner.remover,
		fm.runner.tasker,
		fm.runner.tailer,
	)
}

func (fm *FMFManager) closeWatcherAndStopRunner() {
	fm.watcher.Close()
	fm.runner.CMDCh <- STOP
	<-fm.runner.ErrCh
}

type FMFWatcher struct {
	*fsnotify.Watcher
	grade       *FileMonitor
	hitCount    *FileMonitor
	initialNoti bool
	timeoutSec  uint32
	NotiCh      chan FileMetaFilesEvent
	ErrCh       chan error
}

type FileMetaFilesEvent struct {
	Grade    FileMonitor
	HitCount FileMonitor
	Err      error
}

var (
	ErrTimeout               = errors.New("timeout")
	ErrFsNotifyChannelClosed = errors.New("fsnotify channel closed")
	ErrDirNotExist           = errors.New("watching directory does not exist")
)

func (e FileMetaFilesEvent) String() string {
	return fmt.Sprintf("grade(%s), hitCount(%s)", &e.Grade, &e.HitCount)
}

func NewFMFWatcher(gradeFilePath, hitcountFilePath string,
	initialNoti bool, eventTimeoutSec uint32) (*FMFWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		watcherlogger.Errorf("failed to new fsnotifier, error(%s)", err.Error())
		return nil, err
	}
	gradeDir, gradeFilename := filepath.Split(gradeFilePath)
	gradeDir = filepath.Clean(gradeDir)
	hitcountDir, hitcountFilename := filepath.Split(hitcountFilePath)
	hitcountDir = filepath.Clean(hitcountDir)

	return &FMFWatcher{
		Watcher: watcher,
		grade: &FileMonitor{fsnotify.Event{Name: gradeFilePath, Op: 0},
			gradeFilePath, gradeDir,
			gradeFilename, false, false, time.Now()},
		hitCount: &FileMonitor{fsnotify.Event{Name: hitcountFilePath, Op: 0},
			hitcountFilePath, hitcountDir,
			hitcountFilename, false, false, time.Now()},
		initialNoti: initialNoti,
		timeoutSec:  eventTimeoutSec,
		NotiCh:      make(chan FileMetaFilesEvent),
		ErrCh:       make(chan error, 1),
	}, nil
}

func (fw *FMFWatcher) Watch() error {
	watcherlogger.Infof("started watcher process")
	defer close(fw.NotiCh)
	defer close(fw.ErrCh)
	timeouttm := fw.newTimeoutTimer()
	err := fw.Watcher.Add(fw.grade.Dir)
	if err != nil {
		if err.Error() == "no such file or directory" {
			fw.ErrCh <- ErrDirNotExist
			return ErrDirNotExist
		}
		fw.ErrCh <- err
		return err
	}
	err = fw.Watcher.Add(fw.hitCount.Dir)
	if err != nil {
		if err.Error() == "no such file or directory" {
			fw.ErrCh <- ErrDirNotExist
			return ErrDirNotExist
		}
		fw.ErrCh <- err
		return err
	}
	fw.grade.UpdateExist()
	fw.hitCount.UpdateExist()
	if fw.initialNoti {
		if fw.grade.Exist && fw.hitCount.Exist {
			fw.grade.Update(fsnotify.Write)
			fw.hitCount.Update(fsnotify.Write)
			fme := FileMetaFilesEvent{Grade: *fw.grade, HitCount: *fw.hitCount}
			fw.NotiCh <- fme
			fw.grade.Clear()
			fw.hitCount.Clear()
			timeouttm = fw.newTimeoutTimer()
		}
	}
	for {
		select {
		case event, ok := <-fw.Watcher.Events:
			if !ok {
				fw.ErrCh <- ErrFsNotifyChannelClosed
				return ErrFsNotifyChannelClosed
			}
			watcherlogger.Debugf("[%s]", event)
			if event.Name == fw.grade.Dir ||
				event.Name == fw.hitCount.Dir {
				if event.Op&fsnotify.Remove == fsnotify.Remove ||
					event.Op&fsnotify.Rename == fsnotify.Rename {
					fw.ErrCh <- ErrDirNotExist
					return ErrDirNotExist
				}
			}
			if event.Name == fw.grade.FilePath {
				fw.grade.Update(event.Op)
			}
			if event.Name == fw.hitCount.FilePath {
				fw.hitCount.Update(event.Op)
			}
			if fw.grade.Exist && fw.hitCount.Updated ||
				fw.grade.Updated && fw.hitCount.Exist ||
				fw.grade.Updated && fw.hitCount.Updated {
				fme := FileMetaFilesEvent{Grade: *fw.grade, HitCount: *fw.hitCount}
				fw.NotiCh <- fme
				fw.grade.Clear()
				fw.hitCount.Clear()
				timeouttm = fw.newTimeoutTimer()
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

func (fw *FMFWatcher) newTimeoutTimer() <-chan time.Time {
	if fw.timeoutSec != 0 {
		return time.After(time.Duration(fw.timeoutSec) * time.Second)
	}
	return nil
}

type FMFRunner struct {
	grade                   FileMonitor
	hitCount                FileMonitor
	fmm                     FileMetaPtrMap
	dupFmm                  FileMetaPtrMap
	rhm                     map[string]int
	btwEventsPeriodicRunSec uint32 // run between a event and the next event
	periodicRunSec          uint32 // periodic run
	remover                 *remover.Remover
	tasker                  *tasker.Tasker
	tailer                  *tailer.Tailer
	CMDCh                   chan CMD // command input
	ErrCh                   chan error
}

var (
	ErrStopped = errors.New("stopped")
)

type CMD int

const (
	_        = iota
	STOP CMD = iota
)

func (c CMD) String() string {
	m := map[CMD]string{
		STOP: "stop",
	}
	return m[c]
}

func NewFMFRunner(
	gradeFilePath, hitcountFilePath string,
	btwEventsPeriodicRunSec, periodicRunSec uint32,
	rmr *remover.Remover,
	tskr *tasker.Tasker,
	tlr *tailer.Tailer,
) *FMFRunner {
	gradeDir, gradeFilename := filepath.Split(gradeFilePath)
	gradeDir = filepath.Clean(gradeDir)
	hitcountDir, hitcountFilename := filepath.Split(hitcountFilePath)
	hitcountDir = filepath.Clean(hitcountDir)

	return &FMFRunner{
		grade: FileMonitor{fsnotify.Event{Name: gradeFilePath, Op: 0},
			gradeFilePath, gradeDir,
			gradeFilename, true, true, time.Now()},
		hitCount: FileMonitor{fsnotify.Event{Name: hitcountFilePath, Op: 0},
			hitcountFilePath, hitcountDir,
			hitcountFilename, true, true, time.Now()},
		fmm:    make(FileMetaPtrMap),
		dupFmm: make(FileMetaPtrMap),
		rhm:    make(map[string]int),
		btwEventsPeriodicRunSec: btwEventsPeriodicRunSec,
		periodicRunSec:          periodicRunSec,
		remover:                 rmr,
		tasker:                  tskr,
		tailer:                  tlr,
		CMDCh:                   make(chan CMD, 1),
		ErrCh:                   make(chan error, 1),
	}
}

// https://dave.cheney.net/2013/04/30/curious-channels
// https://stackoverflow.com/questions/35036653/why-doesnt-this-golang-code-to-select-among-multiple-time-after-channels-work
func (fr *FMFRunner) Run(notiCh <-chan FileMetaFilesEvent) error {
	runnerlogger.Infof("started runner process")
	defer close(fr.CMDCh)
	defer close(fr.ErrCh)
	periodictm := fr.newPeriodicRunTimer()
	btwperiodictm := fr.newBtwEventsPeriodicRunTimer()
	for {
		select {
		case fme, open := <-notiCh:
			if !open {
				fr.channelClosedRun()
				notiCh = nil
				continue
			}
			if fme.Err != nil {
				if fme.Err == ErrTimeout {
					fr.eventTimeoutRun(fme)
					btwperiodictm = fr.newBtwEventsPeriodicRunTimer()
				} else {
					fr.errorRun(fme)
				}
				continue
			}
			fr.eventRun(fme)
			btwperiodictm = fr.newBtwEventsPeriodicRunTimer()
		case <-btwperiodictm:
			fr.btwEventsPeriodicRun(FileMetaFilesEvent{Grade: fr.grade, HitCount: fr.hitCount})
			btwperiodictm = fr.newBtwEventsPeriodicRunTimer()
		case <-periodictm:
			fr.periodicRun(FileMetaFilesEvent{Grade: fr.grade, HitCount: fr.hitCount})
			periodictm = fr.newPeriodicRunTimer()
		case cmd := <-fr.CMDCh:
			runnerlogger.Infof("[%s] received command", cmd)
			switch cmd {
			case STOP:
				fr.ErrCh <- ErrStopped
				return ErrStopped
			}
		}
	}
}

func (fr *FMFRunner) newPeriodicRunTimer() <-chan time.Time {
	if fr.periodicRunSec != 0 {
		return time.After(time.Duration(fr.periodicRunSec) * time.Second)
	}
	return nil
}

func (fr *FMFRunner) newBtwEventsPeriodicRunTimer() <-chan time.Time {
	if fr.btwEventsPeriodicRunSec != 0 {
		return time.After(time.Duration(fr.btwEventsPeriodicRunSec) * time.Second)
	}
	return nil
}

func (fr *FMFRunner) channelClosedRun() {
	runnerlogger.Debugf("started channel-closed run")
	defer runnerlogElapased("ended channel-closed run", common.Start())
}

func (fr *FMFRunner) errorRun(fme FileMetaFilesEvent) {
	runnerlogger.Errorf("started error run, error(%s)", fme.Err.Error())
	defer runnerlogElapased("ended error run", common.Start())
}

func (fr *FMFRunner) eventTimeoutRun(fme FileMetaFilesEvent) {
	runnerlogger.Debugf("started timeout run")
	defer runnerlogElapased("ended timeout run", common.Start())
	fr.makeFmm(fme)
	fr.makeRhm()
	fr.removerRun()
	fr.taskerRun()
}

func (fr *FMFRunner) eventRun(fme FileMetaFilesEvent) {
	runnerlogger.Debugf("started event run, (%s)", fme)
	defer runnerlogElapased("ended event run", common.Start())
	fr.makeFmm(fme)
	fr.makeRhm()
	fr.removerRun()
	fr.taskerRun()
}

func (fr *FMFRunner) btwEventsPeriodicRun(fme FileMetaFilesEvent) {
	runnerlogger.Debugf("started periodic run between events")
	defer runnerlogElapased("ended periodic run between events", common.Start())
	fr.makeRhm()
	fr.removerRun()
	fr.taskerRun()
}

func (fr *FMFRunner) periodicRun(fme FileMetaFilesEvent) {
	runnerlogger.Debugf("started periodic run(%s)", fme)
	defer runnerlogElapased("ended periodic run", common.Start())
	fr.makeFmm(fme)
	fr.makeRhm()
	fr.removerRun()
	fr.taskerRun()
}

func (fr *FMFRunner) makeFmm(fme FileMetaFilesEvent) {
	fmm := make(FileMetaPtrMap)
	dupfmm := make(FileMetaPtrMap)
	IPm := make(map[string]int)
	for _, server := range *fr.remover.Servers {
		IPm[server.IP]++
	}
	est := common.Start()
	err := common.MakeAllFileMetas(
		fme.Grade.FilePath,
		fme.HitCount.FilePath,
		fmm, IPm, dupfmm)
	if err != nil {
		runnerlogger.Errorf("failed to make file metas, error(%s)", err.Error())
	}
	runnerlogger.Infof("made file metas(name, grade, size, servers), time(%s)",
		common.Elapsed(est))
	fr.fmm = fmm
	fr.dupFmm = dupfmm
	fr.printFmm()
}

func (fr *FMFRunner) makeRhm() {
	rhm := make(map[string]int)
	var basetm time.Time = time.Now()
	fr.tailer.Tail(basetm, &rhm)
	fr.rhm = rhm
}

func (fr *FMFRunner) removerRun() {
	fr.remover.RunWithInfo(
		remover.FileMetaPtrMap(fr.fmm), remover.FileMetaPtrMap(fr.dupFmm), fr.rhm)
}

func (fr *FMFRunner) taskerRun() {
	fr.tasker.RunWithInfo(
		tasker.FileMetaPtrMap(fr.fmm), fr.rhm)
}

func (fr *FMFRunner) printFmm() {
	log.Printf("file metas ------------\n")
	for _, fm := range fr.fmm {
		log.Printf("\tfm:%s\n", fm)
	}
	log.Printf("----------------\n")
}

func runnerlogElapased(message string, start time.Time) {
	runnerlogger.Infof("%s, time(%s)", message, common.Elapsed(start))
}
