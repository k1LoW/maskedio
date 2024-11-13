package maskedio

import (
	"io"
	"strings"
	"sync"
	"time"
)

var _ io.Writer = (*Writer)(nil)

const defaultRedactMessage = "*****"

type masker struct {
	keywords      []string
	redactMessage string
	replacer      *strings.Replacer
	mu            sync.RWMutex
}

type Writer struct {
	w   io.Writer
	m   *masker
	buf []byte
	mu  sync.Mutex
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: w,
		m: &masker{
			keywords:      nil,
			redactMessage: defaultRedactMessage,
		},
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

	w.m.mu.RLock()
	defer w.m.mu.RUnlock()
	for _, keyword := range w.m.keywords {
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

	s := w.m.mask(string(p))

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
	w.m.mu.Lock()
	defer w.m.mu.Unlock()
	defer w.m.setup()
	w.m.keywords = append(w.m.keywords, keywords...)
}

func (w *Writer) UnsetKeyword(keywords ...string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	defer w.m.setup()
	for _, keyword := range keywords {
		for i, k := range w.m.keywords {
			if k == keyword {
				w.m.keywords = append(w.m.keywords[:i], w.m.keywords[i+1:]...)
			}
		}
	}
}

func (w *Writer) ResetKeywords() {
	w.mu.Lock()
	defer w.mu.Unlock()
	defer w.m.setup()
	w.m.keywords = nil
}

func (w *Writer) SetRedactMessage(redactMessage string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	defer w.m.setup()
	w.m.redactMessage = redactMessage
}

func (m *masker) mask(in string) string {
	return m.replacer.Replace(in)
}

func (m *masker) setup() {
	var reps []string
	for _, keyword := range m.keywords {
		reps = append(reps, keyword, m.redactMessage)
	}
	m.replacer = strings.NewReplacer(reps...)
}
