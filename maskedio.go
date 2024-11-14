package maskedio

import (
	"io"
	"strings"
	"sync"
	"time"
)

var _ io.Writer = (*Writer)(nil)

const defaultRedactMessage = "*****"

type rule struct {
	keywords      []string
	redactMessage string
	replacer      *strings.Replacer
	mu            sync.RWMutex
}

// Writer is a wrapper around io.Writer that masks the output based on the keywords provided.
type Writer struct {
	w                io.Writer
	r                *rule
	buf              []byte
	mu               sync.Mutex
	disableAutoFlush bool
}

// NewWriter returns a new Writer that masks the output based on the keywords provided.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: w,
		r: &rule{
			keywords:      nil,
			redactMessage: defaultRedactMessage,
		},
	}
}

// NewSyncedWriter returns a new Writer with syncronized masking rule.
func (w *Writer) NewSyncedWriter(nw io.Writer) *Writer {
	return &Writer{
		w: nw,
		r: w.r,
	}
}

// NewSameWriter returns a new Writer with same masking rule.
func (w *Writer) NewSameWriter(ww io.Writer) *Writer {
	keywords := make([]string, len(w.r.keywords))
	_ = copy(keywords, w.r.keywords)
	r := &rule{
		keywords:      keywords,
		redactMessage: w.r.redactMessage,
	}
	r.setup()
	return &Writer{
		w: ww,
		r: r,
	}
}

// Write writes the data to the underlying writer, masking the output based on the keywords provided.
func (w *Writer) Write(p []byte) (n int, err error) {
	l := len(p)
	if len(w.buf) > 0 {
		w.mu.Lock()
		p = append(w.buf, p...)
		w.buf = nil
		w.mu.Unlock()
	}

	w.r.mu.RLock()
	defer w.r.mu.RUnlock()
	for _, keyword := range w.r.keywords {
		var kw string
		for _, r := range keyword {
			kw += string(r)
			if strings.HasSuffix(string(p), kw) && len(keyword) > len(kw) {
				w.mu.Lock()
				w.buf = append(w.buf, p...)
				w.mu.Unlock()
				if !w.disableAutoFlush {
					// Auto flush
					go func() {
						time.Sleep(100 * time.Microsecond)
						_ = w.Flush()
					}()
				}
				return l, nil
			}
		}
	}

	s := w.r.mask(string(p))

	if _, err := w.w.Write([]byte(s)); err != nil {
		return 0, err
	}
	return l, nil
}

// Flush writes any buffered data to the underlying writer.
func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) > 0 {
		_, err := w.w.Write(w.buf)
		w.buf = nil
		return err
	}
	return nil
}

// SetKeyword sets the keywords to be masked in the output.
func (w *Writer) SetKeyword(keywords ...string) {
	w.r.mu.Lock()
	defer w.r.mu.Unlock()
	defer w.r.setup()
	w.r.keywords = append(w.r.keywords, keywords...)
}

// UnsetKeyword unsets the keywords to be masked in the output.
func (w *Writer) UnsetKeyword(keywords ...string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	defer w.r.setup()
	for _, keyword := range keywords {
		for i, k := range w.r.keywords {
			if k == keyword {
				w.r.keywords = append(w.r.keywords[:i], w.r.keywords[i+1:]...)
			}
		}
	}
}

// ResetKeywords resets the keywords to be masked in the output.
func (w *Writer) ResetKeywords() {
	w.mu.Lock()
	defer w.mu.Unlock()
	defer w.r.setup()
	w.r.keywords = nil
}

// SetRedactMessage sets the message to be used for redaction.
func (w *Writer) SetRedactMessage(redactMessage string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	defer w.r.setup()
	w.r.redactMessage = redactMessage
}

// DisableAutoFlush disables auto flush.
func (w *Writer) DisableAutoFlush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.disableAutoFlush = true
}

func (r *rule) mask(in string) string {
	return r.replacer.Replace(in)
}

func (r *rule) setup() {
	var reps []string
	for _, keyword := range r.keywords {
		reps = append(reps, keyword, r.redactMessage)
	}
	r.replacer = strings.NewReplacer(reps...)
}
