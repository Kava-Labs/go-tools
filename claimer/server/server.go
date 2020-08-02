package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	RestTargetChain  = "target-chain"
	RestSwapID       = "swap-id"
	RestRandomNumber = "random-number"
)

type ClaimJob struct {
	TargetChain  string
	SwapID       string
	RandomNumber string
}

func NewClaimJob(targetChain, swapID, randomNumber string) ClaimJob {
	return ClaimJob{
		TargetChain:  targetChain,
		SwapID:       swapID,
		RandomNumber: randomNumber,
	}
}

type Server struct {
	Claims chan<- ClaimJob
}

func NewServer(claims chan<- ClaimJob) Server {
	return Server{
		Claims: claims,
	}
}

func (s Server) StartServer() {
	r := mux.NewRouter()
	r.HandleFunc("/claim", s.claim).Methods(http.MethodPost)
	r.HandleFunc("/", s.notFound)
	log.Fatal(http.ListenAndServe(":8080", r))
}

func (s Server) claim(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	targetChain := r.URL.Query().Get(RestTargetChain)
	if len(targetChain) == 0 {
		http.Error(w, fmt.Sprintf("%s is required", RestTargetChain), http.StatusInternalServerError)
		return
	}
	targetChainUpper := strings.ToUpper(targetChain)
	if targetChainUpper != "KAVA" && targetChainUpper != "BINANCE" && targetChainUpper != "BINANCE CHAIN" {
		http.Error(w, fmt.Sprintf("%s must be kava, binance, or binance chain", RestTargetChain), http.StatusInternalServerError)
		return
	}

	swapID := r.URL.Query().Get(RestSwapID)
	if len(swapID) == 0 {
		http.Error(w, fmt.Sprintf("%s is required", RestSwapID), http.StatusInternalServerError)
		return
	}

	randomNumber := r.URL.Query().Get(RestRandomNumber)
	if len(randomNumber) == 0 {
		http.Error(w, fmt.Sprintf("%s is required", RestRandomNumber), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf(`{"message": "request submitted"}`)))

	log.Info(fmt.Sprintf("Received claim request for %s on %s", swapID, targetChain))
	claimJob := NewClaimJob(targetChain, swapID, randomNumber)
	s.Claims <- claimJob
}

func (s Server) notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"message": "page not found"}`))
}
