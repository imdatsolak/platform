/*
FE : The main Service Discovery ServerServer, including authentication, load-balancing and dispatching
Copyright (c) 2018 Imdat Solak
*/
package main

import (
	"configfile"
	"context"
	"cydb"
	"data"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type MyConfig struct {
	Host string `json:"listen_host"`
	Port int    `json:"listen_port"`
}

type SDConfiguration struct {
	Me                MyConfig            `json:"me"`
	Database          cydb.DatabaseConfig `json:"database"`
	HeartbeatInterval int                 `json:"heartbeat_interval"`
}

var apiVersion = "1.0"
var rootURL = "/" + apiVersion
var configuration SDConfiguration
var currentServices data.ServiceInfoList

func isServiceStillAlive(aService data.ServiceInfo) bool {
	var heartbeatURL string
	heartbeatURL = aService.HeartbeatURL
	req, err := http.NewRequest("GET", heartbeatURL, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return true
		}
	}
	data.Logger.Printf("Service ", aService, " NOT ALIVE ANYMORE --- REMOVING")
	return false
}

func allServicesAreStillAvailable() bool {
	var newS data.ServiceInfoList = make(data.ServiceInfoList, 0, 1)

	for _, aService := range currentServices {
		if isServiceStillAlive(aService) {
			newS = append(newS, aService)
		}
	}
	if len(currentServices) != len(newS) {
		currentServices = newS
		return false
	} else {
		return true
	}
}

func regularlyCheckServiceStatus() {
	for {
		if !allServicesAreStillAvailable() {
			cydb.UpdateAvailableServices(currentServices)
		}
		time.Sleep(time.Duration(configuration.HeartbeatInterval) * time.Second)
	}
}

func appendNewService(newService data.ServiceInfo) data.ServiceInfoList {
	var newServiceServer string = newService.Server
	var newServicePort int = newService.Port
	var newS data.ServiceInfoList = make(data.ServiceInfoList, 0, len(currentServices))

	for _, existingService := range currentServices {
		if existingService.Port > 0 && (existingService.Server != newServiceServer || existingService.Port != newServicePort) {
			newS = append(newS, existingService)
		}
	}
	return append(newS, newService)
}

func updateAvailableServices() {
	var las data.ServiceInfoList = cydb.LastAvailableServices()
	if las != nil {
		currentServices = las
		if !allServicesAreStillAvailable() {
			cydb.UpdateAvailableServices(currentServices)
		}
	} else {
		currentServices = make(data.ServiceInfoList, 0, 100)
	}
}

func GetAvailableServices(w http.ResponseWriter, req *http.Request) {
	data.Logger.Printf("SD: Someone asking for AVAILABLE-SERVICES\n")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(currentServices)
}

func RegisterService(w http.ResponseWriter, req *http.Request) {
	var serviceData data.ServiceInfo
	err := json.NewDecoder(req.Body).Decode(&serviceData)
	if err == nil {
		data.Logger.Printf("Registering Service:", serviceData)
		currentServices = appendNewService(serviceData)
		cydb.UpdateAvailableServices(currentServices)
		w.WriteHeader(http.StatusOK)
	} else {
		data.Logger.Printf("Register Err %s", err)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func main() {
	var wait time.Duration
	configuration = SDConfiguration{}
	data.Logger = log.New(os.Stdout, "MARCURIE (sd) - ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if configfile.ReadConfiguration("config.json", &configuration) == false {
		data.Logger.Printf("Missing my config file in local directory")
		os.Exit(1)
	}
	cydb.OpenDatabase(configuration.Database)
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	router := mux.NewRouter()

	router.HandleFunc(rootURL+"/register-service", RegisterService)
	router.HandleFunc(rootURL+"/available-services", GetAvailableServices)

	go regularlyCheckServiceStatus()
	/* Prepare our server */
	myAddr := fmt.Sprintf("%s:%d", configuration.Me.Host, configuration.Me.Port)
	srv := &http.Server{
		Addr: myAddr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router, // Pass our instance of gorilla/mux in.
	}
	updateAvailableServices()

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
