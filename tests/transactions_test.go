//nolint:testpackage
package tests

import (
	"context"
	"math"
	"net/http"

	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
)

//nolint:gochecknoglobals
var exchangeRatesToRub = map[string]float64{
	"RUB": 1.0,
	"USD": 90.0,
	"EUR": 100.0,
	"CNY": 12.3,
	"CHF": 101.0,
	"GBP": 115.0,
	"KZT": 0.18,
	"RSD": 0.83,
}

func (s *IntegrationTestSuite) TestDeposit() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testDeposit",
		Currency:   "USD",
	}

	err := s.db.UpsertUser(context.Background(), existingUser)
	s.Require().NoError(err)

	wallet.UserID = existingUser.UserID

	var createdWallet models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet, existingUser)

	s.Run("deposit transaction successful", func() {
		transaction := models.Transaction{
			ID:         uuid.New(),
			ToWalletID: createdWallet.WalletID,
			Amount:     900.0,
			Currency:   "USD",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/deposit"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusOK, &transaction, nil, existingUser)

		var updatedWallet models.Wallet

		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &updatedWallet, existingUser)

		expectedBalance := transaction.Amount

		s.Require().True(math.Abs(updatedWallet.Balance-expectedBalance) < epsilon)
	})

	s.Run("deposit foreign currency successful", func() {
		transaction := models.Transaction{
			ID:         uuid.New(),
			ToWalletID: createdWallet.WalletID,
			Amount:     500,
			Currency:   "CHF",
		}

		currency := Currency{Name: transaction.Currency, Value: exchangeRatesToRub[transaction.Currency] / exchangeRatesToRub[wallet.Currency]}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &createdWallet, existingUser)

		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString + "/deposit"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusOK, &transaction, nil, existingUser)

		var updatedWallet models.Wallet

		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &updatedWallet, existingUser)

		expectedBalance := createdWallet.Balance + transaction.Amount*currency.Value

		s.Require().True(math.Abs(updatedWallet.Balance-expectedBalance) < epsilon)
	})

	s.Run("deposit negative amount should fail", func() {
		transaction := models.Transaction{
			ID:         uuid.New(),
			ToWalletID: createdWallet.WalletID,
			Amount:     -100.0,
			Currency:   "USD",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/deposit"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusBadRequest, &transaction, nil, existingUser)
	})

	s.Run("unprocessable currency deposit", func() {
		transaction := models.Transaction{
			ID:         uuid.New(),
			ToWalletID: createdWallet.WalletID,
			Amount:     200.0,
			Currency:   "TRY",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/deposit"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusUnprocessableEntity, &transaction, nil, existingUser)
	})

	s.Run("wallet not found", func() {
		transaction := models.Transaction{
			ID:         uuid.New(),
			ToWalletID: uuid.New(),
			Amount:     300.0,
			Currency:   "EUR",
		}

		uuidString := transaction.ToWalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/deposit"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusNotFound, &transaction, nil, existingUser)
	})

	s.Run("wallet belongs to another user", func() {
		transaction := models.Transaction{
			ID:         uuid.New(),
			ToWalletID: createdWallet.WalletID,
			Amount:     438.0,
			Currency:   "CNY",
		}

		otherUser := models.User{UserID: uuid.New()}

		err := s.db.UpsertUser(context.Background(), otherUser)
		s.Require().NoError(err)

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/deposit"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusNotFound, &transaction, nil, otherUser)
	})

	s.Run("wallet is not specified", func() {
		transaction := models.Transaction{
			ID:         uuid.New(),
			ToWalletID: uuid.Nil,
			Amount:     10100.0,
			Currency:   "RUB",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/deposit"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusBadRequest, &transaction, nil, existingUser)
	})

	s.Run("deposit zero amout failed", func() {
		transaction := models.Transaction{
			ID:         uuid.New(),
			ToWalletID: createdWallet.WalletID,
			Amount:     0.0,
			Currency:   "USD",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/deposit"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusBadRequest, &transaction, nil, existingUser)
	})

	s.Run("user is not found in the database", func() {
		transaction := models.Transaction{
			ID:         uuid.New(),
			ToWalletID: createdWallet.WalletID,
			Amount:     500.0,
			Currency:   "USD",
		}

		newUser := models.User{
			UserID: uuid.New(),
		}

		createdWallet.UserID = newUser.UserID
		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/deposit"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusNotFound, &transaction, nil, newUser)
	})
}

func (s *IntegrationTestSuite) TestWithdrawFunds() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testWithdrawFundsWallet",
		Currency:   "RUB",
	}

	err := s.db.UpsertUser(context.Background(), existingUser)
	s.Require().NoError(err)

	wallet.UserID = existingUser.UserID

	var createdWallet models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet, existingUser)

	transaction := models.Transaction{
		ID:         uuid.New(),
		ToWalletID: createdWallet.WalletID,
		Amount:     14000.0,
		Currency:   "RUB",
	}

	uuidString := wallet.WalletID.String()
	walletIDPath := walletPath + "/" + uuidString + "/deposit"

	s.sendRequest(http.MethodPut, walletIDPath, http.StatusOK, &transaction, nil, existingUser)

	s.Run("withdrawal in wallet currency processed succussfully", func() {
		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &createdWallet, existingUser)

		transaction := models.Transaction{
			ID:           uuid.New(),
			FromWalletID: createdWallet.WalletID,
			Amount:       375.0,
			Currency:     "RUB",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/withdrawal"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusOK, &transaction, nil, existingUser)

		var updatedWallet models.Wallet

		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &updatedWallet, existingUser)

		expectedBalance := createdWallet.Balance - transaction.Amount

		s.Require().True(math.Abs(expectedBalance-updatedWallet.Balance) < epsilon)
	})

	s.Run("withdrawal amount in wallet currency exceeds wallet balance", func() {
		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &createdWallet, existingUser)

		transaction := models.Transaction{
			ID:           uuid.New(),
			FromWalletID: createdWallet.WalletID,
			Amount:       14000.0,
			Currency:     "RUB",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/withdrawal"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusBadRequest, &transaction, nil, existingUser)

		var nonmodifiedWallet models.Wallet

		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &nonmodifiedWallet, existingUser)

		s.Require().Equal(nonmodifiedWallet.Balance, createdWallet.Balance)
	})

	s.Run("withdrawal in foreign currency processed succussfully", func() {
		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &createdWallet, existingUser)

		transaction := models.Transaction{
			ID:           uuid.New(),
			FromWalletID: createdWallet.WalletID,
			Amount:       15.0,
			Currency:     "CNY",
		}

		currency := Currency{Name: transaction.Currency, Value: exchangeRatesToRub[transaction.Currency] / exchangeRatesToRub[wallet.Currency]}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/withdrawal"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusOK, &transaction, nil, existingUser)

		var updatedWallet models.Wallet

		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &updatedWallet, existingUser)

		expectedBalance := createdWallet.Balance - transaction.Amount*currency.Value

		s.Require().True(math.Abs(expectedBalance-updatedWallet.Balance) < epsilon)
	})

	s.Run("withdrawal amount in foreign currency exceeds wallet balance", func() {
		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &createdWallet, existingUser)

		transaction := models.Transaction{
			ID:           uuid.New(),
			FromWalletID: createdWallet.WalletID,
			Amount:       10000.0,
			Currency:     "CHF",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/withdrawal"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusBadRequest, &transaction, nil, existingUser)

		var nonmodifiedWallet models.Wallet

		uuidString = createdWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &nonmodifiedWallet, existingUser)

		s.Require().Equal(nonmodifiedWallet.Balance, createdWallet.Balance)
	})

	s.Run("withdraw zero amout failed", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			FromWalletID: createdWallet.WalletID,
			Amount:       0.0,
			Currency:     "RUB",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/withdrawal"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusBadRequest, &transaction, nil, existingUser)
	})

	s.Run("unprocessable currency withdrawal", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			FromWalletID: createdWallet.WalletID,
			Amount:       30.0,
			Currency:     "TRY",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/withdrawal"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusUnprocessableEntity, &transaction, nil, existingUser)
	})

	s.Run("wallet is not specified", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			FromWalletID: uuid.Nil,
			Amount:       10100.0,
			Currency:     "RUB",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/deposit"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusBadRequest, &transaction, nil, existingUser)
	})

	s.Run("wallet not found", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			FromWalletID: uuid.New(),
			Amount:       300.0,
			Currency:     "RUB",
		}

		uuidString := transaction.ToWalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/withdrawal"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusNotFound, &transaction, nil, existingUser)
	})

	s.Run("wallet belongs to another user", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			FromWalletID: createdWallet.WalletID,
			Amount:       733.0,
			Currency:     "RUB",
		}

		otherUser := models.User{UserID: uuid.New()}

		err := s.db.UpsertUser(context.Background(), otherUser)
		s.Require().NoError(err)

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/withdrawal"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusNotFound, &transaction, nil, otherUser)
	})

	s.Run("user is not found in the database", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			FromWalletID: createdWallet.WalletID,
			Amount:       4600.0,
			Currency:     "RUB",
		}

		newUser := models.User{
			UserID: uuid.New(),
		}

		createdWallet.UserID = newUser.UserID
		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/withdrawal"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusNotFound, &transaction, nil, newUser)
	})
}

