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
	DeletedAt time.Time `json:"deleted_at"`
}

type Wallet struct {
	WalletID   uuid.UUID `json:"walletId"`
	UserID     uuid.UUID `json:"userId"`
	WalletName string    `json:"walletName"`
	Balance    float64   `json:"balance"`
	Currency   string    `json:"currency"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// TODO переделать апдейт, поля должны быть *
type WalletUpdate struct {
	WalletName string `json:"walletName"`
	Currency   string `json:"currency"`
}

var (
	ErrWalletNotFound = errors.New("error wallet not found")
)
