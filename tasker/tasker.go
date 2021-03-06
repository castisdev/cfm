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

var tskrlogger common.MLogger

func init() {
	tskrlogger = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "tasker"}
}

type FileMetaPtr *common.FileMeta

// map: file name -> *common.FileMeta
type FileMetaPtrMap map[string]*common.FileMeta

// map: common.Host.addr -> (map : filename -> *common.FileMeta)
type ServerFileMetaPtrMap map[string]FileMetaPtrMap

// map : file name -> Hits
type Freq uint64
type FileFreqMap map[string]Freq

// var sleepSec uint
// var taskTimeout time.Duration

// HostStatus : 서버 heartbeat 상태
type HostStatus int

// task HostStatus const
const (
	NOTOK HostStatus = iota
	OK
)

func (s HostStatus) String() string {
	m := map[HostStatus]string{
		NOTOK: "notok",
		OK:    "ok",
	}
	return m[s]
}

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

// getAllHostStatus :
// 각 src host의 heartbeat 결과가 Status에 저장됨
func (srcs *SrcHosts) getAllHostStatus() {
	for _, src := range *srcs {
		h, ok := heartbeater.Get(src.Addr)
		if ok {
			if h.Status == heartbeater.OK {
				src.Status = OK
				tskrlogger.Debugf("[%s] src, checked heartbeat ok", src)
			} else {
				src.Status = NOTOK
				tskrlogger.Debugf("[%s] src, checked heartbeat not ok", src)
			}
		} else {
			src.Status = NOTOK
			tskrlogger.Debugf("[%s] src, failed to check heartbeat", src)
		}
	}
}

// setSelected :
// src 의 selected 상태를 false로 reset하고,
//
// task list 를 검사해서
// task에서 사용 중인 src 의 selected 상태를 true 변경
func (srcs *SrcHosts) setSelected(curtasks []Task) {
	for _, src := range *srcs {
		src.selected = false
	}
	for _, task := range curtasks {
		for _, src := range *srcs {
			if src.Addr == task.SrcAddr {
				src.selected = true
			}
		}
	}
}

// source 목록에 파라미터로 받은 addr 값의 source가 있는 경우 status 값 반환
// source의 status 와 해당 addr 의 source가 srcs 목록에 있는 지 여부 반환
func (srcs *SrcHosts) getHostStatus(addr string) (HostStatus, bool) {
	for _, src := range *srcs {
		if src.Addr == addr {
			return src.Status, true
		}
	}
	return NOTOK, false
}

// 상태가 OK 이고, selected가 false 인 src 서버 개수 반환
func (srcs *SrcHosts) getSelectableCount() int {
	cnt := 0
	for _, src := range *srcs {
		if src.Status == OK && !src.selected {
			cnt++
		}
	}
	return cnt
}

// selectSourceServer
//
// selected 가 false 이고
//
// status 가 OK 인 source 선택되고, selected가 true로 update
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

// getAllHostStatus :
// 각 dest host의 heartbeat 결과가 Status에 저장됨
func (dsts *DstHosts) getAllHostStatus() {
	for _, dst := range *dsts {
		h, ok := heartbeater.Get(dst.Addr)
		if ok {
			if h.Status == heartbeater.OK {
				dst.Status = OK
				tskrlogger.Debugf("[%s] dst, checked heartbeat ok", dst)
			} else {
				dst.Status = NOTOK
				tskrlogger.Debugf("[%s] dst, checked heartbeat not ok", dst)
			}
		} else {
			dst.Status = NOTOK
			tskrlogger.Debugf("[%s] dst, failed to check heartbeat", dst)
		}
	}
}

