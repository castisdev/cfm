package tasker

import (
	"container/ring"
	"errors"
	"sort"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/heartbeater"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cilog"
)

var sleepSec uint
var taskTimeout time.Duration

// HostStatus : 서버 heartbeat 상태
type HostStatus int

// task HostStatus const
const (
	NOTOK HostStatus = iota
	OK
)

// SrcHost : Source Host
// selected : task에서 src 선택되었는지 여부
// Status : host 상태
type SrcHost struct {
	common.Host
	selected bool
	Status   HostStatus
}

// SrcHosts : SrcHost sturct slice
type SrcHosts []*SrcHost

// DstHost : Destination host
// selected : task에서 src 선택되었는지 여부
// Status : host 상태
type DstHost struct {
	common.Host
	selected bool
	Status   HostStatus
}

// DstHosts : Destination host sturct slice
type DstHosts []*DstHost

// DstServers : 파일 배포 대상 서버 리스트
var DstServers *DstHosts

// SrcServers : 배포할 파일들을 갖고 있는 서버 리스트
var SrcServers *SrcHosts

// Tail :: LB EventLog 를 tailing 하며 SAN 에서 Hit 되는 파일 목록 추출
var Tail *tailer.Tailer

var tasks *Tasks
var advPrefixes []string
var hitcountHistoryFile string
var gradeInfoFile string
var taskCopySpeed string
var ignorePrefixes []string

// GetTaskListInstance is to get global task list structure's addr
func GetTaskListInstance() *Tasks {
	return tasks
}

// NewSrcHosts is constructor of SrcHosts
func NewSrcHosts() *SrcHosts {
	return new(SrcHosts)
}

// NewDstHosts is constructor of DstHosts
func NewDstHosts() *DstHosts {
	return new(DstHosts)
}

// Add is to add host to source servers
// IP, Port, Addr 값 이외 selected, status값은 초기값이 들어감
//
// 서버 순서를 일정하게 유지할 수 있도록 Addr 큰 순서로 sort 함
func (srcs *SrcHosts) Add(s string) error {

	host, err := common.SplitHostPort(s)
	if err != nil {
		return err
	}

	src := SrcHost{host, false, NOTOK}
	*srcs = append(*srcs, &src)

	sort.Slice(*srcs, func(i, j int) bool {
		return (*srcs)[i].Addr > (*srcs)[j].Addr
	})

	return nil
}

// Add : add destination host
// IP, Port, Addr 값 이외 selected, status값은 초기값이 들어감
//
// 서버 순서를 일정하게 유지할 수 있도록 Addr 큰 순서로 sort 함
func (dests *DstHosts) Add(s string) error {

	host, err := common.SplitHostPort(s)
	if err != nil {
		return err
	}

	dest := DstHost{host, false, NOTOK}
	*dests = append(*dests, &dest)

	sort.Slice(*dests, func(i, j int) bool {
		return (*dests)[i].Addr > (*dests)[j].Addr
	})

	return nil
}

// selecteSourceServer
//
// 아직, task 에 사용되지 않았고,
//
// status 가 OK 인 source 선택
//
// 선택된 source: *SrcHost와 선택 여부가 retrun 된다.
func (srcs *SrcHosts) selectSourceServer() (SrcHost, bool) {

	for _, src := range *srcs {

		if src.selected != true && src.Status == OK {
			src.selected = true
			return *src, true
		}

	}
	return SrcHost{}, false
}

// SourcePath : 배포할 파일이 존재하는 경로
var SourcePath *common.SourceDirs

var tasker common.MLogger

func init() {
	sleepSec = 60
	taskTimeout = 30 * time.Minute

	SrcServers = NewSrcHosts()
	DstServers = NewDstHosts()
	SourcePath = common.NewSourceDirs()
	Tail = tailer.NewTailer()

	tasker = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "tasker"}
}

