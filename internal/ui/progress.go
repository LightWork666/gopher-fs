package ui

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// ProgressWriter tracks the number of bytes written and updates a progress bar
type ProgressWriter struct {
	Total      int64
	Current    int64
	Writer     io.Writer
	startTime  time.Time
	lastUpdate time.Time
}

func NewProgressWriter(total int64, w io.Writer) *ProgressWriter {
	return &ProgressWriter{
		Total:     total,
		Writer:    w,
		startTime: time.Now(),
	}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	pw.Current += int64(n)
	pw.printProgress()
	return n, err
}

// ProgressReader tracks the number of bytes read and updates a progress bar
type ProgressReader struct {
	Total      int64
	Current    int64
	Reader     io.Reader
	startTime  time.Time
	lastUpdate time.Time
}

func NewProgressReader(total int64, r io.Reader) *ProgressReader {
	return &ProgressReader{
		Total:     total,
		Reader:    r,
		startTime: time.Now(),
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)
	pr.printProgress()
	return n, err
}

func (pr *ProgressReader) printProgress() {
	// Only update every 100ms or if complete to avoid flashing
	if pr.Current < pr.Total && time.Since(pr.lastUpdate) < 100*time.Millisecond {
		return
	}
	pr.lastUpdate = time.Now()

	percent := float64(pr.Current) / float64(pr.Total) * 100
	width := 40
	completed := int(float64(width) * (float64(pr.Current) / float64(pr.Total)))
	
	bar := strings.Repeat("█", completed) + strings.Repeat("░", width-completed)
	
	// Speed calcs
	duration := time.Since(pr.startTime).Seconds()
	if duration == 0 { duration = 0.0001 } // Prevent division by zero
	speed := float64(pr.Current) / (1024 * 1024) / duration // MB/s
	
	barStr := string(bar)
	fmt.Printf("\r⬇️  Downloading... [%s] %.1f%% (%.2f MB/s)", barStr, percent, speed)
	if pr.Current == pr.Total {
		fmt.Println() // New line on finish
	}
}

func (pw *ProgressWriter) printProgress() {
	// Only update every 100ms or if complete
	if pw.Current < pw.Total && time.Since(pw.lastUpdate) < 100*time.Millisecond {
		return
	}
	pw.lastUpdate = time.Now()

	percent := float64(pw.Current) / float64(pw.Total) * 100
	width := 40
	completed := int(float64(width) * (float64(pw.Current) / float64(pw.Total)))
	
	bar := strings.Repeat("█", completed) + strings.Repeat("░", width-completed)
	
	// Speed calcs
	duration := time.Since(pw.startTime).Seconds()
	if duration == 0 { duration = 0.0001 } // Prevent division by zero
	speed := float64(pw.Current) / (1024 * 1024) / duration // MB/s
	
	fmt.Printf("\r⬆️  Uploading...   [%s] %.1f%% (%.2f MB/s)", bar, percent, speed)
	if pw.Current == pw.Total {
		fmt.Println() 
	}
}
