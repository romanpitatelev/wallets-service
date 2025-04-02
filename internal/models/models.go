package models

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type (
	WalletID uuid.UUID
	UserID   uuid.UUID
	TxID     uuid.UUID
)

type User struct {
	UserID    UserID     `json:"userId"`
	DeletedAt *time.Time `json:"deletedAt"`
}

type Wallet struct {
	WalletID   WalletID   `json:"walletId"`
	UserID     UserID     `json:"userId"`
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
	UserID     UserID `json:"userId"`
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
	ErrInvalidToken         = errors.New("invalid token")
	ErrInvalidSigningMethod = errors.New("invalid signing method")
	ErrWrongUserID          = errors.New("wrong userID")
	ErrZeroAmount           = errors.New("invalid zero-amount transaction")
	ErrNegativeAmount       = errors.New("negative amount transaction")
	ErrSameWallet           = errors.New("same wallet transaction")
	ErrInsufficientFunds    = errors.New("wallet has insufficient funds")
	ErrInvalidTransaction   = errors.New("invalid wallets' data in transaction")
)

type XRRequest struct {
	FromCurrency string `json:"fromCurrency"`
	ToCurrency   string `json:"toCurrency"`
}

type XRResponse struct {
	Rate float64 `json:"rate"`
}

type Claims struct {
	UserID UserID `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type UserInfo struct {
	UserID UserID `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type Transaction struct {
	ID           TxID      `json:"transactionId"`
	Type         string    `json:"type"`
	ToWalletID   WalletID  `json:"toWalletId"`
	FromWalletID WalletID  `json:"fromWalletId"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	CommittedAt  time.Time `json:"committedAt"`
}

func (w *Wallet) Validate() error {
	if w.WalletName == "" {
		return ErrWalletEmptyName
	}

	w.Balance = 0
	w.Active = true

	return nil
}

func (u *UserInfo) Validate(walletUserID UserID) error {
	if walletUserID != u.UserID {
		return ErrWrongUserID
	}

	return nil
}

func (t *Transaction) Validate() error {
	switch {
	case t.Amount == 0:
		return ErrZeroAmount
	case t.Amount < 0:
		return ErrNegativeAmount
	case t.FromWalletID == t.ToWalletID:
		return ErrSameWallet
	default:
		if t.Type == "deposit" {
			if t.ToWalletID == WalletID(uuid.Nil) || t.FromWalletID != WalletID(uuid.Nil) {
				return ErrInvalidTransaction
			}
		}

		if t.Type == "withdraw" {
			if t.ToWalletID != WalletID(uuid.Nil) || t.FromWalletID == WalletID(uuid.Nil) {
				return ErrInvalidTransaction
			}
		}

		if t.Type == "transfer" {
			if t.ToWalletID == WalletID(uuid.Nil) || t.FromWalletID == WalletID(uuid.Nil) {
				return ErrInvalidTransaction
			}
		}
	}

	return nil
}
