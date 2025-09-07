package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	var buf bytes.Buffer

	cmd.Stdout = &buf

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	var s struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	err = json.Unmarshal(buf.Bytes(), &s)
	if err != nil {
		return "", err
	} else if len(s.Streams) == 0 {
		return "", fmt.Errorf("no streams found")
	}

	width := s.Streams[0].Width
	height := s.Streams[0].Height

	ratio := float64(width) / float64(height)

	// Decimal form of ratios for 16:9 and 9:16
	decRatio169 := 16.0 / 9.0
	decRatio916 := 9.0 / 16.0

	// Determines if ratio falls around one of the two within 0.02 tolerance
	if (decRatio169-0.02) < ratio && ratio < (decRatio169+0.02) {
		return "landscape", nil
	} else if (decRatio916-0.02) < ratio && ratio < (decRatio916+0.02) {
		return "portrait", nil
	}

	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {

	outputFilePath := fmt.Sprintf("%v.processing", filePath)

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputFilePath, nil
}
