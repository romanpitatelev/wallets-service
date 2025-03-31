//nolint:testpackage
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

	"github.com/golang-jwt/jwt/v5"
	"github.com/romanpitatelev/wallets-service/internal/broker"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/romanpitatelev/wallets-service/internal/rest"
	"github.com/romanpitatelev/wallets-service/internal/service"
	"github.com/romanpitatelev/wallets-service/internal/store"
	xrclient "github.com/romanpitatelev/wallets-service/internal/xr/xr-client"
	xrserver "github.com/romanpitatelev/wallets-service/internal/xr/xr-server"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/suite"
)

const (
	pgDSN        = "postgresql://postgres:my_pass@localhost:5432/wallets_db"
	port         = 5003
	walletPath   = `/api/v1/wallets`
	xrPort       = 2607
	xrAddress    = "http://localhost:2607"
	kafkaAddress = "localhost:9094"
)

type IntegrationTestSuite struct {
	suite.Suite
	cancelFunc context.CancelFunc
	db         *store.DataStore
	service    *service.Service
	server     *rest.Server
	xrServer   *xrserver.Server
	client     *xrclient.Client
	txProducer *broker.Producer
}

func (s *IntegrationTestSuite) SetupSuite() {
	log.Debug().Msg("starting SetupSuite ...")

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel

	var err error

	s.db, err = store.New(ctx, store.Config{Dsn: pgDSN})
	s.Require().NoError(err)

	log.Debug().Msg("starting new db ...")

	err = s.db.Migrate(migrate.Up)
	s.Require().NoError(err)

	log.Debug().Msg("migrations are ready")

	log.Debug().Msg("starting new producer ...")

	time.Sleep(5 * time.Second)

	s.txProducer, err = broker.NewProducer(broker.ProducerConfig{Addr: kafkaAddress})
	s.Require().NoError(err)

	s.xrServer = xrserver.New(xrPort)

	log.Debug().Msg("xr server is compiled")

	//nolint:testifylint
	go func() {
		err := s.xrServer.Run(ctx)
		s.Require().NoError(err)
	}()

	s.client = xrclient.New(xrclient.Config{ServerAddress: xrAddress})

	log.Debug().Msg("xr client is ready")

	s.service = service.New(
		service.Config{
			StaleWalletDuration: 0,
			PerformCheckPeriod:  0,
		},
		s.db,
		s.client,
		s.txProducer,
	)

	s.server = rest.New(rest.Config{Port: port}, s.service, rest.GetPublicKey())

	//nolint:testifylint
	go func() {
		err = s.server.Run(ctx)
		s.Require().NoError(err)
	}()

	time.Sleep(20 * time.Second)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.cancelFunc()
}

func (s *IntegrationTestSuite) TearDownTest() {
	err := s.db.Truncate(context.Background(), "wallets", "users")
	s.Require().NoError(err)
}

func TestIntegrationSetupSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) sendRequest(method, path string, status int, entity, result any, user models.User) {
	body, err := json.Marshal(entity)
	s.Require().NoError(err)

	requestURL := fmt.Sprintf("http://localhost:%d%s", port, path)
	s.T().Logf("Sending request to %s", requestURL)

	request, err := http.NewRequestWithContext(context.Background(), method,
		fmt.Sprintf("http://localhost:%d%s", port, path), bytes.NewReader(body))
	s.Require().NoError(err, "fail to create request")

	token := s.getToken(user)
	request.Header.Set("Authorization", "Bearer "+token)

	client := http.Client{}

	response, err := client.Do(request)
	s.Require().NoError(err, "fail to execute request")

	s.Require().NotNil(response, "response object is nil")

	defer func() {
		err = response.Body.Close()
		s.Require().NoError(err)
	}()

	s.T().Logf("Response Status Code: %d", response.StatusCode)

	if status != response.StatusCode {
		responseBody, err := io.ReadAll(response.Body)
		s.Require().NoError(err)

		s.T().Logf("Response Body: %s", string(responseBody))

		s.Require().Equal(status, response.StatusCode, "unexpected status code")

		return
	}

	if result == nil {
		return
	}

	err = json.NewDecoder(response.Body).Decode(result)
	s.Require().NoError(err)

}

func (s *IntegrationTestSuite) getToken(user models.User) string {
	claims := models.Claims{
		UserID: user.UserID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	privateKey, err := readPrivateKey()
	s.Require().NoError(err)

	token, err := generateToken(&claims, privateKey)
	s.Require().NoError(err)

	return token
}
