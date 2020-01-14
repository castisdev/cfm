package remover

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/tailer"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func makeGradeInfoFile(dir string, filename string) {
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
	fmt.Fprintf(f, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "filename", "weightcount", "bitrate", "grade", "sumHitCount", "historyCount", "TargetCopyCount")
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "A.mpg", 4144, 1000, 1, 1554, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "B.mpg", 4042, 1000, 1, 1516, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "C.mpg", 3861, 1000, 1, 1448, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "D.mpg", 3493, 1000, 1, 1310, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "E.mpg", 3306, 1000, 1, 1240, 24, 5)
	fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", "F.mpg", 3285, 1000, 1, 1232, 24, 5)

	f.Close()
}

func makeHitcourntHistoryFile(dir string, filename string) {
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
	fmt.Fprintln(f, "A.mpg,1428460337,1000,100,127.0.0.1,2,0,0,0=0 0")
	fmt.Fprintln(f, "B.mpg,1508301143,1000,100,127.0.0.1 127.0.0.2,2,0,0,1=1 2")
	fmt.Fprintln(f, "C.mpg,1508301143,1000,100,127.0.0.1 127.0.0.2,2,0,0,1=1 2")
	fmt.Fprintln(f, "D.mpg,1428460337,1000,100,127.0.0.1,2,0,0,0=0 0")
	fmt.Fprintln(f, "E.mpg,1428460337,1000,100,127.0.0.2,2,0,0,0=0 0")
	fmt.Fprintln(f, "F.mpg,1428460337,1000,100,127.0.0.2,2,0,0,0=0 0")

	f.Close()
}

// 기준 시간 : 1527951611
// 기준 ip : 125.159.40.3
// rising file : F.mpg
// now := time.Now()
func makeRisingHitFile(dir string, filename string, now time.Time, watchTermMin int) {
	// 현재 시각값을 이용하여 N분 전 시각을 구하기 위해선 음수 값이 필요하다.
	from := now.Add(time.Minute * time.Duration(watchTermMin*-1))
	logFileNames := tailer.GetLogFileName(dir, watchTermMin)

	baseTime := from.Unix()
	logFileName := (*logFileNames)[0]
	fp := filepath.Clean(logFileName)

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
	// ------------------------------------------------- 테스트 기준 시각 - 4
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : 3f004a6e-82af-4dce-85ba-9bbf9c7cb8cb, ClientID : 0, GLB IP : 125.144.96.6's file(MCLE901VSGL1500001_K20140915224744.mpg) Request", baseTime-4)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,4,%d,File Not Found, UUID : fffb233a-376a-4c2f-842e-553fb68af9cf, GLB IP : 125.144.161.6, MV6F9001SGL1500001_K20150909214818.mpg", baseTime-4)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.91.87 Selected for Client StreamID : 360527d4-44b3-4b8f-aef7-dbf8fd230d54, ClientID : 0, GLB IP : 125.144.169.6's file(M33E80DTSGL1500001_K20141022144006.mpg) Request", baseTime-4)
	fmt.Fprintln(f)
	// ------------------------------------------------- 테스트 기준 시각 - 2
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : c93a7db2-ccaf-4765-af8d-7ddc2d33a812, ClientID : 0, GLB IP : 125.159.40.5's file(F.mpg) Request", baseTime-2)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,4,%d,File Not Found, UUID : f1add5cf-75ac-41ab-a6ff-85d9e0927762, GLB IP : 125.144.169.6, MK4E7BK2SGL0800014_K20120725124707.mpg", baseTime-2)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.97.67 Selected for Client StreamID : 06fb572e-7602-4231-8670-cb6526603fb0, ClientID : 0, GLB IP : 125.146.8.6's file(M33H90E2SGL1500001_K20171008222635.mpg) Request", baseTime-2)
	fmt.Fprintln(f)
	// ------------------------------------------------- 테스트 기준 시각 - 1
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : 3c61af91-cd6a-4dd6-bc04-5ec6bc78b94f, ClientID : 0, GLB IP : 125.159.40.5's file(MWGI5006SGL1500001_K20180524203234.mpg) Request", baseTime-1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,4,%d,File Not Found, UUID : c585905f-9980-49b1-89bc-97c7140eaa83, GLB IP : 125.159.40.5, M34G80A3SGL1500001_K20160827230242.mpg", baseTime-1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.97.74 Selected for Client StreamID : 7cf6b886-edd2-471b-9cfd-12763a160b0b, GLB IP : 125.159.40.5's file(M34F60QHSGL1500001_K20150701232550.mpg) Request", baseTime-1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.97.77 Selected for Client StreamID : 23dd1489-543b-4051-b07a-e877f8b2e052, GLB IP : 125.147.192.6's file(MW0E6JE3SGL0800014_K20120601193450.mpg) Request", baseTime-1)
	fmt.Fprintln(f)
	// ------------------------------------------------- 테스트 기준 시각
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : 97096b41-afe1-44d8-b57c-e758a70883d9, GLB IP : 125.159.40.5's file(M33F3MA3SGL0800038_K20130326135640.mpg) Request", baseTime)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.91.83 Selected for Client StreamID : aa7de9a1-7d0d-40d5-9586-31dc275a0634, ClientID : 0, GLB IP : 125.147.36.6's file(MADI4008SGL1500001_K20180506231943.mpg) Request", baseTime)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.91.84 Selected for Client StreamID : 1926c2ba-c313-48fb-977a-b7f3fd27ea98, ClientID : 0, GLB IP : 125.148.160.6's file(MEQI405ISGL1500001_K20180509034746.mpg) Request", baseTime)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.97.73 Selected for Client StreamID : f179c61a-d5e0-45b9-b046-a3cd4e3dbbfc, ClientID : 0, GLB IP : 125.147.192.6's file(MIAF51OLSGL1500001_K20150511175323.mpg) Request", baseTime)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : 3894d674-d74b-4eca-a2ea-fafbfa1113a8, ClientID : 0, GLB IP : 125.159.40.5's file(F.mpg) Request", baseTime)
	fmt.Fprintln(f)
	// ------------------------------------------------- 테스트 기준 시각 + 1
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : 8b2e7e4a-270d-4586-85a1-e4284551176d, ClientID : 0, GLB IP : 125.159.40.5's file(F.mpg) Request", baseTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : f1893245-802b-45b1-b0fa-377bc1415b35, ClientID : 0, GLB IP : 125.159.40.5's file(F.mpg) Request", baseTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.93.100 Selected for Client StreamID : 3c68d2a5-354e-4a4b-b181-c724d16cf406, GLB IP : 125.147.36.6's file(MVHF201MSGL1500001_K20150216200556.mpg) Request", baseTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : d41dc03a-91b7-4c14-bb3a-a73823f333e0, ClientID : 0, GLB IP : 125.159.40.5's file(F.mpg) Request", baseTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,4,%d,File Not Found, UUID : d9672fe8-1b39-491f-a8a4-23bf7a6f096c, GLB IP : 125.144.96.6, M0200000SGL1065016_K20100826000000.MPG", baseTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : f8a21815-a75b-4fe4-8f6a-dba984ee7c6e, ClientID : 0, GLB IP : 125.159.40.5's file(F.mpg) Request", baseTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : 7148fadd-edab-4a48-8d27-4bf8c8b74cbd, ClientID : 0, GLB IP : 125.159.40.5's file(F.mpg) Request", baseTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : b121ae29-954c-4990-8e58-e102959d0239, ClientID : 0, GLB IP : 125.159.40.5's file(F.mpg) Request", baseTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.144.93.97 Selected for Client StreamID : 86fd80d3-2691-4274-9172-315d50e90801, ClientID : 0, GLB IP : 125.159.40.5's file(M34I502CSGL1500001_K20180512022857.mpg) Request", baseTime+1)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : 598131b3-8fb2-4415-b0f8-472d52ef054c, ClientID : 0, GLB IP : 125.159.40.5's file(F.mpg) Request", baseTime+1)
	fmt.Fprintln(f)
	// ------------------------------------------------- 테스트 기준 시각 + 2
	fmt.Fprintf(f, "0x40ffff,1,%d,Server 125.159.40.3 Selected for Client StreamID : 0590710c-bd2e-4863-941e-041877328d78, ClientID : 0, GLB IP : 125.159.40.5's file(F.mpg) Request", baseTime+2)
	fmt.Fprintln(f)

	f.Close()
}

