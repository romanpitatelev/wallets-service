//nolint:testpackage
package tests

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
)

func (s *IntegrationTestSuite) TestCreateWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testWalletPost",
		Currency:   "RSD",
	}

	s.Run("created successfully", func() {
		var createdWallet models.Wallet

		s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet)

		s.Require().Equal(wallet.WalletID, createdWallet.WalletID)
		s.Require().Equal(wallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(0.0, createdWallet.Balance)
		s.Require().Equal(wallet.Currency, createdWallet.Currency)
	})
}

func (s *IntegrationTestSuite) TestGetWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testWalletGet",
		Balance:    200.0,
		Currency:   "CHF",
	}

	var createdWallet models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet)

	s.T().Logf("Created Wallet: %+v", createdWallet)

	s.Run("get wallet successful", func() {
		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		var obtainedWallet models.Wallet

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &obtainedWallet)

		s.T().Logf("Wallet obtained: %+v", obtainedWallet)

		s.Require().Equal(createdWallet.WalletID, obtainedWallet.WalletID)
		s.Require().Equal(createdWallet.WalletName, obtainedWallet.WalletName)
		s.Require().Equal(0.0, obtainedWallet.Balance)
		s.Require().Equal(createdWallet.Currency, obtainedWallet.Currency)
	})

	s.Run("wallet not found", func() {
		walletIDNonExistent := uuid.New().String()
		walletIDPath := walletPath + "/" + walletIDNonExistent

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, nil, nil)
	})
}

