package transactionshandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/controller/rest/common"
	"github.com/romanpitatelev/wallets-service/internal/entity"
	"github.com/rs/zerolog/log"
)

type transactionsService interface {
	Deposit(ctx context.Context, transaction entity.Transaction, userID entity.UserID) error
	Withdraw(ctx context.Context, transaction entity.Transaction, userID entity.UserID) error
	Transfer(ctx context.Context, transaction entity.Transaction, userID entity.UserID) error
	GetTransactions(ctx context.Context, request entity.GetWalletsRequest, walletID entity.WalletID, userID entity.UserID) ([]entity.Transaction, error)
}

type Handler struct {
	transactionsService transactionsService
}

func New(transactionsService transactionsService) *Handler {
	return &Handler{
		transactionsService: transactionsService,
	}
}

func (h *Handler) Deposit(w http.ResponseWriter, r *http.Request) {
	var transaction entity.Transaction

	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		http.Error(w, "error", http.StatusBadRequest)

		return
	}

	transaction.Type = "deposit"

	ctx := r.Context()
	userInfo := common.GetUserInfo(ctx)

	if err := transaction.Validate(); err != nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if err := h.transactionsService.Deposit(ctx, transaction, userInfo.UserID); err != nil {
		switch {
		case errors.Is(err, entity.ErrWalletNotFound):
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		case errors.Is(err, entity.ErrWrongCurrency):
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

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var transaction entity.Transaction

	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		http.Error(w, "error", http.StatusBadRequest)

		return
	}

	transaction.Type = "withdraw"

	ctx := r.Context()
	userInfo := common.GetUserInfo(ctx)

	if err := transaction.Validate(); err != nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if err := h.transactionsService.Withdraw(ctx, transaction, userInfo.UserID); err != nil {
		switch {
		case errors.Is(err, entity.ErrWalletNotFound):
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		case errors.Is(err, entity.ErrWrongCurrency):
			http.Error(w, "invalid currency", http.StatusUnprocessableEntity)

			return
		case errors.Is(err, entity.ErrInsufficientFunds):
			http.Error(w, "insufficient funds", http.StatusConflict)

			return
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}
	}
}

func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	var transaction entity.Transaction

	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		http.Error(w, "error", http.StatusBadRequest)
	}

	transaction.Type = "transfer"

	ctx := r.Context()
	userInfo := common.GetUserInfo(ctx)

	if err := transaction.Validate(); err != nil {
		http.Error(w, "transaction validation error", http.StatusBadRequest)

		return
	}

	if err := h.transactionsService.Transfer(ctx, transaction, userInfo.UserID); err != nil {
		switch {
		case errors.Is(err, entity.ErrWalletNotFound):
			http.Error(w, "wallet not found", http.StatusNotFound)

			return
		case errors.Is(err, entity.ErrWrongCurrency):
			http.Error(w, "invalid currency", http.StatusUnprocessableEntity)

			return
		case errors.Is(err, entity.ErrInsufficientFunds):
			http.Error(w, "insufficient funds", http.StatusConflict)

			return
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}
	}
}

func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	request := common.ParseGetRequest(r)
	ctx := r.Context()
	walletIDStr := chi.URLParam(r, "walletId")

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)

		return
	}

	userInfo := common.GetUserInfo(ctx)

	transactions, err := h.transactionsService.GetTransactions(ctx, request, entity.WalletID(walletID), userInfo.UserID)
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
