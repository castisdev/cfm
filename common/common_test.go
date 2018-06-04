package common

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHosts_Add(t *testing.T) {

	hosts := NewHosts()

	hosts.Add("127.0.0.1:18081")
	hosts.Add("127.0.0.2:18081")
	hosts.Add("127.0.0.3:18081")

	assert.Equal(t, 3, len(*hosts))
}

func TestSourceDirs_Add(t *testing.T) {

	srcDirs := NewSourceDirs()

	srcDirs.Add("/data2")
	srcDirs.Add("/data3")

	assert.Equal(t, 2, len(*srcDirs))
}

func TestSourceDirs_IsExistOnSource(t *testing.T) {

	srcDirs := NewSourceDirs()

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
		want    Host
		wantErr bool
	}{
		{"ok", args{"127.0.0.1:1000"}, Host{IP: "127.0.0.1", Port: 1000}, false},
		{"invalid string(double colon)", args{"127.0.0.1:1000:1000"}, Host{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitHostPort(tt.args.str)
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
	ip, err := GetIPv4ByInterfaceName("en0")

	if err != nil {
		t.Errorf("error : (%s)", err)
	}
	assert.Equal(t, "192.168.0.28", ip)
}
