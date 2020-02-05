package fmfm

import (
	"errors"
	"log"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cfm/tasker"
)

// map: file name -> *common.FileMeta
type FileMetaPtrMap map[string]*common.FileMeta

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
	return &FMFRunner{
		grade:    *NewFileMonitor(gradeFilePath),
		hitCount: *NewFileMonitor(hitcountFilePath),
		fmm:      make(FileMetaPtrMap),
		dupFmm:   make(FileMetaPtrMap),
		rhm:      make(map[string]int),
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
func (fr *FMFRunner) Run(eventCh <-chan FileMetaFilesEvent) error {
	runnerlogger.Infof("started runner process")
	defer close(fr.CMDCh)
	defer close(fr.ErrCh)
	periodictm := fr.newPeriodicRunTimer()
	btwperiodictm := fr.newBtwEventsPeriodicRunTimer()
	for {
		select {
		case fme, open := <-eventCh:
			if !open {
				fr.channelClosedRun()
				eventCh = nil
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
