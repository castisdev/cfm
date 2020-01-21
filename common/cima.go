package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

func defaultHttpClient() http.Client {
	return httpClient(600, 5)
}

func httpTimoutClient(timeoutSec uint) http.Client {
	return httpClient(600, timeoutSec)
}

func httpClient(keepaliveTOsec, httpTOsec uint) http.Client {
	keepAliveTimeout := time.Duration(keepaliveTOsec) * time.Second
	timeout := time.Duration(httpTOsec) * time.Second
	defaultTransport := &http.Transport{
		Dial: (&net.Dialer{
			KeepAlive: keepAliveTimeout,
		}).Dial,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}
	return http.Client{
		Transport: defaultTransport,
		Timeout:   timeout,
	}
}

// Heartbeat : host에 heartbeat 요청, 응답받기
// URL : hostip(ipv4):port/hb
// timeoutSec : timeout 값
func Heartbeat(host *Host, timeoutSec uint) (bool, error) {
	serverURL := fmt.Sprintf("http://%s/hb", host.Addr)
	_, urlErr := url.Parse(serverURL)
	if urlErr != nil {
		return false, urlErr
	}
	httpClient := httpTimoutClient(timeoutSec)

	req, reqErr := http.NewRequest(http.MethodHead, serverURL, nil)
	if reqErr != nil {
		return false, reqErr
	}
	res, resErr := httpClient.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if resErr != nil {
		return false, resErr
	}
	// HTTP 커넥션을 재사용할때 메모리 누수를 피하기 위해선
	// 데이터가 필요없더라도 응답 바디를 읽어야함
	// 응답 바디를 읽음. (데이터가 필요 없으므로 devnull에 write)
	// https: //stackoverflow.com/questions/17959732/why-is-go-https-client-not-reusing-connections
	_, err := io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		return false, err
	}

	if res.StatusCode != 200 {
		return false, errors.New(res.Status)
	}
	return true, nil
}

// GetRemoteFileList is to get file list on remote server via CiMonitoringAgent
// URL : hostip(ipv4):port/files
func GetRemoteFileList(host *Host, fileList *[]string) error {

	serverURL := fmt.Sprintf("http://%s/files", host.Addr)
	_, urlErr := url.Parse(serverURL)
	if urlErr != nil {
		return urlErr
	}

	httpClient := defaultHttpClient()

	req, reqErr := http.NewRequest(http.MethodGet, serverURL, nil)
	if reqErr != nil {
		return reqErr
	}

	res, resErr := httpClient.Do(req)
	if resErr != nil {
		return resErr
	}
	if res != nil {
		defer res.Body.Close()
	}

	if res.StatusCode != 200 {
		return errors.New(res.Status)
	}

	body, _ := ioutil.ReadAll(res.Body)

	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		*fileList = append(*fileList, scanner.Text())
	}

	return nil
}

// GetRemoteDiskUsage is to get disk usage on remote server via CiMonitoringAgent
// URL : hostip(ipv4):port/df
func GetRemoteDiskUsage(host *Host, du *DiskUsage) error {

	serverURL := fmt.Sprintf("http://%s/df", host.Addr)
	_, urlErr := url.Parse(serverURL)
	if urlErr != nil {
		return urlErr
	}

	httpClient := defaultHttpClient()

	req, err := http.NewRequest(http.MethodGet, serverURL, nil)
	if err != nil {
		return err
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		return getErr
	}
	if res != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, du)

	return nil
}

// DeleteFileOnRemote is to delete file on remote server via CiMonitoringAgent
// URL : hostip(ipv4):port/files/${name}
func DeleteFileOnRemote(host *Host, fileName string) error {

	serverURL := fmt.Sprintf("http://%s/files/%s", host.Addr, fileName)
	_, urlErr := url.Parse(serverURL)
	if urlErr != nil {
		return urlErr
	}

	httpClient := defaultHttpClient()

	req, err := http.NewRequest(http.MethodDelete, serverURL, nil)
	if err != nil {
		return err
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		return getErr
	}
	if res != nil {
		defer res.Body.Close()
	}
	_, err = io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return errors.New(res.Status)
	}

	return nil
}
