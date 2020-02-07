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

type Manager struct {
	watcher *Watcher
	runner  *Runner
	ErrCh   chan error
}

func NewManager(watcher *Watcher, runner *Runner) *Manager {
	return &Manager{
		watcher: watcher,
		runner:  runner,
		ErrCh:   make(chan error, 1),
	}
}

func (fm *Manager) Tasks() *tasker.Tasks {
	return fm.runner.tasker.Tasks()
}

func (fm *Manager) Manage() {
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

func (fm *Manager) waitWatcher() error {
	err := <-fm.watcher.ErrCh
	return err
}

func (fm *Manager) restart() {
	fm.waitUntilFileExist()
	fm.closeWatcherAndStopRunner()
	newwatcher := fm.cloneWatcher()
	newrunner := fm.cloneRunner()
	fm.watcher = newwatcher
	fm.runner = newrunner
}

func (fm *Manager) restartPollMode() {
	fm.waitUntilFileExist()
	fm.closeWatcherAndStopRunner()
	newwatcher := fm.cloneWatcher()
	newwatcher.mode = POLL
	newrunner := fm.cloneRunner()
	fm.watcher = newwatcher
	fm.runner = newrunner
}

func (fm *Manager) waitUntilFileExist() {
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

func (fm *Manager) closeWatcherAndStopRunner() {
	fm.watcher.Close()
	fm.runner.CMDCh <- STOP
	<-fm.runner.ErrCh
}

func (fm *Manager) cloneWatcher() *Watcher {
	return NewWatcher(
		fm.watcher.grade.FilePath,
		fm.watcher.hitCount.FilePath,
		fm.watcher.initialNoti,
		fm.watcher.timeoutSec,
		fm.watcher.pollingSec,
	)
}

func (fm *Manager) cloneRunner() *Runner {
	return NewRunner(fm.runner.grade.FilePath,
		fm.runner.hitCount.FilePath,
		fm.runner.betweenEventsRunSec,
		fm.runner.periodicRunSec,
		fm.runner.remover,
		fm.runner.tasker,
		fm.runner.tailer,
	)
}
