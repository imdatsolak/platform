package jobs

import (
	"billing"
	"bytes"
	"cydb"
	"data"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"storage"
	"time"
)

const (
	JobStatusCreated        = 0
	JobStatusWaitingForFile = 101
	JobStatusRunning        = 102
	JobStatusDone           = 103
	JobStatusGONE           = 800
	JobStatusNoAccess       = 900
	JobStatusKilled         = 950
	JobStatusHanging        = 960
	JobStatusNONE           = 999
	JobStatusERROR          = 9999
)

const (
	JobTypeObjectDetection         = 100
	JobTypeSTT                     = 101
	JobTypeTTS                     = 102
	JobTypeTextEntityRecognition   = 103
	JobTypeTextClassification      = 104
	JobTypeTextTopicIdentification = 105
	JobTypeTextSentimentAnalysis   = 106
	JobTypeTextSummarization       = 107
)

type JobsConfig struct {
	ServerHost           string `json:"server_host"`
	ServerPort           int    `json:"server_port"`
	ServerName           string `json:"server_name"`
	AvailableServicesURL string `json:"available_services_url"`
}

type JobData struct {
	JobId                int    `json:"job_id"`
	JobStatus            int    `json:"job_status"`
	JobStatusDetails     int    `json:"job_status_details"`
	JobStatusDetsilsText string `json:"job_status_details_text"`
}

type ServerRequest struct {
	ApplicationId         int    `json:"application_id"`
	ApplicationInstanceId int    `json:"application_instance_id"`
	JobId                 int    `json:"job_id"`
	TargetService         int    `json:"job_type"`
	UploadId              string `json:"upload_identifier"`
	Payload               string `json:"payload"`
}

type ServiceIdentification struct {
	ServiceType int `json:"service_type"`
}

var availableServices data.ServiceInfoList
var acceptedServiceTypes map[int]data.ServiceInfo

var configuration JobsConfig
var jobServerRootURL string

/*
 * This function is called as a go-routine and checks every 30 seconds
 * which services are still / again available...
 */
func updateAvailableServices() {
	var avS data.ServiceInfoList
	var ast map[int]data.ServiceInfo
	for {
		/* First retrieve the available services from our Service Discovery Server */
		req, err := http.NewRequest("GET", configuration.AvailableServicesURL, nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				json.NewDecoder(resp.Body).Decode(&avS)
			}
		} else {
			data.Logger.Printf("SD not reachable, can't work without any available services..")
			os.Exit(1)
		}
		availableServices = avS
		/*
		   Now create a list of possible services and their descriptions for later use.
		   We save only ONE description per service type as they must be identical in their
		   description of information that is important to user.
		*/
		ast = make(map[int]data.ServiceInfo)
		for _, s := range availableServices {
			_, stE := ast[s.ServiceType]
			if !stE {
				ast[s.ServiceType] = s
			}
		}
		acceptedServiceTypes = ast
		data.Logger.Printf("JOBS: %d serviceTypes availables\n", len(acceptedServiceTypes))
		time.Sleep(15 * time.Second)
	}
}

func AvailableServices() data.ServiceInfoList {
	return availableServices
}

func InitJobs(c JobsConfig) {
	configuration = c
	acceptedServiceTypes = make(map[int]data.ServiceInfo)
	go updateAvailableServices()
	jobServerRootURL = fmt.Sprintf("http://%s:%d/1.0", configuration.ServerHost, configuration.ServerPort)
}

func runSyncJob(jobData data.TempJobInfo) (int, []byte) {
	var byteBuffer []byte = nil
	var jobServerData ServerRequest = ServerRequest{ApplicationId: jobData.ApplicationId, ApplicationInstanceId: jobData.ApplicationInstanceId, JobId: jobData.JobId, TargetService: jobData.RequestType, UploadId: jobData.UploadIdentifier, Payload: jobData.RequestData}

	jobServerJSON, err := json.Marshal(jobServerData)
	data.Logger.Printf("Will run job")
	if err == nil {
		req, err := http.NewRequest("POST", jobServerRootURL+"/new-job", bytes.NewBuffer(jobServerJSON))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		data.Logger.Printf("Starting 'Client.Do'")
		resp, err := client.Do(req)
		data.Logger.Printf("DONE 'Client.Do'")
		if err == nil {
			defer resp.Body.Close()
			byteBuffer, err = ioutil.ReadAll(resp.Body)
			if resp.StatusCode == http.StatusOK && err == nil {
				data.Logger.Printf("RUNJOB: JobServer ACCEPTED CALL")
				return resp.StatusCode, byteBuffer
			} else {
				data.Logger.Printf("RUNJOB: JobServer returned SC != 200 && SC != 202, code =%d", resp.StatusCode)
				return resp.StatusCode, byteBuffer
			}
		} else {
			data.Logger.Printf("RUNJOB: Could not connect to JobServer")
			return http.StatusInternalServerError, byteBuffer
		}
	} else {
		data.Logger.Printf("RUNJOB: User sent a very bad request :-)")
		return http.StatusBadRequest, byteBuffer
	}
}

