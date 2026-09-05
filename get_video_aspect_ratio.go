package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

type FFProbeOutput struct {
	Streams []FFProbeStream `json:"streams"`
}

type FFProbeStream struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

const (
	landscapeRatio = 16 / 9
	portraitRatio  = 9 / 16
)

func getVideoAspectRatio(filePath string) (string, error) {
	var ffprobeOutput FFProbeOutput
	command := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	buffer := bytes.Buffer{}
	command.Stdout = &buffer

	command.Run()

	jsonDecoder := json.NewDecoder(&buffer)

	if err := jsonDecoder.Decode(&ffprobeOutput); err != nil {
		return "", err
	}

	height := ffprobeOutput.Streams[0].Height
	width := ffprobeOutput.Streams[0].Width
	actualRatio := width / height

	if actualRatio == landscapeRatio {

		return "landscape", nil
	}
	if actualRatio == portraitRatio {

		return "portrait", nil
	}

	return "other", nil
}
