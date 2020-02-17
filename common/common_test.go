package common_test

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/castisdev/cfm/common"
)

func TestHosts_Add(t *testing.T) {

	hosts := common.NewHosts()

	hosts.Add("127.0.0.1:18081")
	hosts.Add("127.0.0.2:18081")
	hosts.Add("127.0.0.3:18081")

	assert.Equal(t, 3, len(*hosts))
}

func TestSourceDirs_Add(t *testing.T) {

	srcDirs := common.NewSourceDirs()

	srcDirs.Add("/data2")
	srcDirs.Add("/data3")

	assert.Equal(t, 2, len(*srcDirs))
}

func TestSourceDirs_IsExistOnSource(t *testing.T) {

	srcDirs := common.NewSourceDirs()

	srcDirs.Add("data2")
	srcDirs.Add("data3")

	for _, dir := range *srcDirs {
		require.Nil(t, os.Mkdir(dir, os.FileMode(0755)))
	}

	if _, err := os.Create("data2/a.mpg"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Create("data3/b.mpg"); err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll("data2")
	defer os.RemoveAll("data3")

	tests := []struct {
		fileName  string
		want      string
		wantExist bool
	}{
		{"a.mpg", "data2/a.mpg", true},
		{"b.mpg", "data3/b.mpg", true},
		{"c.mpg", "", false},
	}

	for _, tt := range tests {
		// got string, got bool
		gs, gb := srcDirs.IsExistOnSource(tt.fileName)
		if !reflect.DeepEqual(gs, tt.want) {
			t.Errorf("got (%s), want (%s)", gs, tt.want)
		}

		if gb != tt.wantExist {
			t.Errorf("got (%t), want (%t)", gb, tt.wantExist)
		}
	}

}

func TestSplitHostPort(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name    string
		args    args
		want    common.Host
		wantErr bool
	}{
		{"ok", args{"127.0.0.1:1000"},
			common.Host{IP: "127.0.0.1", Port: 1000, Addr: "127.0.0.1:1000"}, false,
		},
		{"invalid string(double colon)", args{"127.0.0.1:1000:1000"},
			common.Host{}, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := common.SplitHostPort(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitHostPort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitHostPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetIPv4ByInterfaceName : 이 테스트는 테스트 환경마다 결과 값이 달라질 수 있다.
func TestGetIPv4ByInterfaceName(t *testing.T) {
	ifs, err := net.Interfaces()
	if err != nil {
		t.Errorf("%s", err.Error())
	}
	for _, i := range ifs {
		t.Logf("interface: %s", i.Name)
		ip, err := common.GetIPv4ByInterfaceName(i.Name)
		if err != nil {
			t.Errorf("error : (%s)", err)
		}
		assert.NotEmptyf(t, ip, "ip: %s", ip)
	}
}

func TestGetDiskUsage(t *testing.T) {
	efs := syscall.Statfs_t{
		Bsize:  1000,
		Blocks: 10, // total size
		Bfree:  8,  // freesize
		Bavail: 6,  // avail size
	}
	common.StatfsFunc =
		func(path string, fs *syscall.Statfs_t) error {
			fs.Bsize = efs.Bsize
			fs.Blocks = efs.Blocks
			fs.Bfree = efs.Bfree
			fs.Bavail = efs.Bavail
			return nil
		}
	du, err := common.GetDiskUsage(".")
	assert.Equal(t, nil, err)
	dudesc := fmt.Sprintf("%s", du)
	t.Logf("du: %s", du)

	assert.Equal(t, common.Disksize(uint64(efs.Bsize)*efs.Blocks), du.TotalSize)
	assert.Equal(t, common.Disksize(uint64(efs.Bsize)*efs.Bfree), du.FreeSize)
	assert.Equal(t, du.TotalSize-du.FreeSize, du.UsedSize)
	assert.Equal(t, common.Disksize(uint64(efs.Bsize)*efs.Bavail), du.AvailSize)

	assert.Equal(t, common.Disksize(10000), du.TotalSize)
	assert.Equal(t, common.Disksize(8000), du.FreeSize)
	assert.Equal(t, common.Disksize(2000), du.UsedSize)
	assert.Equal(t, common.Disksize(6000), du.AvailSize)
	assert.Equal(t, uint(25), du.UsedPercent)

	assert.Equal(t, "totalSize(9.8KB), usedSize(2.0KB), availSize(5.9KB), usedPercent(25)", dudesc)
}

func TestGetLimitUsedSizeAndOverUsedSize(t *testing.T) {
	efs := syscall.Statfs_t{
		Bsize:  1000,
		Blocks: 10, // total size
		Bfree:  8,  // freesize
		Bavail: 6,  // avail size
	}
	common.StatfsFunc =
		func(path string, fs *syscall.Statfs_t) error {
			fs.Bsize = efs.Bsize
			fs.Blocks = efs.Blocks
			fs.Bfree = efs.Bfree
			fs.Bavail = efs.Bavail
			return nil
		}
	du, err := common.GetDiskUsage(".")
	t.Logf("du: %s", du)
	assert.Equal(t, nil, err)
	assert.Equal(t, common.Disksize(10000), du.TotalSize)
	assert.Equal(t, common.Disksize(8000), du.FreeSize)
	assert.Equal(t, common.Disksize(2000), du.UsedSize)
	assert.Equal(t, common.Disksize(6000), du.AvailSize)
	assert.Equal(t, uint(25), du.UsedPercent)

	limitsize := du.GetLimitUsedSize(du.UsedPercent)
	assert.Equal(t, du.UsedSize, limitsize)
	oversize := du.GetOverUsedSize(du.UsedPercent)
	assert.Equal(t, common.Disksize(0), oversize)

	limitsize = du.GetLimitUsedSize(20)
	assert.Equal(t, common.Disksize(1600), limitsize)

	oversize = du.GetOverUsedSize(20)
	t.Logf("limit: %s, over: %s", limitsize, oversize)
	assert.Equal(t, common.Disksize(400), oversize)
}