// InitTasks:
//
// 원래 init() 함수 안에 있었는데,
//
// cfw등 다른 모듈에서 tasker package를 사용할 때, init() 함수가 호출될 때
//
// 실행되게 되어 따로 분리함
//
// RunForever() 함수가 호출되기 전에 따로 호출해주어야 함
func InitTasks() {
	tasks = NewTasks()
	tasks.LoadTasks()
}

// RunForever is to run tasker as go routine
func RunForever() {
	destcount := len(*DstServers)
	if destcount == 0 {
		tasker.Debugf("tasker endded, the number of the dest srvers is 0")
		return
	}
	dstIPMap := make(map[string]int)
	for _, dst := range *DstServers {
		dstIPMap[dst.IP]++
	}

	// elapsed time : 소요 시간
	var est time.Time
	for {

		// 0.5 Src heartbeat 검사
		SrcServers.setHostStatus()
		// 0.5.1 Dest heartbeat 검사
		DstServers.setHostStatus()

		CleanTask(tasks)

		// 1. unset selected flag (true->false)
		// 2. task queue 에 있는 src를 할당된 상태로 변경
		SrcServers.setSelected()
		// 2.5 task queue 에 있는 dest를 할당된 상태로 변경
		DstServers.setSelected()

		// status가 OK이고, 아직 배포 task 에 할당안된 source가 없으면
		// task를 더이상 만들지 않음
		srccnt := SrcServers.getSelectableCount()
		if srccnt == 0 {
			tasker.Debugf("no src is available")
			time.Sleep(time.Second * 5)
			continue
		}

		// destination ip 를 round robin 으로 선택하기 위한 ring 생성
		// status가 OK이고,
		// 아직 배포 task 에 할당안된 dest가 없으면
		// task를 더이상 만들지 않음
		okdests := DstServers.getSelectableList()
		if len(okdests) == 0 {
			tasker.Debugf("no dst is available")
			time.Sleep(time.Second * 5)
			continue
		}

		dstRing := ring.New(len(okdests))
		for _, d := range okdests {
			dstRing.Value = d
			dstRing = dstRing.Next()
			tasker.Debugf("[%s] added to available destination servers", d.Addr)
		}

		fileMetaMap := make(map[string]*common.FileMeta)
		duplicatedFileMap := make(map[string]*common.FileMeta)

		est = time.Now()
		// 4. 파일 등급 list 생성
		// gradeinfoFile 과 hitcountHistoryFile로 file meta list 생성
		err := common.MakeAllFileMetas(gradeInfoFile, hitcountHistoryFile,
			fileMetaMap, dstIPMap, duplicatedFileMap)

		if err != nil {
			tasker.Errorf("fail to make file metas, error(%s)", err.Error())
			time.Sleep(time.Second * 5)
			continue
		}
		tasker.Debugf("make file metas, time(%s)", time.Since(est))

		// 5. 높은 등급 순서로 정렬하기 위해 빈 Slice 생성 (map->slice)
		sortedFileList := make([]*common.FileMeta, 0, len(fileMetaMap))

		for _, v := range fileMetaMap {

			// 서버 파일 중 제외 파일 처리
			if common.IsPrefix(v.Name, ignorePrefixes) {
				tasker.Debugf("remove file meta by ignoring prefix, file(%s)", v.Name)
				continue
			}

			// 6. 광고 파일은 제외
			if common.IsADFile(v.Name, advPrefixes) {
				tasker.Debugf("remove adfile from file metas, file(%s)", v.Name)
				continue
			}
			sortedFileList = append(sortedFileList, v)
		}

		// 7. 높은 등급 순서로 정렬 (가장 높은 등급:1)
		sort.Slice(sortedFileList, func(i, j int) bool {
			return sortedFileList[i].Grade < sortedFileList[j].Grade
		})

		// 8. 모든 서버의 파일 리스트 수집
		remoteFileSet := make(map[string]int)
		CollectRemoteFileList(DstServers, remoteFileSet)

		// 9. LB EventLog 에서 특정 IP 에 할당된 파일 목록 추출
		hitMapFromLBLog := make(map[string]int)
		Tail.Tail(time.Now(), &hitMapFromLBLog)

		sortByHit := make([]*common.FileMeta, 0, len(hitMapFromLBLog))
		for fileName, hitCount := range hitMapFromLBLog {

			if common.IsPrefix(fileName, ignorePrefixes) {
				tasker.Debugf("remove file meta by ignoring prefix, file(%s)", fileName)
				continue
			}

			// 9.1. 광고 파일은 제외
			if common.IsADFile(fileName, advPrefixes) {
				tasker.Debugf("remove adfile from rising hit files, file(%s)", fileName)
				continue
			}

			// file size 셋팅 (.hitcount.history 에서 파싱한 정보)
			// .hitcount.history 에 없을 수도 있음(파일 갱신 주기 때문에)
			// 그럴 경우 Size = -1
			fileSize := int64(-1)
			if fm, exists := fileMetaMap[fileName]; exists {
				fileSize = fm.Size
			}

			// file grade 셋팅 (.grade.info 에서 파싱한 정보)
			// .grade.info 에 없을 수도 있음(파일 갱신 주기 때문에)
			// 그럴 경우 Grade = -1
			fileGrade := int32(-1)
			if fm, exists := fileMetaMap[fileName]; exists {
				fileGrade = fm.Grade
			}

			fileMeta := common.FileMeta{
				Name:      fileName,
				Grade:     fileGrade,
				Size:      fileSize,
				RisingHit: hitCount,
			}
			sortByHit = append(sortByHit, &fileMeta)
		}

		// 10. Hit 수가 많은 순서대로 정렬
		sort.Slice(sortByHit, func(i, j int) bool {
			return hitMapFromLBLog[sortByHit[i].Name] > hitMapFromLBLog[sortByHit[j].Name]
		})

		// 11. LB EventLog 에서 찾은 파일을 먼저 배포하고, 그 후에 .grade.info 등급 순으로 배포하기 위한 file list 정렬
		// sortedFileList가 sortByHist를 포함하는 관계라서, 중복이 있으나,
		// task 만들 때, 중복되는 경우 skip 됨
		sortedFileList = append(sortByHit, sortedFileList...)

		for _, file := range sortedFileList {

			// 12. 이미 task queue 에 있는 파일이면 skip
			if _, exists := tasks.FindTaskByFileName(file.Name); exists {
				tasker.Debugf("skip making task, already in task, file(%s)", file.Name)
				continue
			}

			// 13. 이미 remote file list 에 있는 파일이면 skip
			if _, exists := remoteFileSet[file.Name]; exists {
				tasker.Debugf("skip making task, already in remote file list, file(%s)", file.Name)
				continue
			}

			// 14. SAN 에 없는 파일이면 제외
			filePath, exists := SourcePath.IsExistOnSource(file.Name)
			if exists != true {
				tasker.Debugf("skip making task, not found in source paths, file(%s)", file.Name)
				continue
			}

			// 15. src ip 선택, 없으면 loop 종료
			src, exists := SrcServers.selectSourceServer()
			if exists != true {
				tasker.Debugf("stop making task, no src is available")
				break
			}

			// 16. task 생성
			dst := DstHost(dstRing.Value.(DstHost))
			t := tasks.CreateTask(&Task{
				FilePath:  filePath,
				FileName:  file.Name,
				SrcIP:     src.IP,
				DstIP:     dst.IP,
				Grade:     file.Grade,
				CopySpeed: taskCopySpeed,
				SrcAddr:   src.Addr,
				DstAddr:   dst.Addr,
			})
			dstRing = dstRing.Next()

			if file.RisingHit > 0 {
				tasker.Infof("[%d] create task(%s) for risingHit(%d), filePath(%s), file(%s)",
					t.ID, t, file.RisingHit, filePath, file.Name)
			} else {
				tasker.Infof("[%d] create task(%s) for grade(%d), filePath(%s), file(%s)",
					t.ID, t, file.Grade, filePath, file.Name)
			}
		}

		time.Sleep(time.Second * time.Duration(sleepSec))
	}
}

