package fmfm

import (
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/tasker"
	"github.com/castisdev/cilog"
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
		mgrlogger.Errorf("recived an error, error(%s)", rc.Error())
		switch rc {
		// 파일이나 directory가 지워진 경우, 파일이 생길 때까지 기다림
		case ErrNotExist:
			fm.restart()
		// directory가 지워지진 않았지만, unmount 되어서,
		// 더 이상 감시가 이루어지지 않음, 파일이 생길 때까지 기다림
		case ErrDirUnmounted:
			fm.restart()
		// notify 모듈 channel 이 닫힌 경우, poll 모드로 전환해서 재시작
		case ErrEventClosed:
			fm.restartPollMode()
		// notify 모듈 overflow error, poll 모드로 전환해서 재시작
		case ErrEventOverflow:
			fm.restartPollMode()
		// notify 모듈의 다른 종류의 error, poll 모드로 전환해서 재시작
		default:
			fm.restartPollMode()
		}
	}
}

func (fm *FMFManager) waitWatcher() error {
	err := <-fm.watcher.ErrCh
	return err
}

func (fm *FMFManager) restart() {
	fm.waitUntilFileExist()
	fm.closeWatcherAndStopRunner()
	newwatcher := fm.cloneFMFWatcher()
	newrunner := fm.cloneFMFRunner()
	fm.watcher = newwatcher
	fm.runner = newrunner
}

func (fm *FMFManager) restartPollMode() {
	fm.waitUntilFileExist()
	fm.closeWatcherAndStopRunner()
	newwatcher := fm.cloneFMFWatcher()
	newwatcher.mode = POLL
	newrunner := fm.cloneFMFRunner()
	fm.watcher = newwatcher
	fm.runner = newrunner
}

func (fm *FMFManager) waitUntilFileExist() {
	waiting := make(chan struct{})
	go func() {
		defer close(waiting)
		for {
			select {
			case <-time.After(time.Duration(fm.watcher.pollingSec) * time.Second):
				if fm.watcher.grade.UpdateExist() && fm.watcher.hitCount.UpdateExist() {
					return
				}
			}
		}
	}()
	<-waiting
}

func (fm *FMFManager) closeWatcherAndStopRunner() {
	fm.watcher.Close()
	fm.runner.CMDCh <- STOP
	<-fm.runner.ErrCh
}

func (fm *FMFManager) cloneFMFWatcher() *FMFWatcher {
	return NewFMFWatcher(
		fm.watcher.grade.FilePath,
		fm.watcher.hitCount.FilePath,
		fm.watcher.initialNoti,
		fm.watcher.timeoutSec,
		fm.watcher.pollingSec,
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
