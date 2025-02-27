package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/romanpitatelev/wallets-service/configs"
	"github.com/romanpitatelev/wallets-service/internal/rest"
	"github.com/romanpitatelev/wallets-service/internal/service"
	"github.com/romanpitatelev/wallets-service/internal/store"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/suite"
)

const (
	port       = 8081
	walletPath = `/api/v1/wallets`
)

type IntegrationTestSuite struct {
	suite.Suite
	cancelFunc context.CancelFunc
	db         *store.DataStore
	service    *service.Service
	server     *rest.Server
}

func (its *IntegrationTestSuite) SetupSuite() {
	ctx, cancel := context.WithCancel(context.Background())
	its.cancelFunc = cancel

	conf := &configs.Config{
		BindAddress:      ":8081",
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresDatabase: "wallets_db",
		PostgresUser:     "postgres",
		PostgresPassword: "my_pass",
	}

	var err error

	//	conf := configs.NewConfig()

	its.db, err = store.New(ctx, conf)
	its.Require().NoError(err)

	err = its.db.Migrate(migrate.Up)
	its.Require().NoError(err)

	its.service = service.New(its.db)

	its.server = rest.New(its.service)

	go func() {
		err = its.server.Run(ctx)
		its.Require().NoError(err)

	}()

	time.Sleep(50 * time.Millisecond)

}

func (its *IntegrationTestSuite) TearDownSuite() {
	its.cancelFunc()
}

func TestIntegrationSetupSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (its *IntegrationTestSuite) sendRequest(method, path string, status int, entity, result any) {
	body, err := json.Marshal(entity)
	its.Require().NoError(err)

	request, err := http.NewRequestWithContext(context.Background(), method,
		fmt.Sprintf("http://localhost:%d%s", port, path), bytes.NewReader(body))
	its.Require().NoError(err)

	client := http.Client{}

	response, err := client.Do(request)
	its.Require().NoError(err)

	defer func() {
		err = response.Body.Close()
		its.Require().NoError(err)
	}()

	if status != response.StatusCode {
		// TODO: переосмыслить
		responseBody, err := io.ReadAll(response.Body)
		its.Require().NoError(err)

		its.T().Log(responseBody)

		return
	}

	if result == nil {
		return
	}

	err = json.NewDecoder(response.Body).Decode(result)
	its.Require().NoError(err)
}
