package walletshandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/controller/rest/common"
	"github.com/romanpitatelev/wallets-service/internal/entity"
	"github.com/rs/zerolog/log"
)

type walletsService interface {
	CreateWallet(ctx context.Context, wallet entity.Wallet, userID entity.UserID) (entity.Wallet, error)
	GetWallet(ctx context.Context, walletID entity.WalletID, userID entity.UserID) (entity.Wallet, error)
	UpdateWallet(ctx context.Context, walletID entity.WalletID, updatedWallet entity.WalletUpdate, userID entity.UserID) (entity.Wallet, error)
	DeleteWallet(ctx context.Context, walletID entity.WalletID, userID entity.UserID) error
	GetAllWallets(ctx context.Context, request entity.GetWalletsRequest, userID entity.UserID) ([]entity.Wallet, error)
}

type Handler struct {
	walletsService walletsService
}

func New(walletsService walletsService) *Handler {
	return &Handler{
		walletsService: walletsService,
	}
}

func (h *Handler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	var wallet entity.Wallet

	if err := json.NewDecoder(r.Body).Decode(&wallet); err != nil {
		http.Error(w, fmt.Sprintf("%d: %s", http.StatusBadRequest, err), http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := common.GetUserInfo(ctx)

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

	createdWallet, err := h.walletsService.CreateWallet(ctx, wallet, userInfo.UserID)
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

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := common.GetUserInfo(ctx)

	if walletID == uuid.Nil {
		http.Error(w, "walletID empty", http.StatusBadRequest)

		return
	}

	if userInfo.UserID == entity.UserID(uuid.Nil) {
		http.Error(w, "userID empty", http.StatusBadRequest)

		return
	}

	wallet, err := h.walletsService.GetWallet(ctx, entity.WalletID(walletID), userInfo.UserID)
	if err != nil {
		if errors.Is(err, entity.ErrWalletNotFound) {
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

func (h *Handler) UpdateWallet(w http.ResponseWriter, r *http.Request) {
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := common.GetUserInfo(ctx)

	var updatedDecodedWallet entity.WalletUpdate

	if err := json.NewDecoder(r.Body).Decode(&updatedDecodedWallet); err != nil {
		http.Error(w, "error decoding json when updating wallet", http.StatusBadRequest)

		return
	}

	updatedWallet, err := h.walletsService.UpdateWallet(ctx, entity.WalletID(walletID), updatedDecodedWallet, userInfo.UserID)

	switch {
	case errors.Is(err, entity.ErrWalletNotFound):
		http.Error(w, "error wallet not found", http.StatusNotFound)

		return
	case errors.Is(err, entity.ErrWrongCurrency):
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

func (h *Handler) DeleteWallet(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msgf("r in DeleteWallet() is this: %v", r)

	walletIDStr := chi.URLParam(r, "walletId")

	log.Debug().Msgf("walletIDStr in DeleteWallet is: %s", walletIDStr)

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	ctx := r.Context()
	userInfo := common.GetUserInfo(ctx)

	err = h.walletsService.DeleteWallet(ctx, entity.WalletID(walletID), userInfo.UserID)

	switch {
	case errors.Is(err, entity.ErrWalletNotFound):
		http.Error(w, "wallet not found", http.StatusNotFound)

		return
	case errors.Is(err, entity.ErrNonZeroBalanceWallet):
		http.Error(w, "wallet has non-zero balance, deletion forbidden", http.StatusBadRequest)

		return
	case err != nil:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetWallets(w http.ResponseWriter, r *http.Request) {
	request := common.ParseGetRequest(r)
	ctx := r.Context()
	userInfo := common.GetUserInfo(ctx)

	wallets, err := h.walletsService.GetAllWallets(ctx, request, userInfo.UserID)
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