func CreateNewJob(applicationId int, applicationInstanceId int, requestData []byte) (int, data.JobResult, data.UploadInfo) {
	var serviceDescription data.ServiceInfo
	var serviceId ServiceIdentification

	err := json.Unmarshal(requestData, &serviceId)
	if err == nil {
		if _, stE := acceptedServiceTypes[serviceId.ServiceType]; !stE {
			return http.StatusMethodNotAllowed, data.JobResult{}, data.UploadInfo{}
		}

		serviceDescription = acceptedServiceTypes[serviceId.ServiceType]
		data.Logger.Printf("CREATE:: JOB/SERVICE REQUEST of Type %d ", serviceId.ServiceType)
		jobCreated, tempJobData := data.NewTempJobInfoRecord(applicationId, applicationInstanceId, serviceId.ServiceType, requestData)
		if jobCreated {
			if serviceDescription.RequiresUpload {
				newUploadId := storage.CreateNewUploadId()
				data.Logger.Printf("NewUpload ID = %s", newUploadId)
				tempJobData.UploadId = newUploadId
				tempJobData.JobStatus = JobStatusWaitingForFile
				jobId := cydb.AddNewJobInfo(tempJobData)
				if jobId > 0 {
					jobResult := data.JobResult{JobId: jobId, JobStatus: JobStatusWaitingForFile, Payload: ""}
					uploadInfo := data.UploadInfo{UploadId: newUploadId, UploadUntilDate: time.Now().Add(time.Hour)}
					return http.StatusAccepted, jobResult, uploadInfo
				}
				return http.StatusInternalServerError, data.JobResult{}, data.UploadInfo{}
			} else if serviceDescription.IsAsync {
				tempJobData.UploadId = ""
				tempJobData.JobStatus = JobStatusCreated
				jobId := cydb.AddNewJobInfo(tempJobData)
				if jobId > 0 {
					tempJobData.JobId = jobId
					uploadInfo := data.UploadInfo{}
					respCode, jobResult := RunJob(tempJobData)
					return respCode, jobResult, uploadInfo
				}
				return http.StatusInternalServerError, data.JobResult{}, data.UploadInfo{}
			} else {
				tempJobData.UploadId = ""
				tempJobData.JobStatus = JobStatusCreated
				jobId := cydb.AddNewJobInfo(tempJobData)
				if jobId > 0 {
					tempJobData.JobId = jobId
					respCode, jobDataBuffer := runSyncJob(tempJobData)
					cydb.UpdateJobStatus(jobId, JobStatusDone)
					cydb.UpdateJobResultRetrieved(jobId, 1)
					billing.RecordJobDone(tempJobData)
					uploadInfo := data.UploadInfo{}
					jobResult := data.JobResult{JobId: 0, JobStatus: JobStatusDone, Payload: string(jobDataBuffer)}
					return respCode, jobResult, uploadInfo
				}
				return http.StatusInternalServerError, data.JobResult{}, data.UploadInfo{}
			}
		} else {
			return http.StatusInternalServerError, data.JobResult{}, data.UploadInfo{}
		}
	}
	return http.StatusBadRequest, data.JobResult{}, data.UploadInfo{}
}

func runJob(jobData data.TempJobInfo) (int, data.JobResult) {
	var jobStatus data.JobResult
	var jobServerData ServerRequest = ServerRequest{ApplicationId: jobData.ApplicationId, ApplicationInstanceId: jobData.ApplicationInstanceId, JobId: jobData.JobId, TargetService: jobData.RequestType, UploadId: jobData.UploadIdentifier, Payload: jobData.RequestData}

	jobServerJSON, err := json.Marshal(jobServerData)
	data.Logger.Printf("Will run job")
	if err == nil {
		req, err := http.NewRequest("POST", jobServerRootURL+"/new-job", bytes.NewBuffer(jobServerJSON))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		data.Logger.Printf("Starting 'Client.Do'")
		resp, err := client.Do(req)
		data.Logger.Printf("DONE 'Client.Do'")
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK {
				err := json.NewDecoder(resp.Body).Decode(&jobStatus)
				if err == nil {
					data.Logger.Printf("RUNJOB: JobServer ACCEPTED CALL")
					return resp.StatusCode, jobStatus
				} else {
					data.Logger.Printf("RUNJOB: JobServer returned a bad JSON")
					return http.StatusInternalServerError, jobStatus
				}
			} else {
				data.Logger.Printf("RUNJOB: JobServer returned SC != 200 && SC != 202, code =%d", resp.StatusCode)
				return resp.StatusCode, jobStatus
			}
		} else {
			data.Logger.Printf("RUNJOB: Could not connect to JobServer")
			return http.StatusInternalServerError, jobStatus
		}
	} else {
		data.Logger.Printf("RUNJOB: User sent a very bad request :-)")
		return http.StatusBadRequest, jobStatus
	}
}

