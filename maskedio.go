package maskedio

import (
	"io"
	"strings"
	"sync"
	"time"
)

var _ io.Writer = (*Writer)(nil)

type Writer struct {
	w        io.Writer
	keywords []string
	maskword string
	buf      []byte
	replacer *strings.Replacer
	mu       sync.Mutex
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:        w,
		maskword: "*****",
	}
}

func (w *Writer) Write(p []byte) (n int, err error) {
	l := len(p)
	if len(w.buf) > 0 {
		w.mu.Lock()
		p = append(w.buf, p...)
		w.buf = nil
		w.mu.Unlock()
	}

	for _, keyword := range w.keywords {
		var kw string
		for _, r := range keyword {
			kw += string(r)
			if strings.HasSuffix(string(p), kw) && len(keyword) > len(kw) {
				w.mu.Lock()
				w.buf = append(w.buf, p...)
				w.mu.Unlock()
				go func() {
					// Auto flush
					time.Sleep(100 * time.Microsecond)
					_ = w.Flush()
				}()
				return l, nil
			}
		}
	}

	s := w.replacer.Replace(string(p))

	if _, err := w.w.Write([]byte(s)); err != nil {
		return 0, err
	}
	return l, nil
}

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

func (w *Writer) SetKeyword(keywords ...string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.keywords = append(w.keywords, keywords...)
	w.setupReplacer()
}

func (w *Writer) UnsetKeyword(keywords ...string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, keyword := range keywords {
		for i, k := range w.keywords {
			if k == keyword {
				w.keywords = append(w.keywords[:i], w.keywords[i+1:]...)
			}
		}
	}
	w.setupReplacer()
}

func (w *Writer) ResetKeywords() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.keywords = nil
	w.replacer = nil
}

func (w *Writer) SetMaskWord(maskword string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.maskword = maskword
	w.setupReplacer()
}

func (w *Writer) setupReplacer() {
	var reps []string
	for _, keyword := range w.keywords {
		reps = append(reps, keyword, w.maskword)
	}
	w.replacer = strings.NewReplacer(reps...)
}
