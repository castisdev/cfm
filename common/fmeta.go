package common

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Freq uint64
type Hits uint64

// FileMeta is struct to save file meta info
type FileMeta struct {
	Name        string
	Grade       int32
	Size        int64
	RisingHit   int            // LB EventLog 에서 Hit 가 급격하게 오른 파일들의 Hit 수
	ServerCount int            // 이 파일을 가지고 있는 서버 개수
	ServerIPs   map[string]int // 이 파일을 가지고 있는 [서버 IP]개수
	SrcFilePath string         // source file full path : cfm이 구함
}

// NewFileMeta :
// 내부 map을 초기화하는 것 때문에, 그냥 new(FileMeta)하는 것보다 안전함
func NewFileMeta() *FileMeta {
	return &FileMeta{
		Name: "", Grade: 0, Size: -1, RisingHit: 0,
		ServerCount: 0, ServerIPs: map[string]int{}}
}

// NewFileMetaWith :
// 내부 map을 초기화하는 것 때문에, 그냥 new(FileMeta)하는 것보다 안전함
func NewFileMetaWith(filename string, grade int32) *FileMeta {
	return &FileMeta{
		Name: filename, Grade: grade, Size: -1, RisingHit: 0,
		ServerCount: 0, ServerIPs: map[string]int{}}
}

// String : FileMeta to string
func (fm FileMeta) String() string {
	var sl string
	for serverIP, n := range fm.ServerIPs {
		sl = sl + fmt.Sprintf("@%s(%d)", serverIP, n)
	}
	s := fmt.Sprintf(
		"name(%s), srcFilePath(%s), grade(%d), size(%d), risingHit(%d)"+
			", serverCount(%d), serverIPs(%s)",
		fm.Name, fm.SrcFilePath, fm.Grade, fm.Size, fm.RisingHit, fm.ServerCount, sl)
	return s
}

// parseHitcountFileAndUpdateFileMetas
//
// grade.info 파일을 parsing 한 후에 만들어지는 FileMeta map 정보에
//
// hitcount.history file을 parsing 해서 파일 size, 파일 위치 서버 정보 update
//
// parameter로 입력받는 fmm은 grade.info 파일을
// parsing 한 후에 만들어지는 FileMeta map 정보임
//
// fmm에 없는 file 에 대한 meta 정보가 추가되지는 않음
//
// 파일 위치 서버 정보는 입력 파라미터인 serverIPs 중에 속한 경우에만 반영
//
// - 입력한 serverIPs 중 여러 서버에 위치한 파일은 duplicatedFiles에 *FileMeta가 저장됨
//
// - 파일 위치 서버 정보가 필요없는 경우 입력하지 않아도 됨
func parseHitcountFileAndUpdateFileMetas(hitcountfileName string, fmm map[string]*FileMeta,
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
			if fm.ServerIPs == nil { // 예외 처리
				fm.ServerIPs = make(map[string]int)
			}

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

// parseGradeFileAndNewFileMetas
//
// 등급 파일 parsing 해서, 파일이름, 등급 값을 구해서
//
// FileMeta를 new 하고, 구한 값을 저장하고, map 에 저장
//
// 등급 파일 처러할 때, file meta가 처음으로 만들어진다고 가정
//
// - 등급 파일에는 file 이름이 unique 하다고 가정
//
// FileMeta의 Grade값에는 grade.info 파일의 grade column 값이 아니고,
//
// 파일 상의 순서값이 저장됨. 이 순서값은 1부터 시작하고, 1씩 증가함
//
// 파일 이름, 등급 값 이외의 필드 값에는 초기값이 들어감
func parseGradeFileAndNewFileMetas(fileName string, fmm map[string]*FileMeta) error {

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
		// 등급 파일 처러할 때, file meta가 처음으로 만들어진다고 가정
		// - 등급 파일에는 file 이름이 unique 하다고 가정
		fmm[fileName] = NewFileMetaWith(fileName, i)
		i++
	}

	return nil
}

// IsPrefix : prefix 검사
func IsPrefix(f string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(f, prefix) {
			return true
		}
	}
	return false
}

// MakeAllFileMetas :
//
// gradeinfo file과 hitcount history file로 부터
//
// 파일의 meta 정보 목록 만들기
//
// parseGradeFileAndNewFileMeta, parseHitcountFileAndUpdateFileMeta 순으로 호출
//
// -> grade file parsing 이 실패하면, hitcount history file 은 parsing 시도도 하지 않음
func MakeAllFileMetas(gradeInfoFile string, hitcountHistoryFile string,
	fileMetaMap map[string]*FileMeta,
	serverIPMap map[string]int,
	duplicatedFileMap map[string]*FileMeta) error {

	if err := parseGradeFileAndNewFileMetas(gradeInfoFile, fileMetaMap); err != nil {
		s := fmt.Sprintf("fail to parse file(%s), error(%s)", gradeInfoFile, err.Error())
		return errors.New(s)
	}
	// 파일 등급 list에 있는 파일들의 file size, 파일 위치 정보 구하기
	if err := parseHitcountFileAndUpdateFileMetas(hitcountHistoryFile, fileMetaMap,
		serverIPMap, duplicatedFileMap); err != nil {
		s := fmt.Sprintf("fail to parse file(%s), error(%s)", hitcountHistoryFile, err.Error())
		return errors.New(s)
	}
	return nil
}
