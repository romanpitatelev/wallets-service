//nolint:testpackage
package tests

import (
	"context"
	"math"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
)

const (
	balanceTest = 9000.0
	epsilon     = 0.0001
)

type Currency struct {
	Name  string
	Value float64
}

//nolint:gochecknoglobals
var existingUser = models.User{
	UserID: uuid.New(),
}

func (s *IntegrationTestSuite) TestCreateWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     uuid.New(),
		WalletName: "testWalletPost",
		Currency:   "RSD",
	}

	s.Run("user not found", func() {
		s.sendRequest(http.MethodPost, walletPath, http.StatusNotFound, &wallet, nil, existingUser)
	})

	s.Run("created successfully", func() {
		err := s.db.UpsertUser(context.Background(), existingUser)
		s.Require().NoError(err)

		wallet.UserID = existingUser.UserID

		var createdWallet models.Wallet

		s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet, existingUser)

		s.Require().Equal(wallet.WalletID, createdWallet.WalletID)
		s.Require().Equal(wallet.UserID, createdWallet.UserID)
		s.Require().Equal(wallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(0.0, createdWallet.Balance)
		s.Require().Equal(wallet.Currency, createdWallet.Currency)
	})

	s.Run("wallet does not belong to the user", func() {
		err := s.db.UpsertUser(context.Background(), existingUser)
		s.Require().NoError(err)

		otherUser := models.User{
			UserID: uuid.New(),
		}

		err = s.db.UpsertUser(context.Background(), otherUser)
		s.Require().NoError(err)

		wallet.UserID = otherUser.UserID

		s.sendRequest(http.MethodPost, walletPath, http.StatusNotFound, &wallet, nil, existingUser)
	})
}

func (s *IntegrationTestSuite) TestGetWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     existingUser.UserID,
		WalletName: "testWalletGet",
		Balance:    200.0,
		Currency:   "CHF",
	}

	err := s.db.UpsertUser(context.Background(), existingUser)
	s.Require().NoError(err)

	var createdWallet models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet, existingUser)

	s.Run("user not found", func() {
		nonExistentUser := models.User{
			UserID: uuid.New(),
		}

		uuidString := wallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, nil, nil, nonExistentUser)
	})

	s.Run("get wallet successful", func() {
		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &createdWallet, existingUser)

		s.Require().Equal(wallet.WalletID, createdWallet.WalletID)
		s.Require().Equal(wallet.UserID, createdWallet.UserID)
		s.Require().Equal(wallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(0.0, createdWallet.Balance)
		s.Require().Equal(wallet.Currency, createdWallet.Currency)
	})

	s.Run("wallet not found", func() {
		walletIDNonExistent := uuid.New().String()
		walletIDPath := walletPath + "/" + walletIDNonExistent

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, nil, nil, existingUser)
	})

	s.Run("wallet does not belong to the user", func() {
		otherUser := models.User{
			UserID: uuid.New(),
		}

		err := s.db.UpsertUser(context.Background(), otherUser)
		s.Require().NoError(err)

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, nil, nil, otherUser)
	})
}

