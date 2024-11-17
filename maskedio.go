package maskedio

import (
	"io"
	"strings"
	"sync"
	"time"
)

var _ io.Writer = (*Writer)(nil)

const defaultRedactMessage = "*****"

type Rule struct {
	keywords      []string
	redactMessage string
	replacer      *strings.Replacer
	mu            sync.RWMutex
}

func NewRule() *Rule {
	return &Rule{
		keywords:      nil,
		redactMessage: defaultRedactMessage,
		replacer:      strings.NewReplacer(),
	}
}

// Writer is a wrapper around io.Writer that masks the output based on the keywords provided.
type Writer struct {
	w                io.Writer
	r                *Rule
	buf              []byte
	mu               sync.Mutex
	disableAutoFlush bool
}

// NewWriter returns a new Writer that masks the output based on the keywords provided.
func NewWriter(w io.Writer) *Writer {
	return NewRule().NewWriter(w)
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
	r := &Rule{
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

	s := w.r.Mask(string(p))

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

// DisableAutoFlush disables auto flush.
func (w *Writer) DisableAutoFlush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.disableAutoFlush = true
}

// SetKeyword sets the keywords to be masked in the output.
func (w *Writer) SetKeyword(keywords ...string) {
	w.r.SetKeyword(keywords...)
}

// UnsetKeyword unsets the keywords to be masked in the output.
func (w *Writer) UnsetKeyword(keywords ...string) {
	w.r.UnsetKeyword(keywords...)
}

// ResetKeywords resets the keywords to be masked in the output.
func (w *Writer) ResetKeywords() {
	w.r.ResetKeywords()
}

// SetRedactMessage sets the message to be used for redaction.
func (w *Writer) SetRedactMessage(redactMessage string) {
	w.r.SetRedactMessage(redactMessage)
}

// Rule returns the masking rule.
func (w *Writer) Rule() *Rule {
	return w.r
}

// SetRule sets the masking rule.
func (w *Writer) SetRule(r *Rule) {
	w.r = r
}

// Unwrap returns the underlying writer.
func (w *Writer) Unwrap() io.Writer {
	return w.w
}

// NewWriter returns a new Writer that masks the output based on the keywords provided.
func (r *Rule) NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: w,
		r: r,
	}
}

// SetKeyword sets the keywords to be masked in the output.
func (r *Rule) SetKeyword(keywords ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.setup()
	r.keywords = append(r.keywords, keywords...)
}

// UnsetKeyword unsets the keywords to be masked in the output.
func (r *Rule) UnsetKeyword(keywords ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.setup()
	for _, keyword := range keywords {
		for i, k := range r.keywords {
			if k == keyword {
				r.keywords = append(r.keywords[:i], r.keywords[i+1:]...)
			}
		}
	}
}

// ResetKeywords resets the keywords to be masked in the output.
func (r *Rule) ResetKeywords() {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.setup()
	r.keywords = nil
}

// SetRedactMessage sets the message to be used for redaction.
func (r *Rule) SetRedactMessage(redactMessage string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.setup()
	r.redactMessage = redactMessage
}

// Mask masks the input based on the keywords provided.
func (r *Rule) Mask(in string) string {
	return r.replacer.Replace(in)
}

func (r *Rule) setup() {
	var reps []string
	for _, keyword := range r.keywords {
		reps = append(reps, keyword, r.redactMessage)
	}
	r.replacer = strings.NewReplacer(reps...)
}
