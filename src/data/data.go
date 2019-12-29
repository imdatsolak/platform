package data

import (
	"github.com/twinj/uuid"
	"log"
	"time"
)

type TempJobInfo struct {
	JobId                 int
	ApplicationId         int
	ApplicationInstanceId int
	JobUID                string
	JobStatus             int
	RequestType           int
	RequestStartTime      time.Time
	RequestSize           int
	RequestData           string
	UploadId              string
	RequestEndTime        time.Time
	ProcessingTime        int
	JobResultData         string
	JobResultRetrieved    int
	UploadIdentifier      string
}

type JobResult struct {
	JobId     int    `json:"job_id"`
	JobStatus int    `json:"job_status"`
	Payload   string `json:"payload"`
}

type UploadInfo struct {
	UploadId        string    `json:"upload_id"`
	UploadUntilDate time.Time `json:"upload_until"`
}

type StorageServerUploadResponse struct {
	BinaryDataId string    `json:"binary_data_id"`
	Expires      time.Time `json:"expires"`
}

type UploadIdT struct {
	UploadId string `json:"upload_id"`
}

type UploadCheckInfo struct {
	UploadId  string `json:"upload_id"`
	UploadLen int    `json:"data_size"`
	MimeType  string `json:"mime_type"`
}

type ServiceInfo struct {
	ServiceType     int      `json:"service_type"`
	Description     string   `json:"service_description"`
	Server          string   `json:"service_server"`
	Port            int      `json:"service_port"`
	ActionURL       string   `json:"service_action_url"`
	HeartbeatURL    string   `json:"service_heartbeat_url"`
	RequiresUpload  bool     `json:"service_requires_upload"`
	RequestTypes    []string `json:"service_request_types"`
	ReturnsDownload bool     `json:"service_returns_download"`
	ResponseTypes   []string `json:"service_response_types"`
	About           string   `json:"service_about"`
	IsAsync         bool     `json:"service_is_async"`
}

type ServiceInfoList []ServiceInfo

var Logger *log.Logger

func NewTempJobInfoRecord(applicationId int, applicationInstanceId int, jobType int, requestData []byte) (bool, TempJobInfo) {
	var newJob TempJobInfo
	uuid := uuid.NewV4().String()
	if uuid != "" {
		newJob.JobId = -1
		newJob.ApplicationId = applicationId
		newJob.ApplicationInstanceId = applicationInstanceId
		newJob.JobUID = uuid
		newJob.JobStatus = 0
		newJob.RequestType = jobType
		newJob.RequestStartTime = time.Now()
		newJob.RequestData = string(requestData)
		newJob.RequestSize = len(newJob.RequestData)
		newJob.UploadId = ""
		newJob.RequestEndTime = time.Now()
		newJob.ProcessingTime = 0
		newJob.JobResultData = ""
		newJob.UploadIdentifier = ""
		return true, newJob
	} else {
		return false, newJob
	}
}
