package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Spinner displays an animated loading indicator
type Spinner struct {
	message   string
	frames    []string
	interval  time.Duration
	running   bool
	stopCh    chan struct{}
	doneCh    chan struct{}
	mu        sync.Mutex
	startTime time.Time
}

var defaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:  message,
		frames:   defaultFrames,
		interval: 80 * time.Millisecond,
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.startTime = time.Now()
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	go s.spin()
}

func (s *Spinner) spin() {
	defer close(s.doneCh)

	i := 0
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			// Clear the line
			fmt.Print("\r" + strings.Repeat(" ", 60) + "\r")
			return
		case <-ticker.C:
			s.mu.Lock()
			elapsed := time.Since(s.startTime)
			message := s.message
			s.mu.Unlock()

			frame := s.frames[i%len(s.frames)]

			var line string
			if elapsed > 2*time.Second {
				line = fmt.Sprintf("\r%s %s (%ds)",
					Color(Cyan, frame),
					message,
					int(elapsed.Seconds()))
			} else {
				line = fmt.Sprintf("\r%s %s", Color(Cyan, frame), message)
			}
			fmt.Print(line + "   ")
			i++
		}
	}
}

// Stop halts the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	<-s.doneCh
}

// SetMessage updates the spinner message while running
func (s *Spinner) SetMessage(msg string) {
	s.mu.Lock()
	s.message = msg
	s.mu.Unlock()
}

// IsRunning returns whether the spinner is currently active
func (s *Spinner) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
