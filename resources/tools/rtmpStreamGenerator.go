package main

import (
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// Stream represents an RTMP stream
type Stream struct {
	URL    string
	Width  int
	Height int
	FPS    int
}

// startStream starts a synthetic video stream to the RTMP server
func startStream(s Stream, wg *sync.WaitGroup) {
	defer wg.Done()

	// FFmpeg command to generate test pattern and stream to RTMP
	cmd := exec.Command(
		"ffmpeg",
		"-f", "lavfi",
		"-i", fmt.Sprintf("testsrc=size=%dx%d:rate=%d", s.Width, s.Height, s.FPS),
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-f", "flv",
		s.URL,
	)

	fmt.Printf("Starting stream to %s\n", s.URL)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error streaming to %s: %v\n", s.URL, err)
	}
}

func main() {
	var wg sync.WaitGroup

	numStreams := 10 // number of streams to generate
	baseURL := "rtmp://localhost/stream"

	for i := 1; i <= numStreams; i++ {
		stream := Stream{
			URL:    fmt.Sprintf("%s%d", baseURL, i),
			Width:  1920,
			Height: 1080,
			FPS:    30,
		}
		wg.Add(1)
		go startStream(stream, &wg)
		time.Sleep(20 * time.Second)
	}

	wg.Wait()
	fmt.Println("All streams finished.")
}
