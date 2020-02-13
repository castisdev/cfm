package fmfm

import (
	"log"
	"testing"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cfm/tasker"
	"github.com/stretchr/testify/assert"
)

func TestRunnerClone(t *testing.T) {
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil

	DefaultEventTimeoutRuns = []RUN{NOP}
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	clone := runner.clone()

	assert.Equal(t, runner.betweenEventsRunSec, clone.betweenEventsRunSec)
	assert.Equal(t, runner.periodicRunSec, clone.periodicRunSec)
	assert.Equal(t, runner.remover, clone.remover)
	assert.Equal(t, runner.tasker, clone.tasker)
	assert.Equal(t, runner.tailer, clone.tailer)
	assert.Equal(t, runner.RUNFuncs, clone.RUNFuncs)
	assert.Equal(t, runner.SetupRuns, clone.SetupRuns)
}

func TestStop(t *testing.T) {
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil

	DefaultEventTimeoutRuns = []RUN{NOP}
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	eventch := make(chan FileMetaFilesEvent)
	go runner.Run(eventch)

	select {
	case runner.CMDCh <- STOP:
		<-runner.ErrCh
		_, open := <-runner.CMDCh
		assert.Equal(t, false, open)
	}
}

func TestEventTimeoutRun(t *testing.T) {
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil

	DefaultEventTimeoutRuns = []RUN{NOP}
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	eventch := make(chan FileMetaFilesEvent)
	go runner.Run(eventch)

	waitRunnerEventTimeoutRun(t, runner, eventch)
	waitRunnerStop(runner)
}

func TestEventTimeoutRuns(t *testing.T) {
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil

	DefaultEventTimeoutRuns = []RUN{NOP, MakeFMM, MakeRisingHit, RunRemover, RunTasker}
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	eventch := make(chan FileMetaFilesEvent)
	go runner.Run(eventch)

	waitRunnerEventTimeoutRuns(t, runner, eventch)
	waitRunnerStop(runner)
}

func TestEventRun(t *testing.T) {
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil

	DefaultEventRuns = []RUN{NOP}
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	eventch := make(chan FileMetaFilesEvent)
	go runner.Run(eventch)

	waitRunnerEventRun(t, runner, eventch)
	waitRunnerStop(runner)
}

func TestEventRuns(t *testing.T) {
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil

	DefaultEventRuns = []RUN{NOP, PrintFMM, MakeFMM, MakeRisingHit, RunRemover, RunTasker}
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	eventch := make(chan FileMetaFilesEvent)
	go runner.Run(eventch)

	waitRunnerEventRuns(t, runner, eventch)
	waitRunnerStop(runner)
}

// event 가 없을 때, 일정 시간이 지나면 BetweenEventsRun이 실행됨
func TestBetweenEventsRunWithoutEvents(t *testing.T) {
	betweenEventsRunSec := uint32(1)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil

	DefaultBetweenEventsRuns = []RUN{NOP}
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	eventch := make(chan FileMetaFilesEvent)
	go runner.Run(eventch)

	waitRunnerBetweenEventsRunWithoutEvents(t, runner, eventch, 2, betweenEventsRunSec)
	waitRunnerStop(runner)
}

// event 가 일어나고 난 후, 일정 시간이 지나면 BetweenEventsRun이 실행됨
func TestBetweenEventsRunWithEvents(t *testing.T) {
	betweenEventsRunSec := uint32(1)
	periodicRunSec := uint32(0)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil

	DefaultBetweenEventsRuns = []RUN{NOP, MakeFMM, MakeRisingHit, RunRemover, RunTasker}
	DefaultEventRuns = []RUN{PrintFMM}
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	eventch := make(chan FileMetaFilesEvent)
	go runner.Run(eventch)

	waitRunnerBetweenEventsRunWithEvents(t, runner, eventch, 2, betweenEventsRunSec)
	waitRunnerStop(runner)
}

