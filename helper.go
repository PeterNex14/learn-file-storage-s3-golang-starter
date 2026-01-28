package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	type aspectRatio struct {
		Streams		[]struct{
			Width		int		`json:"width"`
			Height		int		`json:"height"`
		} `json:"streams"`
	}
	
	var bytes bytes.Buffer
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmd.Stdout = &bytes
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	aspctRatio := aspectRatio{}
	err = json.Unmarshal(bytes.Bytes(), &aspctRatio)
	if err != nil {
		return "" , err
	}

	if len(aspctRatio.Streams) == 0 {
		return "", errors.New("no streams found in ffprobe output")
	}

	ratio := float64(aspctRatio.Streams[0].Width) / float64(aspctRatio.Streams[0].Height)
	if almostEqual(ratio, 16.0/9.0) {
		return "16:9", nil
	} else if almostEqual(ratio, 9.0/16.0) {
		return "9:16", nil
	}

	return "other", nil
}

func almostEqual(a, b float64) bool {
	if a - b < 0 {
		return b - a < 0.05
	}
	return a - b < 0.05
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputPath, nil

}