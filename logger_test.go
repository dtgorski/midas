// MIT license · Daniel T. Gorski · dtg [at] lengo [dot] org · 09/2019

package midas

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLoggerOutput_1(t *testing.T) {
	var (
		buffer  = &bytes.Buffer{}
		logger  = Logger(buffer)
		wrapped = logger(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(nil)
				if w.Header().Get("foo") != "bar" {
					t.Error("unexpected")
				}
			},
		))
		request = &http.Request{
			URL:        &url.URL{Path: "ä"},
			RemoteAddr: "0.0.0.0:0",
			Method:     "GET",
			Proto:      "HTTP/1.1",
			Header: http.Header{
				"Authorization":   []string{"Basic dXNlcjpwYXNz"},
				"Referer":         []string{"b"},
				"User-Agent":      []string{"d"},
				"X-Forwarded-For": []string{"c"},
				"X-Request-Id":    []string{"~\x7F\x80"},
			},
			TLS: &tls.ConnectionState{
				Version:     0x0303,
				CipherSuite: 0xc030,
			},
		}
		writer = &testResponseWriter{}
	)

	wrapped.ServeHTTP(writer, request)

	p := `^[^|]+ \| 0\.0\.0\.0 \| user \| GET .. HTTP/1\.1 \| 200 \| b \| c \| `
	p += `d \| TLSv1.2 \| ECDHE-RSA-AES256-GCM-SHA384 \| 42 \| 0.000 \| ~..\n$`

	matched, err := regexp.Match(p, buffer.Bytes())
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("unexpected, got %s", buffer.Bytes())
	}
}

func TestLoggerOutput_2(t *testing.T) {
	var (
		buffer  = &bytes.Buffer{}
		logger  = Logger(buffer)
		wrapped = logger(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(42)
				w.(http.Flusher).Flush()
				time.Sleep(time.Second)
			},
		))
		request = &http.Request{
			URL:        &url.URL{Path: "a"},
			RemoteAddr: "0.0.0.0",
			Method:     "GET",
			Proto:      "HTTP/1.1",
			Header:     http.Header{},
		}
		writer = &testResponseWriter{}
	)

	wrapped.ServeHTTP(writer, request)

	p := `^[^|]+ \| 0\.0\.0\.0 \| - \| GET a HTTP/1\.1 \| 42 `
	p += `\| - \| - \| - \| - \| - \| 0 \| 1\.\d{3} \| -\n$`

	matched, err := regexp.Match(p, buffer.Bytes())
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("unexpected, got %s", buffer.Bytes())
	}
}

func TestLoggerDataRace(t *testing.T) {
	var (
		logger  = Logger(ioutil.Discard)
		wrapped = logger(&voidHandler{})
		request = &http.Request{
			URL:        &url.URL{Path: "a"},
			RemoteAddr: "0.0.0.0:0",
			Method:     "GET",
			Proto:      "HTTP/1.1",
			Header: http.Header{
				"Authorization":   []string{"Basic dXNlcjpwYXNz"},
				"Referer":         []string{"b"},
				"User-Agent":      []string{"d"},
				"X-Forwarded-For": []string{"c"},
				"X-Request-Id":    []string{"e"},
			},
		}
		wg = sync.WaitGroup{}
	)
	for i := 0; i < 4e4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wrapped.ServeHTTP(struct{ http.ResponseWriter }{}, request)
		}()
	}
	wg.Wait()
}

func BenchmarkLoggerConcatFullLine(b *testing.B) {
	var (
		logger  = Logger(ioutil.Discard)
		wrapped = logger(&voidHandler{})
		request = &http.Request{
			URL:        &url.URL{Path: strings.Repeat("a", 50)},
			RemoteAddr: "0.0.0.0:0",
			Method:     "GET",
			Proto:      "HTTP/1.1",
			Header: http.Header{
				"Referer":         []string{strings.Repeat("b", 100)},
				"User-Agent":      []string{strings.Repeat("d", 100)},
				"X-Forwarded-For": []string{strings.Repeat("c", 50)},
				"X-Request-Id":    []string{strings.Repeat("e", 50)},
			},
			TLS: &tls.ConnectionState{
				Version:     0x0303,
				CipherSuite: 0xc030,
			},
		}
	)
	for n := 0; n < b.N; n++ {
		wrapped.ServeHTTP(struct{ http.ResponseWriter }{}, request)
	}
}

type testResponseWriter struct{ http.ResponseWriter }

func (testResponseWriter) Header() http.Header {
	return http.Header{"Foo": []string{"bar"}}
}
func (testResponseWriter) Write([]byte) (int, error) {
	return 42, nil
}
func (testResponseWriter) WriteHeader(code int) {}

func (testResponseWriter) Flush() {}

type voidHandler struct{ http.Handler }

func (*voidHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}
