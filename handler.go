package main

import "net/http"

func handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
