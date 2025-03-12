package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type IP struct {
	Address string `json:"address"`
	Count   int    `json:"count"`
}

type User struct {
	UserID    int       `json:"userId"`
	DeletedAt time.Time `json:"deletedAt"`
}

type Wallet struct {
	WalletID   uuid.UUID  `json:"walletId"`
	UserID     uuid.UUID  `json:"userId"`
	WalletName string     `json:"walletName"`
	Balance    float64    `json:"balance"`
	Currency   string     `json:"currency"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	DeletedAt  *time.Time `json:"deletedAt"`
	Active     bool       `json:"active"`
}

type WalletUpdate struct {
	WalletName string `json:"walletName"`
	Currency   string `json:"currency"`
}

type GetWalletsRequest struct {
	Sorting    string `json:"sorting,omitempty"`
	Descending bool   `json:"descending,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Filter     string `json:"filter,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

var (
	ErrWalletEmptyName      = errors.New("wallet name cannot be empty")
	ErrWalletNotFound       = errors.New("error wallet not found")
	ErrWalletUpToDate       = errors.New("wallet is up-to-date")
	ErrZeroValueWallet      = errors.New("zero-value wallet")
	ErrNonZeroBalanceWallet = errors.New("wallet has non-zero balance")
	ErrWrongCurrency        = errors.New("wrong currency")
)

type XRRequest struct {
	FromCurrency string `json:"fromCurrency"`
	ToCurrency   string `json:"toCurrency"`
}

type XRResponse struct {
	Rate float64 `json:"rate"`
}