// CleanTask :
// lock 없이 직접 TaskMap traversal 하던 코드
// 	-> copy해서 사용하도록 수정(copy 할 때 RLock 사용)
// status가 done인 task 삭제
// timeout인 task 삭제
// src의 status가 OK가 아닌 task 삭제
// dest의 status가 OK가 아닌 task 삭제
func CleanTask(tasks *Tasks) {

	tl := make([]Task, 0, 10)
	curtasks := tasks.GetTaskList()

	tasker.Debugf("clean task, current task count(%d)", len(curtasks))
	for _, task := range curtasks {

		if task.Status == DONE {
			tl = append(tl, task)
			tasker.Infof("[%d] with stauts done, delete task(%s) ", task.ID, task)
			continue
		}

		diff := time.Since(time.Unix(int64(task.Mtime), 0))
		if diff > taskTimeout {
			task.Status = TIMEOUT
			tl = append(tl, task)
			tasker.Infof("[%d] delete timeout task(%s)", task.ID, task)
			continue
		}

		srcstatus, srcfound := SrcServers.getHostStatus(task.SrcAddr)
		if !srcfound || srcstatus != OK {
			tl = append(tl, task)
			tasker.Infof("[%d] with srcHost's status NOT OK, delete task(%s)", task.ID, task)
			continue
		}

		dststatus, dstfound := DstServers.getHostStatus(task.DstAddr)
		if !dstfound || dststatus != OK {
			tl = append(tl, task)
			tasker.Infof("[%d] with dstHost's status NOT OK, delete task(%s)", task.ID, task)
			continue
		}
	}

	for _, t := range tl {
		tasks.DeleteTask(t.ID)
	}
}

