package main

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	if err := CompileRegex(); err != nil {
		panic(err)
	}

	// Cron = cron.New()

	if ImportOnStartup {
		Process()
	}

	Cron.AddFunc(CronSchedule, CronProcess)
	fmt.Printf("Next Import: %s\n", Cron.Entries()[0].Schedule.Next(time.Now()))
	Cron.Start()

	select {} // TODO: replace with sync.WaitGroup
	//			https://stackoverflow.com/questions/42752705/prevent-the-main-function-from-terminating-before-goroutines-finish-in-golang
}

func CronProcess() {
	Process()
	fmt.Printf("Next Import: %s\n", Cron.Entries()[0].Schedule.Next(time.Now()))
}

func Process() error {
	LsExecResult, err := Exec(Context, DockerContainerName, []string{"ls", "-1", "/import"})
	if err != nil {
		return err // TODO: replace panic
	}
	LsContents := strings.Split(LsExecResult.StdOut, "\n")

	TotalMessageCount = 0
	TotalWarningCount = 0
	TotalErrorCount = 0

	for _, FileName := range LsContents {
		if JsonFileRegex.MatchString(FileName) {
			if ExecResult, err := ProcessJsonFile(FileName); err != nil {
				fmt.Printf("[%s] Error ❌\n", FileName)
				Notify(fmt.Sprintf("[%s] Import Error", FileName), ExecResult.StdOut)
			}
		}
	}
	if TotalMessageCount+TotalWarningCount+TotalErrorCount > 0 {
		if Response, err := Notify("Data Imported", NotificationMessage); err != nil {
			fmt.Printf("Notification Error: %s\n", Response.Status)
		}
	}
	Response, err := http.Get(UptimeKumaUrl)
	switch {
	case err != nil:
		fmt.Printf("Error: %s\n", err.Error())
	case Response.StatusCode != 200:
		fmt.Printf("UptimeKuma Error: %s\n", Response.Status)
	}
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

	fmt.Printf("[%s] Processed - 🆕%d⚠️%d🚨%d\n", FileName, MessageCount, WarningCount, ErrorCount)

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
