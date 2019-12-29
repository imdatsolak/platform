package cydb

import (
	"data"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"regexp"
	"time"
)

type AccountData struct {
	AccountId          int
	AccountName        string
	Login              string
	Password           string
	BackupEmail        string
	BackupMobilePhone  string
	NotificationsEmail string
	NotificationsPhone string
	State              int
	CreationDate       time.Time
}

type DatabaseConfig struct {
	DBType     string `json:"db_type"`
	DBHost     string `json:"db_host"`
	DBPort     int    `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	Database   string `json:"database"`
	DBFlags    string `json:"db_flags"`
}

var mysql_db *sql.DB
var mysql_db_config DatabaseConfig

func OpenDatabase(config DatabaseConfig) bool {
	var err error
	dbstring := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s", config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.Database, config.DBFlags)
	data.Logger.Printf("DBString = [%s]", dbstring)
	// mysql_db, err = sql.Open("mysql", "cygnusa:cygnusa@tcp(10.0.2.152:3306)/cygnusa?parseTime=true")
	mysql_db, err = sql.Open(config.DBType, dbstring)
	if err != nil {
		return false
	}
	mysql_db.SetConnMaxLifetime(time.Duration(-1))
	return true
}

func CloseDatabase() {
	mysql_db.Close()
}

/*
 * Authentication & Application Related Database Functions
 */

func LoginUser(login string, password string) (bool, AccountData) {
	var accountData AccountData

	return true, accountData
}

func LoginApplication(applicationLogin string, applicationSecret string) (bool, int) {
	var applicationId int = -1

	data.Logger.Printf("applicationLogin = %s, applicationSecret=%s", applicationLogin, applicationSecret)
	err := mysql_db.QueryRow("SELECT applicationId from Applications where applicationLogin = ? and applicationSecret = ? and disabled=0", applicationLogin, applicationSecret).Scan(&applicationId)
	if err == nil && applicationId > 0 {
		return true, applicationId
	}
	return false, -1
}

func getApplicationInstanceId(applicationId int, applicationInstanceUID string) int {
	var applicationInstanceId int = -1

	err := mysql_db.QueryRow("SELECT aInstanceId from ApplicationInstances where applicationId =? and applicationInstanceUID=? and disabled=0", applicationId, applicationInstanceUID).Scan(&applicationInstanceId)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0
		} else {
			return -1
		}
	}
	return applicationInstanceId
}

func RegisterApplicationInstanceIfNeeded(applicationId int, applicationInstanceUID string) (bool, int) {
	var applicationInstanceId int = getApplicationInstanceId(applicationId, applicationInstanceUID)

	if applicationInstanceId > 0 {
		return true, applicationInstanceId
	} else if applicationInstanceId == 0 {
		insert, err := mysql_db.Query("INSERT INTO ApplicationInstances VALUES(0, ?, ?, 0)", applicationId, applicationInstanceUID)
		defer insert.Close()
		if err == nil {
			applicationInstanceId = getApplicationInstanceId(applicationId, applicationInstanceUID)
			if applicationInstanceId > 0 {
				return true, applicationInstanceId
			}
		} else {
			data.Logger.Printf("RegisterApplicationInstance -> INSERT ERROR %s", err)
		}
	}
	return false, -1
}

/*
 * Job Related Database Functions
 */
func AddNewJobInfo(jobData data.TempJobInfo) int {
	var reqData string
	re := regexp.MustCompile(`\r?\n`)
	reqData = re.ReplaceAllString(jobData.RequestData, " ")
	insert, err := mysql_db.Query("INSERT INTO TempJobs VALUES(0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)", jobData.ApplicationId, jobData.ApplicationInstanceId, jobData.JobUID, jobData.JobStatus, jobData.RequestType, jobData.RequestStartTime, jobData.RequestSize, reqData, jobData.UploadId, jobData.RequestEndTime, jobData.ProcessingTime, jobData.JobResultData, jobData.UploadIdentifier)
	if err == nil {
		var jobId int = -1
		defer insert.Close()
		err = mysql_db.QueryRow("SELECT jobId from TempJobs WHERE jobUID = ?", jobData.JobUID).Scan(&jobId)
		if err == nil {
			return jobId
		}
	} else {
		data.Logger.Printf("Err %s", err)
	}
	return -1
}

func DeleteJobInfo(jobId int) {
}

func UpdateJobUploadIdentifier(jobId int, uploadIdentifier string) bool {
	update, err := mysql_db.Query("UPDATE TempJobs SET uploadIdentifier = ? where jobId = ?", uploadIdentifier, jobId)
	if err == nil {
		defer update.Close()
		return true
	}
	return false
}

func JobFullDataForUploadId(uploadId string) (bool, data.TempJobInfo) {
	var resultInfo data.TempJobInfo

	err := mysql_db.QueryRow("SELECT jobId, applicationId, applicationInstanceId, jobUID, jobStatus, requestType, requestStartTime, requestSize, requestData, uploadId, requestEndTime, processingTime, jobResultDataPtr, jobResultRetrieved, uploadIdentifier from TempJobs where uploadId = ?", uploadId).Scan(
		&resultInfo.JobId,
		&resultInfo.ApplicationId,
		&resultInfo.ApplicationInstanceId,
		&resultInfo.JobUID,
		&resultInfo.JobStatus,
		&resultInfo.RequestType,
		&resultInfo.RequestStartTime,
		&resultInfo.RequestSize,
		&resultInfo.RequestData,
		&resultInfo.UploadId,
		&resultInfo.RequestEndTime,
		&resultInfo.ProcessingTime,
		&resultInfo.JobResultData,
		&resultInfo.JobResultRetrieved,
		&resultInfo.UploadIdentifier)
	if err == nil {
		return true, resultInfo
	} else {
		data.Logger.Printf("Err = %s", err)
		return false, resultInfo
	}
}

func JobSummaryForUploadId(uploadId string) (bool, data.TempJobInfo) {
	var resultInfo data.TempJobInfo

	err := mysql_db.QueryRow("SELECT applicationId, applicationInstanceId, uploadId from TempJobs where uploadId = ?", uploadId).Scan(
		&resultInfo.ApplicationId,
		&resultInfo.ApplicationInstanceId,
		&resultInfo.UploadId)
	if err == nil {
		return true, resultInfo
	} else {
		return false, resultInfo
	}
}

func JobSummaryForJobId(jobId int) (bool, data.TempJobInfo) {
	var resultInfo data.TempJobInfo

	err := mysql_db.QueryRow("SELECT jobId, applicationId, applicationInstanceId, jobStatus, uploadId from TempJobs where jobId = ?", jobId).Scan(
		&resultInfo.JobId,
		&resultInfo.ApplicationId,
		&resultInfo.ApplicationInstanceId,
		&resultInfo.JobStatus,
		&resultInfo.UploadId)
	if err == nil {
		return true, resultInfo
	} else {
		return false, resultInfo
	}
}

func JobFullDataForJobId(jobId int) (bool, data.TempJobInfo) {
	var resultInfo data.TempJobInfo

	err := mysql_db.QueryRow("SELECT jobId, applicationId, applicationInstanceId, jobUID, jobStatus, requestType, requestStartTime, requestSize, requestData, uploadId, requestEndTime, processingTime, jobResultDataPtr, jobResultRetrieved, uploadIdentifier from TempJobs where jobId = ?", jobId).Scan(
		&resultInfo.JobId,
		&resultInfo.ApplicationId,
		&resultInfo.ApplicationInstanceId,
		&resultInfo.JobUID,
		&resultInfo.JobStatus,
		&resultInfo.RequestType,
		&resultInfo.RequestStartTime,
		&resultInfo.RequestSize,
		&resultInfo.RequestData,
		&resultInfo.UploadId,
		&resultInfo.RequestEndTime,
		&resultInfo.ProcessingTime,
		&resultInfo.JobResultData,
		&resultInfo.JobResultRetrieved,
		&resultInfo.UploadIdentifier)
	if err == nil {
		return true, resultInfo
	} else {
		data.Logger.Printf("Err = %s", err)
		return false, resultInfo
	}
}

func UpdateJobStatus(jobId int, jobStatus int) {
	update, err := mysql_db.Query("UPDATE TempJobs SET jobStatus = ? where jobId = ?", jobStatus, jobId)
	if err == nil {
		defer update.Close()
	} else {
		data.Logger.Printf("JobStatus Retrieved UPDATE ERROR, %s", err)
	}
}

func UpdateJobResultRetrieved(jobId int, resultRetrieved int) {
	update, err := mysql_db.Query("UPDATE TempJobs SET jobResultRetrieved = ? where jobId = ?", resultRetrieved, jobId)
	if err == nil {
		defer update.Close()
	} else {
		data.Logger.Printf("JobResult Retrieved UPDATE ERROR, %s", err)
	}
}

func RecordJobDoneInDB(jobId int) (bool, int) {
	var reqData string
	var jobData data.TempJobInfo
	var success bool = false

	re := regexp.MustCompile(`\r?\n`)

	success, jobData = JobFullDataForJobId(jobId)
	if success {
		reqData = re.ReplaceAllString(jobData.RequestData, " ")
		insert, err := mysql_db.Query("INSERT INTO DoneJobs VALUES(0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", jobData.JobId, jobData.ApplicationId, jobData.ApplicationInstanceId, jobData.JobUID, jobData.RequestType, jobData.RequestStartTime, jobData.RequestSize, reqData, jobData.RequestEndTime, jobData.ProcessingTime)
		if err == nil {
			jobId = -1
			defer insert.Close()
			err = mysql_db.QueryRow("SELECT doneJobId from DoneJobs WHERE jobUID = ?", jobData.JobUID).Scan(&jobId)
			if err == nil {
				return true, jobId
			}
		} else {
			data.Logger.Printf("UPDATE DoneJobs: Err %s", err)
		}
	}
	return false, -1
}

/*
 * Service Discovery Related Database Methods
 */

func cleanServicesTable() {
	delete, err := mysql_db.Query("DELETE FROM AvailableServices")
	if err == nil {
		defer delete.Close()
	}
}

func UpdateAvailableServices(services data.ServiceInfoList) bool {
	var serviceList []byte
	var serviceJSON string

	serviceList, err := json.Marshal(services)
	if err == nil {
		serviceJSON = string(serviceList)
		cleanServicesTable()
		insert, err := mysql_db.Query("INSERT INTO AvailableServices VALUES(0, ?)", serviceJSON)
		if err == nil {
			defer insert.Close()
			return true
		}
	}
	return false
}

func LastAvailableServices() data.ServiceInfoList {
	var serviceJSON string

	err := mysql_db.QueryRow("SELECT services FROM AvailableServices").Scan(&serviceJSON)
	if err == nil {
		var sil data.ServiceInfoList
		var serviceList []byte = []byte(serviceJSON)
		err = json.Unmarshal(serviceList, &sil)
		if err == nil {
			return sil
		}
	}
	return nil
}
