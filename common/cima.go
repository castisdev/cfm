package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// GetRemoteFileList is to get file list on remote server via CiMonitoringAgent
// URL : /cfm/ls
func GetRemoteFileList(host *Host, fileList *[]string) error {

	serverURL := fmt.Sprintf("http://%s:%d/files", host.IP, host.Port)
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

	body, _ := ioutil.ReadAll(res.Body)

	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		*fileList = append(*fileList, scanner.Text())
	}

	return nil
}

// GetRemoteDiskUsage is to get disk usage on remote server via CiMonitoringAgent
// URL : /cfm/df
func GetRemoteDiskUsage(host *Host, du *DiskUsage) error {

	serverURL := fmt.Sprintf("http://%s:%d/df", host.IP, host.Port)
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
// URL : /cfm/rm?file=${name}
func DeleteFileOnRemote(host *Host, fileName string) error {

	serverURL := fmt.Sprintf("http://%s:%d/files/%s", host.IP, host.Port, fileName)
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

	return nil
}
