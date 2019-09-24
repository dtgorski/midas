// MIT license · Daniel T. Gorski · dtg [at] lengo [dot] org · 09/2019

package midas

import (
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type responseWriter struct {
	wrap http.ResponseWriter
	code int
	sent int
}

var (
	sink io.Writer

	pipe, void = byte('|'), byte('-')
	deli, zero = []byte{' ', pipe, ' '}, []byte("0.000")

	bufPool = sync.Pool{New: func() interface{} { return new([0x200]byte) }}
	escPool = sync.Pool{New: func() interface{} { return new([0x80]byte) }}

	tlsVersion = map[uint16]string{
		0x0300: "SSLv3.0",
		0x0301: "TLSv1.0",
		0x0302: "TLSv1.1",
		0x0303: "TLSv1.2",
		0x0304: "TLSv1.3",
		0x0305: "TLSv1.4",
	}

	cipherSuite = map[uint16]string{
		0x0005: "RSA-RC4-128-SHA",
		0x000a: "RSA-3DES-EDE-CBC-SHA",
		0x002f: "RSA-AES128-CBC-SHA",
		0x0035: "RSA-AES256-CBC-SHA",
		0x003c: "RSA-AES128-CBC-SHA256",
		0x009c: "RSA-AES128-GCM-SHA256",
		0x009d: "RSA-AES256-GCM-SHA384",

		0xc007: "ECDHE-ECDSA-RC4-128-SHA",
		0xc009: "ECDHE-ECDSA-AES128-CBC-SHA",
		0xc00a: "ECDHE-ECDSA-AES256-CBC-SHA",
		0xc011: "ECDHE-RSA-RC4-128-SHA",
		0xc012: "ECDHE-RSA-3DES-EDE-CBC-SHA",
		0xc013: "ECDHE-RSA-AES128-CBC-SHA",
		0xc014: "ECDHE-RSA-AES256-CBC-SHA",
		0xc023: "ECDHE-ECDSA-AES128-CBC-SHA256",
		0xc027: "ECDHE-RSA-AES128-CBC-SHA256",
		0xc02f: "ECDHE-RSA-AES128-GCM-SHA256",
		0xc02b: "ECDHE-ECDSA-AES128-GCM-SHA256",
		0xc030: "ECDHE-RSA-AES256-GCM-SHA384",
		0xc02c: "ECDHE-ECDSA-AES256-GCM-SHA384",
		0xcca8: "ECDHE-RSA-CHACHA20-POLY1305",
		0xcca9: "ECDHE-ECDSA-CHACHA20-POLY1305",

		0x1301: "AES128-GCM-SHA256",
		0x1302: "AES256-GCM-SHA384",
		0x1303: "CHACHA20-POLY1305-SHA256",
	}
)

// Logger is a HTTP middleware decorator factory. This function returns
// an other function to be used to create a new response wrapper on demand.
func Logger(writer io.Writer) func(next http.Handler) http.Handler {
	sink = writer
	return newHandlerFunc
}

func newHandlerFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			res := &responseWriter{wrap: w}

			next.ServeHTTP(res, r)

			buf := bufPool.Get().(*[0x200]byte)
			b := buf[:0]

			b = appendAccessTime(b, now)  // $time_iso8601
			b = appendRemoteAddr(b, r)    // $remote_addr
			b = appendRemoteUser(b, r)    // $remote_user
			b = appendRequest(b, r)       // $request
			b = appendStatus(b, res)      // $status
			b = appendReferer(b, r)       // $http_referer
			b = appendForwardedFor(b, r)  // $http_x_forwarded_for
			b = appendUserAgent(b, r)     // $http_user_agent
			b = appendSSLProtocol(b, r)   // $ssl_protocol
			b = appendSSLCipher(b, r)     // $ssl_cipher
			b = appendBytesSent(b, res)   // $bytes_sent
			b = appendRequestTime(b, now) // $request_time
			b = appendRequestID(b, r)     // $http_x_request_id

			n := len(b) - len(deli)
			b[n] = 0x0a
			_, _ = sink.Write(b[:n+1])

			bufPool.Put(buf)
		},
	)
}

