package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

func (s *Server) CreateWallet(w http.ResponseWriter, r *http.Request) {
	var wallet models.Wallet

	if err := json.NewDecoder(r.Body).Decode(&wallet); err != nil {
		log.Error().Err(err).Msg("failed to decode r.Body in CreateWallet")
		http.Error(w, "error", http.StatusBadRequest)

		return
	}

	wallet.WalletID = uuid.New()
	wallet.Balance = 0

	wallet, err := s.service.CreateWallet(r.Context(), wallet)
	if err != nil {
		log.Error().Err(err).Msg("failed to create wallet")

		return
	}

	w.WriteHeader(http.StatusCreated)
	if err = json.NewEncoder(w).Encode(wallet); err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}

func (s *Server) GetWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	wallet, err := s.service.GetWallet(r.Context(), walletID)
	if err != nil {
		http.Error(w, "failed to get wallet", http.StatusBadRequest)
		log.Error().Err(err).Msg("failed to get wallet info")

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(wallet); err != nil {
		http.Error(w, "failed to encode wallet", http.StatusInternalServerError)

		return
	}
}

func (s *Server) UpdateWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	var wallet models.Wallet

	if err := json.NewDecoder(r.Body).Decode(&wallet); err != nil {
		http.Error(w, "error decoding json when updating wallet", http.StatusBadRequest)

		return
	}

	wallet.WalletID = walletID
	wallet.UpdatedAt = time.Now()
	wallet.Balance = 0

	if err := s.service.UpdateWallet(r.Context(), wallet); err != nil {
		log.Error().Err(err).Msg("failed to update wallet")
	}
}

func (s *Server) DeleteWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	if err := s.service.DeleteWallet(r.Context(), walletID); err != nil {
		log.Error().Err(err).Msg("error deleting wallet")
	}
}

func (s *Server) GetWallets(w http.ResponseWriter, r *http.Request) {
	wallets, err := s.service.GetAllWallets(r.Context())
	if err != nil {
		http.Error(w, "failed to obtain wallets", http.StatusInternalServerError)
	}

	w.Header().Set("content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(wallets); err != nil {
		http.Error(w, "error while encoding wallets info", http.StatusInternalServerError)

		return
	}
}
