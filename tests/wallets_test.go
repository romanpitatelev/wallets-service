package tests

import (
	"context"
	"net/http"
	"time"

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
		CreatedAt:  time.Now(),
	}

	var createdWallet models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet)

	s.Run("get wallet successful", func() {
		uuidString := uuid.UUID(createdWallet.WalletID).String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &createdWallet)

		s.Require().Equal(wallet.WalletID, createdWallet.WalletID)
		s.Require().Equal(wallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(wallet.Balance, createdWallet.Balance)
		s.Require().Equal(wallet.Currency, createdWallet.Currency)
		s.Require().Equal(wallet.CreatedAt, createdWallet.CreatedAt)

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
		updatedWallet := createdWallet
		uuidString := uuid.UUID(createdWallet.WalletID).String()
		walletIDPath := walletPath + "/" + uuidString

		s.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet)

		s.Require().Equal(updatedWallet.WalletName, createdWallet.WalletName)
		s.Require().Equal(updatedWallet.Currency, createdWallet.Currency)
	})
}

func (s *IntegrationTestSuite) TestDeleteWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		WalletName: "testWalletDelete",
		Balance:    534.0,
		Currency:   "RUB",
	}

	var createdWallet models.Wallet

	s.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet)

	s.Run("wallet deletion completed successfully", func() {
		uuidString := createdWallet.WalletID.String()
		walletIDPath := walletPath + "/" + uuidString

		walletBeforeDeletion, err := s.service.GetWallet(context.Background(), createdWallet.WalletID)

		s.Require().NoError(err, "failed to obtain wallet before deletion")
		s.Require().Nil(walletBeforeDeletion.DeletedAt, "deletedAt should be nil before deletion")

		s.sendRequest(http.MethodDelete, walletIDPath, http.StatusNoContent, nil, nil)

		walletAfterDeletion, err := s.service.GetWallet(context.Background(), createdWallet.WalletID)
		s.Require().NoError(err, "failed to obtain wallet after deletion")

		s.Require().NotNil(walletAfterDeletion.DeletedAt, "deletedAt should not be nil after deletion")
		s.Require().True(walletAfterDeletion.DeletedAt.After(walletBeforeDeletion.CreatedAt), "deletedAt should be after createdAt")
	})

	s.Run("wallet not found", func() {
		walletIDNonExistent := uuid.New().String()
		walletIDPath := walletPath + "/" + walletIDNonExistent

		s.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, nil, nil)
	})
}
