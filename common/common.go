package common

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
)

// DiskUsage is a struct for disk usage(df)
type DiskUsage struct {
	TotalSize   int64 `json:"total_size,string"`
	UsedSize    int64 `json:"used_size,string"`
	FreeSize    int64 `json:"free_size,string"`
	UsedPercent int   `json:"used_percent"`
}

/*******************************************************************/

// Host is struct for ip, port set
type Host struct {
	IP   string
	Port int
}

// Hosts is slice of Host structure
type Hosts []*Host

// NewHosts is constructor of Hosts
func NewHosts() *Hosts {
	return new(Hosts)
}

// Add is to add host to remover's host pool
func (hs *Hosts) Add(s string) error {

	host, err := SplitHostPort(s)

	if err != nil {
		return err
	}

	*hs = append(*hs, &host)
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
func (src SourceDirs) IsExistOnSource(fileName string) (string, bool) {

	for _, dir := range src {
		filePath := dir + "/" + fileName
		fileInfo, err := os.Stat(filePath)

		if err != nil {
			continue
		}

		if fileInfo.IsDir() {
			continue
		}

		return dir + "/" + fileName, true
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
	host := Host{IP: ip, Port: port}
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

				switch ip := addr.(type) {

				}
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
