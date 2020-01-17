package common_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/castisdev/cfm/common"
	"github.com/stretchr/testify/assert"
)

func Test_ParseHitcountFile(t *testing.T) {

	tmpFile := "hitcount.history"
	f, err := os.Create(tmpFile)
	if err != nil {
		f.Close()
		t.Errorf("cannot create %s", tmpFile)
	}
	defer os.Remove(tmpFile)

	fmt.Fprintln(f, "historyheader:1524047082")
	fmt.Fprintln(f, "AAAAAAAAAAAAAAAAAA.mpg,1428460337,0,15870,125.159.40.3,2,0,0,0=0 0")
	fmt.Fprintln(f, "CCCCCCCCCCCCCCCCCC_K20180501000000.mpg,1508301143,6399785,920910664,125.144.97.70 125.159.40.3,2,0,0,1=1 2")
	fmt.Fprintln(f, "GGGGGGGGGGGGGGGGGG_K20180501000000.mpg,1502087913,10518227,37699584,125.159.40.3,1,0,0,0=0 0")
	fmt.Fprintln(f, "FFFFFFFFFFFFFFFFFF_K20180501000000.mpg,1508867888,1258200,336050386,125.144.93.97,1,0,0,1=0 0")
	fmt.Fprintln(f, "EEEEEEEEEEEEEEEEEE_K20180501000000.mpg,1500856569,6468780,957662460,125.159.40.3,1,0,0,1=0 0")
	fmt.Fprintln(f, "DDDDDDDDDDDDDDDDDD_K20180501000000.mpg,1384219813,0,100,125.144.91.71,1,0,0,0=0 0")
	fmt.Fprintln(f, "IIIIIIIIIIIIIIIIII_K20180501000000.mpg,1428460341,0,6647,125.159.40.3,2,0,0,0=0 0")
	fmt.Fprintln(f, "BBBBBBBBBBBBBBBBBB_K20180501000000.mpg,1428399564,0,16080,125.159.40.3,2,0,0,0=0 0")
	fmt.Fprintln(f, "HHHHHHHHHHHHHHHHHH_K20180501000000.mpg,1428398692,0,17070,125.159.40.3,2,0,0,0=0 0")
	fmt.Fprintln(f, "STRANGESERVER.mpg,1500856569,10,1234,127.0.0.1 127.0.0.2,2,0,0,0=0 0")
	f.Close()

	// fmm: graceinfo 파일 파싱 후에 만들어진 filemeta map이라고 가정하고 만듬
	fmm := make(map[string]*common.FileMeta)
	fmm["AAAAAAAAAAAAAAAAAA.mpg"] = common.NewFileMetaWith("AAAAAAAAAAAAAAAAAA.mpg", 1)
	fmm["BBBBBBBBBBBBBBBBBB_K20180501000000.mpg"] = common.NewFileMetaWith("BBBBBBBBBBBBBBBBBB_K20180501000000.mpg", 2)
	fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"] = common.NewFileMetaWith("CCCCCCCCCCCCCCCCCC_K20180501000000.mpg", 3)
	fmm["STRANGESERVER.mpg"] = common.NewFileMetaWith("STRANGESERVER.mpg", 99)
	serverIPs := make(map[string]int)
	serverIPs["125.144.97.70"] = 1
	serverIPs["125.159.40.3"] = 1
	serverIPs["125.144.93.97,1"] = 1
	serverIPs["125.144.91.71"] = 1
	duplicatedFiles := make(map[string]*common.FileMeta)
	err = common.ParseHitcountFileAndUpdateFileMetas(tmpFile, fmm, serverIPs, duplicatedFiles)
	if err != nil {
		t.Errorf("parsing error(%s)", err.Error())
	}

	assert.Equal(t, int64(15870), fmm["AAAAAAAAAAAAAAAAAA.mpg"].Size)
	assert.Equal(t, int64(16080), fmm["BBBBBBBBBBBBBBBBBB_K20180501000000.mpg"].Size)
	assert.Equal(t, int64(920910664), fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"].Size)
	assert.Equal(t, int64(1234), fmm["STRANGESERVER.mpg"].Size)

	assert.Equal(t, int32(1), fmm["AAAAAAAAAAAAAAAAAA.mpg"].Grade)
	assert.Equal(t, int32(2), fmm["BBBBBBBBBBBBBBBBBB_K20180501000000.mpg"].Grade)
	assert.Equal(t, int32(3), fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"].Grade)
	assert.Equal(t, int32(99), fmm["STRANGESERVER.mpg"].Grade)

	assert.Equal(t, int(1), fmm["AAAAAAAAAAAAAAAAAA.mpg"].ServerCount)
	assert.Equal(t, int(1), fmm["BBBBBBBBBBBBBBBBBB_K20180501000000.mpg"].ServerCount)
	// CCCCCCCCCCCCCCCCCC_K20180501000000.mpg 파일이 위치한 서버 ip가 두 개임
	assert.Equal(t, int(2), fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"].ServerCount)
	// STRANGESERVER.mpg 는 fmm 에 들어있었지만,
	// 이 파일이 위치한 125.144.91.71 가 serverIPs 에 들어있지 않으므로,
	// FileMeta의 server ip 정보가 update 되지 않고, duplicated map 에도 들어가지 않음
	assert.Equal(t, int(0), fmm["STRANGESERVER.mpg"].ServerCount)

	// DDDDDDDDDDDDDDDDDD_K20180501000000.mpg는 함수 호출 전에 fmm 에 들어있지 않았기 때문에,
	// hitcount 파일에는 있지만, fmm에 추가 되지는 않았음
	fileName := "DDDDDDDDDDDDDDDDDD_K20180501000000.mpg"
	_, exists := fmm[fileName]
	assert.Equal(t, false, exists)

	// CCCCCCCCCCCCCCCCCC_K20180501000000.mpg 의 *FileMeta가 duplicated map 에 들어감
	// pointer 이기 때문에 내부 값들도 같음
	assert.Equal(t, int(1), len(duplicatedFiles))
	assert.Equal(t, duplicatedFiles["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"],
		fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"])
	t.Logf("DuplicatedFile: %s", duplicatedFiles["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"])
	t.Logf("fileMeta: %s", fmm["CCCCCCCCCCCCCCCCCC_K20180501000000.mpg"])

}

func Test_ParseGradeFile(t *testing.T) {

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

	fmm := make(map[string]*common.FileMeta)

	assert.Nil(t, common.ParseGradeFileAndNewFileMetas(tmpFile, fmm))

	assert.Equal(t, int32(1), fmm["AAAAAAAAAAAAAAAAAA.mpg"].Grade)
	assert.Equal(t, int32(2), fmm["BBBBBBBBBBBBBBBBBB_K20140501000000.mpg"].Grade)
	assert.Equal(t, int32(9), fmm["JJJJJJJJJJJJJJJJJJ_K20180501000000.mpg"].Grade)
	assert.Equal(t, int32(18), fmm["vod3-3.mpg"].Grade)

}

func TestIsPrefix(t *testing.T) {

	prefixes := []string{"M64", "MN1"}

	assert.Equal(t, true, common.IsPrefix("M640001.mpg", prefixes))
	assert.Equal(t, true, common.IsPrefix("MN10001.mpg", prefixes))
	assert.Equal(t, false, common.IsPrefix("M650001.mpg", prefixes))
	assert.Equal(t, false, common.IsPrefix("MN20001.mpg", prefixes))
	assert.Equal(t, false, common.IsPrefix("AM64001.mpg", prefixes))
	assert.Equal(t, false, common.IsPrefix("AMN10001.mpg", prefixes))

}
