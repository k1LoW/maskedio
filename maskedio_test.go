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
		{
			[]string{"", "passw0rd"},
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

func TestNewSyncedWriter(t *testing.T) {
	w := NewWriter(new(bytes.Buffer))
	buf := new(bytes.Buffer)
	nw := w.NewSyncedWriter(buf)

	{
		w.SetKeyword("passw0rd")
		if _, err := nw.Write([]byte("password: passw0rd")); err != nil {
			t.Fatal(err)
		}
		nw.Flush()

		got := buf.String()
		want := "password: *****"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		buf.Reset()
	}

	{
		w.SetKeyword("secret")
		if _, err := nw.Write([]byte("password: secret")); err != nil {
			t.Fatal(err)
		}
		nw.Flush()
		got := buf.String()
		want := "password: *****"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		buf.Reset()
	}
}

func TestNewSameWriter(t *testing.T) {
	w := NewWriter(new(bytes.Buffer))
	w.SetKeyword("passw0rd")
	buf := new(bytes.Buffer)
	nw := w.NewSameWriter(buf)

	{
		if _, err := nw.Write([]byte("password: passw0rd")); err != nil {
			t.Fatal(err)
		}
		nw.Flush()

		got := buf.String()
		want := "password: *****"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		buf.Reset()
	}

	{
		w.SetKeyword("secret")
		if _, err := nw.Write([]byte("password: secret")); err != nil {
			t.Fatal(err)
		}
		nw.Flush()
		got := buf.String()
		want := "password: secret"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		buf.Reset()
	}
}
