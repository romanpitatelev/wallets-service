package rest

import (
	"context"
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

type service interface {
	CreateWallet(ctx context.Context, wallet models.Wallet, userID uuid.UUID) (models.Wallet, error)
	GetWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) (models.Wallet, error)
	UpdateWallet(ctx context.Context, walletID uuid.UUID, updatedWallet models.WalletUpdate, userID uuid.UUID) (models.Wallet, error)
	DeleteWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) error
	GetAllWallets(ctx context.Context, request models.GetWalletsRequest, userID uuid.UUID) ([]models.Wallet, error)
	Deposit(ctx context.Context, transaction models.Transaction, userID uuid.UUID) error
	Withdraw(ctx context.Context, transaction models.Transaction, userID uuid.UUID) error
	Transfer(ctx context.Context, transaction models.Transaction, userID uuid.UUID) error
	GetTransactions(ctx context.Context, request models.GetWalletsRequest, walletID uuid.UUID, userID uuid.UUID) ([]models.Transaction, error)
}

func (s *Server) createWallet(w http.ResponseWriter, r *http.Request) {
	var wallet models.Wallet

	if err := json.NewDecoder(r.Body).Decode(&wallet); err != nil {
		http.Error(w, "error", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	err := wallet.Validate()
	if err != nil {
		http.Error(w, "wallet validation error", http.StatusBadRequest)

		return
	}

	err = userInfo.Validate(wallet.UserID)
	if err != nil {
		http.Error(w, "user validation error", http.StatusNotFound)

		return
	}

	createdWallet, err := s.service.CreateWallet(ctx, wallet, userInfo.UserID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err = json.NewEncoder(w).Encode(createdWallet); err != nil {
		log.Warn().Err(err).Msg("failed to encode response")

		return
	}
}

func (s *Server) getWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	if walletID == uuid.Nil {
		http.Error(w, "walletID empty", http.StatusBadRequest)

		return
	}

	if userInfo.UserID == uuid.Nil {
		http.Error(w, "userID empty", http.StatusBadRequest)

		return
	}

	wallet, err := s.service.GetWallet(ctx, walletID, userInfo.UserID)
	if err != nil {
		if errors.Is(err, models.ErrWalletNotFound) {
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		}

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(wallet); err != nil {
		log.Warn().Err(err).Msg("failed to encode response")

		return
	}
}

func (s *Server) updateWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	var updatedDecodedWallet models.WalletUpdate

	if err := json.NewDecoder(r.Body).Decode(&updatedDecodedWallet); err != nil {
		http.Error(w, "error decoding json when updating wallet", http.StatusBadRequest)

		return
	}

	updatedWallet, err := s.service.UpdateWallet(ctx, walletID, updatedDecodedWallet, userInfo.UserID)

	switch {
	case errors.Is(err, models.ErrWalletNotFound):
		http.Error(w, "error wallet not found", http.StatusNotFound)

		return
	case errors.Is(err, models.ErrWrongCurrency):
		http.Error(w, "error wrong currency", http.StatusUnprocessableEntity)

		return
	case err != nil:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(updatedWallet); err != nil {
		log.Warn().Err(err).Msg("failed to encode response")

		return
	}
}

func (s *Server) deleteWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	err = s.service.DeleteWallet(ctx, walletID, userInfo.UserID)

	switch {
	case errors.Is(err, models.ErrWalletNotFound):
		http.Error(w, "wallet not found", http.StatusNotFound)

		return
	case errors.Is(err, models.ErrNonZeroBalanceWallet):
		http.Error(w, "wallet has non-zero balance, deletion forbidden", http.StatusBadRequest)

		return
	case err != nil:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getWallets(w http.ResponseWriter, r *http.Request) {
	request := parseGetRequest(r)
	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	wallets, err := s.service.GetAllWallets(ctx, request, userInfo.UserID)
	if err != nil {
		http.Error(w, "failed to obtain wallets", http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(wallets); err != nil {
		log.Warn().Err(err).Msg("error while encoding wallets info")

		return
	}
}

func parseGetRequest(r *http.Request) models.GetWalletsRequest {
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

func (s *Server) deposit(w http.ResponseWriter, r *http.Request) {
	var transaction models.Transaction

	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		http.Error(w, "error", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	if err := transaction.Validate(); err != nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if transaction.ToWalletID == uuid.Nil || transaction.FromWalletID != uuid.Nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if err := s.service.Deposit(ctx, transaction, userInfo.UserID); err != nil {
		switch {
		case errors.Is(err, models.ErrWalletNotFound):
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		case errors.Is(err, models.ErrWrongCurrency):
			http.Error(w, "invalid currency", http.StatusUnprocessableEntity)

			return
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) withdraw(w http.ResponseWriter, r *http.Request) {
	var transaction models.Transaction

	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		http.Error(w, "error", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	if err := transaction.Validate(); err != nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if transaction.ToWalletID != uuid.Nil || transaction.FromWalletID == uuid.Nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if err := s.service.Withdraw(ctx, transaction, userInfo.UserID); err != nil {
		switch {
		case errors.Is(err, models.ErrWalletNotFound):
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		case errors.Is(err, models.ErrWrongCurrency):
			http.Error(w, "invalid currency", http.StatusUnprocessableEntity)

			return
		case errors.Is(err, models.ErrInsufficientFunds):
			http.Error(w, "insufficient funds", http.StatusConflict)

			return
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}
	}
}

func (s *Server) transfer(w http.ResponseWriter, r *http.Request) {
	var transaction models.Transaction

	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		http.Error(w, "error", http.StatusBadRequest)
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	if err := transaction.Validate(); err != nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if transaction.ToWalletID == uuid.Nil || transaction.FromWalletID == uuid.Nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if err := s.service.Transfer(ctx, transaction, userInfo.UserID); err != nil {
		switch {
		case errors.Is(err, models.ErrWalletNotFound):
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		case errors.Is(err, models.ErrWrongCurrency):
			http.Error(w, "invalid currency", http.StatusUnprocessableEntity)

			return
		case errors.Is(err, models.ErrInsufficientFunds):
			http.Error(w, "insufficient funds", http.StatusConflict)

			return
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}
	}
}

func (s *Server) getTransactions(w http.ResponseWriter, r *http.Request) {
	request := parseGetRequest(r)
	ctx := r.Context()
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	userInfo := s.getUserInfo(ctx)

	transactions, err := s.service.GetTransactions(ctx, request, walletID, userInfo.UserID)
	if err != nil {
		http.Error(w, "failed to obrain transactions", http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(transactions); err != nil {
		log.Warn().Err(err).Msg("error while encoding transactions info")

		return
	}
}
