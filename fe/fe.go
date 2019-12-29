/*
FE : The main Front-End Server, including authentication, load-balancing and dispatching
Copyright (c) 2018 Imdat Solak
*/
package main

import (
	"auth"
	"billing"
	"bytes"
	"configfile"
	"context"
	"cydb"
	"data"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rakyll/magicmime"
	"io/ioutil"
	"jobs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"services"
	"storage"
	"strconv"
	"time"
)

type MyConfig struct {
	InternalHost string `json:"internal_host"`
	InternalPort int    `json:"internal_port"`
	ExternalHost string `json:"external_host"`
	ExternalPort int    `json:"external_port"`
}

type FEConfiguration struct {
	Me       MyConfig                `json:"me"`
	Database cydb.DatabaseConfig     `json:"database"`
	Jobs     jobs.JobsConfig         `json:"jobs"`
	Billing  billing.BillingConfig   `json:"billing"`
	Storage  storage.StorageConfig   `json:"storage"`
	Services services.ServicesConfig `json:"services"`
}

var apiVersion = "1.0"
var rootURL = "/" + apiVersion
var configuration FEConfiguration

/* Auxiliary Functions */
func getAuthTokenFromRequest(req *http.Request) string {
	return ""
}

func getAuthTokenFromURL(req *http.Request) string {
	body, _ := ioutil.ReadAll(req.Body)
	vars := mux.Vars(req)
	authToken := vars["authToken"]
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return authToken
}

/* Authentication Functions */
func checkClientPermission(req *http.Request, w http.ResponseWriter) (bool, string) {
	authToken := getAuthTokenFromURL(req)
	if auth.IsAuthenticated(authToken) {
		return true, authToken
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		return false, ""
	}
}