// setSelected :
// dest 의 selected 상태를 false로 reset하고,
//
// task list 를 검사해서
// task에서 사용 중인 dest 의 selected 상태를 true 변경
func (dsts *DstHosts) setSelected(curtasks []Task) {
	for _, dst := range *dsts {
		dst.selected = false
	}
	for _, task := range curtasks {
		for _, dst := range *dsts {
			if dst.Addr == task.DstAddr {
				dst.selected = true
			}
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
			return dst.Status, true
		}
	}
	return NOTOK, false
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

// DstServers : 파일 배포 대상 서버 리스트
// SrcServers : 배포할 파일들을 갖고 있는 서버 리스트
// Tail :: LB EventLog 를 tailing 하며 SAN 에서 Hit 되는 파일 목록 추출
// SourcePath : 배포할 파일이 존재하는 경로
type Tasker struct {
	sleepSec            uint
	taskTimeout         time.Duration
	SourcePath          *common.SourceDirs
	SrcServers          *SrcHosts
	DstServers          *DstHosts
	Tail                *tailer.Tailer
	tasks               *Tasks
	gradeInfoFile       string
	hitcountHistoryFile string
	taskCopySpeed       string
	ignorePrefixes      []string
}

func NewTasker() *Tasker {
	return &Tasker{
		sleepSec:    60,
		taskTimeout: 30 * time.Minute,
		SourcePath:  common.NewSourceDirs(),
		SrcServers:  NewSrcHosts(),
		DstServers:  NewDstHosts(),
		Tail:        tailer.NewTailer(),
		tasks:       NewTasks(),
	}
}

func NewTaskerWith(
	gradeInfoFile, hitcountHistoryFile string,
	sleepSec uint, taskTimeout time.Duration,
	taskCopySpeed string,
	ignorePrefixes []string) *Tasker {
	return &Tasker{
		sleepSec:            sleepSec,
		taskTimeout:         taskTimeout,
		SourcePath:          common.NewSourceDirs(),
		SrcServers:          NewSrcHosts(),
		DstServers:          NewDstHosts(),
		Tail:                tailer.NewTailer(),
		tasks:               NewTasks(),
		gradeInfoFile:       gradeInfoFile,
		hitcountHistoryFile: hitcountHistoryFile,
		taskCopySpeed:       taskCopySpeed,
		ignorePrefixes:      ignorePrefixes,
	}
}

// Tasks is to get global task list structure's addr
func (tskr *Tasker) Tasks() *Tasks {
	return tskr.tasks
}

// SetTaskTimeout is to set timeout for task
func (tskr *Tasker) SetTaskTimeout(t time.Duration) error {

	if t < 0 {
		return errors.New("can not use negative value")
	}

	tskr.taskTimeout = t
	tskrlogger.Infof("set task timeout(%s)", tskr.taskTimeout)
	return nil
}

// SetHitcountHistoryFile :
func (tskr *Tasker) SetHitcountHistoryFile(f string) {
	tskr.hitcountHistoryFile = f
	tskrlogger.Infof("set hitcountHistory file path(%s)", f)
}

// SetGradeInfoFile :
func (tskr *Tasker) SetGradeInfoFile(f string) {
	tskr.gradeInfoFile = f
	tskrlogger.Infof("set gradeInfo file path(%s)", f)
}

// SetTaskCopySpeed :
func (tskr *Tasker) SetTaskCopySpeed(speed string) {
	tskr.taskCopySpeed = speed
	tskrlogger.Infof("set task copy speed(%s)", speed)
}

// SetSleepSec :
func (tskr *Tasker) SetSleepSec(s uint) {
	tskr.sleepSec = s
	tskrlogger.Infof("set sleepSec(%d)", s)
}

// SetIgnorePrefixes
func (tskr *Tasker) SetIgnorePrefixes(p []string) {
	tskr.ignorePrefixes = p
	tskrlogger.Infof("set ignore prefixes(%v)", p)
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
func (tskr *Tasker) InitTasks() {
	tskr.tasks.LoadTasks()
}

// RunForever is to run tasker as go routine
func (tskr *Tasker) RunForever() {
	for {
		tskr.run(time.Now())
		time.Sleep(time.Second * time.Duration(tskr.sleepSec))
	}
}

// run :
func (tskr *Tasker) run(basetm time.Time) error {
	tskrlogger.Infof("started tasker process")
	defer logElapased("ended tasker process", common.Start())

	destcount := len(*tskr.DstServers)
	if destcount == 0 {
		tskrlogger.Errorf("endded, the number of the dest srvers is 0")
		return errors.New("endded, the number of the dest srvers is 0")
	}
	dstIPMap := make(map[string]int)
	for _, dst := range *tskr.DstServers {
		dstIPMap[dst.IP]++
	}
	fileMetaMap := make(map[string]*common.FileMeta)
	duplicatedFileMap := make(map[string]*common.FileMeta)
	risingHitFileMap := make(map[string]int)

	// 전체 파일 정보 목록 구하기
	// 파일 이름, 파일 등급, file size, 파일 위치 정보 구하기
	// 서버 별로 구할 필요 없음
	// 구하지 못하는 경우, 다음 번 주기로 넘어감
	est := common.Start()
	err := common.MakeAllFileMetas(tskr.gradeInfoFile, tskr.hitcountHistoryFile,
		fileMetaMap, dstIPMap, duplicatedFileMap)

	if err != nil {
		tskrlogger.Errorf("failed to make file metas, error(%s)", err.Error())
		return err
	}
	tskrlogger.Infof("made file metas(name, grade, size, servers), time(%s)", common.Elapsed(est))

	// 급 hit 상승 파일 목록 구하기
	// LB EventLog 에서 특정 IP 에 할당된 파일 목록 추출
	tskr.Tail.Tail(basetm, &risingHitFileMap)

	tskr.runWithInfo(fileMetaMap, risingHitFileMap)

	return nil
}

func (tskr *Tasker) RunWithInfo(
	fileMetaMap FileMetaPtrMap,
	risingHitFileMap map[string]int) {
	tskr.runWithInfo(fileMetaMap, risingHitFileMap)
}

// runWithInfo :
func (tskr *Tasker) runWithInfo(
	fileMetaMap FileMetaPtrMap,
	risingHitFileMap map[string]int) {

	tskrlogger.Infof("started tasker inner process")
	defer logElapased("ended tasker inner process", common.Start())

	// Src heartbeat 검사를 가져옴
	tskr.SrcServers.getAllHostStatus()
	// Dest heartbeat 검사를 가져옴
	tskr.DstServers.getAllHostStatus()

	curtasks := tskr.tasks.GetTaskList()

	// task 정리 후 새로운 task 를다시 구함
	// - DONE task 정리
	// - TIMEOUT 계산해서 TIMEOUT된 task 정리
	// - Src 또는 Dest의 heartbeat 답을 구하지 못한 task 정리
	curtasks = tskr.cleanTask(curtasks)

	// src 할당 상태를 false로 변경
	// task 에서 사용 중인 src 할당 상태를 true로 변경
	// 배포에 할당 가능한 src 서버 개수 구하기
	// 	- status가 OK 이고,
	// 	- 아직 배포 task 에 할당안된 경우 할당 가능
	// 배포에 할당 가능한 src 서버가 없으면 다음 주기로 넘어감
	srccnt := tskr.getAvailableSrcServerCount(curtasks)
	if srccnt == 0 {
		tskrlogger.Infof("no src server is available")
		return
	}

	// dest 할당 상태를 false로 변경
	// task 에서 사용 중인 dest 할당 상태를 true로 변경
	// destination ip 를 round robin 으로 선택하기 위한 ring 생성
	// 배포에 할당 가능한 dest 서버로 ring 생성
	// 	- status가 OK 이고,
	// 	- 아직 배포 task 에 할당안된 경우 할당 가능
	// 배포에 할당 가능한 dest 서버가 없으면 다음 주기로 넘어감
	dstRing := tskr.getAvailableDstServerRing(curtasks)
	if dstRing == nil {
		tskrlogger.Infof("no dst server is available")
		return
	}

	// 모든 dest 서버의 파일 목록 수집
	serverfiles := make(FileFreqMap)
	collectRemoteFileList(tskr.DstServers, serverfiles)

	// 배포 대상이 되는 파일 리스트 만들어서 배포 task 만들기

	// 급 상승 Hit 수가 많은 순서대로 정렬
	// 급 상승 Hit 수가 같으면 높은 등급 순서로 정렬 (가장 높은 등급:1)
	sortedfms := getSortedFileMetaListForTask(fileMetaMap, risingHitFileMap)

	// - grade info, hitcount history 파일에서 file meta를 구할 수 없는 파일 제외
	// - source path에 없는 파일 제외 (SAN 에 없는 파일 제외)
	// - dest 서버에 이미 있는 파일 제외
	// - ignore.prefix로 시작하는 파일 제외 (광고 파일 제외)
	// - task 에 이미 있는 파일 제외
	usedtaskfiles := getFilesInTasks(curtasks)
	for _, fmm := range sortedfms {

		if !tskr.updateFileMetaForSrcFilePath(fmm) {
			tskrlogger.Debugf("ignored by not.found.in.the.source.paths, file(%s)", *fmm)
			continue
		}
		if !tskr.checkForTask(fmm, usedtaskfiles, serverfiles) {
			continue
		}

		// src 서버 선택
		// 	- status가 OK 이고,
		// 	- 아직 배포 task 에 할당안된 경우
		src, exists := tskr.SrcServers.selectSourceServer()
		if exists != true {
			tskrlogger.Debugf("stopped making task, no src is available")
			break
		}

		// task 생성
		dst := DstHost(dstRing.Value.(DstHost))
		t := tskr.tasks.CreateTask(&Task{
			FilePath:  fmm.SrcFilePath,
			FileName:  fmm.Name,
			SrcIP:     src.IP,
			DstIP:     dst.IP,
			Grade:     fmm.Grade,
			CopySpeed: tskr.taskCopySpeed,
			SrcAddr:   src.Addr,
			DstAddr:   dst.Addr,
		})
		dstRing = dstRing.Next()

		if fmm.RisingHit > 0 {
			tskrlogger.Infof("[%d] created task(%s) for risingHit(%d), file(%s)",
				t.ID, t, fmm.RisingHit, *fmm)
		} else {
			tskrlogger.Infof("[%d] created task(%s) for grade(%d), file(%s)",
				t.ID, t, fmm.Grade, *fmm)
		}
	}
}

// cleanTask :
//
// 특정 조건의 task를 tasks(전역변수)에서 삭제
//
// status가 done인 task 삭제
//
// timeout인 task 삭제 : duration(현재 time - task.Mtime)이 taskTimeout보다 큰 경우
//
// src의 status가 OK가 아닌 task 삭제
//
// dest의 status가 OK가 아닌 task 삭제
//
// 작업 후의 task list를 retrun
func (tskr *Tasker) cleanTask(curtasks []Task) []Task {

	tl := make([]int64, 0, len(curtasks))

	for _, task := range curtasks {

		if task.Status == DONE {
			tl = append(tl, task.ID)
			tskrlogger.Infof("[%d] with stauts done, deleted task(%s) ", task.ID, task)
			continue
		}

		diff := time.Since(time.Unix(int64(task.Mtime), 0))
		if diff > tskr.taskTimeout {
			tl = append(tl, task.ID)
			tskrlogger.Infof("[%d] with timeout, deleted task(%s)", task.ID, task)
			continue
		}

		srcstatus, srcfound := tskr.SrcServers.getHostStatus(task.SrcAddr)
		if srcfound {
			tskrlogger.Debugf("[%d][%s] got src status(%s)", task.ID, task.SrcAddr, srcstatus)
		} else {
			tskrlogger.Debugf("[%d][%s] failed to get src status, not found", task.ID, task.SrcAddr)
		}
		if !srcfound || srcstatus != OK {
			tl = append(tl, task.ID)
			tskrlogger.Infof("[%d] with srcHost's status NOTOK, delete task(%s)", task.ID, task)
			continue
		}

		dststatus, dstfound := tskr.DstServers.getHostStatus(task.DstAddr)
		if dstfound {
			tskrlogger.Debugf("[%d][%s] got dst status(%s)", task.ID, task.DstAddr, dststatus)
		} else {
			tskrlogger.Debugf("[%d][%s] failed to get dst status, not found", task.ID, task.DstAddr)
		}
		if !dstfound || dststatus != OK {
			tl = append(tl, task.ID)
			tskrlogger.Infof("[%d] with dstHost's status NOTOK, deleted task(%s)", task.ID, task)
			continue
		}
	}

	tskr.tasks.DeleteTasks(tl)

	return tskr.tasks.GetTaskList()
}

// task list 를 검사해서
//
// src server의 selected 상태 update 하고
//
// task에서 사용하지 않고, status가 ok인
//
// src server 개수 return
func (tskr *Tasker) getAvailableSrcServerCount(curtasks []Task) int {
	tskr.SrcServers.setSelected(curtasks)
	return tskr.SrcServers.getSelectableCount()
}

// task list 를 검사해서
//
// dst server의 selected 상태 update 하고
//
// task에서 사용하지 않고, status가 ok인
//
// sort된 dst server list 를 Ring 으로 만들어서 return
//
// dst server가 없는 경우, nil return 됨
func (tskr *Tasker) getAvailableDstServerRing(curtasks []Task) *ring.Ring {
	dstlist := tskr.getAvailableDstServerList(curtasks)

	dstRing := ring.New(len(dstlist))
	for _, d := range dstlist {
		dstRing.Value = d
		dstRing = dstRing.Next()
		tskrlogger.Debugf("[%s] added to available destination servers", d.Addr)
	}
	return dstRing
}

// task list 를 검사해서
//
// dst server의 selected 상태 update 하고
//
// task에서 사용하지 않고, status가 ok인
//
// sort된 dst server list return
func (tskr *Tasker) getAvailableDstServerList(curtasks []Task) []DstHost {
	tskr.DstServers.setSelected(curtasks)
	return tskr.DstServers.getSelectableList()
}

// collectRemoteFileList is to get file list on remote servers
func collectRemoteFileList(destList *DstHosts, remoteFiles FileFreqMap) {

	for _, dest := range *destList {
		fl := make([]string, 0, 10000)
		err := common.GetRemoteFileList(&dest.Host, &fl)
		if err != nil {
			tskrlogger.Errorf("[%s] failed to get dst server file list, error(%s)", dest, err.Error())
			continue
		}

		tskrlogger.Debugf("[%s] got file list", dest)
		for _, file := range fl {
			remoteFiles[file]++
		}
	}
}

// getSortedFileMetaListForTask
//
// risinghits file의 정보를 file meta 정보에 반영
//
//  - file meta 에 없으면 무시됨
//
// 급 상승 Hit 수가 많은 순서대로 정렬
//
// 급 상승 Hit 수가 같으면 높은 등급 순서로 정렬 (가장 높은 등급:1)
func getSortedFileMetaListForTask(allfmm FileMetaPtrMap,
	risinghits map[string]int) []FileMetaPtr {
	updateFileMetasForRisingHitsFiles(allfmm, risinghits)
	taskfilelist := make([]FileMetaPtr, 0, len(allfmm))
	for _, fmm := range allfmm {
		taskfilelist = append(taskfilelist, fmm)
	}
	// Risinghit 값이 다르면 Risinghit값이 높은 순으로
	// Risinghit 값이 같으면
	// 	Grade 값이 작은(높은 등급) 순으로 정렬 순서로 정렬
	sort.Slice(taskfilelist, func(i, j int) bool {
		if taskfilelist[i].RisingHit != taskfilelist[j].RisingHit {
			return taskfilelist[i].RisingHit > taskfilelist[j].RisingHit
		} else {
			return taskfilelist[i].Grade < taskfilelist[j].Grade
		}
	})

	return taskfilelist
}

func getFilesInTasks(curtasks []Task) FileFreqMap {
	filenames := make(FileFreqMap)
	for _, task := range curtasks {
		filenames[task.FileName]++
	}
	return filenames
}

// updateFileMetasForRisingHitsFiles :
//
// - risinghits map의 file들의 file meta찾아서
// 해당 file meta의 rising hit 값을 update 함
//
// - 일반 file meta의 rising hit 값은 0임
//
// - file meta가 없는 rising hit file은 skip
func updateFileMetasForRisingHitsFiles(allfmm FileMetaPtrMap,
	risinghits map[string]int) {

	for rhfn, hits := range risinghits {
		fmm, ok := allfmm[rhfn]
		if ok {
			fmm.RisingHit = hits
		} else {
			// 전체 file meta 에 없다면 제외
			tskrlogger.Debugf("ignored by not.found.in.the.all.file.metas, file(%s)", rhfn)
		}
	}
}

// file 이 source path 에 있는 지 검사하고 있으면
// file meta의 SrcPath에 update
func (tskr *Tasker) updateFileMetaForSrcFilePath(fmm *common.FileMeta) bool {
	srcFilePath, exists := tskr.SourcePath.IsExistOnSource(fmm.Name)
	if !exists {
		return false
	}
	fmm.SrcFilePath = srcFilePath
	return true
}

// - 이미 배포 대상이 되는 파일은 제외
//
// - 서버에 이미 있는 파일은 제외
//
// - ignore.prefix로 시작하는 파일(광고 파일)은 제외
//
// - source path 값이 비어있으면 제외
func (tskr *Tasker) checkForTask(fmm *common.FileMeta,
	taskfiles FileFreqMap,
	serverfiles FileFreqMap) bool {

	fn := fmm.Name

	// - hitcount.history file에서 구한 서버위치 정보로
	// 어떤 서버인가 이미 있는 파일은 제외
	if fmm.ServerCount > 0 {
		tskrlogger.Debugf("ignored by found.in.the.servers, file(%s)", fmm)
		return false
	}

	// 소스 directory에 없는 파일 제외(SAN 에 없는 파일 제외)
	if fmm.SrcFilePath == "" {
		tskrlogger.Debugf("ignored by not.found.in.the.source.paths, file(%s)", fmm)
		return false
	}

	// ignore.prefix 로 시작하는 파일 제외(광고 파일)
	if common.IsPrefix(fn, tskr.ignorePrefixes) {
		tskrlogger.Debugf("ignored by ignore.prefix, file(%s)", fmm)
		return false
	}

	// 이미 배포 task에 사용되는 파일 제외
	if n, using := taskfiles[fn]; using && n > 0 {
		tskrlogger.Debugf("ignored by found.in.the.tasks, file(%s)", fmm)
		return false
	}

	// - 서버별로 조사한 파일 정보로
	// 어떤 서버인가 이미 이미 있는 파일은 제외
	if n, using := serverfiles[fn]; using && n > 0 {
		tskrlogger.Debugf("ignored by found.in.the.servers, file(%s)", fmm)
		return false
	}

	return true
}

func logElapased(message string, start time.Time) {
	tskrlogger.Infof("%s, time(%s)", message, common.Elapsed(start))
}
