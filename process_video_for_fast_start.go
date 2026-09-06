package main

import (
	"fmt"
	"os/exec"
)

func ProcessVideoForFastStart(filePath string) (string, error) {
	outputFilePath := fmt.Sprintf("%s.processing", filePath)

	fmt.Printf("OUTPUT FILE PATH: %s \n", outputFilePath)
	fmt.Printf("INPUT FILE PATH: %s \n", filePath)

	command := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)

	if err := command.Run(); err != nil {
		return "", err
	}

	return outputFilePath, nil
}
