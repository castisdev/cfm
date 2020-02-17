package common

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"syscall"
	"time"
)

type Disksize uint64

//  DiskUsage :
// |<-------------------------------TOTAL----------------------------------->|
//
// |<----USED------>|<----------------------FREE----------------------->|
//
// |<----USED------>|<----AVAIL----->|<--ReservedForRoot-->|
//  TotalSize : 전체 용량
//  UsedSize : 사용한 용량
//  AvailSize : 사용할 수 있는 용량
//  UsedPercent : 사용한 퍼센트
//		UsedSize / (UsedSize + AvailSize)	퍼센트 반올림값
//  FreeSize : 전체 남은 용량 (사용할 수 있는 용량 + 시스템용 예약 용량)
// 		TotalSize - UserdSize
type DiskUsage struct {
	TotalSize   Disksize `json:"total_size,string"`
	UsedSize    Disksize `json:"used_size,string"`
	AvailSize   Disksize `json:"avail_size,string"`
	UsedPercent uint     `json:"used_percent"`
	FreeSize    Disksize `json:"free_size,string"`
}

/*******************************************************************/

// Host :is struct for ip, port set
// IP : ip
// Port : port
// Addr : ip:port string
type Host struct {
	IP   string `json:"ip,string"`
	Port int    `json:"port,string"`
	Addr string `json:"addr,string"`
}

// Hosts is slice of Host structure
type Hosts []*Host

// NewHosts is constructor of Hosts
func NewHosts() *Hosts {
	return new(Hosts)
}

// Add is to add host to remover's host pool
//
// 서버 순서를 일정하게 유지할 수 있도록 Addr 큰 순서로 sort 함
func (hs *Hosts) Add(s string) error {

	host, err := SplitHostPort(s)

	if err != nil {
		return err
	}

	*hs = append(*hs, &host)

	sort.Slice(*hs, func(i, j int) bool {
		return (*hs)[i].Addr > (*hs)[j].Addr
	})

	return nil
}

/*******************************************************************/

// SourceDirs is slice for source dir
type SourceDirs []string

// NewSourceDirs is constructor of SourceDirs
func NewSourceDirs() *SourceDirs {
	return new(SourceDirs)
}

// Add is to add dir for source
func (src *SourceDirs) Add(dir string) {
	*src = append(*src, dir)
}

// IsExistOnSource is to check existance of file on source dirs
// directory는 무시함; filename으로 directory 이름이 오는 경우
func (src SourceDirs) IsExistOnSource(fileName string) (string, bool) {

	for _, dir := range src {
		filePath := filepath.Join(dir, fileName)
		fileInfo, err := os.Stat(filePath)

		if err != nil {
			continue
		}

		if fileInfo.IsDir() {
			continue
		}

		return filePath, true
	}
	return "", false
}

/*******************************************************************/

// SplitHostPort is to split "IP:Port" string to Host struct
func SplitHostPort(str string) (Host, error) {

	ip, portString, err := net.SplitHostPort(str)
	if err != nil {
		return Host{}, err
	}

	port64, err := strconv.ParseInt(portString, 10, 64)

	if err != nil {
		return Host{}, err
	}

	port := int(port64)
	host := Host{IP: ip, Port: port, Addr: str}
	return host, nil
}

// GetIPv4ByInterfaceName :
func GetIPv4ByInterfaceName(ifname string) (string, error) {

	ifs, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, i := range ifs {
		if i.Name == ifname {

			addrs, err := i.Addrs()

			if err != nil {
				return "", err
			}

			for _, addr := range addrs {
				ip, ok := addr.(*net.IPNet)
				if !ok {
					return "", errors.New("it is not *net.IPNet type")
				}

				if ip.IP.DefaultMask() != nil {
					return ip.IP.String(), nil
				}
			}
		}
	}

	return "", fmt.Errorf("%s not found", ifname)
}

// SetFDLimit :
func SetFDLimit(n uint64) error {
	var rlimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	if err != nil {
		return err
	}
	rlimit.Max = n
	rlimit.Cur = n
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit)
}

// EnableCoreDump :
func EnableCoreDump() error {
	var rlimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_CORE, &rlimit)
	if err != nil {
		return err
	}
	rlimit.Max = math.MaxUint64
	rlimit.Cur = rlimit.Max
	return syscall.Setrlimit(syscall.RLIMIT_CORE, &rlimit)
}

// String : Disksize to strig
func (d Disksize) String() string {
	str := FormatByte(uint64(d))
	return str
}

// FormatCommnasBytes : B표시, 세자리마다 , 사용
// 100000000 -> 100,000,000B
func (d Disksize) FormatBytes() string {
	str := FormatCommasInt64(uint64(d)) + "B"
	return str
}