//nolint:maintidx
func (s *IntegrationTestSuite) TestTransfer() {
	toWallet := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "transferWallet_1",
		Currency:   "RUB",
	}

	fromWallet := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "transferWallet_2",
		Currency:   "RUB",
	}

	fromWalletFX := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "transferWallet_3",
		Currency:   "USD",
	}

	err := s.db.UpsertUser(context.Background(), existingUser)
	s.Require().NoError(err)

	toWallet.UserID = existingUser.UserID
	fromWallet.UserID = existingUser.UserID
	fromWalletFX.UserID = existingUser.UserID

	var createdToWallet models.Wallet

	var createdFromWallet models.Wallet

	var createdFromWalletFX models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &toWallet, &createdToWallet, existingUser)
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &fromWallet, &createdFromWallet, existingUser)
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &fromWalletFX, &createdFromWalletFX, existingUser)

	transactionTo := models.Transaction{
		ID:         uuid.New(),
		ToWalletID: toWallet.WalletID,
		Amount:     1000.0,
		Currency:   "RUB",
	}

	transactionFrom := models.Transaction{
		ID:         uuid.New(),
		ToWalletID: fromWallet.WalletID,
		Amount:     8000.0,
		Currency:   "RUB",
	}

	transactionFromFX := models.Transaction{
		ID:         uuid.New(),
		ToWalletID: fromWalletFX.WalletID,
		Amount:     200.0,
		Currency:   "USD",
	}

	uuidString := createdToWallet.WalletID.String()
	walletIDPath := walletPath + "/" + uuidString + "/deposit"

	s.sendRequest(http.MethodPut, walletIDPath, http.StatusOK, &transactionTo, nil, existingUser)

	uuidString = createdFromWallet.WalletID.String()
	walletIDPath = walletPath + "/" + uuidString + "/deposit"

	s.sendRequest(http.MethodPut, walletIDPath, http.StatusOK, &transactionFrom, nil, existingUser)

	uuidString = createdFromWalletFX.WalletID.String()
	walletIDPath = walletPath + "/" + uuidString + "/deposit"

	s.sendRequest(http.MethodPut, walletIDPath, http.StatusOK, &transactionFromFX, nil, existingUser)

	s.Run("transfer processed successfully", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			ToWalletID:   createdToWallet.WalletID,
			FromWalletID: createdFromWallet.WalletID,
			Amount:       500.0,
			Currency:     "RUB",
		}

		uuidString := createdFromWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/transfer"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusOK, &transaction, nil, existingUser)

		var updatedToWallet models.Wallet

		var updatedFromWallet models.Wallet

		uuidStringTo := createdToWallet.WalletID.String()
		walletIDPathTo := walletPath + "/" + uuidStringTo

		uuidStringFrom := createdFromWallet.WalletID.String()
		walletIDPathFrom := walletPath + "/" + uuidStringFrom

		s.sendRequest(http.MethodGet, walletIDPathTo, http.StatusOK, nil, &updatedToWallet, existingUser)
		s.sendRequest(http.MethodGet, walletIDPathFrom, http.StatusOK, nil, &updatedFromWallet, existingUser)

		expectedBalanceTo := transactionTo.Amount + transaction.Amount
		expectedBalanceFrom := transactionFrom.Amount - transaction.Amount

		s.Require().True(math.Abs(updatedToWallet.Balance-expectedBalanceTo) < epsilon)
		s.Require().True(math.Abs(updatedFromWallet.Balance-expectedBalanceFrom) < epsilon)
	})

	s.Run("zero amount transfer", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			ToWalletID:   createdToWallet.WalletID,
			FromWalletID: createdFromWallet.WalletID,
			Amount:       0.0,
			Currency:     "RUB",
		}

		uuidString := createdFromWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/transfer"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusBadRequest, &transaction, nil, existingUser)
	})

	s.Run("negative amount transfer", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			ToWalletID:   createdToWallet.WalletID,
			FromWalletID: createdFromWallet.WalletID,
			Amount:       -60.0,
			Currency:     "RUB",
		}

		uuidString := createdFromWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/transfer"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusBadRequest, &transaction, nil, existingUser)
	})

	s.Run("transfer amount exceeds source wallet balance", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			ToWalletID:   createdToWallet.WalletID,
			FromWalletID: createdFromWallet.WalletID,
			Amount:       50000.0,
			Currency:     "RUB",
		}

		uuidString := createdFromWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/transfer"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusBadRequest, &transaction, nil, existingUser)
	})

	s.Run("wrong currency transaction failed", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			ToWalletID:   createdToWallet.WalletID,
			FromWalletID: createdFromWallet.WalletID,
			Amount:       5.0,
			Currency:     "EUR",
		}
		uuidString := createdFromWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/transfer"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusUnprocessableEntity, &transaction, nil, existingUser)
	})

	s.Run("target wallet not found", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			ToWalletID:   uuid.New(),
			FromWalletID: createdFromWallet.WalletID,
			Amount:       50.0,
			Currency:     "RUB",
		}
		uuidString := createdFromWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/transfer"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusNotFound, &transaction, nil, existingUser)
	})

	s.Run("source wallet not found", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			ToWalletID:   createdToWallet.WalletID,
			FromWalletID: uuid.New(),
			Amount:       50.0,
			Currency:     "RUB",
		}
		uuidString := createdFromWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/transfer"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusNotFound, &transaction, nil, existingUser)
	})

	s.Run("user does not own the source wallet", func() {
		transaction := models.Transaction{
			ID:           uuid.New(),
			ToWalletID:   createdToWallet.WalletID,
			FromWalletID: createdFromWallet.WalletID,
			Amount:       45.0,
			Currency:     "RUB",
		}

		otherUser := models.User{
			UserID: uuid.New(),
		}

		err = s.db.UpsertUser(context.Background(), otherUser)
		s.Require().NoError(err)

		uuidString := createdFromWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/transfer"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusNotFound, &transaction, nil, otherUser)
	})

	s.Run("successful transfer of funds from fx wallet", func() {
		uuidString = createdFromWalletFX.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &createdFromWalletFX, existingUser)

		uuidString = createdToWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &createdToWallet, existingUser)

		transaction := models.Transaction{
			ID:           uuid.New(),
			ToWalletID:   createdToWallet.WalletID,
			FromWalletID: createdFromWalletFX.WalletID,
			Amount:       40.0,
			Currency:     "USD",
		}

		currency := Currency{Name: transaction.Currency, Value: exchangeRatesToRub[transaction.Currency] / exchangeRatesToRub[createdToWallet.Currency]}

		uuidString := createdFromWalletFX.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/transfer"

		s.sendRequest(http.MethodPut, walletIDPath, http.StatusOK, &transaction, nil, existingUser)

		var updatedWalletFX models.Wallet

		uuidString = createdFromWalletFX.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &updatedWalletFX, existingUser)

		expectedBalanceFrom := createdFromWalletFX.Balance - transaction.Amount

		s.Require().True(math.Abs(updatedWalletFX.Balance-expectedBalanceFrom) < epsilon)

		var updatedWalletTo models.Wallet

		uuidString = createdToWallet.WalletID.String()
		walletIDPath = walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &updatedWalletTo, existingUser)

		expectedBalanceTo := createdToWallet.Balance + transaction.Amount*currency.Value

		s.Require().True(math.Abs(expectedBalanceTo-updatedWalletTo.Balance) < epsilon)
	})
}