func appendAccessTime(b []byte, t time.Time) []byte {
	return append(append(b, t.AppendFormat(b, time.RFC3339)...), deli...)
}

func appendRemoteAddr(b []byte, r *http.Request) []byte {
	for i := len(r.RemoteAddr) - 1; i >= 0; i-- {
		if r.RemoteAddr[i] == ':' {
			return append(append(b, r.RemoteAddr[:i]...), deli...)
		}
	}
	return append(append(b, r.RemoteAddr...), deli...)
}

func appendRemoteUser(b []byte, r *http.Request) []byte {
	if s, _, ok := r.BasicAuth(); ok {
		return append(concat(b, s), deli...)
	}
	return append(append(b, void), deli...)
}

func appendRequest(b []byte, r *http.Request) []byte {
	b = append(append(b, r.Method...), ' ')
	b = append(concat(b, r.URL.Path), ' ')
	return append(append(b, r.Proto...), deli...)
}

func appendStatus(b []byte, log *responseWriter) []byte {
	return append(strconv.AppendInt(b, int64(log.code), 10), deli...)
}

func appendReferer(b []byte, r *http.Request) []byte {
	if s := r.Header["Referer"]; s != nil {
		return append(concat(b, s[0]), deli...)
	}
	return append(append(b, void), deli...)
}

func appendForwardedFor(b []byte, r *http.Request) []byte {
	if s := r.Header["X-Forwarded-For"]; s != nil {
		return append(concat(b, s[0]), deli...)
	}
	return append(append(b, void), deli...)
}

func appendUserAgent(b []byte, r *http.Request) []byte {
	if s := r.Header["User-Agent"]; s != nil {
		return append(concat(b, s[0]), deli...)
	}
	return append(append(b, void), deli...)
}

func appendSSLProtocol(b []byte, r *http.Request) []byte {
	if r.TLS != nil {
		if s, ok := tlsVersion[r.TLS.Version]; ok {
			return append(append(b, s...), deli...)
		}
	}
	return append(append(b, void), deli...)
}

func appendSSLCipher(b []byte, r *http.Request) []byte {
	if r.TLS != nil {
		if s, ok := cipherSuite[r.TLS.CipherSuite]; ok {
			return append(append(b, s...), deli...)
		}
	}
	return append(append(b, void), deli...)
}

func appendBytesSent(b []byte, log *responseWriter) []byte {
	return append(strconv.AppendInt(b, int64(log.sent), 10), deli...)
}

func appendRequestTime(b []byte, t time.Time) []byte {
	if d := time.Since(t).Seconds(); d >= .001 {
		return append(strconv.AppendFloat(b, d, 'f', 3, 64), deli...)
	}
	return append(append(b, zero...), deli...)
}

func appendRequestID(b []byte, r *http.Request) []byte {
	if s := r.Header["X-Request-Id"]; s != nil {
		return append(concat(b, s[0]), deli...)
	}
	return append(append(b, void), deli...)
}

func (w *responseWriter) Header() http.Header {
	return w.wrap.Header()
}

func (w *responseWriter) WriteHeader(code int) {
	w.code = code
	w.wrap.WriteHeader(code)
}

func (w *responseWriter) Write(buf []byte) (int, error) {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	num, err := w.wrap.Write(buf)
	w.sent += num
	return num, err
}

func (w *responseWriter) Flush() {
	if _, ok := w.wrap.(http.Flusher); ok {
		w.wrap.(http.Flusher).Flush()
	}
}

// Appends a string to byte buffer. Control characters below 0x20 and values
// above 0x7E are replaced by a period sign. The underlying buffer offers 128
// bytes capacity. This means, strings longer than 128 bytes will be cut off.
func concat(b []byte, s string) []byte {
	esc := escPool.Get().(*[0x80]byte)
	buf := esc[:]
	for i, num := 0, copy(buf, s); i < num; i++ {
		if buf[i] < 0x20 || buf[i] > 0x7E || buf[i] == pipe {
			b = append(b, '.')
			continue
		}
		b = append(b, buf[i])
	}
	escPool.Put(esc)
	return b
}
