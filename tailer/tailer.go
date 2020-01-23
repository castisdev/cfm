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

// getLogFileName : 파싱해야할 로그 파일명을 현재 시각 기준으로 생성
// 로그 파일명 : EventLog[YYYYmmdd].log
// YYYYmmdd 를 구하기 위해 현재 시각 이용
func GetLogFileName(basetm time.Time, watchDir string, watchTermMin int) *[]string {

	logFileNames := make([]string, 0, 2)

	// base time값을 이용하여 N분 전 시각을 구하기 위해선 음수 값이 필요하다.
	from := basetm.Add(time.Minute * time.Duration(watchTermMin*-1))

	f1 := fmt.Sprintf("%s/EventLog[%s].log", watchDir, from.Format("20060102"))
	f2 := fmt.Sprintf("%s/EventLog[%s].log", watchDir, basetm.Format("20060102"))

	// 시간 순서대로 정렬해야 parsing 로직에서 lastOffset 계산이 정상적으로 된다.
	// 오래된 파일이 제일 앞에 추가되어야 한다.
	logFileNames = append(logFileNames, f1)

	if f2 != f1 {
		logFileNames = append(logFileNames, f2)
	}

	return &logFileNames
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
	tailer.Infof("set watchDir : (%s)", dir)
	t.watchDir = dir
}

// SetWatchIPString :
func (t *Tailer) SetWatchIPString(ip string) {
	tailer.Infof("set watchIpString : (%s)", ip)
	t.watchIPString = ip
}

// SetWatchTermMin :
func (t *Tailer) SetWatchTermMin(minute int) {
	if minute <= 0 {
		tailer.Infof("invalid value, set as default watchTermMin : (%d)", 10)
		t.watchTermMin = 10
	}
	tailer.Infof("set watchTermMin : (%d)", minute)
	t.watchTermMin = minute
}

// SetWatchHitBase :
func (t *Tailer) SetWatchHitBase(baseHit int) {
	if baseHit <= 0 {
		tailer.Infof("set watchHitBase : (%d)", 3)
		t.watchHitBase = 3
	}
	tailer.Infof("set watchHitBase : (%d)", baseHit)
	t.watchHitBase = baseHit
}

// Tail : working on Linux only
//
// 입력받은 시각값의 watchTermMin 전부터의 로그를 tail 한다.
func (t *Tailer) Tail(basetm time.Time, fileMap *map[string]int) {
	tailer.Infof("started tail process")
	defer logElapased("ended tail process", common.Start())

	// 입력받은 시각값을 이용하여 N분 전 시각을 구하기 위해선 음수 값이 필요하다.
	from := basetm.Add(time.Minute * time.Duration(t.watchTermMin*-1))

	logFileNames := t.getLogFileName(basetm)

	for _, file := range *logFileNames {
		startOffset := int64(0)
		readOffset, err := t.parseLBEventLog(file, startOffset, from.Unix(), fileMap)
		if err != nil {
			tailer.Errorf("failed to parse, file(%s),error(%s)", file, err.Error())
			continue
		}
		tailer.Infof("parsed file(%s) from (%d) to (%d)", file, startOffset, readOffset)
	}

	tailer.Debugf("total hit files(%d)", len(*fileMap))
	// Hit 수가 기준 미달일 경우 file list 에서 제외
	for fileName, hitCount := range *fileMap {
		tailer.Debugf("hit file(%s), hit(%d)", fileName, hitCount)
		if hitCount < t.watchHitBase {
			delete(*fileMap, fileName)
		}
	}
	tailer.Debugf("total rising hit files(%d), hits > base hits(%d)",
		len(*fileMap), t.watchHitBase)

	return
}

// getLogFileName : 파싱해야할 로그 파일명을 현재 시각 기준으로 생성
// 로그 파일명 : EventLog[YYYYmmdd].log
// YYYYmmdd 를 구하기 위해 현재 시각 이용
func (t *Tailer) getLogFileName(basetm time.Time) *[]string {
	return GetLogFileName(basetm, t.watchDir, t.watchTermMin)
}

// 다음과 같은 log를 찾기위해서
//
// 0x40ffff,1,1527951607,Server 125.159.40.3 Selected for Client StreamID : 609d8714-096a-475e-994c-135deea7177f, ClientID : 0, GLB IP : 125.159.40.5's file(MZ3I5008SGL1500001_K20180602222428.mpg) Request
//
// 첫번째 필드는 0x40ffff 로 시작하고,
//
// 세번째 필드인 로그 시간이 기준 시간과 같거나 크고,
//
// 네번째 필드에 watchIPString이 있고, file(파일이름)로 되어있는 문자열이
// 있는 경우에
//
// 이와 같은 log line의 count 를 세어서 파일이름의 hitcount 를 구한다.
//
// 파일이름은 숫자와 영문문자와 .과 -와 _로 이루어진 문자열로 구성된다.
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
			// tailer.Debugf("failed to strconv (%s)", ss[2])
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

func logElapased(message string, start time.Time) {
	tailer.Infof("%s, time(%s)", message, common.Elapsed(start))
}