func (s *IntegrationTestSuite) TestGetTransactions() {
	err := s.db.Truncate(context.Background(), "transactions")
	s.Require().NoError(err)

	err = s.db.UpsertUser(context.Background(), existingUser)
	s.Require().NoError(err)

	walletOne := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     existingUser.UserID,
		WalletName: "FirstWallet",
		Currency:   "RUB",
	}

	walletTwo := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     existingUser.UserID,
		WalletName: "SecondWallet",
		Currency:   "USD",
	}

	walletThree := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     existingUser.UserID,
		WalletName: "ThirdWallet",
		Currency:   "RUB",
	}

	createdOne := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletOne, &createdOne, existingUser)

	createdTwo := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletTwo, &createdTwo, existingUser)

	createdThree := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletThree, &createdThree, existingUser)

	transactionOne := models.Transaction{
		ID:         uuid.New(),
		ToWalletID: createdOne.WalletID,
		Amount:     40010.0,
		Currency:   "RUB",
	}

	uuidStringOne := createdOne.WalletID.String()
	walletIDPathOne := walletPath + "/" + uuidStringOne + "/deposit"

	s.sendRequest(http.MethodPut, walletIDPathOne, http.StatusOK, &transactionOne, nil, existingUser)

	transactionTwo := models.Transaction{
		ID:         uuid.New(),
		ToWalletID: createdTwo.WalletID,
		Amount:     204.0,
		Currency:   "USD",
	}

	uuidStringTwo := createdTwo.WalletID.String()
	walletIDPathTwo := walletPath + "/" + uuidStringTwo + "/deposit"

	s.sendRequest(http.MethodPut, walletIDPathTwo, http.StatusOK, &transactionTwo, nil, existingUser)

	transactionThree := models.Transaction{
		ID:         uuid.New(),
		ToWalletID: createdThree.WalletID,
		Amount:     3000.0,
		Currency:   "RUB",
	}

	uuidStringThree := createdThree.WalletID.String()
	walletIDPathThree := walletPath + "/" + uuidStringThree + "/deposit"

	s.sendRequest(http.MethodPut, walletIDPathThree, http.StatusOK, &transactionThree, nil, existingUser)

	transactionFour := models.Transaction{
		ID:           uuid.New(),
		FromWalletID: createdOne.WalletID,
		Amount:       5000.0,
		Currency:     "RUB",
	}

	uuidStringFour := createdOne.WalletID.String()
	walletIDPathFour := walletPath + "/" + uuidStringFour + "/withdrawal"

	s.sendRequest(http.MethodPut, walletIDPathFour, http.StatusOK, &transactionFour, nil, existingUser)

	transactionFive := models.Transaction{
		ID:           uuid.New(),
		ToWalletID:   createdThree.WalletID,
		FromWalletID: createdOne.WalletID,
		Amount:       7500.0,
		Currency:     "RUB",
	}

	uuidStringFive := createdOne.WalletID.String()
	walletIDPathFive := walletPath + "/" + uuidStringFive + "/transfer"

	s.sendRequest(http.MethodPut, walletIDPathFive, http.StatusOK, &transactionFive, nil, existingUser)

	s.Run("get all transactions for walletOne", func() {
		var transactions []models.Transaction

		uuidStrng := createdOne.WalletID.String()
		walletIDPath := walletPath + "/" + uuidStrng + "/transactions"

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &transactions, existingUser)

		s.Require().Len(transactions, 3)
	})

	s.Run("sorted by transacton type with limit 2", func() {
		var transactions []models.Transaction

		uuidStrng := createdOne.WalletID.String()
		walletIDPath := walletPath + "/" + uuidStrng + "/transactions" + "?sorting=transaction_type&limit=2"

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &transactions, existingUser)

		s.Require().Len(transactions, 2)
		s.Require().Equal(transactions[0].ToWalletID, transactionOne.ToWalletID)
		s.Require().Equal(transactions[1].FromWalletID, transactionFive.FromWalletID)
	})

	s.Run("sorted by transacton type with limit 2 and offset 1", func() {
		var transactions []models.Transaction

		uuidStrng := createdOne.WalletID.String()
		walletIDPath := walletPath + "/" + uuidStrng + "/transactions" + "?sorting=transaction_type&limit=2&offset=1"

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &transactions, existingUser)

		s.Require().Len(transactions, 2)
		s.Require().Equal(transactions[0].FromWalletID, transactionFive.FromWalletID)
		s.Require().Equal(transactions[1].FromWalletID, transactionFour.FromWalletID)
	})

	s.Run("sorted by transaction type with limit 2 and offset 1, descending true", func() {
		var transactions []models.Transaction

		uuidStrng := createdOne.WalletID.String()
		walletIDPath := walletPath + "/" + uuidStrng + "/transactions" + "?sorting=transaction_type&limit=2&offset=1&descending=true"

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &transactions, existingUser)

		s.Require().Len(transactions, 2)
		s.Require().Equal(transactions[0].FromWalletID, transactionFive.FromWalletID)
		s.Require().Equal(transactions[1].ToWalletID, transactionOne.ToWalletID)
	})

	s.Run("user does not own any wallets", func() {
		otherUser := models.User{
			UserID: uuid.New(),
		}

		err := s.db.UpsertUser(context.Background(), otherUser)
		s.Require().NoError(err)

		uuidString := createdOne.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString + "/transactions"

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, nil, nil, otherUser)
	})
}
