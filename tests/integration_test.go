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
	"github.com/romanpitatelev/wallets-service/internal/controller/rest"
	transactionshandler "github.com/romanpitatelev/wallets-service/internal/controller/rest/transactions-handler"
	walletshandler "github.com/romanpitatelev/wallets-service/internal/controller/rest/wallets-handler"
	"github.com/romanpitatelev/wallets-service/internal/entity"
	"github.com/romanpitatelev/wallets-service/internal/repository/producer"
	"github.com/romanpitatelev/wallets-service/internal/repository/store"
	transactionsrepo "github.com/romanpitatelev/wallets-service/internal/repository/transactions-repo"
	walletsrepo "github.com/romanpitatelev/wallets-service/internal/repository/wallets-repo"
	xrgrpcclient "github.com/romanpitatelev/wallets-service/internal/repository/xr-grpc-client"
	transactionsservice "github.com/romanpitatelev/wallets-service/internal/usecase/transactions-service"
	walletsservice "github.com/romanpitatelev/wallets-service/internal/usecase/wallets-service"
	xrserver "github.com/romanpitatelev/wallets-service/internal/xr/xr-http/xr-server"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/suite"
)

const (
	pgDSN         = "postgresql://postgres:my_pass@localhost:5432/wallets_db"
	port          = 5003
	walletPath    = "/api/v1/wallets"
	xrPort        = 2607
	xrAddress     = "http://localhost:2607"
	xrgRPCAddress = "http://localhost:2608"
	kafkaAddress  = "localhost:9094"
)

type IntegrationTestSuite struct {
	suite.Suite
	cancelFunc          context.CancelFunc
	db                  *store.DataStore
	walletsrepo         *walletsrepo.Repo
	transactionsrepo    *transactionsrepo.Repo
	walletsservice      *walletsservice.Service
	transactionsservice *transactionsservice.Service
	walletshandler      *walletshandler.Handler
	transactionshandler *transactionshandler.Handler
	server              *rest.Server
	xrServer            *xrserver.Server
	xrRepo              *xrgrpcclient.Client
	txProducer          *producer.Producer
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

	s.walletsrepo = walletsrepo.New(s.db)
	s.transactionsrepo = transactionsrepo.New(s.db, s.walletsrepo)

	log.Debug().Msg("starting new producer ...")

	time.Sleep(5 * time.Second)

	s.txProducer, err = producer.New(producer.ProducerConfig{Addr: kafkaAddress})
	s.Require().NoError(err)

	s.xrServer = xrserver.New(xrPort)

	log.Debug().Msg("xr server is ready")

	//nolint:testifylint
	go func() {
		err := s.xrServer.Run(ctx)
		s.Require().NoError(err)
	}()

	s.xrRepo, err = xrgrpcclient.New(xrgrpcclient.Config{Host: xrgRPCAddress})
	s.Require().NoError(err)

	log.Debug().Msg("xr grpc client is ready")

	s.walletsservice = walletsservice.New(
		walletsservice.Config{
			StaleWalletDuration: 0,
			PerformCheckPeriod:  0,
		},
		s.walletsrepo,
		s.xrRepo,
		s.db,
	)

	s.transactionsservice = transactionsservice.New(
		s.walletsrepo,
		s.transactionsrepo,
		s.xrRepo,
		s.db,
		s.txProducer,
	)

	s.walletshandler = walletshandler.New(s.walletsservice)
	s.transactionshandler = transactionshandler.New(s.transactionsservice)

	s.server = rest.New(rest.Config{Port: port}, s.walletshandler, s.transactionshandler, rest.GetPublicKey())

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
	err := s.db.Truncate(context.Background(), "transactions", "wallets", "users")
	s.Require().NoError(err)
}

func TestIntegrationSetupSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) sendRequest(method, path string, status int, entity, result any, user entity.User) {
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

func (s *IntegrationTestSuite) getToken(user entity.User) string {
	claims := entity.Claims{
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
