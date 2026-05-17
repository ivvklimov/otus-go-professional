package internalhttp

import (
	"net/http"
)

// helloHandler — единственный эндпоинт ДЗ №12.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	// Поддерживаем только GET
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("hello"))
}