// makeFileMetaMap :
// grade 파일과 hitcount 파일을 parsing 해서
// file meta map이 만들어진 상황 simulation
// 전체 파일은 A, B, C, D, E, F 이고,
// 127.0.0.1 에도 있고 127.0.0.2 에도 있는 파일은 B, C 이다.
// all fmm: A.mpg, B.mpg, C.mpg, D.mpg
// dup fmm: B.mpg, C.mpg, E.mpg, F.mpg
func makeFileMetaMap() (FileMetaPtrMap, FileMetaPtrMap) {
	fmm := make(FileMetaPtrMap)

	// put grade, size , and severs
	fmm["A.mpg"] = &common.FileMeta{
		Name:  "A.mpg",
		Grade: 1, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1}}

	fmm["B.mpg"] = &common.FileMeta{
		Name:  "B.mpg",
		Grade: 2, Size: 100, RisingHit: 0,
		ServerCount: 2,
		ServerIPs:   map[string]int{"127.0.0.1": 1, "127.0.0.2": 1}}

	fmm["C.mpg"] = &common.FileMeta{
		Name:  "C.mpg",
		Grade: 3, Size: 100, RisingHit: 0,
		ServerCount: 2,
		ServerIPs:   map[string]int{"127.0.0.1": 1, "127.0.0.2": 1}}

	fmm["D.mpg"] = &common.FileMeta{
		Name:  "D.mpg",
		Grade: 4, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1}}

	fmm["E.mpg"] = &common.FileMeta{
		Name:  "E.mpg",
		Grade: 5, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.2": 1}}

	fmm["F.mpg"] = &common.FileMeta{
		Name:  "F.mpg",
		Grade: 6, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.2": 1}}

	dupfmm := make(FileMetaPtrMap)
	dupfmm["B.mpg"] = fmm["B.mpg"]
	dupfmm["C.mpg"] = fmm["C.mpg"]

	return fmm, dupfmm
}
func makeRisingHitFileMap(files []string) map[string]int {
	risingHitFileMap := make(map[string]int)
	for _, file := range files {
		risingHitFileMap[file] = 1
	}
	return risingHitFileMap
}

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

func cfw(cfwaddr string, du common.DiskUsage, filenames []string) *httptest.Server {
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/df").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			bd, err := json.Marshal(du)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(bd)
		})
	router.Methods("GET").Path("/files").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			for _, fn := range filenames {
				fmt.Fprintln(w, fn)
			}
		})
	router.Methods("DELETE").Path("/files/{fileName}").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			fileName, exists := vars["fileName"]
			if !exists {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			for _, fn := range filenames {
				if fileName == fn {
					w.WriteHeader(http.StatusOK)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		})

	s := &http.Server{
		Addr:         cfwaddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	cfw := httptest.NewUnstartedServer(router)
	l, _ := net.Listen("tcp", cfwaddr)
	cfw.Listener.Close()
	cfw.Listener = l
	cfw.Config = s

	return cfw
}

func Test_FindServersOutOfDiskSpace(t *testing.T) {
	vs1 := "127.0.0.1:18881"
	vs2 := "127.0.0.1:18882"
	vs3 := "127.0.0.1:18883"
	files1 := []string{"A.mpg"}
	files2 := []string{"B.mpg"}
	files3 := []string{"C.mpg"}

	// setup dummy http server
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	cfw1 := cfw(vs1, d1, files1)

	d2 := common.DiskUsage{
		TotalSize: 2000, UsedSize: 1000,
		FreeSize: 1000, AvailSize: 1000, UsedPercent: 50,
	}
	cfw2 := cfw(vs2, d2, files2)

	d3 := common.DiskUsage{
		TotalSize: 3000, UsedSize: 1500,
		FreeSize: 1500, AvailSize: 1500, UsedPercent: 50,
	}
	cfw3 := cfw(vs3, d3, files3)

	cfw1.Start()
	defer cfw1.Close()
	cfw2.Start()
	defer cfw2.Close()
	cfw3.Start()
	defer cfw3.Close()

	hosts := common.NewHosts()
	hosts.Add(vs1)
	hosts.Add(vs2)
	hosts.Add(vs3)

	SetDiskUsageLimitPercent(55)
	dservers := FindServersOutOfDiskSpace(hosts)

	assert.Equal(t, 1, len(dservers))

	if len(dservers) < 1 {
		t.Fatal("cannot find server that run out of disk space")
	}
	overuseServer := dservers[0]
	assert.Equal(t, vs1, overuseServer.Addr)
	assert.Equal(t, uint(60), overuseServer.Du.UsedPercent)

	overUsedSize := overuseServer.Du.GetOverUsedSize(DiskUsageLimitPercent())
	assert.Less(t, uint64(0), uint64(overUsedSize))

	// 50 = 600(current used) - 550(limit used : 1000 * 55%)
	assert.Equal(t, uint64(50), uint64(overUsedSize))
}

func TestSetDiskUsageLimitPercent(t *testing.T) {
	assert.NotNil(t, SetDiskUsageLimitPercent(101))
	assert.Nil(t, SetDiskUsageLimitPercent(0))
	assert.Nil(t, SetDiskUsageLimitPercent(50))
}

func Test_selectFileMetas(t *testing.T) {
	hosts := common.NewHosts()
	vs1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	vs2 := "127.0.0.2:18882"
	files2 := []string{"B.mpg", "C.mpg", "E.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	hosts.Add(vs1)
	hosts.Add(vs2)

	cfw1 := cfw(vs1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()

	cfw2 := cfw(vs2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	allfmm, _ := makeFileMetaMap()

	// sort 되어서 hosts[0]은 vs2가 됨
	host2 := (*hosts)[0]
	host2fmm, err := selectFileMetas(host2, allfmm)
	if err != nil {
		assert.Error(t, err)
	}
	t.Logf("server:%s -> %s", host2, host2fmm)
	assert.Equal(t, 3, len(host2fmm))
	assert.Contains(t, host2fmm, "B.mpg")
	assert.Contains(t, host2fmm, "C.mpg")
	assert.Contains(t, host2fmm, "E.mpg")

	// sort 되어서 hosts[1]은 vs2가 됨
	host1 := (*hosts)[1]
	host1fmm, err := selectFileMetas(host1, allfmm)
	if err != nil {
		assert.Error(t, err)
	}
	t.Logf("server:%s -> %s", host1, host1fmm)
	assert.Equal(t, 4, len(host1fmm))
	assert.Contains(t, host1fmm, "A.mpg")
	assert.Contains(t, host1fmm, "B.mpg")
	assert.Contains(t, host1fmm, "C.mpg")
	assert.Contains(t, host1fmm, "D.mpg")

}

func Test_selectFileMetasS1(t *testing.T) {
	hosts := common.NewHosts()
	vs1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	hosts.Add(vs1)

	cfw1 := cfw(vs1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()

	allfmm, _ := makeFileMetaMap()

	host1 := (*hosts)[0]
	host1fmm, err := selectFileMetas(host1, allfmm)
	if err != nil {
		assert.Error(t, err)
	}
	t.Logf("server:%s -> %s", host1, host1fmm)
	assert.Equal(t, len(host1fmm), 4)
	assert.Contains(t, host1fmm, "A.mpg")
	assert.Contains(t, host1fmm, "B.mpg")
	assert.Contains(t, host1fmm, "C.mpg")
	assert.Contains(t, host1fmm, "D.mpg")
}

func Test_selectFileMetasS2(t *testing.T) {
	hosts := common.NewHosts()

	vs2 := "127.0.0.2:18882"
	files2 := []string{"B.mpg", "C.mpg", "E.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	hosts.Add(vs2)

	cfw2 := cfw(vs2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	allfmm, _ := makeFileMetaMap()

	host2 := (*hosts)[0]
	host2fmm, err := selectFileMetas(host2, allfmm)
	if err != nil {
		assert.Error(t, err)
	}
	t.Logf("server:%s -> %s", host2, host2fmm)
	assert.Equal(t, 3, len(host2fmm))
	assert.Contains(t, host2fmm, "B.mpg")
	assert.Contains(t, host2fmm, "C.mpg")
	assert.Contains(t, host2fmm, "E.mpg")
}

func Test_selectFileMetasS3(t *testing.T) {
	hosts := common.NewHosts()

	vs2 := "127.0.0.2:18882"
	files2 := []string{"B.mpg", "E.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	hosts.Add(vs2)

	cfw2 := cfw(vs2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	allfmm, _ := makeFileMetaMap()

	host2 := (*hosts)[0]
	host2fmm, err := selectFileMetas(host2, allfmm)
	if err != nil {
		assert.Error(t, err)
	}
	t.Logf("server:%s -> %s", host2, host2fmm)

	// Server2 에 B.mpg, C.mpg, E.mpg, F.mpg 가 있다고 meta 가 만들어져있지만,
	// 실제로 Server2 에 B.mpg, E.mpg 와 없으므로
	// server 의 file meta 에는 B.mpg, E.mpg만 들어있음
	assert.Equal(t, 2, len(host2fmm))
	assert.Contains(t, host2fmm, "B.mpg")
	assert.Contains(t, host2fmm, "E.mpg")
}

func Test_selectFileMetasSNull(t *testing.T) {
	hosts := common.NewHosts()

	vs2 := "127.0.0.2:18882"
	files2 := []string{}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	hosts.Add(vs2)

	cfw2 := cfw(vs2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	allfmm, _ := makeFileMetaMap()

	host2 := (*hosts)[0]
	host2fmm, err := selectFileMetas(host2, allfmm)
	if err != nil {
		assert.Error(t, err)
	}
	t.Logf("server:%s -> %s", host2, host2fmm)

	// Server2 에 B.mpg, C.mpg, E.mpg 가 있다고 meta 가 만들어져있지만,
	// 실제로 Server2 에 아무 파일도 없으므로,
	// meta 에도 아무 것도 안들어있음
	assert.Equal(t, 0, len(host2fmm))
}

func Test_getServerFileMetas(t *testing.T) {
	Servers = common.NewHosts()
	vs1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	vs2 := "127.0.0.2:18882"
	files2 := []string{"B.mpg", "C.mpg", "E.mpg", "F.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	Servers.Add(vs1)
	Servers.Add(vs2)

	cfw1 := cfw(vs1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()

	cfw2 := cfw(vs2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	allfmm, _ := makeFileMetaMap()
	ssfmm := getServerFileMetas(allfmm)
	for s, sfmm := range ssfmm {
		t.Logf("server metas:%s -> %s", s, sfmm)
	}

	assert.Equal(t, 2, len(ssfmm))
	s1fmm := ssfmm[vs1]
	assert.Equal(t, 4, len(s1fmm))
	assert.Contains(t, s1fmm, "A.mpg")
	assert.Contains(t, s1fmm, "B.mpg")
	assert.Contains(t, s1fmm, "C.mpg")
	assert.Contains(t, s1fmm, "D.mpg")

	s2fmm := ssfmm[vs2]
	assert.Equal(t, 4, len(s2fmm))
	assert.Contains(t, s2fmm, "B.mpg")
	assert.Contains(t, s2fmm, "C.mpg")
	assert.Contains(t, s2fmm, "E.mpg")
	assert.Contains(t, s2fmm, "F.mpg")

}

func Test_getServerFileMetasServerNull(t *testing.T) {
	Servers = common.NewHosts()
	vs1 := "127.0.0.1:18881"
	files1 := []string{}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	vs2 := "127.0.0.2:18882"
	files2 := []string{}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}

	Servers.Add(vs1)
	Servers.Add(vs2)

	cfw1 := cfw(vs1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()

	cfw2 := cfw(vs2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	allfmm, _ := makeFileMetaMap()
	ssfmm := getServerFileMetas(allfmm)
	for s, sfmm := range ssfmm {
		t.Logf("server metas:%s -> %s", s, sfmm)
	}

	assert.Equal(t, 2, len(ssfmm))
	s1fmm := ssfmm[vs1]
	assert.Equal(t, 0, len(s1fmm))

	s2fmm := ssfmm[vs2]
	assert.Equal(t, 0, len(s2fmm))
}

func Test_updateFileMetasForDuplicatedFiles(t *testing.T) {
	Servers = common.NewHosts()
	vs1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	vs2 := "127.0.0.2:18882"
	files2 := []string{"E.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	Servers.Add(vs1)
	Servers.Add(vs2)
	cfw1 := cfw(vs1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()
	cfw2 := cfw(vs2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	allfmm, dupfmm := makeFileMetaMap()

	ssfmm := getServerFileMetas(allfmm)
	for s, sfmm := range ssfmm {
		t.Logf("server metas:%s -> %s", s, sfmm)
	}

	assert.Equal(t, 2, len(ssfmm))
	s1fmm := ssfmm[vs1]

	// 서버1 에 B.mpg 가 없으므로, s1fmm 에는 B.mpg 는 들어있지 않다.
	assert.Equal(t, 3, len(s1fmm))
	assert.Contains(t, s1fmm, "A.mpg")
	assert.Contains(t, s1fmm, "C.mpg")
	assert.Contains(t, s1fmm, "D.mpg")

	// 서버2 에 B.mpg, C.mpg, F.mpg가 없으므로, s2fmm 에는 B.mpg, C.mpg, F.mpg 는 들어있지 않다.
	s2fmm := ssfmm[vs2]
	assert.Equal(t, 1, len(s2fmm))
	assert.Contains(t, s2fmm, "E.mpg")

	// 전체 파일 정보에서는 B.mpg 가 Server1, Serve2 에 있다고 되어있지만
	// 전체 파일 정보에서는 C.mpg 가 Server1, Serve2 에 있다고 되어있지만
	assert.Equal(t, 2, dupfmm["B.mpg"].ServerCount)
	assert.Equal(t, 2, dupfmm["C.mpg"].ServerCount)

	// updateAllFileMetasForDuplicatedFiles 호출하고 나면,
	updateFileMetasForDuplicatedFiles(dupfmm, ssfmm)

	// Server1, Server2 의 파일 meta 정보가 반영되어
	// B.mpg의 serverCount 값이 2에서 0로 바뀐다.
	assert.Equal(t, 0, dupfmm["B.mpg"].ServerCount)
	// pointer 를 가지고 변경하므로, 전체 파일 meta 에 저장된 정보도 바뀐다.
	assert.Equal(t, 0, allfmm["B.mpg"].ServerCount)

	// Server1, Server2 의 파일 meta 정보가 반영되어
	// C.mpg의 serverCount 값이 2에서 1로 바뀐다.
	assert.Equal(t, 1, dupfmm["C.mpg"].ServerCount)
	// pointer 를 가지고 변경하므로, 전체 파일 meta 에 저장된 정보도 바뀐다.
	assert.Equal(t, 1, allfmm["C.mpg"].ServerCount)

	for s, sfmm := range ssfmm {
		t.Logf("server metas:%s -> %s", s, sfmm)
	}
	t.Logf("duplicated metas:%s", dupfmm)

}

func Test_updateFileMetasForDuplicatedFilesServerNull(t *testing.T) {
	Servers = common.NewHosts()
	vs1 := "127.0.0.1:18881"
	files1 := []string{}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	vs2 := "127.0.0.2:18882"
	files2 := []string{}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	Servers.Add(vs1)
	Servers.Add(vs2)
	cfw1 := cfw(vs1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()
	cfw2 := cfw(vs2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	allfmm, dupfmm := makeFileMetaMap()

	ssfmm := getServerFileMetas(allfmm)
	for s, sfmm := range ssfmm {
		t.Logf("server metas:%s -> %s", s, sfmm)
	}

	assert.Equal(t, 2, len(ssfmm))
	s1fmm := ssfmm[vs1]

	// 서버1 에 아무 파일도 없으므로, s1fmm는 비어있다.
	assert.Equal(t, 0, len(s1fmm))

	// 서버2 에 아무 파일도 없으므로, s2fmm는 비어있다.
	s2fmm := ssfmm[vs2]
	assert.Equal(t, 0, len(s2fmm))

	// 전체 파일 정보에서는 B.mpg 가 Server1, Serve2 에 있다고 되어있지만
	// 전체 파일 정보에서는 C.mpg 가 Server1, Serve2 에 있다고 되어있지만
	assert.Equal(t, 2, dupfmm["B.mpg"].ServerCount)
	assert.Equal(t, 2, dupfmm["C.mpg"].ServerCount)

	// updateAllFileMetasForDuplicatedFiles 호출하고 나면,
	updateFileMetasForDuplicatedFiles(dupfmm, ssfmm)

	// Server1, Server2 의 파일 meta 정보가 반영되어
	// B.mpg의 serverCount 값이 2에서 0로 바뀐다.
	assert.Equal(t, 0, dupfmm["B.mpg"].ServerCount)
	// pointer 를 가지고 변경하므로, 전체 파일 meta 에 저장된 정보도 바뀐다.
	assert.Equal(t, 0, allfmm["B.mpg"].ServerCount)

	// Server1, Server2 의 파일 meta 정보가 반영되어
	// C.mpg의 serverCount 값이 2에서 0로 바뀐다.
	assert.Equal(t, 0, dupfmm["C.mpg"].ServerCount)
	// pointer 를 가지고 변경하므로, 전체 파일 meta 에 저장된 정보도 바뀐다.
	assert.Equal(t, 0, allfmm["C.mpg"].ServerCount)

	for s, sfmm := range ssfmm {
		t.Logf("server metas:%s -> %s", s, sfmm)
	}
	t.Logf("duplicated metas:%s", dupfmm)

	// 중복된 파일이 아닌 파일의 정보는 바뀌지 않는다.
	assert.Equal(t, 1, allfmm["A.mpg"].ServerCount)
	assert.Equal(t, 1, allfmm["D.mpg"].ServerCount)
	assert.Equal(t, 1, allfmm["E.mpg"].ServerCount)
}

func Test_requestRemoveDuplicatedFiles(t *testing.T) {
	Servers = common.NewHosts()
	vs1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	vs2 := "127.0.0.2:18882"
	files2 := []string{"C.mpg", "E.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	Servers.Add(vs1)
	Servers.Add(vs2)
	cfw1 := cfw(vs1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()
	cfw2 := cfw(vs2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	allfmm, dupfmm := makeFileMetaMap()

	ssfmm := getServerFileMetas(allfmm)
	for s, sfmm := range ssfmm {
		t.Logf("server metas:%s -> %s", s, sfmm)
	}

	assert.Equal(t, 2, len(ssfmm))
	s1fmm := ssfmm[vs1]

	// 서버1 에 B.mpg 가 없으므로, s1fmm 에는 B.mpg 는 들어있지 않다.
	assert.Equal(t, 3, len(s1fmm))
	assert.Contains(t, s1fmm, "A.mpg")
	assert.Contains(t, s1fmm, "C.mpg")
	assert.Contains(t, s1fmm, "D.mpg")

	// 서버2 에 B.mpg가 없으므로, s2fmm 에는 B.mpg는 들어있지 않다.
	s2fmm := ssfmm[vs2]
	assert.Equal(t, 2, len(s2fmm))
	assert.Contains(t, s1fmm, "C.mpg")
	assert.Contains(t, s2fmm, "E.mpg")

	// 전체 파일 정보에서는 B.mpg 가 Server1, Serve2 에 있다고 되어있지만
	// 전체 파일 정보에서는 C.mpg 가 Server1, Serve2 에 있다고 되어있지만
	assert.Equal(t, 2, dupfmm["B.mpg"].ServerCount)
	assert.Equal(t, 2, dupfmm["C.mpg"].ServerCount)

	// updateAllFileMetasForDuplicatedFiles 호출하고 나면,
	updateFileMetasForDuplicatedFiles(dupfmm, ssfmm)

	// Server1, Server2 의 파일 meta 정보가 반영되어
	// B.mpg의 serverCount 값이 2에서 0로 바뀐다.
	assert.Equal(t, 0, dupfmm["B.mpg"].ServerCount)
	// pointer 를 가지고 변경하므로, 전체 파일 meta 에 저장된 정보도 바뀐다.
	assert.Equal(t, 0, allfmm["B.mpg"].ServerCount)

	// Server1, Server2 의 파일 meta 정보가 반영되지만
	// C.mpg의 serverCount 값은 그대로 2이다.
	assert.Equal(t, 2, dupfmm["C.mpg"].ServerCount)
	assert.Equal(t, 2, allfmm["C.mpg"].ServerCount)

	for s, sfmm := range ssfmm {
		t.Logf("server metas:%s -> %s", s, sfmm)
	}
	t.Logf("duplicated metas:%s", dupfmm)
	// 중복된 파일이 아닌 파일의 정보는 바뀌지 않는다.
	assert.Equal(t, 1, allfmm["A.mpg"].ServerCount)
	assert.Equal(t, 1, allfmm["D.mpg"].ServerCount)
	assert.Equal(t, 1, allfmm["E.mpg"].ServerCount)

	// Serve1 또는 Server1에 C.mpg 삭제 요청을 날린다.
	requestRemoveDuplicatedFiles(dupfmm, ssfmm)

	// B.mpg의 server count 정보가 0이므로, B.mpg에 대해서는 요청을 하지 않는다.
	// 값도 변함이 없다.
	assert.Equal(t, 0, dupfmm["B.mpg"].ServerCount)

	// 요청이 성공하면,
	// C.mpg의 serverCount 값은 1로 바뀐다.
	assert.Equal(t, 1, dupfmm["C.mpg"].ServerCount)

	t.Logf("duplicated metas:%s", dupfmm)

	// 중복된 파일이 아닌 파일의 정보는 바뀌지 않는다.
	assert.Equal(t, 1, allfmm["A.mpg"].ServerCount)
	assert.Equal(t, 1, allfmm["D.mpg"].ServerCount)
	assert.Equal(t, 1, allfmm["E.mpg"].ServerCount)
}

func Test_getFileListToDeleteForFreeDiskSpace(t *testing.T) {
	s1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	s2 := "127.0.0.2:18882"
	files2 := []string{"B.mpg", "C.mpg", "E.mpg", "F.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	Servers.Add(s1)
	Servers.Add(s2)
	cfw1 := cfw(s1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()
	cfw2 := cfw(s2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	rhfiles := []string{"F.mpg"}
	rhitfmm := makeRisingHitFileMap(rhfiles)
	allfmm, dupfmm := makeFileMetaMap()
	ssfmm := getServerFileMetas(allfmm)
	updateFileMetasForDuplicatedFiles(dupfmm, ssfmm)

	SetDiskUsageLimitPercent(55)
	servers := FindServersOutOfDiskSpace(Servers)

	base := "testsourcefolder"
	SourcePath.Add(base)

	for _, f1 := range files1 {
		createfile(base, f1)
	}
	for _, f2 := range files2 {
		createfile(base, f2)
	}
	deletefile(base, "C.mpg")

	defer deletefile(base, "")

	ignores := []string{"E"}
	SetIgnorePrefixes(ignores)

	D := &common.FileMeta{
		Name:  "D.mpg",
		Grade: 4, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1}}

	B := &common.FileMeta{
		Name:  "B.mpg",
		Grade: 2, Size: 100, RisingHit: 0,
		ServerCount: 2,
		ServerIPs:   map[string]int{"127.0.0.1": 1, "127.0.0.2": 1}}

	for _, server := range servers {
		dels := getFileListToDeleteForFreeDiskSpace(server, ssfmm, rhitfmm)
		t.Logf("files to delete: %s -> %s", server, dels)

		overUsedSize := server.Du.GetOverUsedSize(DiskUsageLimitPercent())
		switch server.Addr {
		// s1 의 경우,
		// 200 만큼 over 해서 사용했으므로, file (size 100) 두 개 지워야 함
		// 등급이 제일 낮은 D.mpg, C.mpg가 지워져야 함
		// C.mpg 는 source path 에 존재하지 않으므로, B.mpg가 지워져야 함
		// D와 B가 지워져야 함
		case s1:
			// 200 = 750(current used) - 550(limit used : 1000 * 55%)
			assert.Equal(t, uint64(200), uint64(overUsedSize))
			assert.Equal(t, 2, len(dels))
			assert.Contains(t, dels, D)
			assert.Contains(t, dels, B)
		// s2 의 경우
		// 50 만큼 over 해서 사용했으므로, file (size 100) 한 개 지워야 함
		// 등급이 제일 낮은 F.mpg가 지워져야 하지만,
		// F.mpg 는 risinghit 파일에 속하므로, E.mpg 가 지워져야 하지만,
		// E.mpg 는 ignore prefix에 속하므로, C.mpg 가 지워져야 하지만,
		// C.mpg 는 source path 에 존재하지 않으므로, B.mpg가 지워져야 함
		// B가 지워져야 함
		case s2:
			// 50 = 600(current used) - 550(limit used : 1000 * 55%)
			assert.Equal(t, uint64(50), uint64(overUsedSize))
			assert.Equal(t, 1, len(dels))
			assert.Contains(t, dels, B)
		}
	}

}

func Test_requestRemoveFilesForFreeDiskSpace(t *testing.T) {
	s1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	s2 := "127.0.0.2:18882"
	files2 := []string{"B.mpg", "C.mpg", "E.mpg", "F.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	Servers.Add(s1)
	Servers.Add(s2)
	cfw1 := cfw(s1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()
	cfw2 := cfw(s2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	rhfiles := []string{"F.mpg"}
	rhitfmm := makeRisingHitFileMap(rhfiles)
	allfmm, dupfmm := makeFileMetaMap()
	ssfmm := getServerFileMetas(allfmm)
	updateFileMetasForDuplicatedFiles(dupfmm, ssfmm)

	SetDiskUsageLimitPercent(55)
	servers := FindServersOutOfDiskSpace(Servers)

	base := "testsourcefolder"
	SourcePath.Add(base)

	for _, f1 := range files1 {
		createfile(base, f1)
	}
	for _, f2 := range files2 {
		createfile(base, f2)
	}
	deletefile(base, "C.mpg")

	defer deletefile(base, "")

	ignores := []string{"E"}
	SetIgnorePrefixes(ignores)

	// s1 의 경우,
	// 200 만큼 over 해서 사용했으므로, file (size 100) 두 개 지워야 함
	// 등급이 제일 낮은 D.mpg, C.mpg가 지워져야 함
	// C.mpg 는 source path 에 존재하지 않으므로, B.mpg가 지워져야 함
	// D와 B가 지워져야 함

	// s2 의 경우
	// 50 만큼 over 해서 사용했으므로, file (size 100) 한 개 지워야 함
	// 등급이 제일 낮은 F.mpg가 지워져야 하지만,
	// F.mpg 는 risinghit 파일에 속하므로, E.mpg 가 지워져야 하지만,
	// E.mpg 는 ignore prefix에 속하므로, C.mpg 가 지워져야 하지만,
	// C.mpg 는 source path 에 존재하지 않으므로, B.mpg가 지워져야 함
	// B가 지워져야 함
	requestRemoveFilesForFreeDiskSpace(servers, ssfmm, rhitfmm)

	// D 는 s1 에서 지워졌으므로, count 가 하나 준다.
	// D.ServerCount : 1 --> 0
	assert.Equal(t, 0, allfmm["D.mpg"].ServerCount)
	// B 는 s1, s2에서 지워졌으므로, count 가 두 개 준다.
	// B.ServerCount : 2 --> 0
	assert.Equal(t, 0, allfmm["B.mpg"].ServerCount)
}

// duplicate 되어서 삭제요청한 파일이 다시 disk 용량으로 삭제대상이 될 순 없는 것이 아닌지 test
func Test_requestRemoveDuplicatedFilesAndgetFileListToDeleteForFreeDiskSpace(t *testing.T) {
	s1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	s2 := "127.0.0.2:18882"
	files2 := []string{"B.mpg", "C.mpg", "E.mpg", "F.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	Servers.Add(s1)
	Servers.Add(s2)

	cfw1 := cfw(s1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()
	cfw2 := cfw(s2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	base := "testsourcefolder"
	SourcePath.Add(base)
	for _, f1 := range files1 {
		createfile(base, f1)
	}
	for _, f2 := range files2 {
		createfile(base, f2)
	}
	deletefile(base, "C.mpg")
	defer deletefile(base, "")

	rhfiles := []string{"F.mpg"}
	rhitfmm := makeRisingHitFileMap(rhfiles)
	allfmm, dupfmm := makeFileMetaMap()
	ssfmm := getServerFileMetas(allfmm)
	updateFileMetasForDuplicatedFiles(dupfmm, ssfmm)

	// sort 순서에 따라서, s2에 B.mpg 삭제 요청을 날린다.
	// sort 순서에 따라서, s2에 C.mpg 삭제 요청을 날린다.
	requestRemoveDuplicatedFiles(dupfmm, ssfmm)
	// 요청이 성공하면, B.mpg의 serverCount 값은 1로 바뀐다.
	assert.Equal(t, 1, allfmm["B.mpg"].ServerCount)
	assert.Equal(t, 1, dupfmm["B.mpg"].ServerCount)

	// 요청이 성공하면, C.mpg의 serverCount 값은 1로 바뀐다.
	assert.Equal(t, 1, allfmm["C.mpg"].ServerCount)
	assert.Equal(t, 1, dupfmm["C.mpg"].ServerCount)

	SetDiskUsageLimitPercent(55)
	servers := FindServersOutOfDiskSpace(Servers)

	ignores := []string{"E"}
	SetIgnorePrefixes(ignores)

	D := &common.FileMeta{
		Name:  "D.mpg",
		Grade: 4, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1}}

	// 원래 server count 가 2 였지만,
	// 중복파일로 s2에서 지워져서 count 가 1로 바뀌고,
	// ServerIPs 의 "127.0.0.2" 값도 0 으로 바뀐다.
	B := &common.FileMeta{
		Name:  "B.mpg",
		Grade: 2, Size: 100, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1, "127.0.0.2": 0}}

	for _, server := range servers {
		dels := getFileListToDeleteForFreeDiskSpace(server, ssfmm, rhitfmm)
		t.Logf("files to delete: %s -> %s", server, dels)

		overUsedSize := server.Du.GetOverUsedSize(DiskUsageLimitPercent())
		switch server.Addr {
		// s1 의 경우,
		// 200 만큼 over 해서 사용했으므로, file (size 100) 두 개 지워야 함
		// 등급이 제일 낮은 D.mpg, C.mpg가 지워져야 함
		// C.mpg 는 source path 에 존재하지 않으므로, B.mpg가 지워져야 함
		// C.mpg 가 중복으로 이미 지워진 경우에도, B.mpg가 지워져야 함
		// D와 B가 지워져야 함
		case s1:
			// 200 = 750(current used) - 550(limit used : 1000 * 55%)
			assert.Equal(t, uint64(200), uint64(overUsedSize))
			assert.Equal(t, 2, len(dels))
			assert.Contains(t, dels, D)
			assert.Contains(t, dels, B)
		// s2 의 경우
		// 50 만큼 over 해서 사용했으므로, file (size 100) 한 개 지워야 함
		// 등급이 제일 낮은 F.mpg가 지워져야 하지만,
		// F.mpg 는 risinghit 파일에 속하므로, E.mpg 가 지워져야 하지만,
		// E.mpg 는 ignore prefix에 속하므로, C.mpg 가 지워져야 하지만,
		// C.mpg 는 source path 에 존재하지 않으므로, B.mpg가 지워져야 함
		// C.mpg 가 중복으로 이미 지워진 경우에도, B.mpg가 지워져야 함
		// B가 지워져야 하지만,
		// B.mpg 가 중복으로 이미 지워졌으므로, 지워질 게 없음
		case s2:
			// 50 = 600(current used) - 550(limit used : 1000 * 55%)
			assert.Equal(t, uint64(50), uint64(overUsedSize))
			assert.Equal(t, 0, len(dels))
		}
	}
}

func Test_RunWithInfo(t *testing.T) {
	s1 := "127.0.0.1:18881"
	files1 := []string{"A.mpg", "B.mpg", "C.mpg", "D.mpg"}
	d1 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 750,
		FreeSize: 250, AvailSize: 250, UsedPercent: 75,
	}
	s2 := "127.0.0.2:18882"
	files2 := []string{"B.mpg", "C.mpg", "E.mpg", "F.mpg"}
	d2 := common.DiskUsage{
		TotalSize: 1000, UsedSize: 600,
		FreeSize: 400, AvailSize: 400, UsedPercent: 60,
	}
	Servers.Add(s1)
	Servers.Add(s2)

	cfw1 := cfw(s1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()
	cfw2 := cfw(s2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	base := "testsourcefolder"
	SourcePath.Add(base)
	for _, f1 := range files1 {
		createfile(base, f1)
	}
	for _, f2 := range files2 {
		createfile(base, f2)
	}
	deletefile(base, "C.mpg")
	defer deletefile(base, "")

	ignores := []string{"E"}
	SetIgnorePrefixes(ignores)

	SetDiskUsageLimitPercent(55)

	rhfiles := []string{"F.mpg"}
	rhitfmm := makeRisingHitFileMap(rhfiles)
	allfmm, dupfmm := makeFileMetaMap()

	remover.Debugf("call 1st RunWithInfo ---------------------------------------")

	RunWithInfo(allfmm, dupfmm, rhitfmm)

	remover.Debugf("after 1st call RunWithInfo ---------------------------------")
	remover.Debugf("B: %s", allfmm["A.mpg"])
	remover.Debugf("B: %s", allfmm["B.mpg"])
	remover.Debugf("C: %s", allfmm["C.mpg"])
	remover.Debugf("C: %s", allfmm["D.mpg"])
	remover.Debugf("C: %s", allfmm["E.mpg"])
	remover.Debugf("C: %s", allfmm["F.mpg"])

	//A.mpg 는 S1에는 그대로 있음
	// S2에는 원래 없었음
	assert.Equal(t, 1, allfmm["A.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["A.mpg"].ServerIPs["127.0.0.2"])

	//B.mpg 는 S1, S2 중복이었기 때문에, S1에는 그대로 있고, S2에서 삭제
	// S1 에서는 disk 용량 부족으로 삭제
	assert.Equal(t, 0, allfmm["B.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["B.mpg"].ServerIPs["127.0.0.2"])

	//C.mpg 는 S1, S2 중복이었기 때문에, S1에는 그대로 있고, S2에서 삭제
	// San에 없는 파일이어서, S1이 disk 부족이지만, 지워지지 않음
	assert.Equal(t, 1, allfmm["C.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["C.mpg"].ServerIPs["127.0.0.2"])

	//D.mpg 는 S1 disk 용량 부족으로 S1에서 삭제
	// S2에는 원래 없었음
	assert.Equal(t, 0, allfmm["D.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["D.mpg"].ServerIPs["127.0.0.2"])

	//E.mpg 는 ignore prefix 때문에 S2에 그대로 있음
	// S1에는 원래 없었음
	assert.Equal(t, 0, allfmm["E.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 1, allfmm["E.mpg"].ServerIPs["127.0.0.2"])

	//F.mpg 는 급 hit 상승 파일이어서 S2에 그대로 있음
	// S1에는 원래 없었음
	assert.Equal(t, 0, allfmm["E.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 1, allfmm["F.mpg"].ServerIPs["127.0.0.2"])

	remover.Debugf("call 2nd RunWithInfo ---------------------------------------")

	//dupfmm 정보는 잘못되어있지만, meta 정보는 update 되어있는 상태,
	//만일 한 번 더 실행한다면,
	RunWithInfo(allfmm, dupfmm, rhitfmm)

	remover.Debugf("after 2nd call RunWithInfo ---------------------------------")
	remover.Debugf("B: %s", allfmm["A.mpg"])
	remover.Debugf("B: %s", allfmm["B.mpg"])
	remover.Debugf("C: %s", allfmm["C.mpg"])
	remover.Debugf("C: %s", allfmm["D.mpg"])
	remover.Debugf("C: %s", allfmm["E.mpg"])
	remover.Debugf("C: %s", allfmm["F.mpg"])

	//S2는 지워질 게 더 이상 없음
	//A.mpg 만 S1 disk 용량 부족으로 S1에서 삭제
	// S2에는 원래 없었음
	assert.Equal(t, 0, allfmm["A.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["A.mpg"].ServerIPs["127.0.0.2"])

	//B.mpg 는 이미 지워짐
	assert.Equal(t, 0, allfmm["B.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["B.mpg"].ServerIPs["127.0.0.2"])

	//C.mpg 는 San에 없는 파일이어서, S1이 disk 부족이지만, 지워지지 않음
	// S2 에서는 이미 지워짐
	assert.Equal(t, 1, allfmm["C.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["C.mpg"].ServerIPs["127.0.0.2"])

	//D.mpg 는 이미 S1에서 지워짐
	// S2에는 원래 없었음
	assert.Equal(t, 0, allfmm["D.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["D.mpg"].ServerIPs["127.0.0.2"])

	//E.mpg 는 ignore prefix 때문에 S2에 그대로 있음
	// S1에는 원래 없었음
	assert.Equal(t, 0, allfmm["E.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 1, allfmm["E.mpg"].ServerIPs["127.0.0.2"])

	//F.mpg 는 급 hit 상승 파일이어서 S2에 그대로 있음
	// S1에는 원래 없었음
	assert.Equal(t, 0, allfmm["E.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 1, allfmm["F.mpg"].ServerIPs["127.0.0.2"])

	//dupfmm 정보는 잘못되어있지만, meta 정보는 update 되어있는 상태,
	//만일 한 번 더 실행한다면,
	// 더 이상 지워질 파일이 없음
	// 2nd call 결과와 같음
	remover.Debugf("call 3rd RunWithInfo ---------------------------------------")

	RunWithInfo(allfmm, dupfmm, rhitfmm)

	remover.Debugf("after 3rd call RunWithInfo ---------------------------------")
	remover.Debugf("B: %s", allfmm["A.mpg"])
	remover.Debugf("B: %s", allfmm["B.mpg"])
	remover.Debugf("C: %s", allfmm["C.mpg"])
	remover.Debugf("C: %s", allfmm["D.mpg"])
	remover.Debugf("C: %s", allfmm["E.mpg"])
	remover.Debugf("C: %s", allfmm["F.mpg"])

	assert.Equal(t, 0, allfmm["A.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["A.mpg"].ServerIPs["127.0.0.2"])

	//B.mpg 는 이미 지워짐
	assert.Equal(t, 0, allfmm["B.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["B.mpg"].ServerIPs["127.0.0.2"])

	//C.mpg 는 San에 없는 파일이어서, S1이 disk 부족이지만, 지워지지 않음
	// S2 에서는 이미 지워짐
	assert.Equal(t, 1, allfmm["C.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["C.mpg"].ServerIPs["127.0.0.2"])

	//D.mpg 는 이미 S1에서 지워짐
	// S2에는 원래 없었음
	assert.Equal(t, 0, allfmm["D.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 0, allfmm["D.mpg"].ServerIPs["127.0.0.2"])

	//E.mpg 는 ignore prefix 때문에 S2에 그대로 있음
	// S1에는 원래 없었음
	assert.Equal(t, 0, allfmm["E.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 1, allfmm["E.mpg"].ServerIPs["127.0.0.2"])

	//F.mpg 는 급 hit 상승 파일이어서 S2에 그대로 있음
	// S1에는 원래 없었음
	assert.Equal(t, 0, allfmm["E.mpg"].ServerIPs["127.0.0.1"])
	assert.Equal(t, 1, allfmm["F.mpg"].ServerIPs["127.0.0.2"])

}
