package common

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// FileMeta is struct to save file meta info
type FileMeta struct {
	Name  string
	Grade int32
	Size  int64
}

// ParseHitcountFile is to parse file name, file size from .hitcount.history
func ParseHitcountFile(fileName string, fmm map[string]*FileMeta) error {

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // remove header
	for scanner.Scan() {
		cols := strings.Split(scanner.Text(), ",")
		fileName := cols[0]
		fileSize := cols[3]

		size, _ := strconv.ParseInt(fileSize, 10, 64)

		// grade.info 에 있는 경우에만 size 값을 채워 넣음
		// grade.info 에는 없고 hitcount.history 에만 있는 파일은 무시(등급을 알수 없으므로)
		if _, exists := fmm[fileName]; exists {
			fmm[fileName].Size = size
		}
	}

	return nil
}

// ParseGradeFile is to parse file name, file grade from .grade.info
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
		cols := strings.Split(scanner.Text(), "\t")
		fileName := cols[0]
		// hitcount.history 에서 파일 사이즈를 가져오게 되는데,
		// grade.info 에 있는 파일이 hitcount.history 에 없을 수 있음
		// size = -1 인 경우 hitcount.history 에 없다고 여기면 됨
		fmm[fileName] = &FileMeta{Name: fileName, Grade: i, Size: -1}
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
