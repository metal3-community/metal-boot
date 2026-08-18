package wsgi

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestBufferingReaderRead(t *testing.T) {
	body := strings.Repeat("x", 100)
	r := newBufferingReader(strings.NewReader(body), 10)

	out := make([]byte, 100)
	n, _ := io.ReadFull(r, out)
	if n != 100 || string(out[:n]) != body {
		t.Fatalf("got %d bytes, want 100", n)
	}
}

func TestBufferingReaderRewind(t *testing.T) {
	body := "hello"
	r := newBufferingReader(strings.NewReader(body), 10)

	out := make([]byte, 5)
	if _, err := io.ReadFull(r, out); err != nil {
		t.Fatal(err)
	}
	if err := r.Rewind(); err != nil {
		t.Fatalf("rewind of fully-buffered body should succeed: %v", err)
	}
	out2 := make([]byte, 5)
	if _, err := io.ReadFull(r, out2); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, out2) {
		t.Fatalf("reread mismatch: %q vs %q", out, out2)
	}
}

func TestBufferingReaderRewindPartial(t *testing.T) {
	body := strings.Repeat("y", 100)
	r := newBufferingReader(strings.NewReader(body), 10)

	out := make([]byte, 50)
	if _, err := io.ReadFull(r, out); err != nil {
		t.Fatal(err)
	}
	if err := r.Rewind(); err == nil {
		t.Fatal("rewind after reading past buffer should fail")
	}
}

func TestBufferingReaderReadline(t *testing.T) {
	r := newBufferingReader(strings.NewReader("line one\nline two\n"), 0)

	buf := make([]byte, 64)
	n, err := r.Readline(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "line one\n" {
		t.Fatalf("got %q", buf[:n])
	}

	n, err = r.Readline(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "line two\n" {
		t.Fatalf("got %q", buf[:n])
	}
}

func TestBootstrapCodeDefault(t *testing.T) {
	cfg := Config{ConfigFile: "/etc/ironic/ironic.conf"}.withDefaults()
	code := cfg.bootstrapCode()

	for _, want := range []string{
		`sys.argv = ["ironic-api", "--config-file", "/etc/ironic/ironic.conf"]`,
		`importlib.import_module("ironic.api.wsgi")`,
		"initialize_wsgi_app",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("bootstrap missing %q:\n%s", want, code)
		}
	}
}

func TestBootstrapCodeOverride(t *testing.T) {
	cfg := Config{Bootstrap: "application = object()"}.withDefaults()
	if got := cfg.bootstrapCode(); got != "application = object()" {
		t.Fatalf("bootstrap override not honored: %q", got)
	}
}

func TestPyLiteral(t *testing.T) {
	if got := pyLiteral(`a"b\c`); got != `"a\"b\\c"` {
		t.Fatalf("got %s", got)
	}
}