func (s *IntegrationTestSuite) TestUpdateWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testWalletUpdate",
		Balance:    300.0,
		Currency:   "RUB",
	}

	err := s.db.UpsertUser(context.Background(), existingUser)
	s.Require().NoError(err)

	wallet.UserID = existingUser.UserID

	var createdWallet models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet, existingUser)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2`,
		balanceTest, createdWallet.WalletID)
	s.Require().NoError(err)

	s.Run("user not found", func() {
		nonExistentUser := models.User{
			UserID: uuid.New(),
		}

		uuidString := wallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusNotFound, &wallet, nil, nonExistentUser)
	})

	s.Run("wallet not found", func() {
		walletIDNonExistent := uuid.New().String()
		walletIDPath := walletPath + "/" + walletIDNonExistent

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, nil, nil, existingUser)
	})

	s.Run("name updated successfully", func() {
		updatedWallet := models.WalletUpdate{
			WalletName: "updatedWalletName",
			UserID:     createdWallet.UserID,
			Currency:   createdWallet.Currency,
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet, existingUser)

		s.Require().Equal(updatedWallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(updatedWallet.UserID, createdWallet.UserID)
		s.Require().Equal(updatedWallet.Currency, createdWallet.Currency)
		s.Require().Equal(createdWallet.Balance, balanceTest)
	})

	s.Run("currency updated successfully", func() {
		cny := Currency{Name: "CNY", Value: 12.3}

		updatedWallet := models.WalletUpdate{
			WalletName: createdWallet.WalletName,
			UserID:     createdWallet.UserID,
			Currency:   cny.Name,
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet, existingUser)

		expectedBalance := balanceTest / cny.Value

		s.Require().Equal(updatedWallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(updatedWallet.UserID, createdWallet.UserID)
		s.Require().Equal(updatedWallet.Currency, createdWallet.Currency)
		s.Require().True(math.Abs(createdWallet.Balance-expectedBalance) < epsilon)
	})

	s.Run("lowercase currency updated successfully", func() {
		rsd := Currency{Name: "rsd", Value: 0.83}

		updatedWallet := models.WalletUpdate{
			WalletName: createdWallet.WalletName,
			UserID:     createdWallet.UserID,
			Currency:   rsd.Name,
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet, existingUser)

		expectedBalance := balanceTest / rsd.Value

		s.Require().Equal(updatedWallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(updatedWallet.UserID, createdWallet.UserID)
		s.Require().Equal(strings.ToUpper(updatedWallet.Currency), createdWallet.Currency)
		s.Require().True(math.Abs(createdWallet.Balance-expectedBalance) < epsilon)
	})

	s.Run("unprocessible currency", func() {
		updatedWallet := models.WalletUpdate{
			WalletName: createdWallet.WalletName,
			UserID:     createdWallet.UserID,
			Currency:   "NIO",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusUnprocessableEntity, &updatedWallet, nil, existingUser)
	})

	s.Run("nothing to update", func() {
		updatedWallet := models.WalletUpdate{
			WalletName: createdWallet.WalletName,
			UserID:     createdWallet.UserID,
			Currency:   createdWallet.Currency,
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet, existingUser)

		s.Require().Equal(updatedWallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(updatedWallet.Currency, createdWallet.Currency)
	})

	s.Run("wallet belongs to another user", func() {
		otherUser := models.User{
			UserID: uuid.New(),
		}

		err := s.db.UpsertUser(context.Background(), otherUser)
		s.Require().NoError(err)

		updatedWallet := models.Wallet{
			WalletID:   uuid.New(),
			UserID:     otherUser.UserID,
			WalletName: "updatedWallet",
			Currency:   "CHF",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusNotFound, &updatedWallet, nil, otherUser)
	})
}

func (s *IntegrationTestSuite) TestDeleteWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     uuid.New(),
		WalletName: "testWalletDelete",
		Balance:    0.0,
		Currency:   "RUB",
		Active:     true,
	}

	err := s.db.UpsertUser(context.Background(), existingUser)
	s.Require().NoError(err)

	wallet.UserID = existingUser.UserID

	var createdWallet models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet, existingUser)

	s.Run("user not found", func() {
		nonExistentUser := models.User{
			UserID: uuid.New(),
		}

		uuidString := wallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodDelete, walletIDPath, http.StatusNotFound, &wallet, nil, nonExistentUser)
	})

	s.Run("wallet deletion completed successfully", func() {
		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodDelete, walletIDPath, http.StatusNoContent, nil, nil, existingUser)
	})

	s.Run("wallet not found", func() {
		walletIDNonExistent := uuid.New().String()
		walletIDPath := walletPath + "/" + walletIDNonExistent

		s.sendRequest(http.MethodDelete, walletIDPath, http.StatusNotFound, nil, nil, existingUser)
	})

	s.Run("balance is non-zero", func() {
		walletNonZero := models.Wallet{
			WalletID:   uuid.New(),
			WalletName: "testDeleteNonZeroBalanceWallet",
			Balance:    0.0,
			Currency:   "USD",
			Active:     true,
		}

		err := s.db.UpsertUser(context.Background(), existingUser)
		s.Require().NoError(err)

		walletNonZero.UserID = existingUser.UserID

		var createdWalletNonZero models.Wallet

		s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletNonZero, &createdWalletNonZero, existingUser)

		err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2`,
			259.0, createdWalletNonZero.WalletID)
		s.Require().NoError(err)

		uuidString := createdWalletNonZero.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodDelete, walletIDPath, http.StatusBadRequest, nil, nil, existingUser)

		var obtainedWallet models.Wallet

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &obtainedWallet, existingUser)

		s.Require().True(obtainedWallet.Active)
		s.Require().Nil(obtainedWallet.DeletedAt)
	})

	s.Run("wallet belongs to another user", func() {
		otherUser := models.User{
			UserID: uuid.New(),
		}

		err := s.db.UpsertUser(context.Background(), otherUser)
		s.Require().NoError(err)

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodDelete, walletIDPath, http.StatusNotFound, nil, nil, otherUser)
	})
}

