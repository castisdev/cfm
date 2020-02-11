package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func writeconfigfile(dir string, filename string, yml []byte) {
	fp := filepath.Join(dir, filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.Create(fp)
	if err != nil {
		f.Close()
		log.Fatal(err)
	}
	fmt.Fprintf(f, "%s", yml)
	f.Close()
}

func deletefile(dir string, filename string) {
	dir = filepath.Clean(dir)
	if dir == "." || dir == ".." {
		log.Fatal(errors.New("do not delete current or parent folder"))
	}
	fp := filepath.Join(dir, filename)
	err := os.RemoveAll(fp)
	if err != nil {
		log.Fatal(err)
	}
}

func TestConfigWatcher(t *testing.T) {
	viper.SetConfigType("yaml")
	var tctbl = []struct {
		yml  []byte
		wfe  bool
		wets uint32
		wpis uint32
	}{
		{yml: []byte(`
  watcher:
    fire_initial_event: true
    event_timeout_sec : 30
    poll_interval_sec : 60
  `), wfe: true, wets: 30, wpis: 60},
		{yml: []byte(`
  watcher:
    fire_initial_event: false
    event_timeout_sec : 0
    poll_interval_sec : 0
  `), wfe: false, wets: 0, wpis: 0},
	}

	for _, tc := range tctbl {
		var c Config
		if err := viper.ReadConfig(bytes.NewBuffer(tc.yml)); err != nil {
			t.Error(err)
		}
		if err := viper.Unmarshal(&c); err != nil {
			t.Error(err)
		}
		assert.Equal(t, tc.wfe, c.Watcher.FireInitialEvent)
		assert.Equal(t, tc.wets, c.Watcher.EventTimeoutSec)
		assert.Equal(t, tc.wpis, c.Watcher.PollingSec)
	}

	dir := "testconfig"
	file := "config.yml"
	defer deletefile(dir, "")
	for _, tc := range tctbl {
		var c Config
		writeconfigfile(dir, file, tc.yml)
		defer deletefile(dir, file)
		viper.SetConfigFile(filepath.Join(dir, file))
		if err := viper.ReadInConfig(); err != nil {
			t.Error(err)
		}
		if err := viper.Unmarshal(&c); err != nil {
			t.Error(err)
		}
		assert.Equal(t, tc.wfe, c.Watcher.FireInitialEvent)
		assert.Equal(t, tc.wets, c.Watcher.EventTimeoutSec)
		assert.Equal(t, tc.wpis, c.Watcher.PollingSec)
	}
}

func TestConfigRunner(t *testing.T) {
	viper.SetConfigType("yaml")
	var tctbl = []struct {
		yml    []byte
		wberis uint32
		wpris  uint32
		wsetup map[string][]string
		wvalid bool
	}{
		{yml: []byte(`
  runner:
    between_events_run_interval_sec: 10
    periodic_run_interval_sec : 40
  `), wberis: 10, wpris: 40, wvalid: true},
		{yml: []byte(`
   runner:
    between_events_run_interval_sec: 0
    periodic_run_interval_sec : 0
  `), wberis: 0, wpris: 0, wvalid: true},
		{yml: []byte(`
  runner:
    between_events_run_interval_sec: 0
    periodic_run_interval_sec : 0
    setup_runs:
      eventRuns: [nop]
  `), wberis: 0, wpris: 0,
			wsetup: map[string][]string{"eventruns": []string{"nop"}}, wvalid: true,
		},
		{yml: []byte(`
      runner:
        between_events_run_interval_sec: 0
        periodic_run_interval_sec : 0
        setup_runs:
          eventRuns: [makeFmm, makeRisingHit, runRemover, runTasker]
          eventTimeoutRuns: [makeFmm, makeRisingHit, runRemover, runTasker]
          betweenEventsRuns: [makeRisingHit, runRemover, runTasker]
          periodicRuns: [nop]
    `), wberis: 0, wpris: 0,
			wsetup: map[string][]string{
				"eventruns":         []string{"makeFmm", "makeRisingHit", "runRemover", "runTasker"},
				"eventtimeoutruns":  []string{"makeFmm", "makeRisingHit", "runRemover", "runTasker"},
				"betweeneventsruns": []string{"makeRisingHit", "runRemover", "runTasker"},
				"periodicruns":      []string{"nop"},
			},
			wvalid: true,
		},
		{yml: []byte(`
      runner:
        between_events_run_interval_sec: 0
        periodic_run_interval_sec : 0
        setup_runs:
          eventRuns: [nop]
          eventTimeoutRuns: [makeFmm, makeRisingHit, runRemover, runTasker]
          someRuns: [nop]
    `), wberis: 0, wpris: 0,
			wsetup: map[string][]string{
				"eventruns":        []string{"nop"},
				"eventtimeoutruns": []string{"makeFmm", "makeRisingHit", "runRemover", "runTasker"},
				"someruns":         []string{"nop"}},
			wvalid: false,
		},
		{yml: []byte(`
      runner:
        between_events_run_interval_sec: 0
        periodic_run_interval_sec : 0
        setup_runs:
          eventRuns: [someRun]
          eventTimeoutRuns: [makeFmm, makeRisingHit, runRemover, runTasker]
    `), wberis: 0, wpris: 0,
			wsetup: map[string][]string{
				"eventruns":        []string{"someRun"},
				"eventtimeoutruns": []string{"makeFmm", "makeRisingHit", "runRemover", "runTasker"},
			},
			wvalid: false,
		},
	}

	for _, tc := range tctbl {
		var c Config
		if err := viper.ReadConfig(bytes.NewBuffer(tc.yml)); err != nil {
			t.Error(err)
		}
		if err := viper.Unmarshal(&c); err != nil {
			t.Error(err)
		}
		assert.Equal(t, tc.wberis, c.Runner.BetweenEventsRunSec)
		assert.Equal(t, tc.wpris, c.Runner.PeriodicRunSec)
		assert.Equal(t, tc.wsetup, c.Runner.SetupRuns)
		if err := c.Runner.validate(); err != nil {
			assert.Equal(t, tc.wvalid, false)
		} else {
			assert.Equal(t, tc.wvalid, true)
		}
	}

	dir := "testconfig"
	file := "config.yml"
	defer deletefile(dir, "")
	for _, tc := range tctbl {
		var c Config
		writeconfigfile(dir, file, tc.yml)
		defer deletefile(dir, file)
		viper.SetConfigFile(filepath.Join(dir, file))
		if err := viper.ReadInConfig(); err != nil {
			t.Error(err)
		}
		if err := viper.Unmarshal(&c); err != nil {
			t.Error(err)
		}
		assert.Equal(t, tc.wberis, c.Runner.BetweenEventsRunSec)
		assert.Equal(t, tc.wpris, c.Runner.PeriodicRunSec)
		assert.Equal(t, tc.wsetup, c.Runner.SetupRuns)
		if err := c.Runner.validate(); err != nil {
			assert.Equal(t, tc.wvalid, false)
		} else {
			assert.Equal(t, tc.wvalid, true)
		}
	}
}

func TestReadConfigValidationConfig(t *testing.T) {
	viper.SetConfigType("yaml")
	var tctbl = []struct {
		yml    []byte
		wc     Config
		wvalid bool
		werror error
	}{
		{
			yml: []byte(`
      # 배포/삭제 제외 파일
      ignore:
        # prefix
        # 기본값 : 없음, 설정해주어야 함, 대소문자 구분
        # 예:
        # KT에서는 광고 파일은 배포/삭제 대상에서 제외해야 한다.
        # KT 에서 사용하는 광고 파일은 이름에 특별한 규칙이 있다. (M64 또는 MN1 으로 시작함)
        # 이를 이용하여 광고 파일 여부를 결정한다.
        # ignore.prefix 를 설정해주어야 한다.
        #   prefixes:
        #   - M64
        #   - MN1
        prefixes:
        - M64
        - MN1

      # 배포할 파일들이 존재하는 경로
      source_dirs:
      - data2
      - data3

      # 파일 우선순위,크기를 구하기 위해 이용하는 파일들의 경로
      hitcount_history_file: data/.hitcount.history
      grade_info_file: data/.grade.info

      # 유효한 log_level : debug,report,info,success,warning,error,fail,exception,critical
      log_level: debug
      log_dir: log

      # remover :
      # cfw의 disk usage를 검사하여 사용량이 storage_usage_limit_percent 이상인 경우,
      # cfw에게 파일을 지우는 요청 하는 모듈
      remover:
        # cfw 의 용량 제한 %, 기본값 : 90, 0<= 값 <= 100
        # cfw 의 disk 사용량이 이 값보다 커지면 cfw에 파일 삭제 요청을 함
        # storage_usage_limit_percent: 90
        storage_usage_limit_percent: 99

        # deprecated
        # 파일 삭제를 위한 검사 후 쉬는 시간(초), 기본값 : 30
        # 이 값이 30초라면 검사하고 30초 쉬고, 검사하고 30초 쉬는 방식으로 동작
        #remover_sleep_sec : 3

      # tasker:
      # 배포 task를 만드는 모듈,
      # 한 번에 소스 갯수만큼의 배포 task가 만들어짐
      tasker:
        # cfw 가 배포 스케줄을 시작해놓고 비정상 종료해버리거나
        # cfw 가 아예 구동이 안되는 등의 예외 상황이 생길 경우
        # 해당 cfw 에 대한 배포 스케줄을 취소해야 한다.
        # 이를 위해 task 마다 timeout 을 설정한다.
        # 배포 task 타임아웃(초), 기본값: 3600
        task_timeout_sec: 30

        # cfw 가 copy 할 때 사용하는 속도 : 기본값 10000000
        task_copy_speed_bps: 10000000

        # deprecated
        # 소스 개수 만큼 배포 task 만들고 나서 쉬는 시간(초), 기본값 : 60
        # 이 값이 60초라면 배포 task 만들고, 60초 쉬고, 만들고 60초 쉬는 방식으로 동작
        #tasker_sleep_sec: 10

      # cfm 의 ip, address, 기본값 127.0.0.1:8080
      listen_addr: 127.0.0.1:7888

      # 파일 우선순위,크기를 구하기 위해 이용하는 파일들 감시 설정
      watcher:
        # 해당 파일의 해당 경로에 존재하면 최초 이벤트 발생하는 설정 : 기본값 : true
        fire_initial_event: true

        # 해당 파일에 아무런 변화도 없을 때 발생하는 timeout event 설정(초):
        # 기본값 : 3600
        event_timeout_sec : 30

        # inotify와 같은 filesystem notify module이 지원되지 않을 때, 사용하는 감시 주기(초)
        # 기본값 : 60
        poll_interval_sec : 60

      # 파일 우선순위,크기를 구하기 위해 이용하는 파일들에 변화가 있을 때
      # remover, tasker를 실행하는 모듈
      runner:
        # 해당 파일에 변경이 없는 동안 주기적으로 실행하는 설정(초): 기본값 : 60
        between_events_run_interval_sec: 10

        # 주기적으로 실행하는 설정(초): 기본값 : 60
        periodic_run_interval_sec : 40

        # 가능한 설정 :
        # name of Runs : [eventRuns, eventTimeoutRuns, betweenEventsRuns, periodicRuns]
        # Run : {nop, makeFmm, makeRisingHit, printFmm, runRemover, runTasker}
        # default 설정
        # setup_runs:
        #   eventRuns: [makeFmm, makeRisingHit, runRemover, runTasker]
        #   eventTimeoutRuns: [makeFmm, makeRisingHit, runRemover, runTasker]
        #   betweenEventsRuns: [makeRisingHit, runRemover, runTasker]
        #   periodicRuns: [nop]
        setup_runs:
          eventRuns: [nop]
          eventTimeoutRuns: [nop]
          betweenEventsRuns: [nop]
          periodicRuns: [nop]

      servers:
        # servers.sources, servers.destinations에 대한
        # heartbeat 타입아웃(초), 기본값: 5
        heartbeat_timeout_sec : 5

        # servers.sources, servers.destinations에 대한
        # heartbeat 검사 쉬는 시간(초), 기본값: 30
        # 이 값이 30인 경우, 검사하고 30초 쉬고, 다시 검사하고 30초 쉬는 방식으로 동작
        heartbeat_interval_sec : 30

        sources: # 배포 시 source 로 선택할 서버
          - 127.0.0.1:8888
          - 127.0.0.1:8889
        destinations: # 배포 대상 서버
          - 127.0.0.1:9888
          - 127.0.0.1:9889
      #    - 172.16.33.52:8889

      # LB EventLog 가 존재하는 경로
      watch_dir: lb_log

      # LB EventLog 에서 watch_term_min 동안의 로그 중
      # watch_ip_string 에 match되는 로그를 찾아
      # 해당 서버에서 서비스 되는 파일들의 hit 수를 구한다.
      # hit 수가 watch_hit_base 보다 크거나 같으면
      # 해당 파일은 grade 에 상관없이 최우선으로 배포한다.
      # 해당 파일은 grade 에 상관없이 삭제 되지 않는다.
      watch_ip_string: 125.159.40.3
      watch_hit_base: 5              # 10분 동안 hit >= 5 이면 배포
      watch_term_min: 10             # 10분 동안의 로그만 파싱
  `),
			wc: Config{
				SourceDirs:          []string{"data2", "data3"},
				HitcountHistoryFile: "data/.hitcount.history",
				GradeInfoFile:       "data/.grade.info",
				LogDir:              "log",
				LogLevel:            "debug",
				Servers: Server{Sources: []string{"127.0.0.1:8888", "127.0.0.1:8889"},
					Destinations:        []string{"127.0.0.1:9888", "127.0.0.1:9889"},
					HeartbeatTimeoutSec: 5,
					HeartbeatSec:        30,
				},
				WatchDir:       "lb_log",
				WatchIPString:  "125.159.40.3",
				WatchTermMin:   10,
				WatchHitBase:   5,
				EnableCoreDump: true,
				ListenAddr:     "127.0.0.1:7888",
				Remover:        Remover{RemoverSleepSec: 30, StorageUsageLimitPercent: 99},
				Tasker:         Tasker{TaskerSleepSec: 60, TaskTimeout: 30, TaskCopySpeedBPS: "10000000"},
				Ignore:         Ignore{Prefixes: []string{"M64", "MN1"}},
				Watcher:        Watcher{FireInitialEvent: true, EventTimeoutSec: 30, PollingSec: 60},
				Runner: Runner{BetweenEventsRunSec: 10, PeriodicRunSec: 40,
					SetupRuns: map[string][]string{"eventruns": []string{"nop"},
						"eventtimeoutruns":  []string{"nop"},
						"betweeneventsruns": []string{"nop"},
						"periodicruns":      []string{"nop"}},
				},
			},
			wvalid: true,
		},
		{ // deafult setting
			yml: []byte(``),
			wc: Config{LogDir: "log",
				LogLevel: "info",
				Servers: Server{
					HeartbeatTimeoutSec: 5,
					HeartbeatSec:        30,
				},
				EnableCoreDump: true,
				ListenAddr:     "127.0.0.1:8080",
				Remover:        Remover{RemoverSleepSec: 30, StorageUsageLimitPercent: 90},
				Tasker:         Tasker{TaskerSleepSec: 60, TaskTimeout: 3600, TaskCopySpeedBPS: "10000000"},
				Watcher:        Watcher{FireInitialEvent: true, EventTimeoutSec: 600, PollingSec: 60},
				Runner:         Runner{BetweenEventsRunSec: 10, PeriodicRunSec: 0},
			},
			wvalid: true,
		},
		{ // invalid loglevel setting
			yml: []byte(`
      log_level: invalidlevel
      `),
			wc: Config{LogDir: "log",
				LogLevel: "invalidlevel",
				Servers: Server{
					HeartbeatTimeoutSec: 5,
					HeartbeatSec:        30,
				},
				EnableCoreDump: true,
				ListenAddr:     "127.0.0.1:8080",
				Remover:        Remover{RemoverSleepSec: 30, StorageUsageLimitPercent: 90},
				Tasker:         Tasker{TaskerSleepSec: 60, TaskTimeout: 3600, TaskCopySpeedBPS: "10000000"},
				Watcher:        Watcher{FireInitialEvent: true, EventTimeoutSec: 600, PollingSec: 60},
				Runner:         Runner{BetweenEventsRunSec: 10, PeriodicRunSec: 0},
			},
			wvalid: false, werror: errors.New("invalid log_level : error(invalid level string [invalidlevel])"),
		},
		{ // invalid ip setting
			yml: []byte(`
      log_level: info
      listen_addr: 127.0.0.1
      `),
			wc: Config{LogDir: "log",
				LogLevel: "info",
				Servers: Server{
					HeartbeatTimeoutSec: 5,
					HeartbeatSec:        30,
				},
				EnableCoreDump: true,
				ListenAddr:     "127.0.0.1",
				Remover:        Remover{RemoverSleepSec: 30, StorageUsageLimitPercent: 90},
				Tasker:         Tasker{TaskerSleepSec: 60, TaskTimeout: 3600, TaskCopySpeedBPS: "10000000"},
				Watcher:        Watcher{FireInitialEvent: true, EventTimeoutSec: 600, PollingSec: 60},
				Runner:         Runner{BetweenEventsRunSec: 10, PeriodicRunSec: 0},
			},
			wvalid: false, werror: errors.New("invalid listen_addr : error(address 127.0.0.1: missing port in address)"),
		},
		{ // invalid src_dir setting
			yml: []byte(`
      log_level: info
      listen_addr: 127.0.0.1
      source_dirs :
      - hello
      `),
			wc: Config{
				SourceDirs: []string{"hello"},
				LogDir:     "log",
				LogLevel:   "info",
				Servers: Server{
					HeartbeatTimeoutSec: 5,
					HeartbeatSec:        30,
				},
				EnableCoreDump: true,
				ListenAddr:     "127.0.0.1",
				Remover:        Remover{RemoverSleepSec: 30, StorageUsageLimitPercent: 90},
				Tasker:         Tasker{TaskerSleepSec: 60, TaskTimeout: 3600, TaskCopySpeedBPS: "10000000"},
				Watcher:        Watcher{FireInitialEvent: true, EventTimeoutSec: 600, PollingSec: 60},
				Runner:         Runner{BetweenEventsRunSec: 10, PeriodicRunSec: 0},
			},
			wvalid: false, werror: errors.New("invalid source_dirs : error(stat hello: no such file or directory)"),
		},
	}
	dir := "testconfig"
	file := "config.yml"
	defer deletefile(dir, "")
	for _, tc := range tctbl {
		writeconfigfile(dir, file, tc.yml)
		defer deletefile(dir, file)
		viper.SetConfigFile(filepath.Join(dir, file))
		c, err := ReadConfig(filepath.Join(dir, file))
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, tc.wc, *c)

		if err := ValidationConfig(*c); err != nil {
			log.Println(err)
			assert.Equal(t,
				tc.werror.Error(),
				err.Error())
			assert.Equal(t, tc.wvalid, false)
		} else {
			assert.Equal(t, tc.wvalid, true)
		}
	}
}
