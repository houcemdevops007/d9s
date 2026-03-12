// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package compose

import (
	"bufio"
	"io"
)

// newScanner creates a line scanner with a large buffer for log lines.
func newScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1024*1024), 1024*1024)
	return s
}
