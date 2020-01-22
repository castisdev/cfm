package remover

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cilog"
)

// map: file name -> *common.FileMeta
type FileMetaPtrMap map[string]*common.FileMeta

// map: common.Host.addr -> (map : filename -> *common.FileMeta)
type ServerFileMetaPtrMap map[string]FileMetaPtrMap

// map : common.Host.addr -> *common.DiskUsage
type ServerDiskUsagePtrMap map[string]*common.DiskUsage

type DServer struct {
	*common.Host
	Du common.DiskUsage
}

// Servers : 파일 삭제 대상 서버 리스트
var Servers *common.Hosts

// SourcePath : 파일 삭제 시 Source 에 없는 파일이면 삭제 대상에서 제외하기 위해 사용
var SourcePath *common.SourceDirs
var diskUsageLimitPercent uint

var hitcountHistoryFile string
var gradeInfoFile string
var sleepSec uint

var ignorePrefixes []string

// Tail :: LB EventLog 를 tailing 하며 SAN 에서 Hit 되는 파일 목록 추출
var Tail *tailer.Tailer

var remover common.MLogger

func init() {
	Servers = common.NewHosts()
	SourcePath = common.NewSourceDirs()

	Tail = tailer.NewTailer()
	sleepSec = 30

	remover = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "remover"}
}

// SetDiskUsageLimitPercent is to set the limitation of disk used size
// min is 0, max is 100
func SetDiskUsageLimitPercent(limit uint) error {
	if limit > 100 {
		return errors.New("disk usage limit percent must be less than 100")
	}

	remover.Infof("set diskUsageLimitPercent(%d)", limit)
	diskUsageLimitPercent = limit

	return nil
}

// DiskUsageLimitPercent : diskUsageLimitPercent 반환
func DiskUsageLimitPercent() uint {
	return diskUsageLimitPercent
}

// SetGradeInfoFile :
func SetGradeInfoFile(f string) {
	remover.Infof("set gradeInfoFile(%s)", f)
	gradeInfoFile = f
}

// SetHitcountHistoryFile :
func SetHitcountHistoryFile(f string) {
	remover.Infof("set hitcountHistoryFile(%s)", f)
	hitcountHistoryFile = f
}

// SetIgnorePrefixes
func SetIgnorePrefixes(p []string) {
	remover.Infof("set ignore prefixes(%v)", p)
	ignorePrefixes = p
}

// SetSleepSec :
func SetSleepSec(s uint) {
	remover.Infof("set sleepSec(%d)", s)
	sleepSec = s
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
	remover.Infof("start remover process")
	defer logElapased("end remover process", common.Start())

	serverIPMap := make(map[string]int)
	for _, server := range *Servers {
		serverIPMap[server.IP]++
	}
	fileMetaMap := make(FileMetaPtrMap)
	duplicatedFileMap := make(FileMetaPtrMap)
	risingHitFileMap := make(map[string]int)
	// 전체 파일 정보 목록 구하기
	// 파일 이름, 파일 등급, file size, 파일 위치 정보 구하기
	// 서버 별로 구할 필요 없음
	// 구하지 못하는 경우, 다음 번 주기로 넘어감
	est := common.Start()
	err := common.MakeAllFileMetas(gradeInfoFile, hitcountHistoryFile,
		fileMetaMap, serverIPMap, duplicatedFileMap)
	if err != nil {
		remover.Errorf("fail to make file metas, error(%s)", err.Error())
		return err
	}
	remover.Infof("make file metas(name, grade, size, servers), time(%s)",
		common.Elapsed(est))

	// 급 hit 상승 파일 목록 구하기
	// LB EventLog 에서 특정 IP 에 할당된 파일 목록 추출
	Tail.Tail(basetm, &risingHitFileMap)

	runWithInfo(fileMetaMap, duplicatedFileMap, risingHitFileMap)

	return nil
}

// runWithInfo :
func runWithInfo(
	fileMetaMap FileMetaPtrMap,
	duplicatedFileMap FileMetaPtrMap,
	risingHitFileMap map[string]int) {

	remover.Infof("start remover inner process")
	defer logElapased("end remover inner process", common.Start())

	// 전체 파일 meta 정보에서
	// 현재 서버(destination)에 있는 파일들의 meta만 골라내서
	// 서버별 파일 meta 정보로 만들기
	serverFileMetaMap := getServerFileMetas(fileMetaMap)

	// destination 서버에 중복 파일 삭제 요청
	requestRemoveDuplicatedFiles(duplicatedFileMap, serverFileMetaMap)

	// disk 용량이 부족한 서버 구하기
	servers := findServersOutOfDiskSpace(Servers)

	// disk 용량이 부족한 server 에 대해서 disk 지워야할 file 목록 만들고,
	// 지워야 할 file 목록이 있는 경우 요청
	requestRemoveFilesForFreeDiskSpace(servers, serverFileMetaMap, risingHitFileMap)
}

