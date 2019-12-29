package storage

import (
	"bytes"
	"data"
	"encoding/json"
	"fmt"
	"github.com/rakyll/magicmime"
	"io/ioutil"
	"net/http"
	"time"
)

type StorageConfig struct {
	ServerHost string `json:"server_host"`
	ServerPort int    `json:"server_port"`
	ServerName string `json:"server_name"`
}

var storageServerConfig StorageConfig
var storageServerRootURL string

func InitStorage(st StorageConfig) {
	storageServerConfig = st
	storageServerRootURL = fmt.Sprintf("http://%s:%d/1.0", st.ServerHost, st.ServerPort)
}

func CreateNewUploadId() string {
	req, err := http.NewRequest("POST", storageServerRootURL+"/new-upload-id", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			storageResponse := data.UploadInfo{}
			err := json.NewDecoder(resp.Body).Decode(&storageResponse)
			data.Logger.Printf("Response from StorageManager = %s", storageResponse.UploadId)
			if err == nil {
				return storageResponse.UploadId
			}
		}
	}
	return ""
}

func CanUploadBinaryData(applicationId int, applicationInstanceId int, uploadId string, binaryDataLen int, mimeType string) bool {
	values := data.UploadCheckInfo{UploadId: uploadId, UploadLen: binaryDataLen, MimeType: mimeType}
	jsonV, err := json.Marshal(values)
	req, err := http.NewRequest("POST", storageServerRootURL+"/can-upload-data", bytes.NewBuffer(jsonV))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		data.Logger.Printf("Body is ok..., resp code = %d", resp.StatusCode)
		if resp.StatusCode == http.StatusOK {
			return true
		}
	}
	data.Logger.Printf("Cannot upload binary data...")
	return false
}

func UploadBinaryData(applicationId int, applicationInstanceId int, uploadId string, binaryData []byte) (bool, string, time.Time) {
	putURL := fmt.Sprintf("%s/upload/%d/%d/%s", storageServerRootURL, applicationId, applicationInstanceId, uploadId)
	contentType, err := magicmime.TypeByBuffer(binaryData)
	if err != nil {
		contentType = "application/octet-stream"
	}
	req, err := http.NewRequest("PUT", putURL, bytes.NewBuffer(binaryData))
	data.Logger.Printf("Prepared PUT Statement %s", putURL)
	if err != nil {
		return false, "", time.Now()
	}
	req.Header.Set("Content-Type", contentType)
	data.Logger.Printf("Set Content Type to %s", contentType)
	client := &http.Client{}
	res, err := client.Do(req)
	data.Logger.Printf("Result of Client.DO = %s", err)
	if err != nil {
		return false, "", time.Now()
	}
	defer res.Body.Close()
	data.Logger.Printf("Checking Status Code %d", res.StatusCode)
	if res.StatusCode == http.StatusOK {
		storageResponse := data.StorageServerUploadResponse{}
		data.Logger.Printf("Checking Json")
		err := json.NewDecoder(res.Body).Decode(&storageResponse)
		if err == nil {
			return true, storageResponse.BinaryDataId, storageResponse.Expires
		}
	}
	return false, "", time.Now()
}

func RetrieveBinaryData(applicationId int, applicationInstanceId int, identifier string) (int, []byte, string, time.Time) {
	getURL := fmt.Sprintf("%s/download/%d/%d/%s", storageServerRootURL, applicationId, applicationInstanceId, identifier)
	req, err := http.NewRequest("GET", getURL, nil)
	data.Logger.Printf("Prepared GET Statement %s", getURL)
	if err != nil {
		return 404, nil, "", time.Now()
	}
	client := &http.Client{}
	res, err := client.Do(req)
	data.Logger.Printf("Result of Client.DO = %s", err)
	if err != nil {
		return 404, nil, "", time.Now()
	}
	defer res.Body.Close()
	data.Logger.Printf("Checking Status Code %d", res.StatusCode)
	if res.StatusCode == http.StatusOK {
		data, err := ioutil.ReadAll(res.Body)
		if err == nil {
			contentType := res.Header.Get("Content-Type")
			return res.StatusCode, data, contentType, time.Now()
		}
	}
	return 404, nil, "", time.Now()
}

func DeleteBinaryData(applicationId int, applicationInstanceId int, identifier string) bool {
	return true
}

func ExpireBinaryData() {
}
