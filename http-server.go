package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
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

func startHttpServer(waitGroup *sync.WaitGroup) {
	http.HandleFunc("/status", serveStatusEndPoint)

	fmt.Printf("json endpoint available at http://localhost:%d/status\n", httpListenPort)
	waitGroup.Done()

	err := http.ListenAndServe(fmt.Sprintf(":%d", httpListenPort), nil)
	switch {
	case errors.Is(err, http.ErrServerClosed):
		os.Exit(1)
	case err != nil:
		os.Exit(1)
	}
}