// CollectRemoteFileList is to get file list on remote servers
func CollectRemoteFileList(destList *DstHosts, remoteFileSet map[string]int) {

	for _, dest := range *destList {
		fl := make([]string, 0, 10000)
		err := common.GetRemoteFileList(&dest.Host, &fl)
		if err != nil {
			tasker.Errorf("[%s] fail to get remote file list, error(%s)", dest, err.Error())
			continue
		}

		tasker.Debugf("[%s] get file list", dest)
		for _, file := range fl {
			remoteFileSet[file]++
		}
	}
}

// SetSleepSec :
func SetSleepSec(s uint) {
	tasker.Infof("set sleepSec(%d)", s)
	sleepSec = s
}

// SetTaskTimeout is to set timeout for task
func SetTaskTimeout(t time.Duration) error {

	if t < 0 {
		return errors.New("can not use negative value")
	}

	taskTimeout = t
	tasker.Infof("set task timeout(%s)", taskTimeout)
	return nil
}

// SetAdvPrefix :
func SetAdvPrefix(p []string) {
	tasker.Debugf("set adv prefixes(%v)", p)
	advPrefixes = p
}

// SetHitcountHistoryFile :
func SetHitcountHistoryFile(f string) {
	tasker.Debugf("set hitcountHistory file path(%s)", f)
	hitcountHistoryFile = f
}

// SetGradeInfoFile :
func SetGradeInfoFile(f string) {
	tasker.Debugf("set gradeInfo file path(%s)", f)
	gradeInfoFile = f
}

// SetTaskCopySpeed :
func SetTaskCopySpeed(speed string) {
	tasker.Debugf("set task copy speed(%s)", speed)
	taskCopySpeed = speed
}

// setHostStatus :
// 각 src host의 heartbeat 결과가 Status에 저장됨
func (srcs *SrcHosts) setHostStatus() {
	for _, src := range *srcs {
		h, ok := heartbeater.Get(src.Addr)
		if ok {
			if h.Status == heartbeater.OK {
				src.Status = OK
				tasker.Debugf("[%s] heartbeat", src)
			} else {
				src.Status = NOTOK
				tasker.Debugf("[%s] fail to heartbeat", src)
			}
		} else {
			src.Status = NOTOK
			tasker.Debugf("[%s] fail to heartbeat", src)
		}
	}
}

