package tailer

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetLogFileName(t *testing.T) {
	basetm := time.Date(2020, time.February, 11, 18, 7, 7, 0, time.UTC)
	dir := "test"
	watchmin := 10
	efile1 := "test/EventLog[20200211].log"
	logFileNames := GetLogFileName(basetm, dir, watchmin)
	t.Logf("%s", (*logFileNames)[0])
	assert.Equal(t, efile1, (*logFileNames)[0])

	basetm = time.Date(2020, time.February, 12, 0, 0, 0, 0, time.UTC)
	dir = "test"
	watchmin = 30
	efile1 = "test/EventLog[20200211].log"
	efile2 := "test/EventLog[20200212].log"

	logFileNames2 := GetLogFileName(basetm, dir, watchmin)
	if len(*logFileNames2) == 1 {
		t.Logf("%s", (*logFileNames2)[0])
		assert.Equal(t, efile1, (*logFileNames2)[0])
	} else {
		t.Logf("%s, %s", (*logFileNames)[0], (*logFileNames2)[1])
		assert.Equal(t, efile1, (*logFileNames2)[0])
		assert.Equal(t, efile2, (*logFileNames2)[1])
	}

}

func TestTailer_parseLBEventLog(t *testing.T) {

	tailer := NewTailer()
	tailer.SetWatchDir(".")
	tailer.SetWatchIPString("125.159.40.3")
	tailer.SetWatchTermMin(-1)
	assert.Equal(t, 10, tailer.watchTermMin)
	tailer.SetWatchTermMin(10)
	assert.Equal(t, 10, tailer.watchTermMin)
	tailer.SetWatchHitBase(0)
	assert.Equal(t, 3, tailer.watchHitBase)
	tailer.SetWatchHitBase(4)
	assert.Equal(t, 4, tailer.watchHitBase)

	tmpFile := "EventLog[20180602].log"

	f, err := os.Create(tmpFile)
	if err != nil {
		f.Close()
		t.Fatalf("cannot create %s", tmpFile)
	}
	defer os.Remove(tmpFile)

	fmt.Fprintln(f, "0x40ffff,1,1527951607,Server 125.159.40.3 Selected for Client StreamID : 3f004a6e-82af-4dce-85ba-9bbf9c7cb8cb, ClientID : 0, GLB IP : 125.144.96.6's file(MCLE901VSGL1500001_K20140915224744.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,4,1527951607,File Not Found, UUID : fffb233a-376a-4c2f-842e-553fb68af9cf, GLB IP : 125.144.161.6, MV6F9001SGL1500001_K20150909214818.mpg")
	fmt.Fprintln(f, "0x40ffff,1,1527951607,Server 125.144.91.87 Selected for Client StreamID : 360527d4-44b3-4b8f-aef7-dbf8fd230d54, ClientID : 0, GLB IP : 125.144.169.6's file(M33E80DTSGL1500001_K20141022144006.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951609,Server 125.159.40.3 Selected for Client StreamID : c93a7db2-ccaf-4765-af8d-7ddc2d33a812, ClientID : 0, GLB IP : 125.159.40.5's file(MXCI5B4LSGL1000051_K20180531174730.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,4,1527951609,File Not Found, UUID : f1add5cf-75ac-41ab-a6ff-85d9e0927762, GLB IP : 125.144.169.6, MK4E7BK2SGL0800014_K20120725124707.mpg")
	fmt.Fprintln(f, "0x40ffff,1,1527951609,Server 125.144.97.67 Selected for Client StreamID : 06fb572e-7602-4231-8670-cb6526603fb0, ClientID : 0, GLB IP : 125.146.8.6's file(M33H90E2SGL1500001_K20171008222635.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951610,Server 125.159.40.3 Selected for Client StreamID : 3c61af91-cd6a-4dd6-bc04-5ec6bc78b94f, ClientID : 0, GLB IP : 125.159.40.5's file(MWGI5006SGL1500001_K20180524203234.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,4,1527951610,File Not Found, UUID : c585905f-9980-49b1-89bc-97c7140eaa83, GLB IP : 125.159.40.5, M34G80A3SGL1500001_K20160827230242.mpg")
	fmt.Fprintln(f, "0x40ffff,1,1527951610,Server 125.144.97.74 Selected for Client StreamID : 7cf6b886-edd2-471b-9cfd-12763a160b0b, GLB IP : 125.159.40.5's file(M34F60QHSGL1500001_K20150701232550.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951610,Server 125.144.97.77 Selected for Client StreamID : 23dd1489-543b-4051-b07a-e877f8b2e052, GLB IP : 125.147.192.6's file(MW0E6JE3SGL0800014_K20120601193450.mpg) Request")
	// ------------------------------------------------- 테스트 기준 시각
	fmt.Fprintln(f, "0x40ffff,1,1527951611,Server 125.159.40.3 Selected for Client StreamID : 97096b41-afe1-44d8-b57c-e758a70883d9, GLB IP : 125.159.40.5's file(M33F3MA3SGL0800038_K20130326135640.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951611,Server 125.144.91.83 Selected for Client StreamID : aa7de9a1-7d0d-40d5-9586-31dc275a0634, ClientID : 0, GLB IP : 125.147.36.6's file(MADI4008SGL1500001_K20180506231943.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951611,Server 125.144.91.84 Selected for Client StreamID : 1926c2ba-c313-48fb-977a-b7f3fd27ea98, ClientID : 0, GLB IP : 125.148.160.6's file(MEQI405ISGL1500001_K20180509034746.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951611,Server 125.144.97.73 Selected for Client StreamID : f179c61a-d5e0-45b9-b046-a3cd4e3dbbfc, ClientID : 0, GLB IP : 125.147.192.6's file(MIAF51OLSGL1500001_K20150511175323.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951611,Server 125.159.40.3 Selected for Client StreamID : 3894d674-d74b-4eca-a2ea-fafbfa1113a8, ClientID : 0, GLB IP : 125.159.40.5's file(MXCI5B4LSGL1000051_K20180531174730.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951612,Server 125.159.40.3 Selected for Client StreamID : 8b2e7e4a-270d-4586-85a1-e4284551176d, ClientID : 0, GLB IP : 125.159.40.5's file(MXCI5B4LSGL1000051_K20180531174730.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951612,Server 125.159.40.3 Selected for Client StreamID : f1893245-802b-45b1-b0fa-377bc1415b35, ClientID : 0, GLB IP : 125.159.40.5's file(MXCI5B4LSGL1000051_K20180531174730.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951612,Server 125.144.93.100 Selected for Client StreamID : 3c68d2a5-354e-4a4b-b181-c724d16cf406, GLB IP : 125.147.36.6's file(MVHF201MSGL1500001_K20150216200556.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951612,Server 125.159.40.3 Selected for Client StreamID : d41dc03a-91b7-4c14-bb3a-a73823f333e0, ClientID : 0, GLB IP : 125.159.40.5's file(MXCI5B4LSGL1000051_K20180531174730.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,4,1527951612,File Not Found, UUID : d9672fe8-1b39-491f-a8a4-23bf7a6f096c, GLB IP : 125.144.96.6, M0200000SGL1065016_K20100826000000.MPG")
	fmt.Fprintln(f, "0x40ffff,1,1527951612,Server 125.159.40.3 Selected for Client StreamID : f8a21815-a75b-4fe4-8f6a-dba984ee7c6e, ClientID : 0, GLB IP : 125.159.40.5's file(MXCI5B4LSGL1000051_K20180531174730.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951612,Server 125.159.40.3 Selected for Client StreamID : 7148fadd-edab-4a48-8d27-4bf8c8b74cbd, ClientID : 0, GLB IP : 125.159.40.5's file(MXCI5B4LSGL1000051_K20180531174730.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951612,Server 125.159.40.3 Selected for Client StreamID : b121ae29-954c-4990-8e58-e102959d0239, ClientID : 0, GLB IP : 125.159.40.5's file(MXCI5B4LSGL1000051_K20180531174730.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951612,Server 125.144.93.97 Selected for Client StreamID : 86fd80d3-2691-4274-9172-315d50e90801, ClientID : 0, GLB IP : 125.159.40.5's file(M34I502CSGL1500001_K20180512022857.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951612,Server 125.159.40.3 Selected for Client StreamID : 598131b3-8fb2-4415-b0f8-472d52ef054c, ClientID : 0, GLB IP : 125.159.40.5's file(MXCI5B4LSGL1000051_K20180531174730.mpg) Request")
	fmt.Fprintln(f, "0x40ffff,1,1527951613,Server 125.159.40.3 Selected for Client StreamID : 0590710c-bd2e-4863-941e-041877328d78, ClientID : 0, GLB IP : 125.159.40.5's file(MXCI5B4LSGL1000051_K20180531174730.mpg) Request")
	f.Close()

	fm := make(map[string]int)
	tailer.parseLBEventLog(tmpFile, 0, int64(1527951611), &fm)

	// 기준 시각 1527951611 보다 이전에 hit 발생
	fileName := "MCLE901VSGL1500001_K20140915224744.mpg"
	v, exists := fm[fileName]
	assert.False(t, exists)

	fileName = "MWGI5006SGL1500001_K20180524203234.mpg"
	v, exists = fm[fileName]
	assert.False(t, exists)

	// 기준 시각 이전에 1 hit, 이후에 9 hit 발생
	fileName = "MXCI5B4LSGL1000051_K20180531174730.mpg"
	v, exists = fm[fileName]
	assert.True(t, exists)
	assert.Equal(t, 9, v)

	// GLB IP 이 포함된 로그 패턴
	// 0x40ffff,1,1527951611,Server 125.159.40.3 Selected for Client StreamID : 97096b41-afe1-44d8-b57c-e758a70883d9, GLB IP : 125.159.40.5's file(M33F3MA3SGL0800038_K20130326135640.mpg) Request
	fileName = "M33F3MA3SGL0800038_K20130326135640.mpg"
	v, exists = fm[fileName]
	assert.True(t, exists)
	assert.Equal(t, 1, v)

	// 1527951611 값 때문에, 어지러움
	// 원래는 10분을 더해서 Tail 을 호출해야 맞을 듯
	// 그래서인지 MXCI5B4LSGL1000051_K20180531174730.mpg의 count가
	// 9 -> 10 이된다.
	result := make(map[string]int)
	tailer.Tail(time.Unix(int64(1527951611), 0), &result)

	t.Logf("%v", result)
	fileName = "MXCI5B4LSGL1000051_K20180531174730.mpg"
	v, exists = result[fileName]
	assert.Equal(t, 1, len(result))
	assert.True(t, exists)
	assert.Equal(t, 10, v)
}
