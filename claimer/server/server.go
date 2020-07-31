package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func StartServer() {
	r := mux.NewRouter()
	r.HandleFunc("/claim", claim).Methods(http.MethodPost)
	r.HandleFunc("/", notFound)
	log.Fatal(http.ListenAndServe(":8080", r))
}

func claim(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// TODO: validate query params
	targetChain := r.URL.Query().Get(RestTargetChain)
	expectedSwapID := r.URL.Query().Get(RestSwapID)
	randomNumber := r.URL.Query().Get(RestRandomNumber)

	w.Write([]byte(fmt.Sprintf(`{"targetChain": "%s", "expectedSwapID": "%s", "randomNumber": "%s" }`,
		targetChain, expectedSwapID, randomNumber)))

	// TODO: pass to pool
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"message": "not found"}`))
}
