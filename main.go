package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

var (
	TotalMessageCount   int
	TotalWarningCount   int
	TotalErrorCount     int
	NotificationMessage string
)

var Cron = cron.New()

func main() {
	LoadEnvVariables()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			healthCheck()
		}
	}

	if err := CompileRegex(); err != nil {
		panic(err)
	}

	var ListenerWaitGroup sync.WaitGroup
	ListenerWaitGroup.Add(1)
	go startHttpServer(&ListenerWaitGroup)
	ListenerWaitGroup.Wait()

	// Cron = cron.New()
	Cron.AddFunc(CronSchedule, CronProcess)

	if ImportOnStartup {
		var StartupWaitGroup sync.WaitGroup
		StartupWaitGroup.Add(1)
		go func() {
			Process()
			StartupWaitGroup.Done()
		}()
		StartupWaitGroup.Wait()
	}

	Cron.Start()
	fmt.Printf("Next Import: %s\n", Cron.Entries()[0].Schedule.Next(time.Now()))

	select {} // TODO: replace with sync.WaitGroup
	//			https://stackoverflow.com/questions/42752705/prevent-the-main-function-from-terminating-before-goroutines-finish-in-golang
}

func healthCheck() {
	response, err := http.Get(fmt.Sprintf("http://localhost:%d/status", httpListenPort))
	switch {
	case err != nil:
		os.Exit(1)
	case response.StatusCode != 200:
		os.Exit(1)
	}
	os.Exit(0)
}

func CronProcess() {
	Process()
	fmt.Printf("Next Import: %s\n", Cron.Entries()[0].Schedule.Next(time.Now()))
}

func Process() error {
	statusEndPoint.ImportRunning = true
	defer func() {
		statusEndPoint.ImportRunning = false
		statusEndPoint.LastImport = time.Now()
	}()

	fmt.Printf(Blue + "Starting Import Job\n" + Reset)
	fmt.Printf(Blue + "-------------------\n" + Reset)
	lsExecResult, err := Exec(Context, DockerContainerName, []string{"ls", "-1", DockerImportDir})
	if err != nil {
		fmt.Println(lsExecResult.StdErr)
		return err // TODO: replace panic
	}
	LsContents := strings.Split(lsExecResult.StdOut, "\n")

	JsonFileCount := 0
	MaxFileNameLength := 0
	for _, FileName := range LsContents {
		if JsonFileRegex.MatchString(FileName) {
			JsonFileCount += 1
			if len(FileName) > MaxFileNameLength {
				MaxFileNameLength = len(FileName)
			}
		}
	}

	TotalMessageCount = 0
	TotalWarningCount = 0
	TotalErrorCount = 0

	CurrentJsonFile := 0
	for _, FileName := range LsContents {
		if JsonFileRegex.MatchString(FileName) {
			CurrentJsonFile += 1

			// Add trailing whitespace to filename so they appear uniform length on terminal output
			PaddedFileName := FileName
			for uniformFileNameLen := true; uniformFileNameLen; uniformFileNameLen = len(PaddedFileName) < MaxFileNameLength {
				PaddedFileName += " "
			}

			fmt.Printf("[%d/%d] %s ", CurrentJsonFile, JsonFileCount, PaddedFileName)
			if ExecResult, err := ProcessJsonFile(FileName); err != nil {
				logID := uuid.NewString()
				os.WriteFile(fmt.Sprintf("/logs/%s.log", logID), []byte(ExecResult.StdOut), 0644) //Todo, add error check
				fmt.Printf(Red+"X"+Reset+"\tError - log stored at /logs/%s.log\n", logID)
				notification{
					Title:   fmt.Sprintf("[%s] Import Error", FileName),
					Message: fmt.Sprintf("%s/logs/%s", httpBaseURL, logID),
				}.Send()
			}
		}
	}
	if TotalMessageCount+TotalWarningCount+TotalErrorCount == 0 {
		fmt.Println(Green + "No new messages - no notification sent" + Reset)
	} else {
		notification{
			Title:   "Data Imported",
			Message: NotificationMessage,
		}.Send()
	}
	Response, err := http.Get(UptimeKumaUrl)
	switch {
	case err != nil:
		fmt.Printf("Error: %s\n", err.Error())
	case Response.StatusCode != 200:
		fmt.Printf("UptimeKuma Error: %s\n", Response.Status)
	}
	fmt.Printf(Blue + "-------------------\n" + Reset)
	fmt.Printf(Blue + "Import Job Finished\n" + Reset)
	return nil
}

func ProcessJsonFile(FileName string) (ExecResult, error) {
	ExecResult, err := Exec(Context, DockerContainerName, []string{
		"php",
		"artisan",
		"importer:import",
		fmt.Sprintf("%s/%s", DockerImportDir, FileName)})
	if err != nil {
		return ExecResult, err
	}
	switch Output := ExecResult.StdOut; {
	case Output == "":

	}

	MessageCountTextArray := MessageCountExtractRegex.FindStringSubmatch(ExecResult.StdOut)
	WarningCountTextArray := WarningCountExtractRegex.FindStringSubmatch(ExecResult.StdOut)
	ErrorCountTextArray := ErrorCountExtractRegex.FindStringSubmatch(ExecResult.StdOut)

	if len(MessageCountTextArray) != 2 ||
		len(WarningCountTextArray) != 2 ||
		len(ErrorCountTextArray) != 2 {
		// Notify()
		return ExecResult, fmt.Errorf("TextArrays do not all contain 2 elements")
	}
	MessageCount, errMessageCount := strconv.Atoi(MessageCountTextArray[1])
	WarningCount, errWarningCount := strconv.Atoi(WarningCountTextArray[1])
	ErrorCount, errErrorCount := strconv.Atoi(ErrorCountTextArray[1])
	if err := errors.Join(errMessageCount, errWarningCount, errErrorCount); err != nil {
		return ExecResult, err
	}

	TotalMessageCount += MessageCount
	TotalWarningCount += WarningCount
	TotalErrorCount += ErrorCount

	fmt.Printf(Green+"✓"+Reset+"\t%d NEW %d WARNINGS %d ERRORS\n", MessageCount, WarningCount, ErrorCount)

	if MessageCount+WarningCount+ErrorCount == 0 {
		return ExecResult, nil
	}

	DisplayName := strings.ToUpper(strings.TrimSuffix(FileName, filepath.Ext(FileName)))

	NotificationMessageLine := fmt.Sprintf("[%s]", DisplayName)
	if MessageCount > 0 {
		NotificationMessageLine += fmt.Sprintf(" 🆕%d", MessageCount)
	}
	if WarningCount > 0 {
		NotificationMessageLine += fmt.Sprintf(" ⚠️%d", WarningCount)
	}
	if ErrorCount > 0 {
		NotificationMessageLine += fmt.Sprintf(" 🚨%d", ErrorCount)
	}
	if NotificationMessage != "" {
		NotificationMessage += "\n"
	}
	NotificationMessage += NotificationMessageLine
	return ExecResult, nil

}