// getServerFileMetas :
// 전체 파일 meta map 중에
// 서버 별로 있는 파일에 대한 meta map을 구해서 반환
// 서버 파일 목록을 구하다 에러가 난 경우, 해당 서버의 목록은 비어있게 됨
func getServerFileMetas(allfmm FileMetaPtrMap) ServerFileMetaPtrMap {
	sfmm := make(ServerFileMetaPtrMap)
	for _, server := range *Servers {
		sfms, err := selectFileMetas(server, allfmm)
		sfmm[server.Addr] = sfms
		if err != nil {
			remover.Errorf("[%s] fail to get server file meatas, erorr(%s)",
				server, err.Error())
		}
	}
	return sfmm
}

// selectFileMetas :
// param 으로 받은 file meta들 중에,
// server에 있는 file들의 file meta pointer만 모아놓은 map을 반환함
func selectFileMetas(server *common.Host,
	fileMetaMap FileMetaPtrMap) (FileMetaPtrMap, error) {

	sfm := make(FileMetaPtrMap)

	fl := make([]string, 0, 10000)
	err := common.GetRemoteFileList(server, &fl)
	if err != nil {
		s := fmt.Sprintf("fail to get remote file list, error(%s)", err.Error())
		return sfm, errors.New(s)
	}
	for _, filename := range fl {
		// 예외처리 : 아직 해당 서버의 파일 내용이 반영이 안된 상황 등으로 인해서
		// 서버에 있지만 전체 파일 목록에서 찾을 수 없으면 제외
		fm, ok := fileMetaMap[filename]
		if !ok {
			// remover.Debugf("[%s] skip file, with no grade or no size, file(%s)", server, filename)
			continue
		}
		// 등급이나 크기 중 하나만 있거나, 값이 잘못된 경우 제외
		if fm.Grade <= 0 || fm.Size <= 0 {
			// remover.Debugf("[%s] skip file, with wrong grade(%d) or wrong size(%d),"+
			// 	" file(%s)", server, fm.Grade, fm.Size, filename)
			continue
		}
		// 예외처리
		// 파일이 위치한 server 가 없다면 제외
		if fm.ServerCount <= 0 {
			//remover.Debugf("[%s] skip file, not found in the servers, file(%s)", server,  filename)
			continue
		}
		// 예외처리
		// 파일이 위치한 server list에 현재 서버가 없다면 제외
		n, exist := fm.ServerIPs[server.IP]
		if !exist || (exist && n == 0) {
			// remover.Debugf("[%s] skip file, not found in the servers, file(%s)"+
			// 	", exist(%t), count(%d)",
			// 	server, filename, exist, n)
			continue
		}

		sfm[filename] = fm
	}
	return sfm, nil
}

// requestRemoveDuplicatedFiles:
//
// 서버 파일 중에 중복 서버 list에 있는 파일 목록 구해서 delete 요청 하기
//
// disk 용량이 모자라는 경우 파일 삭제 요청 시와 같은 예외 사항이 있음
//
// - hit수가 급증가한 파일 제외
//
// - ingnore prefix 를 갖는 파일 제외
//
// - SAN 에 없는 파일 제외
func requestRemoveDuplicatedFiles(duplicatedFileMap FileMetaPtrMap,
	ssfms ServerFileMetaPtrMap) {

	// 서버별 파일 meta 정보를 가지고
	// 중복된 파일 meta 정보의 서버 정보 update
	updateFileMetasForDuplicatedFiles(duplicatedFileMap, ssfms)

	for dfn, dfm := range duplicatedFileMap {
		for _, server := range *Servers {
			// 해당 서버에 중복 파일이 없을 때
			if n, found := dfm.ServerIPs[server.IP]; !found || n == 0 {
				remover.Debugf("[%s] skip requesting to delete duplicated"+
					", not found in the server, file(%s)",
					server, dfm)
				continue
			}
			fm, ok := ssfms[server.Addr][dfn]
			// 중복 파일 meta에 update 한 후라면 필요없는 예외 처리
			// 해당 서버에 중복 파일이 없을 때
			if !ok {
				remover.Debugf("[%s] skip requesting to delete duplicated"+
					", not found in the server, file(%s)",
					server, dfn)
				continue
			}
			// 제외 대상 파일 처리
			if common.IsPrefix(fm.Name, ignorePrefixes) {
				remover.Debugf("[%s] skip file by ignore.prefix, file(%s)", server, fm.Name)
				continue
			}
			// SAN 에 없는 파일이면 삭제 대상에서 제외
			if _, exists := SourcePath.IsExistOnSource(fm.Name); exists != true {
				remover.Debugf("[%s] skip file by not.found.in.the.source.paths, file(%s)", server, fm.Name)
				continue
			}
			// 중복된 파일이 아니면 제외
			if fm.ServerCount < 2 {
				remover.Debugf("[%s] skip requesting to delete duplicated"+
					", one copy left in the servers, file(%s)",
					server, dfn)
				continue
			}
			if err := common.DeleteFileOnRemote(server, fm.Name); err != nil {
				remover.Errorf("[%s] fail to request to delete duplicated"+
					", file(%s), error(%s)",
					server, fm, err.Error())
				continue
			}
			remover.Infof("[%s] request to delete duplicated, file(%s)", server, fm)
			// 현재 server에 delete 요청 성공한 파일에 대해서
			// file meta 정보에서 현재 서버 정보 삭제
			// - file meta 정보를 다시 읽지 않고 현재 file meta 정보를 가지고,
			// 	 copy수가 하나가 될 때까지만 삭제요청을 하기 위해서 현재 file meta 정보에 반영
			if fm.ServerIPs[server.IP] > 0 {
				fm.ServerIPs[server.IP]--

				if fm.ServerCount > 0 {
					fm.ServerCount--
				}
			}
		}
	}
}

