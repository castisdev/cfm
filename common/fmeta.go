package common

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// FileMeta is struct to save file meta info
type FileMeta struct {
	Name        string
	Grade       int32
	Size        int64
	RisingHit   int            // LB EventLog 에서 Hit 가 급격하게 오른 파일들의 Hit 수
	ServerCount int            // 이 파일을 가지고 있는 서버 개수
	ServerIPs   map[string]int // 이 파일을 가지고 있는 [서버 IP]개수
}

// ParseHitcountFile :
// hitcount.history file을 parsing 해서
// 파일 이름을 key로 하는 map 에
// 파일 size, 파일 위치 서버 정보 반영
// 파일 위치 서버 정보는 serverIPs 중에 속한 경우에만 반영
// 여러 서버에 위치한 파일은 duplicatedFiles 에도 저장함
func ParseHitcountFile(hitcountfileName string, fmm map[string]*FileMeta,
	serverIPs map[string]int, duplicatedFiles map[string]*FileMeta) error {

	file, err := os.Open(hitcountfileName)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // remove header
	for scanner.Scan() {

		if scanner.Text() == "" {
			continue
		}

		cols := strings.Split(scanner.Text(), ",")

		if len(cols) < 5 {
			continue
		}

		fileName := cols[0]
		fileSize := cols[3]
		vodIPList := cols[4]

		size, _ := strconv.ParseInt(fileSize, 10, 64)

		// 이미 file meta map에 저장되어있는 경우에만 parsing한 값을 update
		if fm, exists := fmm[fileName]; exists {
			fm.Size = size
			fm.ServerCount = 0
			vodIPs := strings.Split(vodIPList, " ")
			for _, vodIP := range vodIPs {
				if _, found := serverIPs[vodIP]; found {
					fm.ServerIPs[vodIP]++
					fm.ServerCount++
				}
			}
			if fm.ServerCount > 1 {
				duplicatedFiles[fileName] = fm
			}
		}
	}

	return nil
}

// ParseGradeFile is to parse file name, file grade from .grade.info
// 등급 파일 parsing 해서, FileMeta map에 파일이름, 등급 저장
func ParseGradeFile(fileName string, fmm map[string]*FileMeta) error {

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // remove header
	i := int32(1)
	for scanner.Scan() {
		if scanner.Text() == "" {
			continue
		}
		cols := strings.Split(scanner.Text(), "\t")
		fileName := cols[0]
		// hitcount.history 에서 파일 사이즈를 가져오게 되는데,
		// grade.info 에 있는 파일이 hitcount.history 에 없을 수 있음
		// size = -1 인 경우 hitcount.history 에 없다고 여기면 됨
		fmm[fileName] = &FileMeta{
			Name: fileName, Grade: i, Size: -1,
			ServerCount: 0, ServerIPs: map[string]int{}}
		i++
	}

	return nil
}

// IsADFile :
func IsADFile(f string, adPrefixes []string) bool {

	for _, prefix := range adPrefixes {
		if strings.HasPrefix(f, prefix) {
			return true
		}
	}

	return false
}

// String : FileMeta to string
func (fm FileMeta) String() string {
	var sl string
	for serverIP, n := range fm.ServerIPs {
		sl = sl + fmt.Sprintf("@%s(%d)", serverIP, n)
	}
	s := fmt.Sprintf(
		"Name(%s), Grade(%d), Size(%d), RisingHit(%d), SeverCount(%d), ServerIPs(%s)",
		fm.Name, fm.Grade, fm.Size, fm.RisingHit, fm.ServerCount, sl)

	return s
}

// GetAllFileMetas :
// gradeinfo file과 hitcount history file로 부터
// 파일의 meta 정보 목록 만들기
// ParseGradeFile, ParseHitcountFile 호출
func MakeAllFileMetas(gradeInfoFile string, hitcountHistoryFile string,
	fileMetaMap map[string]*FileMeta,
	serverIPMap map[string]int,
	duplicatedFileMap map[string]*FileMeta) error {

	if err := ParseGradeFile(gradeInfoFile, fileMetaMap); err != nil {
		s := fmt.Sprintf("fail to parse file(%s), error(%s)", gradeInfoFile, err.Error())
		return errors.New(s)
	}
	// 파일 등급 list에 있는 파일들의 file size, 파일 위치 정보 구하기
	if err := ParseHitcountFile(hitcountHistoryFile, fileMetaMap,
		serverIPMap, duplicatedFileMap); err != nil {
		s := fmt.Sprintf("fail to parse file(%s), error(%s)", hitcountHistoryFile, err.Error())
		return errors.New(s)
	}
	return nil
}
