package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type notification struct {
	Title        string        `json:"title"`
	Message      string        `json:"message"`
	Priority     *int          `json:"priority"`
	GotifyExtras *gotifyExtras `json:"extras"`
}

type gotifyExtras struct {
	GotifyClientNotification *gotifyClientNotification `json:"client::notification"`
	GotifyClientDisplay      *gotifyClientDisplay      `json:"client::display"`
	GotifyAndroidAction      *gotifyAndroidAction      `json:"android::action"`
}

type gotifyClientNotification struct {
	GotifyClick *gotifyClick `json:"click"`
}

type gotifyClientDisplay struct {
	ContentType string `json:"contentType"`
}

type gotifyAndroidAction struct {
	GotifyOnReceive *gotifyOnReceive `json:"onReceive"`
}

type gotifyOnReceive struct {
	IntentURL string `json:"intentUrl"`
}

type gotifyClick struct {
	Url string `json:"url"`
}

func (n notification) Send() {
	var err error
	switch {
	case GotifyUrl != "":
		err = n.sendGotify()
	default:
		fmt.Println()
		return
	}
	if err != nil {
		fmt.Printf(Red+"Notification error:"+Reset+" %s\n", err.Error())
	} else {
		fmt.Printf(Green+"Notification sent:"+Reset+" %s\n", n.Title)
	}
}

func (n notification) sendGotify() error {
	if n.Priority == nil {
		n.Priority = &GotifyPriority
	}
	marshalled, err := json.Marshal(n)
	if err != nil {
		return err
	}
	response, err := http.Post(GotifyUrl, "application/json", bytes.NewReader(marshalled))
	switch {
	case err != nil:
		return err
	case response.StatusCode != 200:
		return fmt.Errorf("status: %d message: %s", response.StatusCode, response.Status)
	}
	return nil
}

// func Notify(Title string, Message string) {
// 	if GotifyUrl != "" {
// 		if err := NotifyGotify(Title, Message, GotifyPriority); err != nil {
// 			fmt.Printf(Red+"[Gotify] error: %s\n"+Reset, err.Error())
// 		} else {
// 			fmt.Printf(Green+"[Gotify] Notification sent: %s\n"+Reset, Title)
// 		}
// 	}
// }

// func NotifyGotify(Title string, Message string, Priority int) error {
// 	type GotifyPOST struct {
// 		Title    string `json:"title"`
// 		Message  string `json:"message"`
// 		Priority int    `json:"priority"`
// 	}
// 	PostBody := GotifyPOST{
// 		Title:    Title,
// 		Message:  Message,
// 		Priority: Priority,
// 	}

// 	Marshalled, err := json.Marshal(PostBody)
// 	if err != nil {
// 		return err
// 	}
// 	Response, err := http.Post(GotifyUrl, "application/json", bytes.NewReader(Marshalled))
// 	if err != nil {
// 		return err
// 	}
// 	if Response.StatusCode != 200 {
// 		return fmt.Errorf("status: %d message: %s", Response.StatusCode, Response.Status)
// 	}
// 	return nil
// }
