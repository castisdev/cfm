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

type FileMetaPtr *common.FileMeta

// map: file name -> *common.FileMeta
type FileMetaPtrMap map[string]*common.FileMeta

// map: common.Host.addr -> (map : filename -> *common.FileMeta)
type ServerFileMetaPtrMap map[string]FileMetaPtrMap

// map : file name -> Hits
type Freq uint64
type FileFreqMap map[string]Freq

var sleepSec uint
var taskTimeout time.Duration

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

// DstServers : 파일 배포 대상 서버 리스트
var DstServers *DstHosts

// SrcServers : 배포할 파일들을 갖고 있는 서버 리스트
var SrcServers *SrcHosts

// Tail :: LB EventLog 를 tailing 하며 SAN 에서 Hit 되는 파일 목록 추출
var Tail *tailer.Tailer

var tasks *Tasks
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

// SetTaskTimeout is to set timeout for task
func SetTaskTimeout(t time.Duration) error {

	if t < 0 {
		return errors.New("can not use negative value")
	}

	taskTimeout = t
	tasker.Infof("set task timeout(%s)", taskTimeout)
	return nil
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

// SetSleepSec :
func SetSleepSec(s uint) {
	tasker.Infof("set sleepSec(%d)", s)
	sleepSec = s
}

// SetIgnorePrefixes
func SetIgnorePrefixes(p []string) {
	tasker.Debugf("set ignore prefixes(%v)", p)
	ignorePrefixes = p
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
	for {
		run(time.Now())
		time.Sleep(time.Second * time.Duration(sleepSec))
	}
}

// run :
func run(basetm time.Time) error {
	tasker.Infof("start tasker process")
	defer logElapased("end tasker process", common.Start())

	destcount := len(*DstServers)
	if destcount == 0 {
		tasker.Errorf("tasker endded, the number of the dest srvers is 0")
		return errors.New("tasker endded, the number of the dest srvers is 0")
	}
	fileMetaMap := make(map[string]*common.FileMeta)
	duplicatedFileMap := make(map[string]*common.FileMeta)
	risingHitFileMap := make(map[string]int)

	// 4. 파일 등급 list 생성
	// gradeinfoFile 과 hitcountHistoryFile로 file meta list 생성
	est := common.Start()
	err := common.MakeAllFileMetas(gradeInfoFile, hitcountHistoryFile,
		fileMetaMap, map[string]int{}, duplicatedFileMap)

	if err != nil {
		tasker.Errorf("fail to make file metas, error(%s)", err.Error())
		return err
	}
	tasker.Infof("make file metas(name, grade, size, servers), time(%s)", common.Elapsed(est))

	// 급 hit 상승 파일 목록 구하기
	// LB EventLog 에서 특정 IP 에 할당된 파일 목록 추출
	Tail.Tail(basetm, &risingHitFileMap)

	runWithInfo(fileMetaMap, risingHitFileMap)

	return nil
}

// runWithInfo :
func runWithInfo(
	fileMetaMap FileMetaPtrMap,
	risingHitFileMap map[string]int) {

	tasker.Infof("start tasker inner process")
	defer logElapased("end tasker inner process", common.Start())

	// Src heartbeat 검사를 가져옴
	SrcServers.getAllHostStatus()
	// Dest heartbeat 검사를 가져옴
	DstServers.getAllHostStatus()

	curtasks := tasks.GetTaskList()

	// task 정리 후 새로운 task 를다시 구한다.
	curtasks = cleanTask(curtasks)

	// unset selected flag (true->false)
	// task queue 에 있는 src를 할당된 상태로 변경
	// status가 OK이고, 아직 배포 task 에 할당안된 source가 없으면
	// task를 더이상 만들지 않음
	srccnt := getAvailableSrcServerCount(curtasks)
	if srccnt == 0 {
		tasker.Debugf("no src is available")
		return
	}

	// task queue 에 있는 dest를 할당된 상태로 변경
	// destination ip 를 round robin 으로 선택하기 위한 ring 생성
	// status가 OK이고,
	// 아직 배포 task 에 할당안된 dest가 없으면
	// task를 더이상 만들지 않음
	dstRing := getAvailableDstServerRing(curtasks)
	if dstRing == nil {
		tasker.Debugf("no dst is available")
		return
	}

	// 모든 서버의 파일 리스트 수집
	remoteFileSet := make(FileFreqMap)
	collectRemoteFileList(DstServers, remoteFileSet)

	// 높은 등급 순서로 정렬하기 위해 빈 Slice 생성 하고 배포 대상이 되는파일 return
	// ignore.prefix로 시작하는 파일 배포 제외 : 광고 파일 배포 제외
	// .grade.info, .hitcount.history 에서 file meta를 구할 수 없는 파일은 배포에서 제외
	// task 에 이미 있는 파일 제외
	// source path에 없는 파일 제외 :SAN 에 없는 파일 배포에서 제외
	//   source path에 있는 파일이면, source path를 file meta에 update
	// 급 상승 Hit 수가 많은 순서대로 정렬
	// 높은 등급 순서로 정렬 (가장 높은 등급:1)
	sortedFileList := getFileMetaListForTask(fileMetaMap, risingHitFileMap,
		curtasks, remoteFileSet)

	for _, file := range sortedFileList {

		// src ip 선택, 없으면 loop 종료
		src, exists := SrcServers.selectSourceServer()
		if exists != true {
			tasker.Debugf("stop making task, no src is available")
			break
		}

		// task 생성
		dst := DstHost(dstRing.Value.(DstHost))
		t := tasks.CreateTask(&Task{
			FilePath:  file.SrcFilePath,
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
			tasker.Infof("[%d] create task(%s) for risingHit(%d), file(%s)",
				t.ID, t, file.RisingHit, *file)
		} else {
			tasker.Infof("[%d] create task(%s) for grade(%d), file(%s)",
				t.ID, t, file.Grade, *file)
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
func cleanTask(curtasks []Task) []Task {

	tl := make([]int64, 0, len(curtasks))

	for _, task := range curtasks {

		if task.Status == DONE {
			tl = append(tl, task.ID)
			tasker.Infof("[%d] with stauts done, delete task(%s) ", task.ID, task)
			continue
		}

		diff := time.Since(time.Unix(int64(task.Mtime), 0))
		if diff > taskTimeout {
			tl = append(tl, task.ID)
			tasker.Infof("[%d] with timeout, delete task(%s)", task.ID, task)
			continue
		}

		srcstatus, srcfound := SrcServers.getHostStatus(task.SrcAddr)
		if srcfound {
			tasker.Debugf("[%d][%s] get src status(%s)", task.ID, task.SrcAddr, srcstatus)
		} else {
			tasker.Debugf("[%d][%s] fail to get src status, not found", task.ID, task.SrcAddr)
		}
		if !srcfound || srcstatus != OK {
			tl = append(tl, task.ID)
			tasker.Infof("[%d] with srcHost's status NOTOK, delete task(%s)", task.ID, task)
			continue
		}

		dststatus, dstfound := DstServers.getHostStatus(task.DstAddr)
		if dstfound {
			tasker.Debugf("[%d][%s] get dst status(%s)", task.ID, task.DstAddr, dststatus)
		} else {
			tasker.Debugf("[%d][%s] fail to get dst status, not found", task.ID, task.DstAddr)
		}
		if !dstfound || dststatus != OK {
			tl = append(tl, task.ID)
			tasker.Infof("[%d] with dstHost's status NOTOK, delete task(%s)", task.ID, task)
			continue
		}
	}

	tasks.DeleteTasks(tl)

	return tasks.GetTaskList()
}

// task list 를 검사해서
//
// src server의 selected 상태 update 하고
//
// task에서 사용하지 않고, status가 ok인
//
// src server 개수 return
func getAvailableSrcServerCount(curtasks []Task) int {
	SrcServers.setSelected(curtasks)
	return SrcServers.getSelectableCount()
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
func getAvailableDstServerRing(curtasks []Task) *ring.Ring {
	dstlist := getAvailableDstServerList(curtasks)

	dstRing := ring.New(len(dstlist))
	for _, d := range dstlist {
		dstRing.Value = d
		dstRing = dstRing.Next()
		tasker.Debugf("[%s] added to available destination servers", d.Addr)
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
func getAvailableDstServerList(curtasks []Task) []DstHost {
	DstServers.setSelected(curtasks)
	return DstServers.getSelectableList()
}

// getAllHostStatus :
// 각 src host의 heartbeat 결과가 Status에 저장됨
func (srcs *SrcHosts) getAllHostStatus() {
	for _, src := range *srcs {
		h, ok := heartbeater.Get(src.Addr)
		if ok {
			if h.Status == heartbeater.OK {
				src.Status = OK
				tasker.Debugf("[%s] src, heartbeat ok", src)
			} else {
				src.Status = NOTOK
				tasker.Debugf("[%s] src, fail to heartbeat", src)
			}
		} else {
			src.Status = NOTOK
			tasker.Debugf("[%s] src, fail to heartbeat, fail to get heartbeat result", src)
		}
	}
}

// setSelected :
// src 의 selected 상태를 false로 reset하고,
//
// task list 를 검사해서
// task에서 사용 중인 src 의 selected 상태를 true 변경
//
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
				tasker.Debugf("[%s] dst, heartbeat ok", dst)
			} else {
				dst.Status = NOTOK
				tasker.Debugf("[%s] dst, fail to heartbeat", dst)
			}
		} else {
			dst.Status = NOTOK
			tasker.Debugf("[%s] dst, fail to heartbeat, fail to get heartbeat result", dst)
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
			return dst.Status, true
		}
	}
	return NOTOK, false
}

// collectRemoteFileList is to get file list on remote servers
func collectRemoteFileList(destList *DstHosts, remoteFiles FileFreqMap) {

	for _, dest := range *destList {
		fl := make([]string, 0, 10000)
		err := common.GetRemoteFileList(&dest.Host, &fl)
		if err != nil {
			tasker.Errorf("[%s] fail to get remote file list, error(%s)", dest, err.Error())
			continue
		}

		tasker.Debugf("[%s] get file list", dest)
		for _, file := range fl {
			remoteFiles[file]++
		}
	}
}

// getFileMetaListForTask:
//
// 높은 등급 순서로 정렬하기 위해 빈 Slice 생성 하고 배포 대상이 되는파일 return
//
// ignore.prefix로 시작하는 파일 배포 제외 : 광고 파일 배포 제외
//
// .grade.info, .hitcount.history 에서 file meta를 구할 수 없는 파일은 배포에서 제외
//
// task 에 이미 있는 파일 제외
//
// source path에 있는 지 검사해서 없는 파일 제외 : SAN 에 없는 파일 배포에서 제외
//
// 급 상승 Hit 수가 많은 순서대로 정렬
//
// 높은 등급 순서로 정렬 (가장 높은 등급:1)
func getFileMetaListForTask(allfmm FileMetaPtrMap,
	risinghits map[string]int, curtasks []Task,
	serverfiles FileFreqMap) []FileMetaPtr {

	usedtaskfiles := getFilesInTasks(curtasks)
	updateFileMetasForRisingHitsFiles(allfmm, risinghits)

	taskfilelist := make([]FileMetaPtr, 0, len(allfmm))
	for _, fmm := range allfmm {
		if !updateFileMetaForSrcFilePath(fmm) {
			tasker.Debugf("skip file by not.found.in.the.source.paths, file(%s)", fmm)
			continue
		}
		if !checkForTask(fmm, usedtaskfiles, serverfiles) {
			continue
		}
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
// - file meta가 없는 rising hit file은 결과에서 제외됨
func updateFileMetasForRisingHitsFiles(allfmm FileMetaPtrMap,
	risinghits map[string]int) {

	for rhfn, hits := range risinghits {
		fmm, ok := allfmm[rhfn]
		if ok {
			fmm.RisingHit = hits
		} else {
			// 전체 file meta 에 없다면 제외
			tasker.Debugf("skip file by not.found.in.the.all.file.metas, file(%s)", rhfn)
		}
	}
}

// file 이 source path 에 있는 지 검사하고 있으면
// file meta의 SrcPath에 update
func updateFileMetaForSrcFilePath(fmm *common.FileMeta) bool {
	srcFilePath, exists := SourcePath.IsExistOnSource(fmm.Name)
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
func checkForTask(fmm *common.FileMeta,
	taskfiles FileFreqMap,
	serverfiles FileFreqMap) bool {

	fn := fmm.Name

	// ignore.prefix 로 시작하는 파일 제외(광고 파일)
	if common.IsPrefix(fn, ignorePrefixes) {
		tasker.Debugf("skip file by ignore.prefix, file(%s)", fmm)
		return false
	}

	// 이미 배포 task에 사용되는 파일 제외
	if n, using := taskfiles[fn]; using && n > 0 {
		tasker.Debugf("skip file, found in the tasks, file(%s)", fmm)
		return false
	}

	// - 서버에 이미 있는 파일은 제외
	if n, using := serverfiles[fn]; using && n > 0 {
		tasker.Debugf("skip file, found in the servers, file(%s)", fmm)
		return false
	}

	// 소스 directory에 없는 파일 제외(SAN 에 없는 파일 제외)
	if fmm.SrcFilePath == "" {
		tasker.Debugf("skip file by not.found.in.the.source.paths, file(%s)", fmm)
		return false
	}
	return true
}

func logElapased(message string, start time.Time) {
	tasker.Infof("%s, time(%s)", message, common.Elapsed(start))
}
