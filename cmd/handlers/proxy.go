package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"log/slog"
)

const percentile = 80

type Host struct {
	ID            string
	State         State
	OpenExpiresAt time.Time
	FailuresCount int
}

type State string

const (
	StateClosed   State = "closed"
	StateOpen           = "open"
	StateHalfOpen       = "half-open"
)

type handlerCtx struct {
	httpClient      *http.Client
	lru             *lru.Cache[string, Host]
	longPooling     bool
	maxFailedTries  int
	openStateExpiry time.Duration
}

func New(longPooling bool, maxFailedTries int, openStateExpiry time.Duration) *handlerCtx {
	lru, _ := lru.New[string, Host](100)
	return &handlerCtx{httpClient: http.DefaultClient, lru: lru, longPooling: longPooling, maxFailedTries: maxFailedTries, openStateExpiry: openStateExpiry}
}

//// Why we need this? Maybe add request-id to logs
//	rID := request.Header.Get("X-Request-ID")
//	if rID == "" {
//		rID = "unable-to-correlate"
//	}

func (hc *handlerCtx) PassThrough(writer http.ResponseWriter, req *http.Request) {
	host, ok := hc.lru.Get(req.URL.Host)
	if !ok {
		host.ID = req.URL.Host
		host.State = StateClosed
	}
	if host.State == StateOpen && time.Now().After(host.OpenExpiresAt) {
		host.State = StateHalfOpen
	}

	switch host.State {
	case StateOpen:
		hc.fail(writer, req)
		slog.Info("gofuse: attempt failed", "host", req.URL.Host, "state", host.State)
	case StateClosed, StateHalfOpen:
		resp := hc.proxy(req)
		hc.updateHostStatus(host, resp)
		if resp.StatusCode < 500 {
			hc.rewrite(writer, resp)
		}
	}
}

// TODO: Mention long-pooling feature in README.md
func (hc *handlerCtx) fail(writer http.ResponseWriter, req *http.Request) {
	timeout := fetchTimeout(req.Header)
	if hc.longPooling && timeout > 0 {
		// Long pooling
		slog.Info("gofuse: hold", "host", req.URL.Host, "timeout", timeout)
		<-time.After((timeout * percentile) / 100)
	}
	writer.WriteHeader(http.StatusServiceUnavailable)
	_, _ = writer.Write([]byte(fmt.Sprintf("{\"error\":gofuse open for host: %s}", req.URL.Host)))
}

func (hc *handlerCtx) proxy(request *http.Request) *http.Response {
	// Flush requestURI, requestURI must stay unmodified, (RFC 2616, Section 5.1)
	request.RequestURI = ""
	// Assign origin protocol
	request.URL.Scheme = request.Header.Get("X-Forwarded-Proto")
	if request.URL.Scheme == "" {
		request.URL.Scheme = "https"
	}

	resp, err := hc.httpClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	slog.Info("gofuse: received response", "host", request.URL.Host, "status", resp.Status)

	return resp
}

func (hc *handlerCtx) updateHostStatus(host Host, response *http.Response) {
	isRequestSuccessful := response.StatusCode < 500
	switch host.State {
	case StateClosed:
		if isRequestSuccessful && host.FailuresCount > 0 {
			host.FailuresCount--
		}
		if !isRequestSuccessful {
			host.FailuresCount++
			if host.FailuresCount >= hc.maxFailedTries {
				slog.Info("gofuse: becoming open", "host", host.ID, "failureCount", host.FailuresCount)
				host.State = StateOpen
				host.OpenExpiresAt = time.Now().Add(hc.openStateExpiry)
			}
		}
	case StateHalfOpen:
		if isRequestSuccessful {
			host.FailuresCount--
			if host.FailuresCount == 0 {
				slog.Info("gofuse: becoming closed", "host", host.ID)
				host.State = StateClosed
			}
		} else {
			host.State = StateOpen
			host.OpenExpiresAt = time.Now().Add(hc.openStateExpiry)
		}
	}
	hc.lru.Add(host.ID, host)
}

func (hc *handlerCtx) rewrite(writer http.ResponseWriter, response *http.Response) {
	for k, v := range response.Header {
		writer.Header()[k] = v
	}
	writer.Header()["Via"] = []string{"gofuse"}

	bytes, err := io.ReadAll(response.Body)
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)

	if err != nil {
		log.Fatal(err)
	}
	_, err = writer.Write(bytes)
	if err != nil {
		log.Fatal(err)
	}
}

func fetchTimeout(header http.Header) time.Duration {
	val, ok := header["Request-Timeout"]
	if !ok || len(val) == 0 {
		return 0
	}
	tm := val[0]
	tmNum, err := strconv.Atoi(tm)
	if err != nil {
		return 0
	}
	return time.Duration(tmNum) * time.Millisecond
}