// 실제 서버의 파일 유무가 반영된 서버별 file meta 정보를 이용하여,
// 전체 서버 file meta 를 이용해서 만들어진
// 중복 서버 file(여러 서버에 있는) meta 정보의 서버 목록 정보 update
func updateFileMetasForDuplicatedFiles(duplicatedFileMap FileMetaPtrMap,
	ssfmm ServerFileMetaPtrMap) {
	for _, server := range *Servers {
		for dfn, _ := range duplicatedFileMap {
			// 중복 파일로 등록되어있는 파일에 대해서
			// 해당 서버의 파일 목록에서 찾을 수 없지만,
			// 해당 파일의 서버 리스트에는 해당 서버가 있는 경우
			_, insfmm := ssfmm[server.Addr][dfn]
			_, indfm := duplicatedFileMap[dfn].ServerIPs[server.IP]
			if !insfmm && indfm {
				// 서버 리스트에서 해당 서버의 duplicate 수 줄여주고
				// count도 줄여준다.
				if duplicatedFileMap[dfn].ServerIPs[server.IP] > 0 {
					duplicatedFileMap[dfn].ServerIPs[server.IP]--

					if duplicatedFileMap[dfn].ServerCount > 0 {
						duplicatedFileMap[dfn].ServerCount--
					}
				}
			}
		}
	}
}

// findServersOutOfDiskSpace
// 서버 중에 disk 용량이 충분하지 않는 서버 구함
func findServersOutOfDiskSpace(serverList *common.Hosts) []DServer {
	s := make([]DServer, 0, len(*serverList))
	for _, server := range *serverList {
		du := new(common.DiskUsage)
		err := common.GetRemoteDiskUsage(server, du)
		if err != nil {
			remover.Errorf("[%s] fail to get disk usage, error(%s)", server, err.Error())
			continue
		}
		// limit used size 까지 사용하지 않았으면 skip
		limitUsedSize := du.GetLimitUsedSize(diskUsageLimitPercent)
		if du.UsedSize <= limitUsedSize {
			remover.Debugf("[%s] enough disk space, used(%s) <= limit(%s)"+
				", limitPercent(%d), diskUsage(%s)",
				server, du.UsedSize, limitUsedSize, diskUsageLimitPercent, du)
			continue
		}

		remover.Debugf("[%s] not enough disk space(%s), used(%s) > limit(%s)"+
			", limitPercent(%d), diskUsage(%s)",
			server, du.UsedSize-limitUsedSize, du.UsedSize, limitUsedSize, diskUsageLimitPercent, du)

		ds := DServer{server, *du}
		s = append(s, ds)
	}

	return s
}

