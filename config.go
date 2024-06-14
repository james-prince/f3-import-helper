package main

import (
	"fmt"
	"os"
	"strconv"
)

var (
	DockerContainerName string
	DockerImportDir     string
	CronSchedule        string
	GotifyUrl           string
	GotifyPriority      int
	UptimeKumaUrl       string
	ImportOnStartup     bool
	httpListenPort      int
)

func LoadEnvVariables() {
	fmt.Println(Magenta + "Loading ENV Variables" + Reset)
	fmt.Println(Magenta + "---------------------" + Reset)
	EnvVar{Key: "DOCKER_CONTAINER_NAME", Required: true}.applyToString(&DockerContainerName)
	EnvVar{Key: "DOCKER_IMPORT_DIR", Required: true}.applyToString(&DockerImportDir)
	EnvVar{Key: "CRON_SCHEDULE", DefaultString: "@midnight"}.applyToString(&CronSchedule)
	EnvVar{Key: "GOTIFY_URL"}.applyToString(&GotifyUrl)
	EnvVar{Key: "GOTIFY_PRIORITY", DefaultInt: 5}.applyToInt(&GotifyPriority)
	EnvVar{Key: "UPTIMEKUMA_URL"}.applyToString(&UptimeKumaUrl)
	EnvVar{Key: "IMPORT_ON_STARTUP"}.applyToBool(&ImportOnStartup)
	EnvVar{Key: "HTTP_LISTEN_PORT", DefaultInt: 80}.applyToInt(&httpListenPort)
	fmt.Println(Magenta + "---------------------" + Reset)
}

type EnvVar struct {
	Key           string
	Required      bool
	DefaultString string
	DefaultInt    int
	DefaultBool   bool
}

func (e EnvVar) applyToString(Pointer *string) {
	if EnvValue, EnvFound := os.LookupEnv(e.Key); EnvFound {
		*Pointer = EnvValue
		fmt.Printf(Green+"%s: \"%s\"\n"+Reset, e.Key, *Pointer)
		return
	}
	e.exitIfRequired()
	*Pointer = e.DefaultString
	fmt.Printf(Cyan+"%s: \"%s\" (Default)\n"+Reset, e.Key, *Pointer)
}

func (e EnvVar) applyToBool(Pointer *bool) {
	if EnvValue, EnvFound := os.LookupEnv(e.Key); EnvFound {
		if BoolValue, err := strconv.ParseBool(EnvValue); err != nil {
			fmt.Printf(Red+"%s: %s\n"+Reset, e.Key, err.Error())
		} else {
			*Pointer = BoolValue
			fmt.Printf(Green+"%s: %t\n"+Reset, e.Key, *Pointer)
			return
		}
	}
	e.exitIfRequired()
	*Pointer = e.DefaultBool
	fmt.Printf(Cyan+"%s: %t (Default)\n"+Reset, e.Key, *Pointer)
}

func (e EnvVar) applyToInt(Pointer *int) {
	if EnvValue, EnvFound := os.LookupEnv(e.Key); EnvFound {
		if IntValue, err := strconv.Atoi(EnvValue); err != nil {
			fmt.Printf(Red+"%s: %s\n"+Reset, e.Key, err.Error())
		} else {
			*Pointer = IntValue
			fmt.Printf(Green+"%s: %d\n"+Reset, e.Key, *Pointer)
			return
		}
	}
	e.exitIfRequired()
	*Pointer = e.DefaultInt
	fmt.Printf(Cyan+"%s: %d (Default)\n"+Reset, e.Key, *Pointer)
}

func (e EnvVar) exitIfRequired() {
	if e.Required {
		fmt.Printf(Red+"%s ENV not set - exiting application\n"+Reset, e.Key)
		os.Exit(1)
	}
}