/* File Functions */
func UploadFile(w http.ResponseWriter, req *http.Request) {
	data.Logger.Printf("UploadFile called")
	clientPermitted, authToken := checkClientPermission(req, w)
	if clientPermitted {
		vars := mux.Vars(req)
		uploadId := vars["uploadId"]
		data.Logger.Printf("UploadID=%d, authToken=%s", uploadId, authToken)
		_, applicationId, applicationInstanceId := auth.DecodeAndCheckAuthToken(authToken)
		binaryData, err := ioutil.ReadAll(req.Body)
		if err == nil && jobs.CanUploadBinaryData(applicationId, applicationInstanceId, uploadId) {
			if storage.CanUploadBinaryData(applicationId, applicationInstanceId, uploadId, 1024, "image/jpg") {
				success, identifier, _ := storage.UploadBinaryData(applicationId, applicationInstanceId, uploadId, binaryData)
				if success {
					responseCode, jobData := jobs.JobDataUploaded(uploadId, identifier)
					if responseCode == http.StatusAccepted {
						responseCode, jobResponse := jobs.RunJob(jobData)
						if responseCode == http.StatusAccepted {
							w.Header().Set("Content-Type", "application/json; charset=utf-8")
							w.WriteHeader(responseCode)
							json.NewEncoder(w).Encode(jobResponse)
						} else {
							data.Logger.Printf("RunJob-Error")
							w.WriteHeader(responseCode)
						}
					} else {
						data.Logger.Printf("JobDataUploaded")
						w.WriteHeader(responseCode)
					}
				} else {
					w.WriteHeader(http.StatusInsufficientStorage)
				}
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}
}

func DownloadFile(w http.ResponseWriter, req *http.Request) {
	data.Logger.Printf("Download called")
	clientPermitted, authToken := checkClientPermission(req, w)
	if clientPermitted {
		vars := mux.Vars(req)
		identifier := vars["identifier"]
		data.Logger.Printf("UploadID=%d, authToken=%s", identifier, authToken)
		_, applicationId, applicationInstanceId := auth.DecodeAndCheckAuthToken(authToken)
		responseCode, data, contentType, _ := storage.RetrieveBinaryData(applicationId, applicationInstanceId, identifier)
		if responseCode == http.StatusOK {
			w.Header().Set("Content-Type", contentType)
			w.Write(data)
		} else {
			w.WriteHeader(responseCode)
		}
	}
}

/* Job Related Functions */
func getJobInfo(authToken string, w http.ResponseWriter, req *http.Request) (bool, int, int, int) {
	_, applicationId, applicationInstanceId := auth.DecodeAndCheckAuthToken(authToken)
	vars := mux.Vars(req)
	jobId, err := strconv.Atoi(vars["jobId"])
	if err == nil {
		return true, applicationId, applicationInstanceId, jobId
	}
	w.WriteHeader(http.StatusBadRequest)
	return false, 0, 0, 0
}

func JobNew(w http.ResponseWriter, req *http.Request) {
	data.Logger.Printf("JobNew called")
	clientPermitted, authToken := checkClientPermission(req, w)
	if clientPermitted {
		requestBody, _ := ioutil.ReadAll(req.Body)
		_, applicationId, applicationInstanceId := auth.DecodeAndCheckAuthToken(authToken)
		httpResponse, jobData, storageUploadInfo := jobs.CreateNewJob(applicationId, applicationInstanceId, requestBody)
		if httpResponse == http.StatusOK {
			var decodedResult interface{}
			err := json.Unmarshal([]byte(jobData.Payload), &decodedResult)
			if err == nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(httpResponse)
				json.NewEncoder(w).Encode(decodedResult)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else if httpResponse == http.StatusAccepted || httpResponse == http.StatusCreated {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(httpResponse)
			switch jobData.JobStatus {
			case jobs.JobStatusWaitingForFile:
				json.NewEncoder(w).Encode(storageUploadInfo)
			default:
				json.NewEncoder(w).Encode(jobData)
			}
		} else {
			w.WriteHeader(httpResponse)
		}
	}
}

func JobStatus(w http.ResponseWriter, req *http.Request) {
	data.Logger.Printf("JOB-Status called")
	clientPermitted, authToken := checkClientPermission(req, w)
	if clientPermitted {
		success, applicationId, applicationInstanceId, jobId := getJobInfo(authToken, w, req)
		if success {
			httpResponse, jobStatus := jobs.JobStatus(applicationId, applicationInstanceId, jobId)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(httpResponse)
			json.NewEncoder(w).Encode(jobStatus)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

func JobResult(w http.ResponseWriter, req *http.Request) {
	data.Logger.Printf("JOB-Result called")
	clientPermitted, authToken := checkClientPermission(req, w)
	if clientPermitted {
		success, applicationId, applicationInstanceId, jobId := getJobInfo(authToken, w, req)
		if success {
			var decodedResult interface{}
			httpResponse, jobResult := jobs.JobResult(applicationId, applicationInstanceId, jobId)
			if jobResult != nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
			}
			w.WriteHeader(httpResponse)

			if jobResult != nil {
				err := json.Unmarshal(jobResult, &decodedResult)
				if err == nil {
					json.NewEncoder(w).Encode(decodedResult)
				}
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

func JobDelete(w http.ResponseWriter, req *http.Request) {
	data.Logger.Printf("JOBDelete called")
	clientPermitted, authToken := checkClientPermission(req, w)
	if clientPermitted {
		success, applicationId, applicationInstanceId, jobId := getJobInfo(authToken, w, req)
		if success {
			httpResponse := jobs.DeleteJob(applicationId, applicationInstanceId, jobId)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(httpResponse)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

/* Main Functions */
func GoHome(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, "http://www.marcurie.eu/", 301)
}

func Authenticate(w http.ResponseWriter, req *http.Request) {
	data.Logger.Printf("Authenticate called")
	var authRequestJSON auth.AuthenticationRequest
	err := json.NewDecoder(req.Body).Decode(&authRequestJSON)
	if err != nil {
		data.Logger.Print("ISO: Error Unmarshalling: %d", err)
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		success, authResponse := auth.Authenticate(authRequestJSON)
		if success {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(w).Encode(*authResponse)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}
}

func AvailableServices(w http.ResponseWriter, req *http.Request) {
	data.Logger.Printf("AvailableServices called")
	clientPermitted, _ := checkClientPermission(req, w)
	if clientPermitted {
		var emptyArray []string = make([]string, 0, 1)
		var retServices []interface{}
		var serviceInfo interface{}
		availableServices := jobs.AvailableServices()
		for _, service := range availableServices {
			err := json.Unmarshal([]byte(service.About), &serviceInfo)
			if err == nil {
				retServices = append(retServices, serviceInfo)
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if retServices != nil {
			json.NewEncoder(w).Encode(retServices)
		} else {
			json.NewEncoder(w).Encode(emptyArray)
		}
	} else {
		w.WriteHeader(http.StatusUnauthorized)
	}
}

func main() {
	var wait time.Duration
	data.Logger = log.New(os.Stdout, "MARCURIE (fe) - ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if err := magicmime.Open(magicmime.MAGIC_MIME_TYPE | magicmime.MAGIC_SYMLINK | magicmime.MAGIC_ERROR); err != nil {
		data.Logger.Fatal(err)
	}
	defer magicmime.Close()
	configuration = FEConfiguration{}
	if configfile.ReadConfiguration("config.json", &configuration) == false {
		data.Logger.Printf("Missing my config file in local directory")
		os.Exit(1)
	}
	cydb.OpenDatabase(configuration.Database)
	storage.InitStorage(configuration.Storage)
	jobs.InitJobs(configuration.Jobs)
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	router := mux.NewRouter()
	router.HandleFunc("/", GoHome)

	/* AUTHENTICATION METHODS */
	router.HandleFunc(rootURL+"/auth", Authenticate)

	/* Available Services & Info */
	router.HandleFunc(rootURL+"/available-services/{authToken}", AvailableServices)

	/* Job Related Methods */
	router.HandleFunc(rootURL+"/job/new/{authToken}", JobNew)
	router.HandleFunc(rootURL+"/job/status/{authToken}/{jobId}", JobStatus)
	router.HandleFunc(rootURL+"/job/result/{authToken}/{jobId}", JobResult)
	router.HandleFunc(rootURL+"/job/delete/{authToken}/{jobId}", JobDelete)

	/* UPLOAD METHODS */
	router.HandleFunc(rootURL+"/upload/{authToken}/{uploadId}", UploadFile)
	router.HandleFunc(rootURL+"/download/{authToken}/{identifier}", DownloadFile)

	/* Prepare our server */
	myAddr := fmt.Sprintf("%s:%d", configuration.Me.InternalHost, configuration.Me.InternalPort)
	srv := &http.Server{
		Addr: myAddr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router, // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			data.Logger.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	data.Logger.Println("shutting down")
	cydb.CloseDatabase()
	os.Exit(0)
}
