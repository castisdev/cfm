package fmfm

import (
	"log"
	"testing"

	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cfm/tasker"
)

func waitRunnerStop(r *Runner) {
	select {
	case r.CMDCh <- STOP:
		<-r.ErrCh
		return
	}
}

func nOPTest(t *testing.T, e FileMetaFilesEvent) {
	log.Println("---nOPTest:", e)
}

func makeFMMTest(t *testing.T, e FileMetaFilesEvent) {
	log.Println("---makeFMMTest:", e)
}

func makeRisingHitTest(t *testing.T, e FileMetaFilesEvent) {
	log.Println("---makeRisingHitTest:", e)
}

func printFMMTest(t *testing.T, e FileMetaFilesEvent) {
	log.Println("---printFMMTest:", e)
}

func runRemoverTest(t *testing.T, e FileMetaFilesEvent) {
	log.Println("---runRemoverTest:", e)
}

func runTaskerTest(t *testing.T, e FileMetaFilesEvent) {
	log.Println("---runTaskerTest:", e)
}

func newTestRunFuns(t *testing.T) map[RUN]func(*Runner, FileMetaFilesEvent) {
	runFuncs := make(map[RUN]func(*Runner, FileMetaFilesEvent))
	runFuncs[NOP] = func(r *Runner, e FileMetaFilesEvent) { nOPTest(t, e) }
	runFuncs[MakeFMM] = func(r *Runner, e FileMetaFilesEvent) { makeFMMTest(t, e) }
	runFuncs[MakeRisingHit] = func(r *Runner, e FileMetaFilesEvent) { makeRisingHitTest(t, e) }
	runFuncs[PrintFMM] = func(r *Runner, e FileMetaFilesEvent) { printFMMTest(t, e) }
	runFuncs[RunRemover] = func(r *Runner, e FileMetaFilesEvent) { runRemoverTest(t, e) }
	runFuncs[RunTasker] = func(r *Runner, e FileMetaFilesEvent) { runTaskerTest(t, e) }
	return runFuncs
}

func waitRunnerEventTimeoutRun(t *testing.T, r *Runner, ch chan FileMetaFilesEvent) {
	to := FileMetaFilesEvent{Err: ErrTimeout}
	select {
	case ch <- to:
	}
}

func TestRun(t *testing.T) {
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil

	DefaultEventTimeoutRuns = []RUN{NOP}

	runner := NewRunner("", "", betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)
	eventch := make(chan FileMetaFilesEvent)
	go runner.Run(eventch)

	runner.RUNFuncs = newTestRunFuns(t)
	// runner.SetupRuns = SetupRuns{
	// 	EventRuns:         []RUN{NOP},
	// 	EventTimeoutRuns:  []RUN{MakeFMM, MakeRisingHit, RunRemover, RunTasker},
	// 	BetweenEventsRuns: []RUN{NOP},
	// 	PeriodicRuns:      []RUN{NOP},
	// }

	waitRunnerEventTimeoutRun(t, runner, eventch)
	waitRunnerStop(runner)
}
