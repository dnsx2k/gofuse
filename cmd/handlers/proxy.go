package handlers

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/dnsx2k/gofuse/pkg/circuitbreaker"
	"github.com/dnsx2k/gofuse/pkg/settings"
	lru "github.com/hashicorp/golang-lru/v2"
)

const percentile = 80

type handlerCtx struct {
	httpClient *http.Client
	lru        *lru.Cache[string, circuitbreaker.Host]
	settings   map[string]settings.ClientConfiguration
}

func New(settings map[string]settings.ClientConfiguration) *handlerCtx {
	lru, _ := lru.New[string, circuitbreaker.Host](100)
	return &handlerCtx{httpClient: http.DefaultClient, lru: lru, settings: settings}
}

func (hc *handlerCtx) PassThrough(writer http.ResponseWriter, req *http.Request) {
	host, ok := hc.lru.Get(req.URL.Host)
	if !ok {
		host = hc.getHostWithSettings(req.URL.Host)
	}
	if host.State == circuitbreaker.StateOpen && time.Now().After(host.OpenExpiresAt) {
		host.State = circuitbreaker.StateHalfOpen
	}

	switch host.State {
	case circuitbreaker.StateOpen:
		hc.fail(writer, host)
		slog.Info("gofuse: attempt failed", "host", req.URL.Host, "state", host.State)
	case circuitbreaker.StateClosed, circuitbreaker.StateHalfOpen:
		resp := hc.proxy(req)
		hc.updateHostStatus(host, resp)
		if resp.StatusCode < 500 {
			hc.rewrite(writer, resp)
		}
	}
}

func (hc *handlerCtx) fail(writer http.ResponseWriter, host circuitbreaker.Host) {
	timeout := host.Settings.Timeout
	if host.Settings.LongPooling && timeout > 0 {
		// Long pooling
		slog.Info("gofuse: hold", "host", host.ID, "timeout", timeout)
		<-time.After((timeout * percentile) / 100)
	}
	writer.WriteHeader(http.StatusServiceUnavailable)
	_, _ = writer.Write([]byte(fmt.Sprintf("{\"error\":gofuse open for host: %s}", host.ID)))
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

func (hc *handlerCtx) getHostWithSettings(host string) circuitbreaker.Host {
	settings, ok := hc.settings[host]
	if !ok {
		settings = hc.settings["default"]
	}
	return circuitbreaker.Host{
		ID:       host,
		State:    circuitbreaker.StateClosed,
		Settings: &settings,
	}
}

func (hc *handlerCtx) updateHostStatus(host circuitbreaker.Host, response *http.Response) {
	isRequestSuccessful := response.StatusCode < 500
	switch host.State {
	case circuitbreaker.StateClosed:
		if isRequestSuccessful && host.FailuresCount > 0 {
			host.FailuresCount--
		}
		if !isRequestSuccessful {
			host.FailuresCount++
			if host.FailuresCount >= host.Settings.MaxFailedTries {
				slog.Info("gofuse: becoming open", "host", host.ID, "failureCount", host.FailuresCount)
				host.State = circuitbreaker.StateOpen
				host.OpenExpiresAt = time.Now().Add(host.Settings.OpenTTL)
			}
		}
	case circuitbreaker.StateHalfOpen:
		if isRequestSuccessful {
			host.FailuresCount--
			if host.FailuresCount == 0 {
				slog.Info("gofuse: becoming closed", "host", host.ID)
				host.State = circuitbreaker.StateClosed
			}
		} else {
			host.State = circuitbreaker.StateOpen
			host.OpenExpiresAt = time.Now().Add(host.Settings.OpenTTL)
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