func TestPeriodicRun(t *testing.T) {
	betweenEventsRunSec := uint32(0)
	periodicRunSec := uint32(1)
	var rmr *remover.Remover = nil
	var tskr *tasker.Tasker = nil
	var tlr *tailer.Tailer = nil

	DefaultPeriodicRuns = []RUN{NOP}
	runner := NewRunner(betweenEventsRunSec, periodicRunSec, rmr, tskr, tlr)

	eventch := make(chan FileMetaFilesEvent)
	go runner.Run(eventch)

	waitRunnerPeriodicRun(t, runner, eventch, 2, periodicRunSec)
	waitRunnerStop(runner)
}

func waitRunnerStop(r *Runner) {
	select {
	case r.CMDCh <- STOP:
		<-r.ErrCh
		return
	}
}

func waitRunnerEventTimeoutRun(t *testing.T, r *Runner, ch chan FileMetaFilesEvent) {
	r.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			assert.Equal(t, ErrTimeout, e.Err)
			log.Println("eventtimeoutoutrun:", e)
		},
	}
	to := FileMetaFilesEvent{Err: ErrTimeout}
	select {
	case ch <- to:
	}
}

func waitRunnerEventTimeoutRuns(t *testing.T, r *Runner, ch chan FileMetaFilesEvent) {
	runcount := 0
	r.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, ErrTimeout, e.Err)
			assert.Equal(t, 1, runcount)
			log.Println("eventtimeoutrun:NOP, run count", runcount)
		},
		MakeFMM: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, ErrTimeout, e.Err)
			assert.Equal(t, 2, runcount)
			log.Println("eventtimeoutrun:MakeFMM, run count", runcount)
		},
		MakeRisingHit: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, ErrTimeout, e.Err)
			assert.Equal(t, 3, runcount)
			log.Println("eventtimeoutrun:MakeRisingHit, run count", runcount)
		},
		RunRemover: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, ErrTimeout, e.Err)
			assert.Equal(t, 4, runcount)
			log.Println("eventtimeoutrun:RunRemover, run count", runcount)
		},
		RunTasker: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, ErrTimeout, e.Err)
			assert.Equal(t, 5, runcount)
			log.Println("eventtimeoutrun:RunTasker, run count", runcount)
		},
	}
	to := FileMetaFilesEvent{Err: ErrTimeout}
	select {
	case ch <- to:
	}
}

func waitRunnerEventRun(t *testing.T, r *Runner, ch chan FileMetaFilesEvent) {
	r.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			assert.Equal(t, nil, e.Err)
			log.Println("eventrun:", e)
		},
	}
	e := FileMetaFilesEvent{}
	select {
	case ch <- e:
	}
}

func waitRunnerEventRuns(t *testing.T, r *Runner, ch chan FileMetaFilesEvent) {
	runcount := 0
	r.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, nil, e.Err)
			assert.Equal(t, 1, runcount)
			log.Println("eventrun:NOP, run count", runcount)
		},
		PrintFMM: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, nil, e.Err)
			assert.Equal(t, 2, runcount)
			log.Println("eventrun:PrintFMM, run count", runcount)
		},
		MakeFMM: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, nil, e.Err)
			assert.Equal(t, 3, runcount)
			log.Println("eventrun:MakeFMM, run count", runcount)
		},
		MakeRisingHit: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, nil, e.Err)
			assert.Equal(t, 4, runcount)
			log.Println("eventrun:MakeRisingHit, run count", runcount)
		},
		RunRemover: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, nil, e.Err)
			assert.Equal(t, 5, runcount)
			log.Println("eventrun:RunRemover, run count", runcount)
		},
		RunTasker: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, nil, e.Err)
			assert.Equal(t, 6, runcount)
			log.Println("eventrun:RunTasker, run count", runcount)
		},
	}
	e := FileMetaFilesEvent{}
	select {
	case ch <- e:
	}
}

