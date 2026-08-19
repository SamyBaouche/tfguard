package ui

import (
	"fmt"
	"io"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner is a line-rewriting animated indicator for long-running work.
// It is a no-op when style is disabled (CI / pipes / tests).
type Spinner struct {
	w       io.Writer
	style   Style
	mu      sync.Mutex
	stopCh  chan struct{}
	doneCh  chan struct{}
	message string
	active  bool
}

// NewSpinner creates a spinner writing to w.
func NewSpinner(w io.Writer, style Style) *Spinner {
	return &Spinner{w: w, style: style}
}

// Start begins animation with the given message. Safe to call repeatedly.
func (s *Spinner) Start(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.message = message
	if !s.style.Enabled() {
		fmt.Fprintf(s.w, "  · %s...\n", message)
		return
	}
	if s.active {
		return
	}
	s.active = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	go s.loop()
}

// Done stops the spinner and prints a green check line.
func (s *Spinner) Done(detail string) {
	s.finish(s.style.Green("✓"), detail)
}

// Succeed is an alias of Done.
func (s *Spinner) Succeed(detail string) {
	s.Done(detail)
}

// Fail stops the spinner and prints a red cross line.
func (s *Spinner) Fail(detail string) {
	s.finish(s.style.Red("✗"), detail)
}

func (s *Spinner) finish(mark, detail string) {
	s.mu.Lock()
	msg := s.message
	wasActive := s.active && s.style.Enabled()
	if s.active && s.style.Enabled() {
		close(s.stopCh)
		s.active = false
		done := s.doneCh
		s.mu.Unlock()
		<-done
	} else {
		s.active = false
		s.mu.Unlock()
	}

	line := fmt.Sprintf("  %s %s", mark, msg)
	if detail != "" {
		line += s.style.Dim("  ·  " + detail)
	}
	if wasActive {
		fmt.Fprintf(s.w, "\r\033[2K%s\n", line)
	} else if s.style.Enabled() {
		fmt.Fprintln(s.w, line)
	} else {
		// Non-TTY already printed "· msg..." on Start; append result.
		fmt.Fprintf(s.w, "  %s %s", mark, msg)
		if detail != "" {
			fmt.Fprintf(s.w, "  ·  %s", detail)
		}
		fmt.Fprintln(s.w)
	}
}

func (s *Spinner) loop() {
	defer close(s.doneCh)
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	i := 0
	for {
		frame := spinnerFrames[i%len(spinnerFrames)]
		s.mu.Lock()
		msg := s.message
		s.mu.Unlock()
		fmt.Fprintf(s.w, "\r\033[2K  %s %s", s.style.Cyan(frame), msg)
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			i++
		}
	}
}
