// Package httpstat traces HTTP latency infomation (DNSLookup, TCP Connection and so on) on any golang HTTP request.
// It uses `httptrace` package. Just create `go-httpstat` powered `context.Context` and give it your `http.Request` (no big code modification is required).
package httpstat

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"time"
)

// Result stores httpstat info.
type Result struct {
	// The following are duration for each phase
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	ContentTransfer  time.Duration

	// The followings are timeline of request
	NameLookup    time.Duration
	Connect       time.Duration
	Pretransfer   time.Duration
	StartTransfer time.Duration
	Total         time.Duration

	t0 time.Time
	t5 time.Time // need to be provided from outside

	dnsStart      time.Time
	dnsDone       time.Time
	tcpStart      time.Time
	tcpDone       time.Time
	tlsStart      time.Time
	tlsDone       time.Time
	serverStart   time.Time
	serverDone    time.Time
	transferStart time.Time
	transferDone  time.Time // need to be provided from outside

	// isTLS is true when connection seems to use TLS
	isTLS bool

	// isReused is true when connection is reused (keep-alive)
	isReused bool

	// ConnectedTo is the address of the server that was connected to.
	ConnectedTo net.Addr
}

func (r *Result) Durations() map[string]time.Duration {
	return map[string]time.Duration{
		"DNSLookup":        r.DNSLookup,
		"TCPConnection":    r.TCPConnection,
		"TLSHandshake":     r.TLSHandshake,
		"ServerProcessing": r.ServerProcessing,
		"ContentTransfer":  r.ContentTransfer,

		"NameLookup":    r.NameLookup,
		"Connect":       r.Connect,
		"Pretransfer":   r.Connect,
		"StartTransfer": r.StartTransfer,
		"Total":         r.Total,
	}
}

// Format formats stats result.
func (r Result) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "DNS lookup:        %4d ms\n",
				int(r.DNSLookup/time.Millisecond))
			fmt.Fprintf(s, "TCP connection:    %4d ms\n",
				int(r.TCPConnection/time.Millisecond))
			fmt.Fprintf(s, "TLS handshake:     %4d ms\n",
				int(r.TLSHandshake/time.Millisecond))
			fmt.Fprintf(s, "Server processing: %4d ms\n",
				int(r.ServerProcessing/time.Millisecond))

			if !r.t5.IsZero() {
				fmt.Fprintf(s, "Content transfer:  %4d ms\n\n",
					int(r.ContentTransfer/time.Millisecond))
			} else {
				fmt.Fprintf(s, "Content transfer:  %4s ms\n\n", "-")
			}

			fmt.Fprintf(s, "Name Lookup:    %4d ms\n",
				int(r.NameLookup/time.Millisecond))
			fmt.Fprintf(s, "Connect:        %4d ms\n",
				int(r.Connect/time.Millisecond))
			fmt.Fprintf(s, "Pre Transfer:   %4d ms\n",
				int(r.Pretransfer/time.Millisecond))
			fmt.Fprintf(s, "Start Transfer: %4d ms\n",
				int(r.StartTransfer/time.Millisecond))

			if !r.t5.IsZero() {
				fmt.Fprintf(s, "Total:          %4d ms\n",
					int(r.Total/time.Millisecond))
			} else {
				fmt.Fprintf(s, "Total:          %4s ms\n", "-")
			}
			return
		}

		fallthrough
	case 's', 'q':
		var b bytes.Buffer
		first := true
		for k, v := range r.Durations() {
			if first {
				first = false
			} else {
				b.Write([]byte(", "))
			}
			b.WriteString(k)
			b.Write([]byte(": "))
			// Handle when End function is not called
			if (k == "ContentTransfer" || k == "Total") && r.t5.IsZero() {
				b.WriteString("- ms")
				continue
			}
			b.WriteString(strconv.FormatInt(int64(v/time.Millisecond), 10))
			b.Write([]byte(" ms"))
		}
		b.WriteTo(s)
	}
}

// WithHTTPStat is a wrapper of httptrace.WithClientTrace. It records the
// time of each httptrace hooks.
func WithHTTPStat(ctx context.Context, r *Result) context.Context {
	return withClientTrace(ctx, r)
}