func waitRunnerBetweenEventsRunWithoutEvents(t *testing.T, r *Runner,
	ch chan FileMetaFilesEvent, N, duSec uint32) {
	start := common.Start()
	btwecount := uint32(0)
	r.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			btwecount++
			assert.Equal(t, nil, e.Err)
			diff := common.Elapsed(start) - time.Second*time.Duration(duSec*btwecount+1)
			log.Println("event duration:", diff)
			assert.True(t, diff < time.Duration(1)*time.Millisecond)
			log.Println("betweeneventsrun:", "count:", btwecount)
		},
	}
	select {
	case <-time.After(time.Duration(duSec*N+1) * time.Second):
		assert.True(t, N == btwecount || btwecount == N+1)
	}
}

func waitRunnerBetweenEventsRunWithEvents(t *testing.T, r *Runner,
	ch chan FileMetaFilesEvent, N, duSec uint32) {
	start := common.Start()
	btwecount := uint32(0)
	ecount := uint32(0)
	runcount := 0
	r.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			btwecount++
			assert.Equal(t, nil, e.Err)
			diff := common.Elapsed(start) - time.Second*time.Duration(duSec*btwecount+ecount+1)
			log.Println("event duration:", diff)
			assert.True(t, diff < time.Duration(1)*time.Millisecond)

			runcount++
			assert.Equal(t, 1, runcount)
			log.Println("betweeneventsrun:NOP", "count:", btwecount)
			log.Println("betweeneventsrun:NOP, run count", runcount)
		},
		MakeFMM: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, nil, e.Err)
			assert.Equal(t, 2, runcount)
			log.Println("betweeneventsrun:MakeFMM, run count", runcount)
		},
		MakeRisingHit: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, nil, e.Err)
			assert.Equal(t, 3, runcount)
			log.Println("betweeneventsrun:MakeRisingHit, run count", runcount)
		},
		RunRemover: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, nil, e.Err)
			assert.Equal(t, 4, runcount)
			log.Println("betweeneventsrun:RunRemover, run count", runcount)
		},
		RunTasker: func(r *Runner, e FileMetaFilesEvent) {
			runcount++
			assert.Equal(t, nil, e.Err)
			assert.Equal(t, 5, runcount)
			log.Println("betweeneventsrun:RunTasker, run count", runcount)
			runcount = 0
		},
		PrintFMM: func(r *Runner, e FileMetaFilesEvent) {
			ecount++
			assert.Equal(t, nil, e.Err)
			diff := common.Elapsed(start) - time.Second*time.Duration(ecount+duSec*btwecount)
			log.Println("event duration:", diff)
			assert.True(t, diff < time.Duration(1)*time.Millisecond)
			log.Println("eventrun:", "count:", ecount)
		},
	}
	e := FileMetaFilesEvent{}
	select {
	case ch <- e:
	}
	select {
	case <-time.After(time.Duration(duSec*N+1) * time.Second):
		assert.True(t, N == btwecount || btwecount == N+1)
	}
	select {
	case ch <- e:
	}
	select {
	case <-time.After(time.Duration(duSec*N+1) * time.Second):
		assert.True(t, N*2 == btwecount || btwecount == N*2+1)
	}
}

func waitRunnerPeriodicRun(t *testing.T, r *Runner,
	ch chan FileMetaFilesEvent, N, duSec uint32) {
	start := common.Start()
	periodiccount := uint32(0)
	r.RUNFuncs = map[RUN]func(*Runner, FileMetaFilesEvent){
		NOP: func(r *Runner, e FileMetaFilesEvent) {
			periodiccount++
			assert.Equal(t, nil, e.Err)
			diff := common.Elapsed(start) - time.Second*time.Duration(duSec*periodiccount+1)
			log.Println("event duration:", diff)
			assert.True(t, diff < time.Duration(1)*time.Millisecond)
			log.Println("periodcrun:", "count:", periodiccount)
		},
	}
	select {
	case <-time.After(time.Duration(duSec*N+1) * time.Second):
		assert.True(t, N == periodiccount || periodiccount == N+1)
	}
}
