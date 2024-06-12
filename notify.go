package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func Notify(Title string, Message string) (*http.Response, error) {
	if GotifyUrl != "" {
		Response, err := NotifyGotify(Title, Message, GotifyPriority)
		return Response, err
	}
	return &http.Response{}, nil
}

func NotifyGotify(Title string, Message string, Priority int) (*http.Response, error) {
	type GotifyPOST struct {
		Title    string `json:"title"`
		Message  string `json:"message"`
		Priority int    `json:"priority"`
	}
	PostBody := GotifyPOST{
		Title:    Title,
		Message:  Message,
		Priority: Priority,
	}

	Marshalled, err := json.Marshal(PostBody)
	if err != nil {
		return nil, err
	}
	Response, err := http.Post(GotifyUrl, "application/json", bytes.NewReader(Marshalled))
	if err != nil {
		return nil, err
	}
	if Response.StatusCode != 200 {
		return nil, fmt.Errorf("status: %d message: %s", Response.StatusCode, Response.Status)
	}
	return Response, nil
}
