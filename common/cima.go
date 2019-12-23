package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// Heartbeat : host에 heartbeat 요청, 응답받기
// URL : hostip(ipv4):port/hb
// timeoutSec : timeout 값
func Heartbeat(host *Host, timeoutSec uint) (bool, error) {
	serverURL := fmt.Sprintf("http://%s/hb", host.Addr)
	_, urlErr := url.Parse(serverURL)
	if urlErr != nil {
		return false, urlErr
	}
	httpClient := http.Client{
		Timeout: time.Second * time.Duration(timeoutSec),
	}

	req, reqErr := http.NewRequest(http.MethodHead, serverURL, nil)
	if reqErr != nil {
		return false, reqErr
	}
	res, resErr := httpClient.Do(req)
	if resErr != nil {
		return false, resErr
	}
	defer res.Body.Close()

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

	httpClient := http.Client{
		Timeout: time.Second * 2,
	}

	req, reqErr := http.NewRequest(http.MethodGet, serverURL, nil)
	if reqErr != nil {
		return reqErr
	}

	res, resErr := httpClient.Do(req)
	if resErr != nil {
		return resErr
	}
	defer res.Body.Close()

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

	httpClient := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, serverURL, nil)
	if err != nil {
		return err
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		return getErr
	}
	defer res.Body.Close()

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

	httpClient := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodDelete, serverURL, nil)
	if err != nil {
		return err
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		return getErr
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return errors.New(res.Status)
	}

	return nil
}
