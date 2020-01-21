package common

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createfile(dir string, filename string) {
	fp := filepath.Join(dir, filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func deletefile(dir string, filename string) {
	dir = filepath.Clean(dir)
	if dir == "." || dir == ".." {
		log.Fatal(errors.New("do not delete current or parent folder"))
	}
	fp := filepath.Join(dir, filename)
	err := os.RemoveAll(fp)
	if err != nil {
		log.Fatal(err)
	}
}

func makeHitcountFile(dir string, filename string) {
	fp := filepath.Join(dir, filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.Create(fp)
	if err != nil {
		f.Close()
		log.Fatal(err)
	}
	fmt.Fprintln(f, "historyheader:1524047082")
	fmt.Fprintln(f, "AAAAAAAAAAAAAAAAAA.mpg,1428460337,0,15870,125.159.40.3,2,0,0,0=0 0")
	fmt.Fprintln(f, "CCCCCCCCCCCCCCCCCC_K20180501000000.mpg,1508301143,6399785,920910664,125.144.97.70 125.159.40.3,2,0,0,1=1 2")
	fmt.Fprintln(f, "GGGGGGGGGGGGGGGGGG_K20180501000000.mpg,1502087913,10518227,37699584,125.159.40.3,1,0,0,0=0 0")
	fmt.Fprintln(f, "FFFFFFFFFFFFFFFFFF_K20180501000000.mpg,1508867888,1258200,336050386,125.144.93.97,1,0,0,1=0 0")
	fmt.Fprintln(f, "EEEEEEEEEEEEEEEEEE_K20180501000000.mpg,1500856569,6468780,957662460,125.159.40.3,1,0,0,1=0 0")
	fmt.Fprintln(f, "DDDDDDDDDDDDDDDDDD_K20180501000000.mpg,1384219813,0,100,125.144.91.71,1,0,0,0=0 0")
	fmt.Fprintln(f, "IIIIIIIIIIIIIIIIII_K20180501000000.mpg,1428460341,0,6647,125.159.40.3,2,0,0,0=0 0")
	fmt.Fprintln(f, "BBBBBBBBBBBBBBBBBB_K20180501000000.mpg,1428399564,0,16080,127.0.0.1,2,0,0,0=0 0")
	fmt.Fprintln(f, "HHHHHHHHHHHHHHHHHH_K20180501000000.mpg,1428398692,0,17070,125.159.40.3,2,0,0,0=0 0")
	fmt.Fprintln(f, "STRANGESERVER.mpg,1500856569,10,1234,127.0.0.1 127.0.0.2,2,0,0,0=0 0")
	fmt.Fprintln(f, "NOWHERE.mpg,1500856569,10,9999,,2,0,0,0=0 0")
	f.Close()
}

func Test_parseHitcountFile(t *testing.T) {
	hcdir := "testhitcountfolder"
	hcfile := "hitcount.history"
	makeHitcountFile(hcdir, hcfile)
	defer deletefile(hcdir, "")

	hitcountHistoryFile := filepath.Join(hcdir, hcfile)

	// fmm: graceinfo 파일 파싱 후에 만들어진 filemeta map이라고 가정하고 만듬
	fmm := make(map[string]*FileMeta)
	fmm["AAAAAAAAAAAAAAAAAA.mpg"] = NewFileMetaWith("AAAAAAAAAAAAAAAAAA.mpg", 1)
	fmm["BBBBBBBBBBBBBBBBBB_K20180501000000.mpg"] = NewFileMetaWith("BBBBBBBBBBBBBBBBBB_K20180501000000.mpg", 2)
	fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"] = NewFileMetaWith("CCCCCCCCCCCCCCCCCC_K20180501000000.mpg", 3)
	fmm["STRANGESERVER.mpg"] = NewFileMetaWith("STRANGESERVER.mpg", 99)
	fmm["NOWHERE.mpg"] = NewFileMetaWith("NOWHERE.mpg", 100)
	serverIPs := make(map[string]int)
	serverIPs["125.144.97.70"] = 1
	serverIPs["125.159.40.3"] = 1
	serverIPs["125.144.93.97,1"] = 1
	serverIPs["125.144.91.71"] = 1
	duplicatedFiles := make(map[string]*FileMeta)

	// fmm 에 미리 들어있지 않는 file 에 대한 meta는 새로 만들어지지 않음
	err := parseHitcountFileAndUpdateFileMetas(hitcountHistoryFile, fmm, serverIPs, duplicatedFiles)
	if err != nil {
		t.Errorf("parsing error(%s)", err.Error())
	}

	assert.Equal(t, int64(15870), fmm["AAAAAAAAAAAAAAAAAA.mpg"].Size)
	assert.Equal(t, int64(16080), fmm["BBBBBBBBBBBBBBBBBB_K20180501000000.mpg"].Size)
	assert.Equal(t, int64(920910664), fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"].Size)
	assert.Equal(t, int64(1234), fmm["STRANGESERVER.mpg"].Size)

	// server 정보가 없어도 등록됨
	assert.Equal(t, int64(9999), fmm["NOWHERE.mpg"].Size)

	assert.Equal(t, int32(1), fmm["AAAAAAAAAAAAAAAAAA.mpg"].Grade)
	assert.Equal(t, int32(2), fmm["BBBBBBBBBBBBBBBBBB_K20180501000000.mpg"].Grade)
	assert.Equal(t, int32(3), fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"].Grade)
	assert.Equal(t, int32(99), fmm["STRANGESERVER.mpg"].Grade)
	assert.Equal(t, int32(100), fmm["NOWHERE.mpg"].Grade)

	assert.Equal(t, int(1), fmm["AAAAAAAAAAAAAAAAAA.mpg"].ServerCount)

	// BBBBBBBBBBBBBBBBBB_K20180501000000.mpg 는 파일 정보가 127.0.0.1 인데,
	// parseHitcountFileAndUpdateFileMetas 의 parameter로 입력하지 않았으므로,
	// server count 정보와 server ips 정보가 없음
	assert.Equal(t, int(0), fmm["BBBBBBBBBBBBBBBBBB_K20180501000000.mpg"].ServerCount)
	assert.Equal(t, int(0), len(fmm["BBBBBBBBBBBBBBBBBB_K20180501000000.mpg"].ServerIPs))

	// CCCCCCCCCCCCCCCCCC_K20180501000000.mpg 파일이 위치한 서버 ip가 두 개임
	assert.Equal(t, int(2), fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"].ServerCount)

	// STRANGESERVER.mpg 는 fmm 에 들어있었지만,
	// 이 파일이 위치한 125.144.91.71 가 serverIPs 에 들어있지 않으므로,
	// FileMeta의 server ip 정보가 update 되지 않고, duplicated map 에도 들어가지 않음
	assert.Equal(t, int(0), fmm["STRANGESERVER.mpg"].ServerCount)

	// server 정보가 없어도 등록됨
	assert.Equal(t, int(0), fmm["NOWHERE.mpg"].ServerCount)

	// DDDDDDDDDDDDDDDDDD_K20180501000000.mpg는 함수 호출 전에 fmm 에 들어있지 않았기 때문에,
	// hitcount 파일에는 있지만, fmm에 추가 되지는 않았음
	fileName := "DDDDDDDDDDDDDDDDDD_K20180501000000.mpg"
	_, exists := fmm[fileName]
	assert.Equal(t, false, exists)
	// IIIIIIIIIIIIIIIIII_K20180501000000.mpg 마찬가지
	fileName = "IIIIIIIIIIIIIIIIII_K20180501000000.mpg"
	_, exists = fmm[fileName]
	assert.Equal(t, false, exists)
	// GGGGGGGGGGGGGGGGGG_K20180501000000.mpg 마찬가지
	fileName = "GGGGGGGGGGGGGGGGGG_K20180501000000.mpg"
	_, exists = fmm[fileName]
	assert.Equal(t, false, exists)
	// FFFFFFFFFFFFFFFFFF_K20180501000000.mpg 마찬가지
	fileName = "FFFFFFFFFFFFFFFFFF_K20180501000000.mpg"
	_, exists = fmm[fileName]
	assert.Equal(t, false, exists)

	// CCCCCCCCCCCCCCCCCC_K20180501000000.mpg 의 *FileMeta가 duplicated map 에 들어감
	// pointer 이기 때문에 내부 값들도 같음
	assert.Equal(t, int(1), len(duplicatedFiles))
	assert.Equal(t, duplicatedFiles["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"],
		fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"])
	t.Logf("DuplicatedFile: %s", duplicatedFiles["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"])
	t.Logf("fileMeta: %s", fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"])
}

func Test_parseGradeFile(t *testing.T) {

	tmpFile := "grade.info"

	f, err := os.Create(tmpFile)
	if err != nil {
		f.Close()
		t.Errorf("cannot create %s", tmpFile)
	}
	defer os.Remove(tmpFile)

	fmt.Fprintf(f, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "filename", "weightcount", "bitrate", "grade", "sumHitCount", "historyCount", "TargetCopyCount")
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "AAAAAAAAAAAAAAAAAA.mpg", 4144, 6439600, 1, 1554, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "BBBBBBBBBBBBBBBBBB_K20140501000000.mpg", 4042, 6468052, 1, 1516, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "CCCCCCCCCCCCCCCCCC_K20180501000000.mpg", 3861, 6443013, 1, 1448, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "DDDDDDDDDDDDDDDDDD_K20180501000000.mpg", 3493, 6443011, 1, 1310, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "EEEEEEEEEEEEEEEEEE_K20180501000000.mpg", 3306, 6443019, 1, 1240, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "FFFFFFFFFFFFFFFFFF_K20180501000000.mpg", 3285, 6443056, 1, 1232, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "GGGGGGGGGGGGGGGGGG_K20180501000000.mpg", 3245, 6443011, 1, 1217, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "HHHHHHHHHHHHHHHHHH_K20180501000000.mpg", 3226, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "JJJJJJJJJJJJJJJJJJ_K20180501000000.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "vod1-1.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "vod1-2.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "vod1-3.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "vod2-1.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "vod2-2.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "vod2-3.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "vod3-1.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "vod3-2.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "vod3-3.mpg", 3225, 6443017, 1, 1210, 24, 5)
	f.Close()

	fmm := make(map[string]*FileMeta)

	assert.Nil(t, parseGradeFileAndNewFileMetas(tmpFile, fmm))

	assert.Equal(t, int32(1), fmm["AAAAAAAAAAAAAAAAAA.mpg"].Grade)
	assert.Equal(t, int32(2), fmm["BBBBBBBBBBBBBBBBBB_K20140501000000.mpg"].Grade)
	assert.Equal(t, int32(9), fmm["JJJJJJJJJJJJJJJJJJ_K20180501000000.mpg"].Grade)
	assert.Equal(t, int32(18), fmm["vod3-3.mpg"].Grade)

}

// grade info file 에는 file name이 unique 하다고 가정한다.
// file 이름이 중복되면, 제일 나중 정보만 사용된다.
func Test_parseGradeFileWithDuplicatedFileNames(t *testing.T) {

	tmpFile := "grade.info"

	f, err := os.Create(tmpFile)
	if err != nil {
		f.Close()
		t.Errorf("cannot create %s", tmpFile)
	}
	defer os.Remove(tmpFile)

	fmt.Fprintf(f, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "filename", "weightcount", "bitrate", "grade", "sumHitCount", "historyCount", "TargetCopyCount")
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "A.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "B.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "C.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "A.mpg", 3225, 6443017, 1, 1210, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "A.mpg", 3225, 6443017, 1, 1210, 24, 5)
	f.Close()

	fmm := make(map[string]*FileMeta)
	assert.Nil(t, parseGradeFileAndNewFileMetas(tmpFile, fmm))

	assert.Equal(t, int32(2), fmm["B.mpg"].Grade)
	assert.Equal(t, int32(3), fmm["C.mpg"].Grade)

	// A의 Grade 는 1이 아니고 5가 된다.
	// 제일 마지막 나온 A의 grade 값이 5이기 때문
	assert.Equal(t, int32(5), fmm["A.mpg"].Grade)
}

func TestIsPrefix(t *testing.T) {

	prefixes := []string{"M64", "MN1"}

	assert.Equal(t, true, IsPrefix("M640001.mpg", prefixes))
	assert.Equal(t, true, IsPrefix("MN10001.mpg", prefixes))
	assert.Equal(t, false, IsPrefix("M650001.mpg", prefixes))
	assert.Equal(t, false, IsPrefix("MN20001.mpg", prefixes))
	assert.Equal(t, false, IsPrefix("AM64001.mpg", prefixes))
	assert.Equal(t, false, IsPrefix("AMN10001.mpg", prefixes))
}
