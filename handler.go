package main

import "net/http"

func handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/embeddings" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}
