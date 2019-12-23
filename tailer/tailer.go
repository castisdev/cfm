package tailer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cilog"
)

// Tailer :
type Tailer struct {
	watchIPString string
	watchTermMin  int
	watchDir      string
	watchHitBase  int
}

var tailer common.MLogger

func init() {
	tailer = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "tailer"}
}

// NewTailer :
func NewTailer() *Tailer {
	// from - 값이 아닌 경우 -10 으로 설정
	return &Tailer{
		watchHitBase:  10,
		watchTermMin:  10,
		watchIPString: "125.159.40.3",
		watchDir:      "/var/log/castis/lb_log",
	}
}

// SetWatchDir :
func (t *Tailer) SetWatchDir(dir string) {
	tailer.Debugf("set watch dir : (%s)", dir)
	t.watchDir = dir
}

// SetWatchIPString :
func (t *Tailer) SetWatchIPString(ip string) {
	tailer.Debugf("set watch ip string : (%s)", ip)
	t.watchIPString = ip
}

// SetWatchTermMin :
func (t *Tailer) SetWatchTermMin(minute int) {
	if minute <= 0 {
		tailer.Debugf("invalid value, set as default watch term min : (%d)", 10)
		t.watchTermMin = 10
	}
	tailer.Debugf("set watch term min : (%d)", minute)
	t.watchTermMin = minute
}

// SetWatchHitBase :
func (t *Tailer) SetWatchHitBase(baseHit int) {
	if baseHit <= 0 {
		tailer.Debugf("set watch hit base : (%d)", 3)
		t.watchHitBase = 3
	}
	tailer.Debugf("set watch hit base : (%d)", baseHit)
	t.watchHitBase = baseHit
}

// Tail : working on Linux only
func (t *Tailer) Tail(fileMap *map[string]int) {

	now := time.Now()
	// 현재 시각값을 이용하여 N분 전 시각을 구하기 위해선 음수 값이 필요하다.
	from := now.Add(time.Minute * time.Duration(t.watchTermMin*-1))

	logFileNames := t.getLogFileName()

	for _, file := range *logFileNames {

		readOffset, err := t.parseLBEventLog(file, int64(0), from.Unix(), fileMap)
		if err != nil {
			tailer.Errorf("fail to parse,file(%s),error(%s)", file, err.Error())
			continue
		}
		tailer.Debugf("parse file(%s) from (0) to (%d)", file, readOffset)
	}

	tailer.Debugf("hit file count(%d) in LB log", len(*fileMap))
	// Hit 수가 기준 미달일 경우 file list 에서 제외
	for fileName, hitCount := range *fileMap {
		if hitCount < t.watchHitBase {
			//tailer.Debugf("deleted in rising hit, file(%s), hit(%d)", fileName, hitCount)
			delete(*fileMap, fileName)
		}
		tailer.Debugf("rising hit file(%s), hit(%d)", fileName, hitCount)
	}

	return
}

// getLogFileName : 파싱해야할 로그 파일명을 현재 시각 기준으로 생성
// 로그 파일명 : EventLog[YYYYmmdd].log
// YYYYmmdd 를 구하기 위해 현재 시각 이용
func (t *Tailer) getLogFileName() *[]string {

	logFileNames := make([]string, 0, 2)

	now := time.Now()
	// 현재 시각값을 이용하여 N분 전 시각을 구하기 위해선 음수 값이 필요하다.
	from := now.Add(time.Minute * time.Duration(t.watchTermMin*-1))

	f1 := fmt.Sprintf("%s/EventLog[%s].log", t.watchDir, from.Format("20060102"))
	f2 := fmt.Sprintf("%s/EventLog[%s].log", t.watchDir, now.Format("20060102"))

	// 시간 순서대로 정렬해야 parsing 로직에서 lastOffset 계산이 정상적으로 된다.
	// 오래된 파일이 제일 앞에 추가되어야 한다.
	logFileNames = append(logFileNames, f1)

	if f2 != f1 {
		logFileNames = append(logFileNames, f2)
	}

	return &logFileNames
}

func (t *Tailer) parseLBEventLog(fileName string, offset int64, baseTime int64, fileMap *map[string]int) (int64, error) {

	f, err := os.Open(fileName)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	f.Seek(offset, io.SeekStart)

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {

		b := scanner.Bytes()
		offset += int64(len(b) + 1) // line 끝에 \n 가 있으므로 +1, windows 의 경우 \r\n 이므로 +2 를 해줘야 함

		line := string(b)

		if line == "" {
			continue
		}

		ss := strings.FieldsFunc(line, func(r rune) bool {
			if r == ',' {
				return true
			}
			return false
		})

		if ss[0] != "0x40ffff" {
			// tailer.Debugf("(%s) != 0x40ffff", ss[0])
			continue
		}

		logTime, err := strconv.ParseInt(ss[2], 10, 64)
		if err != nil {
			// tailer.Debugf("fail to strconv (%s)", ss[2])
			continue
		}

		if logTime < baseTime {
			// tailer.Debugf("logTime(%d) < baseTime(%d)", logTime, baseTime)
			continue
		}

		matched, err := regexp.MatchString(t.watchIPString, ss[3])
		if err != nil {
			// tailer.Debugf("regexp match error(%s)", err.Error())
			continue
		}

		if matched {
			re := regexp.MustCompile(`file\(([a-zA-Z0-9_\.]+)\)`)
			file := re.FindStringSubmatch(line)
			if len(file) != 0 {
				(*fileMap)[file[1]]++
				// tailer.Debugf("found %s", file[1])
			}
		}
	}

	return offset, nil
}