func (s *IntegrationTestSuite) TestUpdateWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testWalletUpdate",
		Balance:    300.0,
		Currency:   "RUB",
	}

	var createdWallet models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet)

	s.Run("wallet not found", func() {
		walletIDNonExistent := uuid.New().String()
		walletIDPath := walletPath + "/" + walletIDNonExistent

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, nil, nil)
	})

	s.Run("name updated successfully", func() {
		updatedWallet := models.WalletUpdate{
			WalletName: "updatedWalletName",
			Currency:   createdWallet.Currency,
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet)

		s.Require().Equal(updatedWallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(updatedWallet.Currency, createdWallet.Currency)
	})

	s.Run("currency updated successfully", func() {
		updatedWallet := models.WalletUpdate{
			WalletName: createdWallet.WalletName,
			Currency:   "CNY",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet)

		s.Require().Equal(updatedWallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(updatedWallet.Currency, createdWallet.Currency)
	})

	s.Run("nothing to update", func() {
		updatedWallet := models.WalletUpdate{
			WalletName: createdWallet.WalletName,
			Currency:   createdWallet.Currency,
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		var updateWalletAnother models.Wallet

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &updateWalletAnother)

		s.Require().Equal(updatedWallet.WalletName, updateWalletAnother.WalletName)
		s.Require().Equal(updatedWallet.Currency, updateWalletAnother.Currency)
	})
}

func (s *IntegrationTestSuite) TestDeleteWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testWalletDelete",
		Balance:    0.0,
		Currency:   "RUB",
		Active:     true,
	}

	var createdWallet models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet)

	s.Run("wallet deletion completed successfully", func() {
		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodDelete, walletIDPath, http.StatusNoContent, nil, nil)
	})

	s.Run("wallet not found", func() {
		walletIDNonExistent := uuid.New().String()
		walletIDPath := walletPath + "/" + walletIDNonExistent

		s.sendRequest(http.MethodDelete, walletIDPath, http.StatusNotFound, nil, nil)
	})

	s.Run("balance is non-zero", func() {
		walletNonZero := models.Wallet{
			WalletID:   uuid.New(),
			WalletName: "testDeleteNonZeroBalanceWallet",
			Balance:    0.0,
			Currency:   "USD",
			Active:     true,
		}

		var createdWalletNonZero models.Wallet

		s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletNonZero, &createdWalletNonZero)

		err := s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2`,
			259.0, createdWalletNonZero.WalletID)
		s.Require().NoError(err)

		uuidString := createdWalletNonZero.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodDelete, walletIDPath, http.StatusBadRequest, nil, nil)

		var obtainedWallet models.Wallet

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &obtainedWallet)

		s.Require().True(obtainedWallet.Active)
		s.Require().Nil(obtainedWallet.DeletedAt)
	})
}

func (s *IntegrationTestSuite) TestGetWallets() {
	err := s.db.Truncate(context.Background(), "wallets")
	s.Require().NoError(err)

	var arrWallets []models.Wallet

	walletOne := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "FirstWallet",
		Currency:   "RUB",
	}
	arrWallets = append(arrWallets, walletOne)

	walletTwo := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "SecondWallet",
		Currency:   "TRY",
	}
	arrWallets = append(arrWallets, walletTwo)

	walletThree := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "ThirdWallet",
		Currency:   "CNY",
	}
	arrWallets = append(arrWallets, walletThree)

	walletFour := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "FourthWallet",
		Currency:   "HUF",
	}
	arrWallets = append(arrWallets, walletFour)

	walletFive := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "FifthWallet",
		Currency:   "KZT",
	}
	arrWallets = append(arrWallets, walletFive)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2`,
		259.0, walletOne.WalletID)
	s.Require().NoError(err)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2`,
		359.0, walletTwo.WalletID)
	s.Require().NoError(err)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2`,
		459.0, walletThree.WalletID)
	s.Require().NoError(err)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2`,
		559.0, walletFour.WalletID)
	s.Require().NoError(err)

	err = s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2`,
		659.0, walletFive.WalletID)
	s.Require().NoError(err)

	createdOne := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletOne, &createdOne)

	createdTwo := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletTwo, &createdTwo)

	createdThree := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletThree, &createdThree)

	createdFour := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletFour, &createdFour)

	createdFive := models.Wallet{}
	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &walletFive, &createdFive)

	s.Run("read successfully", func() {
		var wallets []models.Wallet

		s.sendRequest(http.MethodGet, walletPath, http.StatusOK, nil, &wallets)

		s.Require().Len(wallets, len(arrWallets))
	})

	s.Run("sorted by name with limit 2", func() {
		var wallets []models.Wallet

		someWalletPath := walletPath + "?sorting=wallet_name&limit=2"

		s.sendRequest(http.MethodGet, someWalletPath, http.StatusOK, nil, &wallets)

		s.Require().Len(wallets, 2)
		s.Require().Equal(wallets[0].Balance, walletFive.Balance)
		s.Require().Equal(wallets[1].WalletID, walletOne.WalletID)
	})

	s.Run("sorted by name with limit 2 and offset 2", func() {
		var wallets []models.Wallet

		someWalletPath := walletPath + "?sorting=wallet_name&limit=2&offset=2"

		s.sendRequest(http.MethodGet, someWalletPath, http.StatusOK, nil, &wallets)

		s.Require().Len(wallets, 2)
		s.Require().Equal(wallets[0].Balance, walletFour.Balance)
		s.Require().Equal(wallets[1].WalletID, walletTwo.WalletID)
	})

	s.Run("sorted by name with limit 2 and offset 2", func() {
		var wallets []models.Wallet

		someWalletPath := walletPath + "?sorting=wallet_name&limit=2&offset=2"

		s.sendRequest(http.MethodGet, someWalletPath, http.StatusOK, nil, &wallets)

		s.Require().Len(wallets, 2)
		s.Require().Equal(wallets[0].Currency, walletFour.Currency)
		s.Require().Equal(wallets[1].WalletID, walletTwo.WalletID)
	})

	s.Run("sorted by name with limit 2 and offset 2, descending true", func() {
		var wallets []models.Wallet

		someWalletPath := walletPath + "?sorting=wallet_name&limit=2&offset=2&descending=true"

		s.sendRequest(http.MethodGet, someWalletPath, http.StatusOK, nil, &wallets)

		s.Require().Len(wallets, 2)
		s.Require().Equal(wallets[0].Balance, walletFour.Balance)
		s.Require().Equal(wallets[1].WalletName, walletOne.WalletName)
	})
}
