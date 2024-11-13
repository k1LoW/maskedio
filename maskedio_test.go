package maskedio

import (
	"bytes"
	"fmt"
	"testing"
)

func TestWriter(t *testing.T) {
	tests := []struct {
		keywords []string
		in       []string
		wantN    int
		want     string
	}{
		{
			[]string{"passw0rd"},
			[]string{"password: passw0rd"},
			18,
			"password: *****",
		},
		{
			[]string{"passw0rd"},
			[]string{"password: pass", "w0rd"},
			18,
			"password: *****",
		},
		{
			[]string{"passw0rd", "secret"},
			[]string{"password: passw0rd or secret"},
			28,
			"password: ***** or *****",
		},
		{
			[]string{"secret", "passw0rd"},
			[]string{"password: passw0rd or secret"},
			28,
			"password: ***** or *****",
		},
		{
			[]string{"secret", "passw0rd"},
			[]string{"password: pass", "w0rd or secret"},
			28,
			"password: ***** or *****",
		},
		{
			[]string{"passw1rd", "passw0rd"},
			[]string{"password: pass", "w0rd"},
			18,
			"password: *****",
		},
		{
			[]string{"passw0rd"},
			[]string{"password: pass"},
			14,
			"password: pass",
		},
	}
	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			w := NewWriter(buf)
			w.SetKeyword(tt.keywords...)
			var gotN int
			for _, in := range tt.in {
				n, err := w.Write([]byte(in))
				if err != nil {
					t.Fatal(err)
				}
				gotN += n
			}
			if gotN != tt.wantN {
				t.Errorf("got %d, want %d", gotN, tt.wantN)
			}
			if err := w.Flush(); err != nil {
				t.Fatal(err)
			}

			got := buf.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
