package tasker

import (
	"container/ring"
	"errors"
	"sort"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cilog"
)

var maxTaskCount int
var taskTimeout time.Duration

// SrcHost 구조체에 src 선택 여부를 알기 위한 bool 변수 추가
type SrcHost struct {
	common.Host
	selected bool
}

// SrcHosts : SrcHost sturct slice
type SrcHosts []*SrcHost

// DstServers : 파일 배포 대상 서버 리스트
var DstServers *common.Hosts

// SrcServers : 배포할 파일들을 갖고 있는 서버
var SrcServers *SrcHosts

var tasks *Tasks
var advPrefixes []string
var hitcountHistoryFile string
var gradeInfoFile string

// GetTaskListInstance is to get global task list structure's addr
func GetTaskListInstance() *Tasks {
	return tasks
}

// NewHosts is constructor of Hosts
func NewHosts() *SrcHosts {
	return new(SrcHosts)
}

// Add is to add host to source servers
func (srcs *SrcHosts) Add(s string) error {

	host, err := common.SplitHostPort(s)
	src := SrcHost{host, false}

	if err != nil {
		return err
	}

	*srcs = append(*srcs, &src)
	return nil
}

func (srcs *SrcHosts) selectSourceServer() (string, bool) {

	for _, src := range *srcs {

		if src.selected != true {
			src.selected = true
			return src.IP, true
		}

	}
	return "", false
}

// SourcePath : 배포할 파일이 존재하는 경로
var SourcePath *common.SourceDirs

func init() {

	maxTaskCount = 10
	taskTimeout = 30 * time.Minute

	SrcServers = NewHosts()
	DstServers = common.NewHosts()
	SourcePath = common.NewSourceDirs()

	tasks = NewTasks()
}

// RunForever is to run tasker as go routine
func RunForever() {

	// destination ip 를 round robin 으로 선택하기 위한 ring 생성
	dstRing := ring.New(len(*DstServers))
	for _, s := range *DstServers {
		dstRing.Value = s.IP
		dstRing = dstRing.Next()
		cilog.Debugf("add to ring (%s)", s.IP)
	}

	// elapsed time : 소요 시간
	var est time.Time

	for {

		CleanTask(tasks)

		// 1. unset selected flag (true->false)
		for _, src := range *SrcServers {
			src.selected = false
		}

		// 2. task queue 에 있는 src ip는 할당된 상태로 변경
		n := 0
		for _, task := range tasks.TaskMap {
			for _, src := range *SrcServers {
				if src.IP == task.SrcIP {
					src.selected = true
					n++
				}
			}
		}

		if n == len(*SrcServers) {
			cilog.Debugf("src is full")
			time.Sleep(time.Second * 5)
			continue
		}

		// 4. 파일 등급 list 생성
		fileMetaMap := make(map[string]*common.FileMeta)

		est = time.Now()
		if err := common.ParseGradeFile(gradeInfoFile, fileMetaMap); err != nil {
			cilog.Debugf("fail to parse file(%s), error(%s)", gradeInfoFile, err.Error())
			break
		} else {
			cilog.Debugf("parse file(%s),time(%s)", gradeInfoFile, time.Since(est))
		}

		est = time.Now()
		if err := common.ParseHitcountFile(hitcountHistoryFile, fileMetaMap); err != nil {
			cilog.Debugf("fail to parse file(%s), error(%s)", hitcountHistoryFile, err.Error())
			break
		} else {
			cilog.Debugf("parse file(%s),time(%s)", hitcountHistoryFile, time.Since(est))
		}

		// 5. 모든 서버의 파일 리스트 수집
		remoteFileSet := make(map[string]int)
		CollectRemoteFileList(DstServers, remoteFileSet)

		// 6. 높은 등급 순서로 정렬하기 위해 빈 Slice 생성 (map->slice)
		sortedFileList := make([]*common.FileMeta, 0, len(fileMetaMap))

		for _, v := range fileMetaMap {

			// 7. 광고 파일은 제외
			if common.IsADFile(v.Name, advPrefixes) {
				continue
			}
			sortedFileList = append(sortedFileList, v)
		}

		// 8. 높은 등급 순서로 정렬 (가장 높은 등급:1)
		sort.Slice(sortedFileList, func(i, j int) bool {
			return sortedFileList[i].Grade < sortedFileList[j].Grade
		})

		for _, file := range sortedFileList {

			// 9. 이미 task queue 에 있는 파일이면 skip
			if _, exists := tasks.FindTaskByFileName(file.Name); exists {
				//cilog.Debugf("%s is already in task queue", file.Name)
				continue
			}

			// 10. 이미 remote file list 에 있는 파일이면 skip
			if _, exists := remoteFileSet[file.Name]; exists {
				//cilog.Debugf("%s is already in remote file list", file.Name)
				continue
			}

			// 11. SAN 에 없는 파일이면 제외
			filePath, exists := SourcePath.IsExistOnSource(file.Name)
			if exists != true {
				//cilog.Debugf("%s not found in sources", file.Name)
				continue
			}

			// 12. src ip 선택, 없으면 loop 종료
			srcIP, exists := SrcServers.selectSourceServer()
			if exists != true {
				cilog.Debugf("src is full")
				break
			}

			// 13. task 생성
			dstIP := string(dstRing.Value.(string))
			t := tasks.CreateTask(&Task{FilePath: filePath, FileName: file.Name, SrcIP: srcIP, DstIP: dstIP, Grade: file.Grade})
			cilog.Infof("create task,ID(%d),Grade(%d),FilePath(%s),SrcIP(%s),DstIP(%s),Ctime(%d),Mtime(%d)",
				t.ID, t.Grade, t.FilePath, t.SrcIP, t.DstIP, t.Ctime, t.Mtime)
			dstRing = dstRing.Next()
		}

		time.Sleep(time.Second * 5)
	}
}