// requestRemoveFilesForFreeDiskSpace:
//
// - hit수가 급증가한 파일 제외
//
// - ingnore prefix 를 갖는 파일 제외
//
// - SAN 에 없는 파일 제외
func requestRemoveFilesForFreeDiskSpace(servers []DServer,
	ssfmm ServerFileMetaPtrMap, rhitfmm map[string]int) {

	for _, server := range servers {
		fileListToDelete := getFileListToDeleteForFreeDiskSpace(server,
			ssfmm, rhitfmm)
		if len(fileListToDelete) == 0 {
			remover.Infof("[%s] no file to request to delete", server)
			continue
		}

		deletingSize := common.Disksize(0)
		for _, fm := range fileListToDelete {
			// 예외처리
			if fm.ServerCount <= 0 {
				remover.Debugf("[%s] skip requesting to delete"+
					" by not.found.in.the.server, file(%s)", server, fm)
				continue
			}
			// 예외처리
			if n, exist := fm.ServerIPs[server.IP]; !exist || n <= 0 {
				remover.Debugf("[%s] skip requsting to delete"+
					" by not.found.in.the.server, file(%s)", server, fm)
				continue
			}
			if err := common.DeleteFileOnRemote(server.Host, fm.Name); err != nil {
				remover.Errorf("[%s] fail to request to delete, file(%s), error(%s)",
					server, fm, err.Error())
			} else {
				remover.Infof("[%s] request to delete, file(%s)", server, fm)
				deletingSize = deletingSize + common.Disksize(fm.Size)
				// 현재 server에 delete 요청 성공한 파일에 대해서
				// file meta 정보에서 현재 서버 정보 삭제
				// file meta 정보를 다시 읽지 않고
				// 현재 file meta 정보에 반영 최신 정보를 반영함.
				// 현재는 file meta 정보를 매번 다시 읽고,
				// 제일 마지막에 이 함수를 부르고 있어서
				// test 코드에서 검증용으로 사용할 빼고는 필요없는 코드임
				{
					if fm.ServerIPs[server.IP] > 0 {
						fm.ServerIPs[server.IP]--

						if fm.ServerCount > 0 {
							fm.ServerCount--
						}
					}
				}
			}
		}
		if deletingSize > 0 {
			remover.Infof("[%s] request for free disk space(%s / %s)", server,
				deletingSize, server.Du.GetOverUsedSize(diskUsageLimitPercent))
		}
	}
}

// getFileListToDeleteForFreeDiskSpace :
//
// disk 용량 확보를 위해서
//
// 서버의 파일 목록 중 지워야할 목록 구하기 :
//
// - hit수가 급증가한 파일 제외
//
// - ingnore prefix 를 갖는 파일 제외
//
// - SAN 에 없는 파일 제외
//
// - 낮은 등급 순으로 disk 여유 용량이 확보될 때까지
// 지워야 할 파일 구하기
func getFileListToDeleteForFreeDiskSpace(server DServer,
	ssfmm ServerFileMetaPtrMap,
	rhitfmm map[string]int) []*common.FileMeta {

	sfmm, ok := ssfmm[server.Addr]
	// 예외처리: 서버 파일 목록이 없는 경우
	if !ok {
		remover.Debugf("[%s] no file to delete", server)
		return []*common.FileMeta{}
	}

	fileList := make([]*common.FileMeta, 0, len(sfmm))
	for filename, fm := range sfmm {
		// 예외처리
		if fm.ServerCount <= 0 {
			remover.Debugf("[%s] skip file by not.found.in.the.server, file(%s)", server, filename)
			continue
		}
		// 예외처리
		if n, exist := fm.ServerIPs[server.IP]; !exist || n <= 0 {
			remover.Debugf("[%s] skip file by not.found.in.the.server, file(%s)", server, filename)
			continue
		}
		// 제외 대상 파일 처리
		if common.IsPrefix(filename, ignorePrefixes) {
			remover.Debugf("[%s] skip file by ignore.prefix, file(%s)", server, filename)
			continue
		}
		// SAN 에 없는 파일이면 삭제 대상에서 제외
		if _, exists := SourcePath.IsExistOnSource(filename); exists != true {
			remover.Debugf("[%s] skip file by not.found.in.the.source.paths, file(%s)", server, filename)
			continue
		}
		// 급 hit 상승 파일 목록에 속하는 파일이면 삭제 대상에 제외
		if _, exists := rhitfmm[filename]; exists {
			remover.Debugf("[%s] skip file by rising.hit.file, file(%s)", server, filename)
			continue
		}
		fileList = append(fileList, fm)
	}
	if len(fileList) == 0 {
		remover.Debugf("[%s] no file to delete", server)
		return fileList
	}

	// 낮은 등급순(숫자는 큰 순서로 999999 -> 1 순으로) 정렬
	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].Grade > fileList[j].Grade
	})

	fileListToDelete := make([]*common.FileMeta, 0, len(fileList)/10)
	// 용량 확보될때까지 삭제할 파일에 추가
	overUsedSize := server.Du.GetOverUsedSize(diskUsageLimitPercent)
	deletingSize := common.Disksize(0)
	for _, fm := range fileList {
		if deletingSize >= overUsedSize {
			break
		}
		deletingSize += common.Disksize(fm.Size)
		remover.Debugf("[%s] deleting size(%s / %s), add file to delete-list"+
			", file(%s)", server, common.Disksize(deletingSize), overUsedSize, fm)
		fileListToDelete = append(fileListToDelete, fm)
	}

	return fileListToDelete
}

func logElapased(message string, start time.Time) {
	remover.Infof("%s, time(%s)", message, common.Elapsed(start))
}
