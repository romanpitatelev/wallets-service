package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

const (
	DefaultLimit = 25
)

func (s *Server) CreateWallet(w http.ResponseWriter, r *http.Request) {
	var wallet models.Wallet

	if err := json.NewDecoder(r.Body).Decode(&wallet); err != nil {
		log.Error().Err(err).Msg("failed to decode r.Body in CreateWallet")
		http.Error(w, "error", http.StatusBadRequest)

		return
	}

	validWallet, err := s.ValidateWallet(wallet)
	if err != nil {
		log.Error().Err(err).Msg("wallet has failed validation check")
		http.Error(w, "validation error", http.StatusBadRequest)

		return
	}

	createdWallet, err := s.service.CreateWallet(r.Context(), validWallet)
	if err != nil {
		log.Error().Err(err).Msg("failed to create wallet")
		http.Error(w, "error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err = json.NewEncoder(w).Encode(createdWallet); err != nil {
		log.Warn().Err(err).Msg("failed to encode response")

		return
	}
}

func (s *Server) GetWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		log.Error().Err(err).Str("walletId", walletIDStr).Msg("Failed to parse walletId")
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	wallet, err := s.service.GetWallet(r.Context(), walletID)
	if err != nil {
		if errors.Is(err, models.ErrWalletNotFound) {
			log.Error().Err(err).Msg("wallet not found in GetWallet()")
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		}

		log.Error().Err(err).Msg("failed to get wallet info in GetWallet()")
		http.Error(w, "failed to get wallet", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(wallet); err != nil {
		log.Warn().Err(err).Msg("failed to encode response")

		return
	}
}

func (s *Server) UpdateWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		log.Error().Err(err).Str("walletId", walletIDStr).Msg("Failed to parse walletId")
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	var updatedDecodedWallet models.WalletUpdate

	if err := json.NewDecoder(r.Body).Decode(&updatedDecodedWallet); err != nil {
		log.Error().Err(err).Msg("failed to decode updated wallet")
		http.Error(w, "error decoding json when updating wallet", http.StatusBadRequest)

		return
	}

	updatedWallet, err := s.service.UpdateWallet(r.Context(), walletID, updatedDecodedWallet)

	switch {
	case errors.Is(err, models.ErrWalletNotFound):
		log.Error().Err(err).Msg("wallet not found in UpdateWallet()")
		http.Error(w, "error wallet not found", http.StatusNotFound)

		return
	case errors.Is(err, models.ErrWrongCurrency):
		log.Error().Err(err).Msg("wrong currency error in UpdateWallet()")
		http.Error(w, "error wrong currency", http.StatusUnprocessableEntity)

		return
	case err != nil:
		log.Error().Err(err).Msg("failed to update due to internal server error in UpdateWallet()")
		http.Error(w, "failed to update wallet", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(updatedWallet); err != nil {
		log.Warn().Err(err).Msg("failed to encode response")

		return
	}
}

func (s *Server) DeleteWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		log.Error().Err(err).Str("walletId", walletIDStr).Msg("Failed to parse walletId")
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	err = s.service.DeleteWallet(r.Context(), walletID)

	switch {
	case errors.Is(err, models.ErrWalletNotFound):
		log.Error().Err(err).Msg("wallet not found")
		http.Error(w, "wallet not found", http.StatusNotFound)

		return
	case errors.Is(err, models.ErrNonZeroBalanceWallet):
		log.Error().Err(err).Msg("deletion forbidden")
		http.Error(w, "wallet has non-zero balance, deletion forbidden", http.StatusBadRequest)

		return
	case err != nil:
		log.Error().Err(err).Msg("error deleting wallet")
		http.Error(w, "error deleting wallet", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) GetWallets(w http.ResponseWriter, r *http.Request) {
	request := ParseGetRequest(r)
	ctx := r.Context()

	wallets, err := s.service.GetAllWallets(ctx, request)
	if err != nil {
		log.Error().Err(err).Msg("failed to obtain wallets")
		http.Error(w, "failed to obtain wallets", http.StatusNotFound)

		return
	}

	w.Header().Set("content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(wallets); err != nil {
		log.Warn().Err(err).Msg("error while encoding wallets info")

		return
	}
}

func ParseGetRequest(r *http.Request) models.GetWalletsRequest {
	queryParams := r.URL.Query()

	parameters := models.GetWalletsRequest{
		Sorting: queryParams.Get("sorting"),
		Filter:  queryParams.Get("filter"),
	}

	var (
		limit  int64
		offset int64
	)

	if d := queryParams.Get("descending"); d != "" {
		parameters.Descending, _ = strconv.ParseBool(d)
	}

	if l := queryParams.Get("limit"); l != "" {
		if limit, _ = strconv.ParseInt(l, 0, 64); limit == 0 {
			limit = DefaultLimit
		}

		parameters.Limit = int(limit)
	} else {
		parameters.Limit = DefaultLimit
	}

	if o := queryParams.Get("offset"); o != "" {
		offset, _ = strconv.ParseInt(o, 0, 64)
		parameters.Offset = int(offset)
	}

	return parameters
}

func (s *Server) ValidateWallet(wallet models.Wallet) (models.Wallet, error) {
	if wallet.WalletName == "" {
		return wallet, models.ErrWalletEmptyName
	}

	wallet.Balance = 0
	wallet.Active = true

	return wallet, nil
}