// CleanTask is to delete task which is timeout or done
func CleanTask(tasks *Tasks) {

	tl := make([]Task, 0, 10)

	for _, task := range tasks.TaskMap {

		if task.Status == DONE {
			tl = append(tl, *task)
			continue
		}

		diff := time.Since(time.Unix(task.Mtime, 0))
		if diff > taskTimeout {
			task.Status = TIMEOUT
			tl = append(tl, *task)
		}
	}

	for _, t := range tl {

		switch t.Status {
		case DONE:
			cilog.Successf("task is done,ID(%d),Grade(%d),FilePath(%s),SrcIP(%s),DstIP(%s),Ctime(%d),Mtime(%d),Status(%s)",
				t.ID, t.Grade, t.FilePath, t.SrcIP, t.DstIP, t.Ctime, t.Mtime, t.Status)
		case TIMEOUT:
			cilog.Warningf("task timeout!!,ID(%d),Grade(%d),FilePath(%s),SrcIP(%s),DstIP(%s),Ctime(%d),Mtime(%d),Status(%s)",
				t.ID, t.Grade, t.FilePath, t.SrcIP, t.DstIP, t.Ctime, t.Mtime, t.Status)
		default:
			cilog.Warningf("unexpected task status,ID(%d),Grade(%d),FilePath(%s),SrcIP(%s),DstIP(%s),Ctime(%d),Mtime(%d),Status(%s)",
				t.ID, t.Grade, t.FilePath, t.SrcIP, t.DstIP, t.Ctime, t.Mtime, t.Status)
		}
		tasks.DeleteTask(t.ID)
	}
}

// CollectRemoteFileList is to get file list on remote servers
func CollectRemoteFileList(hostList *common.Hosts, remoteFileSet map[string]int) {

	for _, host := range *hostList {
		fl := make([]string, 0, 10000)
		err := common.GetRemoteFileList(host, &fl)

		for _, file := range fl {
			remoteFileSet[file]++
		}

		if err != nil {
			cilog.Errorf("fail to connect to (%s:%d)", host.IP, host.Port)
		} else {
			cilog.Debugf("get remote(%s:%d) file list", host.IP, host.Port)
		}
	}
}

// SetTaskTimeout is to set timeout for task
func SetTaskTimeout(t time.Duration) error {

	if t < 0 {
		return errors.New("can not use negative value")
	}

	taskTimeout = t
	cilog.Infof("set task timeout : (%s)", taskTimeout)
	return nil
}

// SetAdvPrefix :
func SetAdvPrefix(p []string) {
	advPrefixes = p
}

// SetHitcountHistoryFile :
func SetHitcountHistoryFile(f string) {
	hitcountHistoryFile = f
}

// SetGradeInfoFile :
func SetGradeInfoFile(f string) {
	gradeInfoFile = f
}