// setSelected :
// tasks에서 사용 중이지 않은 src 의 selected 상태를 false로 변경
// tasks에서 사용 중인 src 의 selected 상태를 true 변경
// lock 없이 직접 TaskMap traversal 하던 코드
// 	-> copy해서 사용하도록 수정(copy 할 때 RLock 사용)
// 1. unset selected flag (true->false)
// 2. task queue 에 있는 src ip는 할당된 상태로 변경
func (srcs *SrcHosts) setSelected() int {
	for _, src := range *srcs {
		src.selected = false
	}
	selectedCount := 0
	curtasks := tasks.GetTaskList()
	for _, task := range curtasks {
		for _, src := range *srcs {
			if src.Addr == task.SrcAddr {
				src.selected = true
				selectedCount++
			}
		}
	}
	return selectedCount
}

// source 목록에 파라미터로 받은 addr 값의 source가 있는 경우 status 값 반환
// source의 status 와 해당 addr 의 source가 srcs 목록에 있는 지 여부 반환
func (srcs *SrcHosts) getHostStatus(addr string) (HostStatus, bool) {
	for _, src := range *srcs {
		if src.Addr == addr {
			tasker.Debugf("[%s] get src status(%d)", src, src.Status)
			return src.Status, true
		}
	}
	tasker.Debugf("[%s] fail to get status, not found", addr)
	return OK, false
}

func (srcs *SrcHosts) getSelectableCount() int {
	cnt := 0
	for _, src := range *srcs {
		if src.Status == OK && !src.selected {
			cnt++
		}
	}
	return cnt
}

// setSelected :
// tasks에서 사용 중이지 않은 dest 의 selected 상태를 false로 변경
// tasks에서 사용 중인 dest 의 selected 상태를 true 변경
func (dsts *DstHosts) setSelected() int {
	for _, dst := range *dsts {
		dst.selected = false
	}
	selectedCount := 0
	curtasks := tasks.GetTaskList()
	for _, task := range curtasks {
		for _, dst := range *dsts {
			if dst.Addr == task.DstAddr {
				dst.selected = true
				selectedCount++
			}
		}
	}
	return selectedCount
}

// setHostStatus :
// 각 dest host의 heartbeat 결과가 Status에 저장됨
func (dsts *DstHosts) setHostStatus() {
	for _, dst := range *dsts {
		h, ok := heartbeater.Get(dst.Addr)
		if ok {
			if h.Status == heartbeater.OK {
				dst.Status = OK
				tasker.Debugf("[%s] heartbeat", dst)
			} else {
				dst.Status = NOTOK
				tasker.Debugf("[%s] fail to heartbeat", dst)
			}
		} else {
			dst.Status = NOTOK
			tasker.Debugf("[%s] fail to heartbeat", dst)
		}
	}
}

// getSelectableList :
// status가 OK이고,
// task 에 할당되지 않은 sort된 dest list 반환
func (dsts *DstHosts) getSelectableList() (rl []DstHost) {
	for _, dst := range *dsts {
		if dst.Status == OK && !dst.selected {
			rl = append(rl, *dst)
			tasker.Debugf("[%s] added in selectable dst list", dst)
		}
	}
	sort.Slice(rl, func(i, j int) bool {
		return rl[i].Addr > rl[j].Addr
	})
	return rl
}

// destination 목록에 파라미터로 받은 addr 값의 destination이 있는 경우 status 값 반환
// destination의 status 와 해당 addr 의 destination이 srcs 목록에 있는 지 여부 반환
func (dsts *DstHosts) getHostStatus(addr string) (HostStatus, bool) {
	for _, dst := range *dsts {
		if dst.Addr == addr {
			tasker.Debugf("[%s] get dst status(%d)", dst, dst.Status)
			return dst.Status, true
		}
	}
	tasker.Debugf("[%s] fail to get status, not found", addr)
	return OK, false
}

// SetIgnorePrefixes
func SetIgnorePrefixes(p []string) {
	tasker.Debugf("set ignore prefixes(%v)", p)
	ignorePrefixes = p
}
