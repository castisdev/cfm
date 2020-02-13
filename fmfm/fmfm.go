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
	CMDCh   chan CMD
	ErrCh   chan error
}

func NewManager(watcher *Watcher, runner *Runner) *Manager {
	return &Manager{
		watcher: watcher,
		runner:  runner,
		CMDCh:   make(chan CMD, 1),
		ErrCh:   make(chan error, 1),
	}
}

func (fm *Manager) Tasks() *tasker.Tasks {
	return fm.runner.tasker.Tasks()
}

func (fm *Manager) Manage() {
	defer close(fm.CMDCh)
	defer close(fm.ErrCh)
	for {
		go fm.watcher.Watch()
		go fm.runner.Run(fm.watcher.NotiCh)
		rc := fm.waitWatcher()
		mgrlogger.Infof("received an error, error(%s)", rc.Error())
		switch rc {
		// 파일이나 directory가 지워진 경우, 파일이 생길 때까지 기다림
		case ErrNotExist:
			err := fm.restart()
			if err != nil && err == ErrStopped {
				fm.stopWatcherAndRunner()
				fm.ErrCh <- ErrStopped
				return
			}
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
		case ErrStopped:
			fm.stopWatcherAndRunner()
			fm.ErrCh <- ErrStopped
			return
		// notify 모듈의 다른 종류의 error, poll 모드로 전환해서 재시작
		default:
			fm.restartPollMode()
		}
	}
}

func (fm *Manager) waitWatcher() error {
	for {
		select {
		case err := <-fm.watcher.ErrCh:
			return err
		case cmd := <-fm.CMDCh:
			mgrlogger.Debugf("[%s] received command", cmd)
			switch cmd {
			case STOP:
				fm.ErrCh <- ErrStopped
				return ErrStopped
			}
		}
	}
}

func (fm *Manager) restart() error {
	err := fm.waitUntilFileExist()
	if err != nil {
		return err
	}
	fm.stopWatcher()
	fm.stopRunner()
	newwatcher := fm.cloneWatcher()
	newrunner := fm.cloneRunner()
	fm.watcher = newwatcher
	fm.runner = newrunner
	return nil
}

func (fm *Manager) restartPollMode() {
	fm.stopWatcher()
	fm.stopRunner()
	newwatcher := fm.cloneWatcher()
	newwatcher.mode = POLL
	newrunner := fm.cloneRunner()
	fm.watcher = newwatcher
	fm.runner = newrunner
}

func (fm *Manager) stopWatcherAndRunner() {
	fm.stopWatcher()
	fm.stopRunner()
}

func (fm *Manager) stopWatcher() {
	if !fm.watcher.isCloseDoneCh() {
		close(fm.watcher.doneCh)
	}
	<-fm.watcher.ErrCh
	fm.watcher.Close()
}

func (fm *Manager) stopRunner() {
	fm.runner.CMDCh <- STOP
	<-fm.runner.ErrCh
}

func (fm *Manager) waitUntilFileExist() error {
	waiting := make(chan error)
	go func() {
		defer close(waiting)
		var poll uint32 = 1
		if fm.watcher.pollingSec > 0 {
			poll = fm.watcher.pollingSec
		}
		for {
			select {
			case <-time.After(time.Duration(poll) * time.Second):
				if fm.watcher.grade.UpdateExist() && fm.watcher.hitCount.UpdateExist() {
					waiting <- nil
					return
				}
			case cmd := <-fm.CMDCh:
				mgrlogger.Debugf("[%s] received command", cmd)
				switch cmd {
				case STOP:
					waiting <- ErrStopped
					return
				}
			}
		}
	}()
	err := <-waiting
	return err
}

func (fm *Manager) cloneWatcher() *Watcher {
	return fm.watcher.clone()
}

func (fm *Manager) cloneRunner() *Runner {
	return fm.runner.clone()
}
