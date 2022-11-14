package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	RestTargetChain    = "target-chain"
	RestSwapID         = "swap-id"
	RestRandomNumber   = "random-number"
	TargetKava         = "KAVA"
	TargetBinance      = "BINANCE"
	TargetBinanceChain = "BINANCE CHAIN"
)

// ClaimJob defines a claim request received by the server
type ClaimJob struct {
	ID           string
	TargetChain  string
	SwapID       string
	RandomNumber string
}

// NewClaimJob instantiates a new instance of ClaimJob
func NewClaimJob(id, targetChain, swapID, randomNumber string) ClaimJob {
	return ClaimJob{
		ID:           id,
		TargetChain:  targetChain,
		SwapID:       swapID,
		RandomNumber: randomNumber,
	}
}

// Server that accepts HTTP POST claim requests on '/claim' and passes them to the Claims channel
type Server struct {
	Claims     chan<- ClaimJob
	httpServer *http.Server
}

// NewServer instantiates a new instance of Server
func NewServer(claims chan<- ClaimJob) *Server {
	return &Server{
		Claims: claims,
	}
}

// Start starts the server
func (s *Server) Start() error {
	r := mux.NewRouter()
	r.HandleFunc("/claim", s.claim).Methods(http.MethodPost)
	r.HandleFunc("/status", s.status)
	r.HandleFunc("/", s.notFound)
	s.httpServer = &http.Server{Addr: ":8080", Handler: r}
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// claim handles requests to submit a claim for a bep3 swap
func (s *Server) claim(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	log.WithFields(log.Fields{
		"request_id": requestID,
		"url":        r.URL.String(),
	}).Info("claim request received")

	w.Header().Set("Content-Type", "application/json")

	targetChain := r.URL.Query().Get(RestTargetChain)
	if len(targetChain) == 0 {
		http.Error(w, fmt.Sprintf("%s is required", RestTargetChain), http.StatusBadRequest)
		return
	}
	targetChainUpper := strings.ToUpper(targetChain)
	if targetChainUpper != TargetKava && targetChainUpper != TargetBinance && targetChainUpper != TargetBinanceChain {
		http.Error(w, fmt.Sprintf("%s must be kava, binance, or binance chain", RestTargetChain), http.StatusBadRequest)
		return
	}

	swapID := r.URL.Query().Get(RestSwapID)
	if len(swapID) == 0 {
		http.Error(w, fmt.Sprintf("%s is required", RestSwapID), http.StatusBadRequest)
		return
	}

	randomNumber := r.URL.Query().Get(RestRandomNumber)
	if len(randomNumber) == 0 {
		http.Error(w, fmt.Sprintf("%s is required", RestRandomNumber), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"message": "claim request received, attempting to process..."}`)))

	claimJob := NewClaimJob(requestID, targetChain, swapID, randomNumber)
	s.Claims <- claimJob

	log.WithFields(log.Fields{
		"request_id":   requestID,
		"swap_id":      swapID,
		"target_chain": targetChain,
	}).Info(fmt.Sprintf("claim request submitted to queue for processing"))
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"message": "page not found"}`))
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy"}`))
}
