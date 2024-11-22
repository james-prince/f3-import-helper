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
	default:
		os.Exit(0)
	}
}

func CronProcess() {
	Process()
	fmt.Printf("Next Import: %s\n", Cron.Entries()[0].Schedule.Next(time.Now()))
}

func GetErrorMessage(ExitCode int) string {
	switch ExitCode {
	case 1:
		return "Generic error or unspecified problem during import"
	case 64:
		return "Cannot connect to Firefly III"
	case 65:
		return "Invalid path provided"
	case 66:
		return "Path is not allowed"
	case 67:
		return "There are no files in the provided directory"
	case 68:
		return "Cannot read configuration file"
	case 69:
		return "Cannot parse configuration file"
	case 70:
		return "The importable file cannot be found"
	case 71:
		return "The importable file cannot be read"
	case 72:
		return "Too many errors processing the data in the importable file"
	case 73:
		return "Nothing was imported during this run"
	}
	return ""
}

func Process() error {
	statusEndPoint.ImportRunning = true
	defer func() {
		statusEndPoint.ImportRunning = false
		statusEndPoint.LastImport = time.Now()
	}()

	fmt.Printf(Blue + "Starting Import Job\n" + Reset)
	fmt.Printf(Blue + "-------------------\n" + Reset)

	files, err := getDockerDirContents(DockerImportDir, ".json")
	if err != nil {
		return err
	}

	JsonFileCount := len(files)
	MaxFileNameLength := 0
	for _, dockerFileContent := range files {
		if len(dockerFileContent.FileName) > MaxFileNameLength {
			MaxFileNameLength = len(dockerFileContent.FileName)
		}
	}

	TotalMessageCount = 0
	TotalWarningCount = 0
	TotalErrorCount = 0

	for index, dockerFileContent := range files {
		// Add trailing whitespace to filename so they appear uniform length on terminal output
		PaddedFileName := dockerFileContent.FileName
		for {
			if len(PaddedFileName) >= MaxFileNameLength {
				break
			}
			PaddedFileName += " "
		}
		formatString := "%0" + fmt.Sprintf("%d", len(strconv.Itoa(JsonFileCount))) + "d"

		fmt.Printf("["+formatString+"/"+formatString+"] %s ", index+1, JsonFileCount, PaddedFileName)
		if ExecResult, err := ProcessJsonFile(dockerFileContent.FilePath); err != nil {

			DisplayName := strings.ToUpper(strings.TrimSuffix(dockerFileContent.FileName, filepath.Ext(dockerFileContent.FileName)))

			NotificationMessageLine := fmt.Sprintf("- %s: ExitCode %d (%s)", DisplayName, ExecResult.ExitCode, GetErrorMessage(ExecResult.ExitCode))

			logID, err := recordLog(ExecResult.StdOut)
			if err == nil {
				NotificationMessageLine += fmt.Sprintf(" [View Log](%s/logs/%s)", httpBaseURL, logID)
			}

			if NotificationMessage != "" {
				NotificationMessage += "\n"
			}
			NotificationMessage += NotificationMessageLine
			NotificationMessageLine = ""

			fmt.Printf(Red+"X"+Reset+" | Error - log stored at /logs/%s.log\n", logID)

			TotalErrorCount += 1
			// os.WriteFile(fmt.Sprintf("/logs/%s.log", logID), []byte(ExecResult.StdOut), 0644) //Todo, add error check
			// fmt.Printf(Red+"X"+Reset+" | Error - log stored at /logs/%s.log\n", logID)
			// notificationMessage := fmt.Sprintf("Log stored at **/logs/%s.log**\n\n[Open log in browser](%s/logs/%s)", logID, httpBaseURL, logID)
			// notification{
			// 	Title:   fmt.Sprintf("[%s] Import Error", dockerFileContent.FileName),
			// 	Message: notificationMessage,
			// 	GotifyExtras: &gotifyExtras{
			// 		GotifyClientDisplay: &gotifyClientDisplay{
			// 			GotifyContentType: "text/markdown"}},
			// }.Send()
		}
		// }
	}

	if TotalMessageCount+TotalWarningCount+TotalErrorCount == 0 {
		fmt.Println(Green + "No new messages - no notification sent" + Reset)
	} else {
		notification{
			Title:   "Data Imported",
			Message: NotificationMessage,
			GotifyExtras: &gotifyExtras{
				GotifyClientDisplay: &gotifyClientDisplay{
					GotifyContentType: "text/markdown"}},
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

func recordLog(message string) (string, error) {
	logID := uuid.NewString()
	err := os.WriteFile(fmt.Sprintf("/logs/%s.log", logID), []byte(message), 0644) //Todo, add error check
	return logID, err

}

func ProcessJsonFile(FilePath string) (ExecResult, error) {
	FileName := filepath.Base(FilePath)
	ExecResult, err := Exec(Context, DockerContainerName, "www-data", []string{
		"php",
		"artisan",
		"importer:import",
		FilePath})
	if err != nil {
		return ExecResult, err
	}
	// if ExecResult.ExitCode != 0 {
	// 	return ExecResult, nil
	// }
	// switch Output := ExecResult.StdOut; {
	// case Output == "":

	// }

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

	fmt.Printf(Green+"✓"+Reset+" | %d NEW %d WARNINGS %d ERRORS\n", MessageCount, WarningCount, ErrorCount)

	if MessageCount+WarningCount+ErrorCount == 0 {
		return ExecResult, nil
	}

	DisplayName := strings.ToUpper(strings.TrimSuffix(FileName, filepath.Ext(FileName)))

	NotificationMessageLine := fmt.Sprintf("- %s:", DisplayName)
	if MessageCount > 0 {
		NotificationMessageLine += fmt.Sprintf(" %d NEW", MessageCount)
	}
	if WarningCount > 0 {
		NotificationMessageLine += fmt.Sprintf(" %d WARNINGS", WarningCount)
	}
	if ErrorCount > 0 {
		NotificationMessageLine += fmt.Sprintf(" %d ERRORS", ErrorCount)
	}
	if NotificationMessage != "" {
		NotificationMessage += "\n"
	}
	NotificationMessage += NotificationMessageLine
	NotificationMessageLine = ""
	return ExecResult, nil

}
