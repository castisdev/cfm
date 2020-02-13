package fmfm

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cfm/tasker"
)

// map: file name -> *common.FileMeta
type FileMetaPtrMap map[string]*common.FileMeta

type Runner struct {
	fmm                 FileMetaPtrMap
	dupFmm              FileMetaPtrMap
	rhm                 map[string]int
	betweenEventsRunSec uint32 // run between a event and the next event
	periodicRunSec      uint32 // periodic run
	remover             *remover.Remover
	tasker              *tasker.Tasker
	tailer              *tailer.Tailer
	CMDCh               chan CMD // command input
	ErrCh               chan error
	RUNFuncs            map[RUN]func(*Runner, FileMetaFilesEvent)
	SetupRuns           SetupRuns
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

func newRunFuns() map[RUN]func(*Runner, FileMetaFilesEvent) {
	runFuncs := make(map[RUN]func(*Runner, FileMetaFilesEvent))
	runFuncs[NOP] = func(r *Runner, fme FileMetaFilesEvent) {}
	runFuncs[MakeFMM] = func(r *Runner, fme FileMetaFilesEvent) { r.makeFmm(fme) }
	runFuncs[MakeRisingHit] = func(r *Runner, fme FileMetaFilesEvent) { r.makeRhm() }
	runFuncs[PrintFMM] = func(r *Runner, fme FileMetaFilesEvent) { r.printFmm() }
	runFuncs[RunRemover] = func(r *Runner, fme FileMetaFilesEvent) { r.runRemover() }
	runFuncs[RunTasker] = func(r *Runner, fme FileMetaFilesEvent) { r.runTasker() }
	return runFuncs
}

type SetupRuns map[RUNS][]RUN

type RUNS int

const (
	_         RUNS = iota
	EventRuns      = iota
	EventTimeoutRuns
	BetweenEventsRuns
	PeriodicRuns
)

func (r RUNS) String() string {
	m := map[RUNS]string{
		EventRuns:         "EVENTRUNS",
		EventTimeoutRuns:  "EVENTTIMEOUTRUNS",
		BetweenEventsRuns: "BETWEENEVENTSRUNS",
		PeriodicRuns:      "PERIODICRUNS",
	}
	return m[r]
}

func ToRuns(runs string) RUNS {
	m := map[string]RUNS{
		"EVENTRUNS":         EventRuns,
		"EVENTTIMEOUTRUNS":  EventTimeoutRuns,
		"BETWEENEVENTSRUNS": BetweenEventsRuns,
		"PERIODICRUNS":      PeriodicRuns,
	}
	return m[strings.ToUpper(runs)]
}

type RUN int

const (
	_   RUN = iota
	NOP     = iota
	PrintFMM
	MakeFMM
	MakeRisingHit
	RunRemover
	RunTasker
)

func (r RUN) String() string {
	m := map[RUN]string{
		NOP:           "NOP",
		PrintFMM:      "PRINTFMM",
		MakeFMM:       "MAKEFMM",
		MakeRisingHit: "MAKERISINGHIT",
		RunRemover:    "RUNREMOVER",
		RunTasker:     "RUNTASKER",
	}
	return m[r]
}

func ToRun(run string) RUN {
	m := map[string]RUN{
		"NOP":           NOP,
		"PRINTFMM":      PrintFMM,
		"MAKEFMM":       MakeFMM,
		"MAKERISINGHIT": MakeRisingHit,
		"RUNREMOVER":    RunRemover,
		"RUNTASKER":     RunTasker,
	}
	return m[strings.ToUpper(run)]
}

func defaultSetupRuns() SetupRuns {
	return SetupRuns{
		EventRuns:         DefaultEventRuns,
		EventTimeoutRuns:  DefaultEventTimeoutRuns,
		BetweenEventsRuns: DefaultBetweenEventsRuns,
		PeriodicRuns:      DefaultPeriodicRuns,
	}
}

var (
	DefaultEventRuns         = []RUN{MakeFMM, MakeRisingHit, RunRemover, RunTasker}
	DefaultEventTimeoutRuns  = DefaultEventRuns
	DefaultBetweenEventsRuns = []RUN{MakeRisingHit, RunRemover, RunTasker}
	DefaultPeriodicRuns      = []RUN{}
)

func ToSetupRuns(setup map[string][]string) SetupRuns {
	newsetup := defaultSetupRuns()
	for nameofruns, runs := range setup {
		if len(runs) > 0 {
			newruns := make([]RUN, 0)
			for _, run := range runs {
				newruns = append(newruns, ToRun(run))
			}
			newsetup[ToRuns(nameofruns)] = newruns
		}
	}
	return newsetup
}

func NewRunner(
	betweenEventsRunSec, periodicRunSec uint32,
	rmr *remover.Remover,
	tskr *tasker.Tasker,
	tlr *tailer.Tailer,
) *Runner {
	return &Runner{
		fmm:                 make(FileMetaPtrMap),
		dupFmm:              make(FileMetaPtrMap),
		rhm:                 make(map[string]int),
		betweenEventsRunSec: betweenEventsRunSec,
		periodicRunSec:      periodicRunSec,
		remover:             rmr,
		tasker:              tskr,
		tailer:              tlr,
		CMDCh:               make(chan CMD, 1),
		ErrCh:               make(chan error, 1),
		RUNFuncs:            newRunFuns(),
		SetupRuns:           defaultSetupRuns(),
	}
}

func (fr *Runner) clone() *Runner {
	nr := NewRunner(
		fr.betweenEventsRunSec,
		fr.periodicRunSec,
		fr.remover,
		fr.tasker,
		fr.tailer,
	)
	nr.RUNFuncs = fr.RUNFuncs
	nr.SetupRuns = fr.SetupRuns
	return nr
}

// https://dave.cheney.net/2013/04/30/curious-channels
// https://stackoverflow.com/questions/35036653/why-doesnt-this-golang-code-to-select-among-multiple-time-after-channels-work
func (fr *Runner) Run(eventCh <-chan FileMetaFilesEvent) error {
	runnerlogger.Infof("started runner process")
	defer close(fr.CMDCh)
	defer close(fr.ErrCh)
	periodictm := fr.newPeriodicRunTimer()
	btwperiodictm := fr.newBetweenEventsRunTimer()
	for {
		select {
		case fme, open := <-eventCh:
			if !open {
				eventCh = nil
				continue
			}
			if fme.Err != nil {
				if fme.Err == ErrTimeout {
					fr.eventTimeoutRun(fme)
					btwperiodictm = fr.newBetweenEventsRunTimer()
				} else {
					fr.errorRun(fme)
				}
				continue
			}
			fr.eventRun(fme)
			btwperiodictm = fr.newBetweenEventsRunTimer()
		case <-btwperiodictm:
			fr.betweenEventsRun(FileMetaFilesEvent{})
			btwperiodictm = fr.newBetweenEventsRunTimer()
		case <-periodictm:
			fr.periodicRun(FileMetaFilesEvent{})
			periodictm = fr.newPeriodicRunTimer()
		case cmd := <-fr.CMDCh:
			runnerlogger.Debugf("[%s] received command", cmd)
			switch cmd {
			case STOP:
				fr.ErrCh <- ErrStopped
				return ErrStopped
			}
		}
	}
}

func (fr *Runner) newPeriodicRunTimer() <-chan time.Time {
	if fr.periodicRunSec != 0 {
		return time.After(time.Duration(fr.periodicRunSec) * time.Second)
	}
	return nil
}

func (fr *Runner) newBetweenEventsRunTimer() <-chan time.Time {
	if fr.betweenEventsRunSec != 0 {
		return time.After(time.Duration(fr.betweenEventsRunSec) * time.Second)
	}
	return nil
}

func (fr *Runner) eventTimeoutRun(fme FileMetaFilesEvent) {
	runnerlogger.Infof("started timeout run")
	defer runnerlogElapased("ended timeout run", common.Start())

	for _, r := range fr.SetupRuns[EventTimeoutRuns] {
		fr.RUNFuncs[r](fr, fme)
	}
}

func (fr *Runner) errorRun(fme FileMetaFilesEvent) {
	runnerlogger.Infof("started error run, error(%s)", fme.Err.Error())
	defer runnerlogElapased("ended error run", common.Start())
}

func (fr *Runner) eventRun(fme FileMetaFilesEvent) {
	runnerlogger.Infof("started event run")
	defer runnerlogElapased("ended event run", common.Start())

	for _, r := range fr.SetupRuns[EventRuns] {
		fr.RUNFuncs[r](fr, fme)
	}
}

func (fr *Runner) betweenEventsRun(fme FileMetaFilesEvent) {
	runnerlogger.Infof("started between events run")
	defer runnerlogElapased("ended between events run", common.Start())

	for _, r := range fr.SetupRuns[BetweenEventsRuns] {
		fr.RUNFuncs[r](fr, fme)
	}
}

func (fr *Runner) periodicRun(fme FileMetaFilesEvent) {
	runnerlogger.Infof("started periodic run")
	defer runnerlogElapased("ended periodic run", common.Start())

	for _, r := range fr.SetupRuns[PeriodicRuns] {
		fr.RUNFuncs[r](fr, fme)
	}
}

func (fr *Runner) makeFmm(fme FileMetaFilesEvent) {
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
		return
	}
	runnerlogger.Infof("made file metas(name, grade, size, servers), time(%s)",
		common.Elapsed(est))
	fr.fmm = fmm
	fr.dupFmm = dupfmm
}

func (fr *Runner) makeRhm() {
	rhm := make(map[string]int)
	var basetm time.Time = time.Now()
	fr.tailer.Tail(basetm, &rhm)
	fr.rhm = rhm
}

func (fr *Runner) runRemover() {
	fr.remover.RunWithInfo(
		remover.FileMetaPtrMap(fr.fmm), remover.FileMetaPtrMap(fr.dupFmm), fr.rhm)
}

func (fr *Runner) runTasker() {
	fr.tasker.RunWithInfo(
		tasker.FileMetaPtrMap(fr.fmm), fr.rhm)
}

func (fr *Runner) printFmm() {
	log.Printf("file metas ------------\n")
	for _, fm := range fr.fmm {
		log.Printf("\tfm:%s\n", fm)
	}
	log.Printf("----------------\n")
}

func runnerlogElapased(message string, start time.Time) {
	runnerlogger.Infof("%s, time(%s)", message, common.Elapsed(start))
}
