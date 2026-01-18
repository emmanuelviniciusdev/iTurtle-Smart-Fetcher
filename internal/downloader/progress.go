package downloader

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// ProgressPrinter handles turtle-themed progress output
type ProgressPrinter struct {
	writer     io.Writer
	turtlePos  int
	lastUpdate time.Time
	turtles    []string
}

// NewProgressPrinter creates a new turtle progress printer
func NewProgressPrinter(w io.Writer) *ProgressPrinter {
	return &ProgressPrinter{
		writer:     w,
		turtlePos:  0,
		lastUpdate: time.Now(),
		turtles: []string{
			"🐢",
			"🐢",
			"🐢",
			"🐢",
		},
	}
}

// PrintStart prints the start of a download operation
func (p *ProgressPrinter) PrintStart(operation string) {
	fmt.Fprintf(p.writer, "\n🐢 %s...\n", operation)
}

// PrintProgress prints an animated progress indicator
func (p *ProgressPrinter) PrintProgress(message string) {
	// Only update animation every 200ms to avoid flickering
	if time.Since(p.lastUpdate) < 200*time.Millisecond {
		return
	}
	p.lastUpdate = time.Now()

	// Cycle through turtle positions
	p.turtlePos = (p.turtlePos + 1) % 4

	// Create animation frames
	var animation string
	switch p.turtlePos {
	case 0:
		animation = "🐢    "
	case 1:
		animation = " 🐢   "
	case 2:
		animation = "  🐢  "
	case 3:
		animation = "   🐢 "
	}

	// Print with carriage return to overwrite
	fmt.Fprintf(p.writer, "\r%s %s", animation, message)
}

// PrintComplete prints a completion message
func (p *ProgressPrinter) PrintComplete(message string, count int) {
	if count == 1 {
		fmt.Fprintf(p.writer, "\n✅ %s (1 file)\n", message)
	} else {
		fmt.Fprintf(p.writer, "\n✅ %s (%d files)\n", message, count)
	}
}

// PrintFile prints a completed file with turtle
func (p *ProgressPrinter) PrintFile(filename string) {
	// Truncate long filenames
	display := filename
	if len(display) > 60 {
		display = display[:57] + "..."
	}
	fmt.Fprintf(p.writer, "   🐢 %s\n", display)
}

// PrintError prints an error message
func (p *ProgressPrinter) PrintError(message string) {
	fmt.Fprintf(p.writer, "\n❌ %s\n", message)
}

// PrintWarning prints a warning message
func (p *ProgressPrinter) PrintWarning(message string) {
	fmt.Fprintf(p.writer, "\n⚠️  %s\n", message)
}

// PrintSection prints a section header
func (p *ProgressPrinter) PrintSection(title string) {
	border := strings.Repeat("─", len(title)+4)
	fmt.Fprintf(p.writer, "\n┌%s┐\n│  %s  │\n└%s┘\n", border, title, border)
}

// ClearLine clears the current line
func (p *ProgressPrinter) ClearLine() {
	fmt.Fprintf(p.writer, "\r%s\r", strings.Repeat(" ", 80))
}
