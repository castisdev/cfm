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
	"github.com/gorilla/mux"
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
			w.WriteHeader(http.StatusOK)
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

// makeFileMetaMap :
// grade 파일과 hitcount 파일을 parsing 해서
// file meta map이 만들어진 상황 simulation
// 전체 파일은 A, B, C, D 이고,
// 127.0.0.1 에도 있고 127.0.0.2 에도 있는 파일은 B, C 이다.
// all fmm: A.mpg, B.mpg, C.mpg, D.mpg, E.mpg
// dup fmm: B.mpg, C.mpg
func makeFileMetaMap() (FileMetaPtrMap, FileMetaPtrMap) {
	fmm := make(FileMetaPtrMap)

	// put grade, size , and severs
	fmm["A.mpg"] = &common.FileMeta{
		Name:  "A.mpg",
		Grade: 1, Size: 1000, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1}}

	fmm["B.mpg"] = &common.FileMeta{
		Name:  "B.mpg",
		Grade: 2, Size: 1000, RisingHit: 0,
		ServerCount: 2,
		ServerIPs:   map[string]int{"127.0.0.1": 1, "127.0.0.2": 1}}

	fmm["C.mpg"] = &common.FileMeta{
		Name:  "C.mpg",
		Grade: 3, Size: 1000, RisingHit: 0,
		ServerCount: 2,
		ServerIPs:   map[string]int{"127.0.0.1": 1, "127.0.0.2": 1}}

	fmm["D.mpg"] = &common.FileMeta{
		Name:  "D.mpg",
		Grade: 4, Size: 1000, RisingHit: 0,
		ServerCount: 1,
		ServerIPs:   map[string]int{"127.0.0.1": 1}}

	fmm["E.mpg"] = &common.FileMeta{
		Name:  "E.mpg",
		Grade: 5, Size: 1000, RisingHit: 0,
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

	host1 := (*hosts)[0]
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

	host2 := (*hosts)[1]
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

	// Server2 에 B.mpg, C.mpg, E.mpg 가 있다고 meta 가 만들어져있지만,
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
	files2 := []string{"B.mpg", "C.mpg", "E.mpg"}
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
	assert.Equal(t, 3, len(s2fmm))
	assert.Contains(t, s2fmm, "B.mpg")
	assert.Contains(t, s2fmm, "C.mpg")
	assert.Contains(t, s2fmm, "E.mpg")

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

	// 서버2 에 B.mpg, C.mpg가 없으므로, s2fmm 에는 B.mpg, C.mpg 는 들어있지 않다.
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

func Test_requestFreeDisk(t *testing.T) {
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
	Servers.Add(vs1)
	Servers.Add(vs2)
	cfw1 := cfw(vs1, d1, files1)
	cfw1.Start()
	defer cfw1.Close()
	cfw2 := cfw(vs2, d2, files2)
	cfw2.Start()
	defer cfw2.Close()

	rhfiles := []string{"D.mpg", "F.mpg"}
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
	defer deletefile(base, "")

	ignores := []string{}
	SetIgnorePrefixes(ignores)
	for _, server := range servers {
		dels := getFileListToFreeDiskSpace(server, ssfmm, rhitfmm)
		t.Logf("files: %s -> %s", server, dels)
	}
}
