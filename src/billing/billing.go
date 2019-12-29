package billing

import (
	"cydb"
	"data"
)

type BillingConfig struct {
	ServerHost string `json:"server_host"`
	ServerPort int    `json:"server_port"`
}

func RecordJobDone(jobInfo data.TempJobInfo) bool {
	success, _ := cydb.RecordJobDoneInDB(jobInfo.JobId)
	return success
}