func (s *IntegrationTestSuite) TestGetWallets() {
	err := s.db.Truncate(context.Background(), "wallets")
	s.Require().NoError(err)

	err = s.db.UpsertUser(context.Background(), existingUser)
	s.Require().NoError(err)

	var arrWallets []models.Wallet

	walletOne := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     existingUser.UserID,
		WalletName: "FirstWallet",
		Currency:   "RUB",
	}
	arrWallets = append(arrWallets, walletOne)

	walletTwo := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     existingUser.UserID,
		WalletName: "SecondWallet",
		Currency:   "TRY",
	}
	arrWallets = append(arrWallets, walletTwo)

	walletThree := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     existingUser.UserID,
		WalletName: "ThirdWallet",
		Currency:   "CNY",
	}
	arrWallets = append(arrWallets, walletThree)

	walletFour := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     existingUser.UserID,
		WalletName: "FourthWallet",
		Currency:   "HUF",
	}
	arrWallets = append(arrWallets, walletFour)

	walletFive := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     existingUser.UserID,
		WalletName: "FifthWallet",
		Currency:   "KZT",
	}
	arrWallets = append(arrWallets, walletFive)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2 AND user_id = $3`,
		259.0, walletOne.WalletID, walletOne.UserID)
	s.Require().NoError(err)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2 AND user_id = $3`,
		359.0, walletTwo.WalletID, walletTwo.UserID)
	s.Require().NoError(err)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2 AND user_id = $3`,
		459.0, walletThree.WalletID, walletThree.UserID)
	s.Require().NoError(err)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2 AND user_id = $3`,
		559.0, walletFour.WalletID, walletFour.UserID)
	s.Require().NoError(err)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2 AND user_id = $3`,
		659.0, walletFive.WalletID, walletFive.UserID)
	s.Require().NoError(err)

	createdOne := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletOne, &createdOne, existingUser)

	createdTwo := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletTwo, &createdTwo, existingUser)

	createdThree := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletThree, &createdThree, existingUser)

	createdFour := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletFour, &createdFour, existingUser)

	createdFive := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletFive, &createdFive, existingUser)

	s.Run("read successfully", func() {
		var wallets []models.Wallet

		s.sendRequest(http.MethodGet, walletPath, http.StatusOK, nil, &wallets, existingUser)

		s.Require().Len(wallets, len(arrWallets))
	})

	s.Run("sorted by name with limit 2", func() {
		var wallets []models.Wallet

		someWalletPath := walletPath + "?sorting=wallet_name&limit=2"

		s.sendRequest(http.MethodGet, someWalletPath, http.StatusOK, nil, &wallets, existingUser)

		s.Require().Len(wallets, 2)
		s.Require().Equal(wallets[0].UserID, walletFive.UserID)
		s.Require().Equal(wallets[1].WalletID, walletOne.WalletID)
	})

	s.Run("sorted by name with limit 2 and offset 2", func() {
		var wallets []models.Wallet

		someWalletPath := walletPath + "?sorting=wallet_name&limit=2&offset=2"

		s.sendRequest(http.MethodGet, someWalletPath, http.StatusOK, nil, &wallets, existingUser)

		s.Require().Len(wallets, 2)
		s.Require().Equal(wallets[0].UserID, walletFour.UserID)
		s.Require().Equal(wallets[1].WalletID, walletTwo.WalletID)
	})

	s.Run("sorted by name with limit 2 and offset 2", func() {
		var wallets []models.Wallet

		someWalletPath := walletPath + "?sorting=wallet_name&limit=2&offset=2"

		s.sendRequest(http.MethodGet, someWalletPath, http.StatusOK, nil, &wallets, existingUser)

		s.Require().Len(wallets, 2)
		s.Require().Equal(wallets[0].Currency, walletFour.Currency)
		s.Require().Equal(wallets[1].WalletID, walletTwo.WalletID)
	})

	s.Run("sorted by name with limit 2 and offset 2, descending true", func() {
		var wallets []models.Wallet

		someWalletPath := walletPath + "?sorting=wallet_name&limit=2&offset=2&descending=true"

		s.sendRequest(http.MethodGet, someWalletPath, http.StatusOK, nil, &wallets, existingUser)

		s.Require().Len(wallets, 2)
		s.Require().Equal(wallets[0].Balance, walletFour.Balance)
		s.Require().Equal(wallets[1].WalletName, walletOne.WalletName)
	})

	s.Run("user does not own any wallets", func() {
		otherUser := models.User{
			UserID: uuid.New(),
		}

		err := s.db.UpsertUser(context.Background(), otherUser)
		s.Require().NoError(err)

		var wallets []models.Wallet

		s.sendRequest(http.MethodGet, walletPath, http.StatusOK, nil, &wallets, otherUser)

		s.Require().Len(wallets, 0)
	})
}
