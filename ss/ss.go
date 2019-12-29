/*
SS : The main StorageServer
Copyright (c) 2018 Imdat Solak
*/
package main

import (
	"bytes"
	"context"
	"data"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rakyll/magicmime"
	"github.com/twinj/uuid"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

var apiVersion = "1.0"
var rootURL = "/" + apiVersion

var storagePath string = "/tmp/ul"
var storageTTL time.Duration = time.Hour

var globalUploadId int = 100
var globalCreateId int = 100

func CreateNewUploadId() string {
	var newUID string
	newUID = uuid.NewV4().String()
	data.Logger.Printf("new Upload ID: %s", newUID)
	return newUID
}

func CanUploadBinaryData(applicationId int, applicationInstanceId int, uploadId string, binaryDataLen int, mimeType string) bool {
	_, err := uuid.Parse(uploadId)
	if err == nil {
		return true
	}
	return false
}

func storeBinaryData(applicationId int, applicationInstanceId int, identifier string, binaryData []byte, ttl int64) (bool, time.Time) {
	filename := fmt.Sprintf("%s/%d_%d_%s", storagePath, applicationId, applicationInstanceId, identifier)
	f, err := os.Create(filename)
	if err != nil {
		data.Logger.Print("Could not create file")
		return false, time.Now()
	}
	defer f.Close()
	f.Write(binaryData)
	return true, time.Now().Add(time.Duration(ttl) * time.Second)
}

func storeUploadedBinaryData(applicationId int, applicationInstanceId int, uploadId string, binaryData []byte, ttl int64) (success bool, identifier string, expires time.Time) {
	saved, expires := storeBinaryData(applicationId, applicationInstanceId, uploadId, binaryData, ttl)
	return saved, uploadId, expires
}

func retrieveBinaryData(applicationId int, applicationInstanceId int, identifier string) ([]byte, string, time.Time) {
	var contentType string
	filename := fmt.Sprintf("%s/%d_%d_%s", storagePath, applicationId, applicationInstanceId, identifier)
	content, err := ioutil.ReadFile(filename)
	if err == nil {
		data.Logger.Printf("File %s found and opened", filename)
		data.Logger.Printf("Content Length: %d...", len(content))
		if len(content) > 0 {
			contentType, err = magicmime.TypeByBuffer(content)
			if err != nil {
				contentType = "application/octet-stream"
			}
			return content, contentType, time.Now().Add(storageTTL)
		}
	} else {
		data.Logger.Printf("Could not open file %s, err = %s", identifier, err)
	}
	return nil, "", time.Now()
}

func doDeleteBinaryData(applicationId int, applicationInstanceId int, identifier string) bool {
	return true
}

/* Auxiliary Functions */

func getVars(req *http.Request) map[string]string {
	body, _ := ioutil.ReadAll(req.Body)
	vars := mux.Vars(req)
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return vars
}

func uploadBinaryDataInt(w http.ResponseWriter, req *http.Request, applicationId int, applicationInstanceId int, uploadId string) {
	binaryData, err := ioutil.ReadAll(req.Body)
	if err == nil && CanUploadBinaryData(applicationId, applicationInstanceId, uploadId, 1024, "image/jpg") {
		if len(binaryData) == 0 {
			w.WriteHeader(http.StatusExpectationFailed)
		} else {
			var ttl int64 = 900 // 15 Minutes time for keeping the data
			success, identifier, expires := storeUploadedBinaryData(applicationId, applicationInstanceId, uploadId, binaryData, ttl)
			if success {
				data.Logger.Printf("Successfully saved data with identifier = %s", identifier)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				json.NewEncoder(w).Encode(data.StorageServerUploadResponse{BinaryDataId: identifier, Expires: expires})
			} else {
				data.Logger.Printf("Could not store file = %s", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	} else {
		data.Logger.Printf("Reading Upload Body Failed = %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

/* File API Functions */
func CanUploadData(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func UploadBinaryData(w http.ResponseWriter, req *http.Request) {
	var vars = getVars(req)
	uploadId := vars["uploadId"]
	applicationId, err := strconv.Atoi(vars["applicationId"])
	applicationInstanceId, err := strconv.Atoi(vars["applicationInstanceId"])
	data.Logger.Printf("Upload Request received uploadId: %s, applicationId: %d, applicationInstanceId %d", uploadId, applicationId, applicationInstanceId)
	if err == nil {
		uploadBinaryDataInt(w, req, applicationId, applicationInstanceId, uploadId)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func GetBinaryData(w http.ResponseWriter, req *http.Request) {
	var vars = getVars(req)
	identifier := vars["identifier"]
	applicationId, _ := strconv.Atoi(vars["applicationId"])
	applicationInstanceId, _ := strconv.Atoi(vars["applicationInstanceId"])
	binaryData, contentType, expires := retrieveBinaryData(applicationId, applicationInstanceId, identifier)
	if len(binaryData) > 0 && expires.After(time.Now()) {
		w.Header().Set("Content-Type", contentType)
		w.Write(binaryData)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func DeleteBinaryData(w http.ResponseWriter, req *http.Request) {
	var vars = getVars(req)
	identifier := vars["identifier"]
	applicationId, _ := strconv.Atoi(vars["applicationId"])
	applicationInstanceId, _ := strconv.Atoi(vars["applicationInstanceId"])
	if doDeleteBinaryData(applicationId, applicationInstanceId, identifier) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func UploadBinaryDataInternal(w http.ResponseWriter, req *http.Request) {
	var vars = getVars(req)
	applicationId, _ := strconv.Atoi(vars["applicationId"])
	applicationInstanceId, _ := strconv.Atoi(vars["applicationInstanceId"])
	uploadId := CreateNewUploadId()
	uploadBinaryDataInt(w, req, applicationId, applicationInstanceId, uploadId)
}

func NewUploadId(w http.ResponseWriter, req *http.Request) {
	uploadId := CreateNewUploadId()
	if uploadId != "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(data.UploadIdT{UploadId: uploadId})
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func ExpireBinaryData() {
}

/*
We can handle the authentication probably better using a
MUX Middleware (check out https://github.com/gorilla/mux#middleware)
*/
func main() {
	var wait time.Duration
	uuid.Init()
	data.Logger = log.New(os.Stdout, "MARCURIE (ss) - ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	if err := magicmime.Open(magicmime.MAGIC_MIME_TYPE | magicmime.MAGIC_SYMLINK | magicmime.MAGIC_ERROR); err != nil {
		data.Logger.Fatal(err)
	}
	defer magicmime.Close()
	router := mux.NewRouter()

	/* EXTERNAL API-CALLS*/
	router.HandleFunc(rootURL+"/new-upload-id", NewUploadId)
	router.HandleFunc(rootURL+"/can-upload-data", CanUploadData)
	router.HandleFunc(rootURL+"/upload/{applicationId}/{applicationInstanceId}/{uploadId}", UploadBinaryData)
	router.HandleFunc(rootURL+"/download/{applicationId}/{applicationInstanceId}/{identifier}", GetBinaryData)
	router.HandleFunc(rootURL+"/delete-data/{applicationId}/{applicationInstanceId}/{identifier}", DeleteBinaryData)
	router.HandleFunc(rootURL+"/store-data/{applicationId}/{applicationInstanceId}", UploadBinaryDataInternal)

	/* Prepare our server */
	srv := &http.Server{
		Addr: "0.0.0.0:9500",
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
	os.Exit(0)

}
