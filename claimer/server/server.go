package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kava-labs/go-tools/claimer/claimer"
	"github.com/kava-labs/go-tools/claimer/types"
	log "github.com/sirupsen/logrus"
)

// Server that accepts HTTP POST claim requests on '/claim' and passes them to the Claims channel
type Server struct {
	Dispatcher *claimer.Dispatcher
	httpServer *http.Server
}

// NewServer instantiates a new instance of Server
func NewServer(dispatcher *claimer.Dispatcher) *Server {
	return &Server{
		Dispatcher: dispatcher,
	}
}

// Start starts the server
func (s *Server) Start() error {
	r := mux.NewRouter()
	r.HandleFunc("/claim", s.claim).Methods(http.MethodPost)
	r.HandleFunc("/status", s.status)
	r.HandleFunc("/", s.notFound)

	healthChecker := healthCheckHandler(
		context.Background(),
		log.New(),
		s.Dispatcher,
	)
	r.HandleFunc("/health", healthChecker)

	s.httpServer = &http.Server{Addr: ":8080", Handler: handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r)}
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

	targetChain := r.URL.Query().Get(types.RestTargetChain)
	if len(targetChain) == 0 {
		http.Error(w, fmt.Sprintf("%s is required", types.RestTargetChain), http.StatusBadRequest)
		return
	}
	targetChainUpper := strings.ToUpper(targetChain)
	if targetChainUpper != types.TargetKava && targetChainUpper != types.TargetBinance && targetChainUpper != types.TargetBinanceChain {
		http.Error(w, fmt.Sprintf("%s must be kava, binance, or binance chain", types.RestTargetChain), http.StatusBadRequest)
		return
	}

	swapID := r.URL.Query().Get(types.RestSwapID)
	if len(swapID) == 0 {
		http.Error(w, fmt.Sprintf("%s is required", types.RestSwapID), http.StatusBadRequest)
		return
	}

	randomNumber := r.URL.Query().Get(types.RestRandomNumber)
	if len(randomNumber) == 0 {
		http.Error(w, fmt.Sprintf("%s is required", types.RestRandomNumber), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "claim request received, attempting to process..."}`))

	claimJob := types.NewClaimJob(requestID, targetChain, swapID, randomNumber)
	s.Dispatcher.JobQueue() <- claimJob

	log.WithFields(log.Fields{
		"request_id":   requestID,
		"swap_id":      swapID,
		"target_chain": targetChain,
	}).Info("claim request submitted to queue for processing")
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
