package tests

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
)

func (its *IntegrationTestSuite) TestCreateWallet() {

	wallet := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     uuid.New(),
		WalletName: "testWalletPost",
		Balance:    100.0,
		Currency:   "RSD",
		CreatedAt:  time.Now(),
	}

	its.Run("created successfully", func() {
		createdWallet := models.Wallet{}

		its.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet)

		its.Require().Equal(wallet.WalletID, createdWallet.WalletID)
		its.Require().Equal(wallet.WalletName, createdWallet.WalletName)
		its.Require().Equal(wallet.Balance, createdWallet.Balance)
		its.Require().Equal(wallet.Currency, createdWallet.Currency)
		its.Require().Equal(wallet.CreatedAt, createdWallet.CreatedAt)
	})
}

func (its *IntegrationTestSuite) TestGetWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     uuid.New(),
		WalletName: "testWalletGet",
		Balance:    200.0,
		Currency:   "CHF",
		CreatedAt:  time.Now(),
	}

	createdWallet := models.Wallet{}

	its.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet)

	its.Run("get wallet successful", func() {
		uuidString := uuid.UUID(createdWallet.WalletID).String()
		walletIDPath := walletPath + "/" + uuidString

		its.sendRequest(http.MethodGet, walletIDPath, http.StatusOK, nil, &createdWallet)

		its.Require().Equal(wallet.WalletID, createdWallet.WalletID)
		its.Require().Equal(wallet.WalletName, createdWallet.WalletName)
		its.Require().Equal(wallet.Balance, createdWallet.Balance)
		its.Require().Equal(wallet.Currency, createdWallet.Currency)
		its.Require().Equal(wallet.CreatedAt, createdWallet.CreatedAt)

	})

	its.Run("wallet not found", func() {
		walletIDNonExistent := uuid.New().String()
		walletIDPath := walletPath + "/" + walletIDNonExistent

		its.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, &wallet, nil)
	})
}

func (its *IntegrationTestSuite) TestUpdateWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     uuid.New(),
		WalletName: "testWalletUpdate",
		Balance:    300.0,
		Currency:   "RUB",
	}

	createdWallet := models.Wallet{}

	its.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet)

	its.Run("name updated successfully", func() {
		updatedWallet := models.Wallet{
			WalletID:   createdWallet.WalletID,
			UserID:     createdWallet.UserID,
			WalletName: "updatedWalletName",
			Balance:    createdWallet.Balance,
			Currency:   createdWallet.Currency,
		}

		uuidString := uuid.UUID(createdWallet.WalletID).String()
		walletIDPath := walletPath + "/" + uuidString

		its.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet)

		its.Require().Equal(updatedWallet.WalletName, createdWallet.WalletName)
	})

	its.Run("wallet not found", func() {
		walletIDNonExistent := uuid.New().String()
		walletIDPath := walletPath + "/" + walletIDNonExistent

		its.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, nil, nil)
	})

	its.Run("balance updated successfully", func() {
		updatedWallet := models.Wallet{
			WalletID:   createdWallet.WalletID,
			UserID:     createdWallet.UserID,
			WalletName: createdWallet.WalletName,
			Balance:    900,
			Currency:   createdWallet.Currency,
		}

		uuidString := uuid.UUID(createdWallet.WalletID).String()
		walletIDPath := walletPath + "/" + uuidString

		its.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet)

		its.Require().Equal(updatedWallet.Balance, createdWallet.Balance)
	})

	its.Run("currency updated successfully", func() {
		updatedWallet := models.Wallet{
			WalletID:   createdWallet.WalletID,
			UserID:     createdWallet.UserID,
			WalletName: createdWallet.WalletName,
			Balance:    createdWallet.Balance,
			Currency:   "CNY",
		}

		uuidString := uuid.UUID(createdWallet.WalletID).String()
		walletIDPath := walletPath + "/" + uuidString

		its.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet)

		its.Require().Equal(updatedWallet.Currency, createdWallet.Currency)
	})

	its.Run("nothing to update", func() {
		updatedWallet := createdWallet
		uuidString := uuid.UUID(createdWallet.WalletID).String()
		walletIDPath := walletPath + "/" + uuidString

		its.sendRequest(http.MethodPatch, walletIDPath, http.StatusOK, &updatedWallet, &createdWallet)

		its.Require().Equal(updatedWallet.WalletName, createdWallet.WalletName)
		its.Require().Equal(updatedWallet.Currency, createdWallet.Currency)
	})

}

func (its *IntegrationTestSuite) TestDeleteWallet() {
	wallet := models.Wallet{
		WalletID:   uuid.New(),
		UserID:     uuid.New(),
		WalletName: "testWalletDelete",
		Balance:    534.0,
		Currency:   "RUB",
	}

	createdWallet := models.Wallet{}

	its.sendRequest(http.MethodPost, walletPath, http.StatusCreated, &wallet, &createdWallet)

	its.Run("wallet deletion completed successfully", func() {
		uuidString := uuid.UUID(createdWallet.WalletID).String()
		walletIDPath := walletPath + "/" + uuidString

		walletBeforeDeletion, err := its.service.GetWallet(context.Background(), createdWallet.WalletID)

		its.Require().NoError(err, "failed to obtain wallet before deletion")
		its.Require().Nil(walletBeforeDeletion.DeletedAt, "deletedAt should be nil before deletion")

		its.sendRequest(http.MethodDelete, walletIDPath, http.StatusNoContent, nil, nil)

		walletAfterDeletion, err := its.service.GetWallet(context.Background(), createdWallet.WalletID)
		its.Require().NoError(err, "failed to obtain wallet after deletion")

		its.Require().NotNil(walletAfterDeletion.DeletedAt, "deletedAt should not be nil after deletion")
		its.Require().True(walletAfterDeletion.DeletedAt.After(walletBeforeDeletion.CreatedAt), "deletedAt should be after createdAt")
	})

	its.Run("wallet not found", func() {
		walletIDNonExistent := uuid.New().String()
		walletIDPath := walletPath + "/" + walletIDNonExistent

		its.sendRequest(http.MethodGet, walletIDPath, http.StatusNotFound, nil, nil)
	})
}
