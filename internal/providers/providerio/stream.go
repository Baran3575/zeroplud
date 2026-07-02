package providerio

import (
	"bufio"
	"io"
	"strings"
)

// SSEEvent represents a single Server-Sent Event.
type SSEEvent struct {
	Event string
	Data  string
	ID    string
}

// SSEDecoder reads SSE events from an io.Reader.
type SSEDecoder struct {
	scanner *bufio.Scanner
}

func NewSSEDecoder(r io.Reader) *SSEDecoder {
	return &SSEDecoder{scanner: bufio.NewScanner(r)}
}

// Next reads the next complete SSE event (delimited by blank line).
// Returns io.EOF when the stream ends.
func (d *SSEDecoder) Next() (SSEEvent, error) {
	var event SSEEvent
	for d.scanner.Scan() {
		line := d.scanner.Text()
		if line == "" {
			return event, nil
		}
		if strings.HasPrefix(line, "event: ") {
			event.Event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			event.Data = strings.TrimPrefix(line, "data: ")
		} else if strings.HasPrefix(line, "id: ") {
			event.ID = strings.TrimPrefix(line, "id: ")
		}
	}
	return event, d.scanner.Err()
}