func JobDataUploaded(uploadId string, uploadIdentifier string) (int, data.TempJobInfo) {
	success, jobData := cydb.JobFullDataForUploadId(uploadId)
	jobData.UploadIdentifier = uploadIdentifier
	if success {
		return http.StatusAccepted, jobData
	} else {
		return http.StatusNotFound, jobData
	}
}

func RunJob(jobData data.TempJobInfo) (int, data.JobResult) {
	return runJob(jobData)
}

func JobStatus(applicationId int, applicationInstanceId int, jobId int) (int, data.JobResult) {
	success, jobInfo := cydb.JobSummaryForJobId(jobId)
	if success {
		if jobInfo.ApplicationId == applicationId && jobInfo.ApplicationInstanceId == applicationInstanceId && jobInfo.JobId == jobId {
			var jobStatus data.JobResult
			var jobStatusRequest string = fmt.Sprintf("{\"job_id\": %d}", jobId)
			req, err := http.NewRequest("POST", jobServerRootURL+"/status", bytes.NewBuffer([]byte(jobStatusRequest)))
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{}
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK {
					err := json.NewDecoder(resp.Body).Decode(&jobStatus)
					if err == nil {
						cydb.UpdateJobStatus(jobId, jobStatus.JobStatus)
						return resp.StatusCode, jobStatus
					} else {
						data.Logger.Printf("JOBSTATUS: JobServer returned a bad JSON")
						return http.StatusInternalServerError, jobStatus
					}
				} else {
					data.Logger.Printf("JOBSTATUS: JobServer returned SC != 200 && SC != 202, code =%d", resp.StatusCode)
					return resp.StatusCode, data.JobResult{JobId: -1, JobStatus: JobStatusERROR, Payload: ""}
				}
			} else {
				data.Logger.Printf("JOBSTATUS: Could not connect to JobServer")
				return http.StatusInternalServerError, jobStatus
			}
		} else { // Don't try accessing other user's data
			return http.StatusUnauthorized, data.JobResult{}
		}
	} else {
		return http.StatusNotFound, data.JobResult{}
	}
}

func JobResult(applicationId int, applicationInstanceId int, jobId int) (int, []byte) {
	success, jobInfo := cydb.JobSummaryForJobId(jobId)
	if success {
		if jobInfo.ApplicationId == applicationId && jobInfo.ApplicationInstanceId == applicationInstanceId && jobInfo.JobId == jobId {
			var jobStatusRequest string = fmt.Sprintf("{\"job_id\": %d}", jobId)
			req, err := http.NewRequest("POST", jobServerRootURL+"/result", bytes.NewBuffer([]byte(jobStatusRequest)))
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{}
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					byteBuffer, err := ioutil.ReadAll(resp.Body)
					data.Logger.Printf("JOBRESULT: RESULT = %s", string(byteBuffer))
					if err == nil {
						cydb.UpdateJobStatus(jobId, JobStatusDone)
						cydb.UpdateJobResultRetrieved(jobId, 1)
						billing.RecordJobDone(jobInfo)
						return resp.StatusCode, byteBuffer
					} else {
						data.Logger.Printf("JOBRESULT: JobServer returned a bad JSON")
						return http.StatusInternalServerError, nil
					}
				} else {
					data.Logger.Printf("JOBRESULT: JobServer returned SC != 200 && SC != 202, code =%d", resp.StatusCode)
					return resp.StatusCode, nil
				}
			} else {
				data.Logger.Printf("JOBRESULT: Could not connect to JobServer")
				return http.StatusInternalServerError, nil
			}
		} else { // Don't try accessing other user's data
			return http.StatusUnauthorized, nil
		}
	} else {
		return http.StatusNotFound, nil
	}
}

func DeleteJob(applicationId int, applicationInstanceId int, jobId int) int {
	return http.StatusOK
}

func ExpireJobs() int {
	return http.StatusOK
}

func CanUploadBinaryData(applicationId int, applicationInstanceId int, uploadId string) bool {
	success, jobInfo := cydb.JobSummaryForUploadId(uploadId)
	if success && jobInfo.ApplicationId == applicationId && jobInfo.ApplicationInstanceId == applicationInstanceId && jobInfo.UploadId == uploadId {
		return true
	} else {
		return false
	}
}
