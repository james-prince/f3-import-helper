package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type endPoint struct {
	LastImport    time.Time `json:"lastImport"`
	NextImport    time.Time `json:"nextImport"`
	ImportRunning bool      `json:"importRunning"`
}

var statusEndPoint endPoint

func serveStatusEndPoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	statusEndPoint.NextImport = Cron.Entries()[0].Schedule.Next(time.Now())

	if err := json.NewEncoder(w).Encode(statusEndPoint); err != nil {
		fmt.Println(err.Error())
	}
}

func serveLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text") //Check text is valid content type
	logID := strings.TrimPrefix(r.URL.Path, "/logs/")
	logFileBytes, err := os.ReadFile(fmt.Sprintf("/logs/%s.log", logID))
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(logFileBytes)
}

func startHttpServer(waitGroup *sync.WaitGroup) {
	http.HandleFunc("/status", serveStatusEndPoint)
	http.HandleFunc("/logs/", serveLog)

	fmt.Printf("json endpoint available at %s/status\n", httpBaseURL)
	waitGroup.Done()

	err := http.ListenAndServe(fmt.Sprintf(":%d", httpListenPort), nil)
	switch {
	case errors.Is(err, http.ErrServerClosed):
		os.Exit(1)
	case err != nil:
		os.Exit(1)
	}
}
