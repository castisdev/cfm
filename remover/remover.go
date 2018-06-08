package remover

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cilog"
)

// Servers : 파일 삭제 대상 서버 리스트
var Servers *common.Hosts

// SourcePath : 파일 삭제 시 Source 에 없는 파일이면 삭제 대상에서 제외하기 위해 사용
var SourcePath *common.SourceDirs
var diskUsageLimitPercent int

var advPrefixes []string
var hitcountHistoryFile string
var gradeInfoFile string

func init() {
	Servers = common.NewHosts()
	SourcePath = common.NewSourceDirs()
}

// RunForever is to run remover as go routine
func RunForever() {
	for {

		var diskUsageMap = make(map[string]*common.DiskUsage)
		var fileMetaMap = make(map[string]*common.FileMeta)
		isAlreadyParsed := false

		// diskUsageMap's key = ip:port
		// 1. disk 용량 확인
		collectRemoteDiskUsage(Servers, diskUsageMap)

		for ipPort, diskUsage := range diskUsageMap {

			// 2. 지워야 하는 사이즈 구하고, 용량 충분하면 skip
			allowedDiskSize := diskUsage.TotalSize * int64(diskUsageLimitPercent) / 100
			sizeToDelete := diskUsage.UsedSize - allowedDiskSize
			if sizeToDelete <= 0 {
				cilog.Debugf("size to delete is less than 0 (%d), so skip (%s)", sizeToDelete, ipPort)
				continue
			}
			cilog.Debugf("size to delete is (%d) on (%s),total(%d),allowedSize(%d)", sizeToDelete, ipPort, diskUsage.TotalSize, allowedDiskSize)

			// grade.info 와 hitcount.history 의 라인 수가 상당하여 한번만 파싱하도록
			if isAlreadyParsed == false {

				// 3. 파일 등급 list 생성
				if err := common.ParseGradeFile(gradeInfoFile, fileMetaMap); err != nil {
					cilog.Errorf("fail to parse file(%s), error(%s)", gradeInfoFile, err.Error())
					break
				}
				cilog.Infof("parse file(%s)", gradeInfoFile)

				if err := common.ParseHitcountFile(hitcountHistoryFile, fileMetaMap); err != nil {
					cilog.Errorf("fail to parse file(%s), error(%s)", hitcountHistoryFile, err.Error())
					break
				}
				cilog.Infof("parse file(%s)", hitcountHistoryFile)

				isAlreadyParsed = true
			}

			host, err := common.SplitHostPort(ipPort)

			if err != nil {
				cilog.Exceptionf("cannot split string(%s) to host,error(%s)", ipPort, err.Error())
				continue
			}

			// 4. 서버별 파일 리스트 확인
			fl := make([]string, 0, 10000)
			err = common.GetRemoteFileList(&host, &fl)

			if err != nil {
				cilog.Errorf("fail to get remote file list (%s:%d), error(%s)", host.IP, host.Port, err.Error())
				break
			}
			cilog.Debugf("get remote file list (%s:%d)", host.IP, host.Port)

			// 5. 서버별 파일 리스트 구조 변환 []string -> []common.FileMeta
			var fileList []common.FileMeta
			for _, file := range fl {
				// 5-1. 광고 파일은 제외
				if common.IsADFile(file, advPrefixes) {
					continue
				}
				fileList = append(fileList, common.FileMeta{Name: file, Grade: -1, Size: -1})
			}

			// 6. 파일들의 등급 확인
			for i, file := range fileList {
				if _, exists := fileMetaMap[file.Name]; exists {
					fileList[i].Grade = fileMetaMap[file.Name].Grade
				}
			}

			// 7. 파일들의 크기 확인
			for i, file := range fileList {
				if _, exists := fileMetaMap[file.Name]; exists {
					fileList[i].Size = fileMetaMap[file.Name].Size
				}
			}

			// 8. 낮은 등급순으로 정렬 (999999->1 순으로)
			sort.Slice(fileList, func(i, j int) bool {
				return fileList[i].Grade > fileList[j].Grade
			})

			fileListToDelete := make([]string, 0, 10)
			// 9. 용량 확보될때까지 삭제할 파일 걸러내고
			for _, file := range fileList {

				if sizeToDelete <= 0 {
					break
				}

				// hitcount.history 에 없어서 파일 크기를 못구한 경우 size = -1
				// hitcount.history 에 없다면(hitcount.history 반영 전) 이제 막 서버에 생성된 파일이라는 의미
				// 신규 생성된 파일들은 삭제 대상에서 제외
				if file.Size < 0 {
					continue
				}
				// 10. SAN 에 없는 파일이면 삭제 대상에서 제외하고
				if _, exists := SourcePath.IsExistOnSource(file.Name); exists != true {
					continue
				}

				sizeToDelete -= file.Size
				fileListToDelete = append(fileListToDelete, file.Name)
			}

			for _, file := range fileListToDelete {

				if err := common.DeleteFileOnRemote(&host, file); err != nil {
					cilog.Errorf("fail to delete file(%s) on (%s:%d),error(%s)", file, host.IP, host.Port, err.Error())
				} else {
					if grade, exists := fileMetaMap[file]; exists {
						cilog.Successf("success to delete,file(%s),grade(%d),server(%s:%d)", file, grade.Grade, host.IP, host.Port)
					} else {
						cilog.Successf("success to delete,file(%s),server(%s:%d)", file, host.IP, host.Port)
					}
				}
			}
		}

		time.Sleep(time.Second * 3)
	}
}

func collectRemoteDiskUsage(hostList *common.Hosts, diskUsageMap map[string]*common.DiskUsage) {

	for _, host := range *hostList {
		du := new(common.DiskUsage)
		err := common.GetRemoteDiskUsage(host, du)
		if err != nil {
			cilog.Errorf("fail to connect to (%s:%d),error(%s)", host.IP, host.Port, err.Error())
		} else {
			cilog.Debugf("get remote(%s:%d) disk usage", host.IP, host.Port)
		}

		key := fmt.Sprintf("%s:%d", host.IP, host.Port)
		diskUsageMap[key] = du
	}
}

// SetDiskUsageLimitPercent is to set the limitation of disk used size
// min is 0, max is 100
func SetDiskUsageLimitPercent(limit int) error {

	if limit < 0 {
		return errors.New("limit must be greater than 0")
	}

	if limit > 100 {
		return errors.New("limit must be less than 100")
	}

	diskUsageLimitPercent = limit

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