// String : DiskUsage to strig
func (du DiskUsage) String() string {
	s := fmt.Sprintf(
		"totalSize(%s), usedSize(%s), availSize(%s), usedPercent(%d)",
		du.TotalSize, du.UsedSize, du.AvailSize, du.UsedPercent)
	return s
}

// String : Host to strig
func (h Host) String() string {
	s := fmt.Sprintf(
		"%s", h.Addr)
	return s
}

// FormatCommasInt64 : float to strig
// 3자리마다 ',' 사용, 소수점 3자리까지 표현
// 100000000 -> 100,000,000
func FormatCommasInt64(num uint64) string {
	str := strconv.FormatUint(num, 10)
	re := regexp.MustCompile("(\\d+)(\\d{3})")
	for i := 0; i < (len(str)-1)/3; i++ {
		str = re.ReplaceAllString(str, "$1,$2")
	}
	return str
}

// FormatCommasFloat64 : float to strig
// 3자리마다 ',' 사용, 소수점 3자리까지 표현
// 100000000.123756789 -> 100,000,000.124
// 100000000 -> 100,000,000.000
func FormatCommasFloat64(num float64) string {
	str := strconv.FormatFloat(num, 'f', 3, 64)
	re := regexp.MustCompile("(\\d+)(\\d{3})")
	for i := 0; i < (len(str)-1)/3; i++ {
		str = re.ReplaceAllString(str, "$1,$2")
	}
	return str
}

// FormatByte
// 값이 커질 수록 단위가 달라짐
// 1023 -> 1,023B
// 1024 -> 1.0KB
func FormatByte(b uint64) string {
	const unit = uint64(1024)
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	str := fmt.Sprintf("%.1f%cB",
		float64(b)/float64(div), "KMGTPE"[exp])

	re := regexp.MustCompile("(\\d+)(\\d{3})")
	for i := 0; i < (len(str)-1)/3; i++ {
		str = re.ReplaceAllString(str, "$1,$2")
	}
	return str
}

var (
	StatfsFunc func(string, *syscall.Statfs_t) error = syscall.Statfs
)

// GetDiskUsage :
//
// |<-------------------------------f_blocks---------------------------------->|
//
// |<----USED------>|<----------------------f_bfree--------------------->|
//
// |<----USED------>|<----f_bavail---->|<--ReservedForRoot-->|
//
// total size = f_blocks * block_size
//
// free size = f_bfree * block_size
//
// avail size = f_bavail * block_size
//
// used size = total size - free size
//
// used percent = ( used size ) / ( used size + avail size)
//
// reserved space is for system partitions
func GetDiskUsage(path string) (DiskUsage, error) {
	du := DiskUsage{}
	fs := syscall.Statfs_t{}
	err := StatfsFunc(path, &fs)
	if err != nil {
		return du, err
	}
	du.TotalSize = Disksize(fs.Blocks * uint64(fs.Bsize))
	du.FreeSize = Disksize(fs.Bfree * uint64(fs.Bsize))
	du.UsedSize = du.TotalSize - du.FreeSize
	du.AvailSize = Disksize(fs.Bavail * uint64(fs.Bsize))

	var used_f float64
	if float64(du.UsedSize+du.AvailSize) == float64(0) {
		used_f = float64(0)
	} else {
		used_f = float64(du.UsedSize) / float64(du.UsedSize+du.AvailSize)
	}
	du.UsedPercent = uint(used_f*100.0 + 0.5)
	return du, nil
}

// GetLimitUsedSize :
// total avail size = ( used + avail )
// limit fraction = limit used / total avail size
// limit used = limit fraction * total avail size
func (du DiskUsage) GetLimitUsedSize(limitPercent uint) Disksize {
	totalAvailSize := du.UsedSize + du.AvailSize
	limitPraction := float64(limitPercent) / float64(100)
	limitUsed := limitPraction * float64(totalAvailSize)
	return Disksize(limitUsed)
}

// GetOverUsedSize :
// 오버 사용량 = 사용량 - 제한 사용량 ( 사용량 > 제한 사용량 )
// 사용량이 제한 사용량보다 적을 때는 0 반환
func (du DiskUsage) GetOverUsedSize(limitPercent uint) Disksize {
	limitUsed := du.GetLimitUsedSize(limitPercent)
	if du.UsedSize > limitUsed {
		return du.UsedSize - limitUsed
	}
	return 0
}

func Start() time.Time {
	return time.Now()
}

func Elapsed(start time.Time) time.Duration {
	return time.Since(start)
}
