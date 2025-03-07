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

		var updateWalletAnother models.Wallet

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &updateWalletAnother)

		s.Require().Equal(updatedWallet.WalletName, updateWalletAnother.WalletName)
		s.Require().Equal(updatedWallet.Currency, updateWalletAnother.Currency)
	})

	s.Run("currency updated successfully", func() {
		updatedWallet := models.WalletUpdate{
			WalletName: createdWallet.WalletName,
			Currency:   "CNY",
		}

		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		var updateWalletAnother models.Wallet

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &updateWalletAnother)

		s.Require().Equal(updatedWallet.WalletName, updateWalletAnother.WalletName)
		s.Require().Equal(updatedWallet.Currency, updateWalletAnother.Currency)
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
	var arrWallets []models.Wallet

	walletOne := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testWalletOne",
		Currency:   "RUB",
	}
	arrWallets = append(arrWallets, walletOne)

	walletTwo := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testWalletTwo",
		Currency:   "TRY",
	}
	arrWallets = append(arrWallets, walletTwo)

	walletThree := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testWalletThre",
		Currency:   "CNY",
	}
	arrWallets = append(arrWallets, walletThree)

	createWallet := func(w models.Wallet) models.Wallet {
		var createdWallet models.Wallet
		s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &w, &createdWallet)
		return createdWallet
	}

	for i := 0; i < len(arrWallets); i++ {
		createdWallet := createWallet(arrWallets[i])
		err := s.db.Exec(context.Background(), `UPDATE wallets SET balance = $1 WHERE wallet_id = $2`,
			259.0+float64(i*100), createdWallet.WalletID)
		s.Require().NoError(err)
	}

	s.Run("read successfully with pagination", func() {
		var response models.GetWalletsResponse

		s.sendRequest(http.MethodGet, walletPath, http.StatusOK, nil, &response)

		s.Require().Len(response.Wallets, len(arrWallets))
		s.Require().Equal(len(arrWallets), response.TotalCount)
	})

}
