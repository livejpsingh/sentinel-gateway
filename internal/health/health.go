package health

import (
	"net/http"
	"sync/atomic"
)

type Readiness struct {
	ready atomic.Bool
}

func NewReadiness() *Readiness {
	readiness := &Readiness{}
	readiness.ready.Store(true)
	return readiness
}

func (r *Readiness) Set(ready bool) {
	r.ready.Store(ready)
}

func LivenessHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
}

func ReadinessHandler(readiness *Readiness) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !readiness.ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
}
