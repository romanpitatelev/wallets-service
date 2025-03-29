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
	WithdrawFunds(ctx context.Context, transaction models.Transaction, userID uuid.UUID) error
	Transfer(ctx context.Context, transaction models.Transaction, userID uuid.UUID) error
	GetTransactions(ctx context.Context, request models.GetWalletsRequest, walletID uuid.UUID, userID uuid.UUID) ([]models.Transaction, error)
}

func (s *Server) createWallet(w http.ResponseWriter, r *http.Request) {
	var wallet models.Wallet

	if err := json.NewDecoder(r.Body).Decode(&wallet); err != nil {
		log.Info().Err(err).Msg("failed to decode r.Body in createWallet")
		http.Error(w, "error", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	err := wallet.Validate()
	if err != nil {
		log.Info().Err(err).Msg("wallet has failed validation check")
		http.Error(w, "wallet validation error", http.StatusBadRequest)

		return
	}

	err = userInfo.Validate(wallet.UserID)
	if err != nil {
		log.Info().Err(err).Msg("user has failed validation check")
		http.Error(w, "user validation error", http.StatusNotFound)

		return
	}

	createdWallet, err := s.service.CreateWallet(r.Context(), wallet, userInfo.UserID)
	if err != nil {
		log.Info().Err(err).Msg("failed to create wallet")
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

func (s *Server) getWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		log.Info().Err(err).Str("walletId", walletIDStr).Msg("Failed to parse walletId")
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

	wallet, err := s.service.GetWallet(r.Context(), walletID, userInfo.UserID)
	if err != nil {
		if errors.Is(err, models.ErrWalletNotFound) {
			log.Info().Err(err).Msg("wallet not found in getWallet()")
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		}

		log.Info().Err(err).Msg("failed to get wallet info in getWallet()")
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

func (s *Server) updateWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		log.Info().Err(err).Str("walletId", walletIDStr).Msg("Failed to parse walletId")
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	var updatedDecodedWallet models.WalletUpdate

	if err := json.NewDecoder(r.Body).Decode(&updatedDecodedWallet); err != nil {
		log.Info().Err(err).Msg("failed to decode updated wallet")
		http.Error(w, "error decoding json when updating wallet", http.StatusBadRequest)

		return
	}

	updatedWallet, err := s.service.UpdateWallet(r.Context(), walletID, updatedDecodedWallet, userInfo.UserID)

	switch {
	case errors.Is(err, models.ErrWalletNotFound):
		log.Info().Err(err).Msg("wallet not found in updateWallet()")
		http.Error(w, "error wallet not found", http.StatusNotFound)

		return
	case errors.Is(err, models.ErrWrongCurrency):
		log.Info().Err(err).Msg("wrong currency error in updateWallet()")
		http.Error(w, "error wrong currency", http.StatusUnprocessableEntity)

		return
	case err != nil:
		log.Info().Err(err).Msg("failed to update due to internal server error in updateWallet()")
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

func (s *Server) deleteWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		log.Info().Err(err).Str("walletId", walletIDStr).Msg("Failed to parse walletId")
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	err = s.service.DeleteWallet(r.Context(), walletID, userInfo.UserID)

	switch {
	case errors.Is(err, models.ErrWalletNotFound):
		log.Info().Err(err).Msg("wallet not found")
		http.Error(w, "wallet not found", http.StatusNotFound)

		return
	case errors.Is(err, models.ErrNonZeroBalanceWallet):
		log.Info().Err(err).Msg("deletion forbidden")
		http.Error(w, "wallet has non-zero balance, deletion forbidden", http.StatusBadRequest)

		return
	case err != nil:
		log.Info().Err(err).Msg("error deleting wallet")
		http.Error(w, "error deleting wallet", http.StatusInternalServerError)

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
		log.Info().Err(err).Msg("failed to obtain wallets")
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
		log.Info().Err(err).Msg("failed to decode r.Body in deposit()")
		http.Error(w, "error", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	if err := transaction.Validate(); err != nil {
		log.Info().Err(err).Msg("deposit transaction failed")
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if transaction.ToWalletID == uuid.Nil || transaction.FromWalletID != uuid.Nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if err := s.service.Deposit(r.Context(), transaction, userInfo.UserID); err != nil {
		switch {
		case errors.Is(err, models.ErrWalletNotFound):
			log.Info().Err(err).Msg("wallet not found")
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		case errors.Is(err, models.ErrWrongCurrency):
			log.Info().Err(err).Msg("currency not supported")
			http.Error(w, "invalid currency", http.StatusUnprocessableEntity)

			return
		default:
			log.Info().Err(err).Msg("error depositing funds")
			http.Error(w, "error depositing funds", http.StatusInternalServerError)

			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) withdrawFunds(w http.ResponseWriter, r *http.Request) {
	var transaction models.Transaction

	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		log.Info().Err(err).Msg("failed to decode r.Body in withdrawFunds()")
		http.Error(w, "error", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	if err := transaction.Validate(); err != nil {
		log.Info().Err(err).Msg("withdrawal failed")
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if transaction.ToWalletID != uuid.Nil || transaction.FromWalletID == uuid.Nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if err := s.service.WithdrawFunds(r.Context(), transaction, userInfo.UserID); err != nil {
		switch {
		case errors.Is(err, models.ErrWalletNotFound):
			log.Info().Err(err).Msg("wallet not found")
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		case errors.Is(err, models.ErrWrongCurrency):
			log.Info().Err(err).Msg("currency not supported")
			http.Error(w, "invalid currency", http.StatusUnprocessableEntity)

			return
		case errors.Is(err, models.ErrInsufficientFunds):
			log.Info().Err(err).Msg("insufficient funds error")
			http.Error(w, "invalid currency", http.StatusBadRequest)

			return
		default:
			log.Info().Err(err).Msg("error depositing funds")
			http.Error(w, "error withdrawing funds", http.StatusInternalServerError)

			return
		}
	}
}

func (s *Server) transfer(w http.ResponseWriter, r *http.Request) {
	var transaction models.Transaction

	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		log.Info().Err(err).Msg("failed to decode r.Body in transfer()")
		http.Error(w, "error", http.StatusBadRequest)
	}

	ctx := r.Context()
	userInfo := s.getUserInfo(ctx)

	if err := transaction.Validate(); err != nil {
		log.Info().Err(err).Msg("withdrawal failed")
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if transaction.ToWalletID == uuid.Nil || transaction.FromWalletID == uuid.Nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if err := s.service.Transfer(r.Context(), transaction, userInfo.UserID); err != nil {
		switch {
		case errors.Is(err, models.ErrWalletNotFound):
			log.Info().Err(err).Msg("wallet not found")
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		case errors.Is(err, models.ErrWrongCurrency):
			log.Info().Err(err).Msg("currency unsupported")
			http.Error(w, "invalid currency", http.StatusUnprocessableEntity)

			return
		case errors.Is(err, models.ErrInsufficientFunds):
			log.Info().Err(err).Msg("insufficient funds in the source wallet")
			http.Error(w, "invalid currency", http.StatusBadRequest)

			return
		default:
			log.Info().Err(err).Msg("error depositing funds")
			http.Error(w, "error withdrawing funds", http.StatusInternalServerError)

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
		log.Info().Err(err).Str("walletId", walletIDStr).Msg("Failed to parse walletId")
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	userInfo := s.getUserInfo(ctx)

	transactions, err := s.service.GetTransactions(ctx, request, walletID, userInfo.UserID)
	if err != nil {
		log.Info().Err(err).Msg("failed to obtain transactions")
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
