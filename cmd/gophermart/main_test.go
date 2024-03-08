package main_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dlomanov/go-diploma-tpl/config"
	"github.com/dlomanov/go-diploma-tpl/internal/app"
	"github.com/dlomanov/go-diploma-tpl/internal/infra/httpserver"
	"github.com/docker/go-connections/nat"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

const (
	startupDelay    = 3 * time.Second
	teardownTimeout = 10 * time.Second
	posgresStartup  = 5 * time.Second
)

type (
	AppSuite struct {
		suite.Suite
		appAPI     *appAPI
		logger     *zap.Logger
		pgc        *postgres.PostgresContainer
		teardown   func()
		teardownCh chan error
	}
	appAPI struct {
		client *resty.Client
	}
	userModel struct {
		login string
		pass  string
	}
	orderModel struct {
		Number     string          `json:"number"`
		Status     string          `json:"status"`
		Accrual    decimal.Decimal `json:"accrual"`
		UploadedAt time.Time       `json:"uploaded_at"`
	}
	balanceModel struct {
		Current   decimal.Decimal `json:"current"`
		Withdrawn decimal.Decimal `json:"withdrawn"`
	}
	withdrawalModel struct {
		Order       string          `json:"order"`
		Sum         decimal.Decimal `json:"sum"`
		ProcessedAt time.Time       `json:"processed_at"`
	}
)

func TestAppSuite(t *testing.T) {
	suite.Run(t, new(AppSuite))
}

func (s *AppSuite) SetupSuite() {
	logger := zaptest.NewLogger(s.T(), zaptest.Level(zap.DebugLevel))
	cfg := config.NewFromYAML()

	ctx, cancel := context.WithCancel(context.Background())
	s.logger = logger
	s.teardown = cancel
	s.teardownCh = make(chan error, 1)
	s.appAPI = newAppAPI(cfg)

	s.pgc, *cfg = createPosgres(s.T(), *cfg)

	if err := app.RunMigration(logger, cfg); err != nil {
		require.NoError(s.T(), err)
	}
	go func() {
		server := s.startAccrualServer(ctx, cfg)
		err := <-server.Notify()
		s.logger.Error("accrual server stopped", zap.Error(err))

	}()
	go func() {
		err := app.Run(ctx, logger, cfg)
		select {
		case s.teardownCh <- err:
		default:
		}
	}()

	time.Sleep(startupDelay)
}

func (s *AppSuite) TearDownSuite() {
	timeout, cancel := context.WithTimeout(context.Background(), teardownTimeout)
	defer cancel()

	s.teardown()

	if err := s.pgc.Terminate(timeout); err != nil {
		s.logger.Error("failed to terminate postgres container", zap.Error(err))
	}
	var err error
	select {
	case err = <-s.teardownCh:
		require.NoError(s.T(), err)
	case <-timeout.Done():
		s.logger.Error("teardown timeout")
	}
}

func (s *AppSuite) TestApp() {
	var (
		users = []userModel{
			{login: "user0", pass: ""},  // invalid
			{login: "user1", pass: "1"}, // valid
			{login: "user2", pass: "2"}, // valid
			{login: "user3", pass: "3"}, // not registered
		}
		orderNumber1       = "6200000000000005"
		orderNumber2       = "4242424242424242"
		invalidOrderNumber = "471629309440"
		statusCode         int
		orders             []orderModel
		withdrawals        []withdrawalModel
		balance            balanceModel
	)

	// 0 UNAUTHORIZED
	statusCode, _ = s.appAPI.getOrders(s.T())
	require.Equal(s.T(), http.StatusUnauthorized, statusCode, "0 UNAUTHORIZED: get orders should return 401, got %d", statusCode)

	statusCode = s.appAPI.createOrder(s.T(), "")
	require.Equal(s.T(), http.StatusUnauthorized, statusCode, "0 UNAUTHORIZED: create order should return 401, got %d", statusCode)

	statusCode, _ = s.appAPI.getBalance(s.T())
	require.Equal(s.T(), http.StatusUnauthorized, statusCode, "0 UNAUTHORIZED: get balance should return 401, got %d", statusCode)

	statusCode, _ = s.appAPI.getWithdrawals(s.T())
	require.Equal(s.T(), http.StatusUnauthorized, statusCode, "0 UNAUTHORIZED: get withdrawals should return 401, got %d", statusCode)

	statusCode = s.appAPI.withdraw(s.T(), "", decimal.Zero)
	require.Equal(s.T(), http.StatusUnauthorized, statusCode, "0 UNAUTHORIZED: withdraw should return 401, got %d", statusCode)

	// 1 AUTH
	// 1.1 Register endpoint
	require.Equal(s.T(), http.StatusBadRequest, s.appAPI.register(s.T(), users[0]), "1.1 AUTH: invalid user[0] registration should return 400, got %d", statusCode)
	require.Equal(s.T(), http.StatusOK, s.appAPI.register(s.T(), users[1]), "1.1 AUTH: valid user[1] registration should return 200, got %d", statusCode)
	require.Equal(s.T(), http.StatusOK, s.appAPI.register(s.T(), users[2]), "1.1 AUTH: valid user[2] registration should return 200, got %d", statusCode)
	require.Equal(s.T(), http.StatusConflict, s.appAPI.register(s.T(), users[1]), "1.1 AUTH: duplicate user registration should return 409, got %d", statusCode)

	// 1.2 Login endpoint
	require.Equal(s.T(), http.StatusBadRequest, s.appAPI.login(s.T(), users[0]), "1.2 AUTH: invalid user[0] login should return 400, got %d", statusCode)
	require.Equal(s.T(), http.StatusUnauthorized, s.appAPI.login(s.T(), users[3]), "1.2 AUTH: unknown user[3] login should return 401, got %d", statusCode)
	require.Equal(s.T(), http.StatusOK, s.appAPI.login(s.T(), users[1]), "1.2 AUTH: valid user[1] login should return 200, got %d", statusCode)

	// 2 ORDER

	// 2.0 Before order creation
	statusCode, _ = s.appAPI.getOrders(s.T())
	require.Equal(s.T(), http.StatusNoContent, statusCode, "2.0 ORDER: get orders should return 204, got %d", statusCode)

	// 2.1 Create order
	require.Equal(s.T(), http.StatusUnprocessableEntity, s.appAPI.createOrder(s.T(), ""), "2.1 ORDER: create order with empty number should return 422, got %d", statusCode)
	require.Equal(s.T(), http.StatusUnprocessableEntity, s.appAPI.createOrder(s.T(), invalidOrderNumber), "2.1 ORDER: create order with invalid number should return 422, got %d", statusCode)
	require.Equal(s.T(), http.StatusAccepted, s.appAPI.createOrder(s.T(), orderNumber1), "2.1 ORDER: create order should return 202, got %d", statusCode)
	require.Equal(s.T(), http.StatusOK, s.appAPI.createOrder(s.T(), orderNumber1), "2.1 ORDER: duplicate order creation should return 200, got %d", statusCode)
	require.Equal(s.T(), http.StatusOK, s.appAPI.login(s.T(), users[2]), "2.1 ORDER: valid user[2] login should return 200, got %d", statusCode)
	require.Equal(s.T(), http.StatusConflict, s.appAPI.createOrder(s.T(), orderNumber1), "2.1 ORDER: create order that belongs to another user should return 409, got %d", statusCode)

	// 2.2 Get orders
	statusCode, orders = s.appAPI.getOrders(s.T())
	require.Equal(s.T(), http.StatusNoContent, statusCode, "2.2 ORDER: get orders of user with no orders should return 204, got %d", statusCode)
	require.Empty(s.T(), orders, "2.2 ORDER: get orders of user with no orders should return no orders")

	require.Equal(s.T(), http.StatusOK, s.appAPI.login(s.T(), users[1]), "2.2 ORDER: valid user[1] login should return 200, got %d", statusCode)

	statusCode, orders = s.appAPI.getOrders(s.T())
	require.Equal(s.T(), http.StatusOK, statusCode, "2.2 ORDER: get orders of users with order should return 200, got %d", statusCode)
	require.Len(s.T(), orders, 1, "2.2 ORDER: get orders of users with order should return single order, got %d", len(orders))
	order := orders[0]
	require.Equal(s.T(), orderNumber1, order.Number, "2.2 ORDER: order number should be equal to orderNumber1, got %s", order.Number)
	require.NotEmpty(s.T(), order.Status, "2.2 ORDER: order status should not be empty")
	require.NotEmpty(s.T(), order.UploadedAt, "2.2 ORDER: order uploaded_at should not be empty")

	// 2.3 Get balance
	// wait for order processing
	for i := 0; i < 10; i++ {
		statusCode, balance = s.appAPI.getBalance(s.T())
		require.Equal(s.T(), http.StatusOK, statusCode, "2.3 ORDER: get balance should return 200, got %d", statusCode)
		if !balance.Current.Equal(decimal.Zero) {
			break
		}
		time.Sleep(1 * time.Second)
	}
	require.True(s.T(), balance.Current.GreaterThan(decimal.Zero), "2.3 ORDER: balance should be greater than zero, got %s", balance.Current)
	require.True(s.T(), balance.Withdrawn.Equal(decimal.Zero), "2.3 ORDER: withdrawn should be zero, got %s", balance.Withdrawn)

	// order should be processed
	statusCode, orders = s.appAPI.getOrders(s.T())
	require.Equal(s.T(), http.StatusOK, statusCode, "2.3 ORDER: get orders should return 200, got %d", statusCode)
	require.Len(s.T(), orders, 1, "2.3 ORDER: get orders should return single order, got %d", len(orders))
	order = orders[0]
	require.Equal(s.T(), orderNumber1, order.Number, "2.3 ORDER: order number should be equal to orderNumber1, got %s", order.Number)
	require.Equal(s.T(), order.Status, "PROCESSED", "2.3 ORDER: order status should be PROCESSED, got %s", order.Status)
	require.True(s.T(), order.Accrual.Equal(balance.Current), "2.3 ORDER: order accrual should be equal to balance, got %s", order.Accrual)
	require.NotEmpty(s.T(), order.UploadedAt, "order uploaded_at should not be empty")

	// 3 WITHDRAW
	balanceAmount := balance.Current

	// 3.0 Before withdraw
	statusCode, withdrawals = s.appAPI.getWithdrawals(s.T())
	require.Equal(s.T(), http.StatusNoContent, statusCode, "3.0 WITHDRAW: get withdrawals should return 204, got %d", statusCode)
	require.Empty(s.T(), withdrawals, "3.0 WITHDRAW: get withdrawals should return no withdrawals")

	// 3.1 Withdraw
	require.Equal(s.T(), http.StatusUnprocessableEntity, s.appAPI.withdraw(s.T(), order.Number, decimal.NewFromInt(-1)), "3.1 WITHDRAW: withdraw negative amount should return 422, got %d", statusCode)
	require.Equal(s.T(), http.StatusUnprocessableEntity, s.appAPI.withdraw(s.T(), order.Number, decimal.Zero), "3.1 WITHDRAW: withdraw zero amount should return 422, got %d", statusCode)
	require.Equal(s.T(), http.StatusConflict, s.appAPI.withdraw(s.T(), order.Number, decimal.NewFromInt(1)), "3.1 WITHDRAW: withdraw for existing order should return 409, got %d", statusCode)
	require.Equal(s.T(), http.StatusUnprocessableEntity, s.appAPI.withdraw(s.T(), invalidOrderNumber, decimal.NewFromInt(1)), "3.1 WITHDRAW: withdraw for invalid order should return 422, got %d", statusCode)
	require.Equal(s.T(), http.StatusPaymentRequired, s.appAPI.withdraw(s.T(), orderNumber2, balanceAmount.Add(balanceAmount)), "3.1 WITHDRAW: withdraw more than balance should return 402, got %d", statusCode)

	require.Equal(s.T(), http.StatusOK, s.appAPI.withdraw(s.T(), orderNumber2, balanceAmount), "3.1 WITHDRAW: withdraw whole balance should return 200, got %d", statusCode)
	statusCode, balance = s.appAPI.getBalance(s.T())
	require.Equal(s.T(), http.StatusOK, statusCode, "3.1 WITHDRAW: get balance should return 200, got %d", statusCode)
	require.True(s.T(), balance.Current.Equal(decimal.Zero), "3.1 WITHDRAW: balance should be zero, got %s", balance.Current)
	require.True(s.T(), balance.Withdrawn.Equal(balanceAmount), "3.2 WITHDRAW: withdrawn should be equal to old balance, got %s", balance.Withdrawn)

	// 3.2 Get withdrawals
	statusCode, withdrawals = s.appAPI.getWithdrawals(s.T())
	require.Equal(s.T(), http.StatusOK, statusCode, "3.2 WITHDRAW: get withdrawals should return 200, got %d", statusCode)
	require.Len(s.T(), withdrawals, 1, "3.2 WITHDRAW: get withdrawals should return single withdrawal, got %d", len(withdrawals))
	withdrawal := withdrawals[0]
	require.Equal(s.T(), withdrawal.Order, orderNumber2, "3.2 WITHDRAW: withdrawal order should be equal to orderNumber2, got %s", withdrawal.Order)
	require.True(s.T(), withdrawal.Sum.Equal(balanceAmount), "3.2 WITHDRAW: withdrawal sum should be equal to old balance, got %s", withdrawal.Sum)
	require.NotEmpty(s.T(), withdrawal.ProcessedAt, "3.2 WITHDRAW: withdrawal processed_at should not be empty")
}

func (s *AppSuite) startAccrualServer(
	ctx context.Context,
	cfg *config.Config) *httpserver.Server {
	router := chi.NewRouter()

	type response struct {
		Order   string          `json:"order"`
		Status  string          `json:"status"`
		Accrual decimal.Decimal `json:"accrual,omitempty"`
		Attempt uint8           `json:"-"`
	}
	storage := make(map[string]response)

	router.Get("/api/orders/{number}", func(w http.ResponseWriter, r *http.Request) {
		number := chi.URLParam(r, "number")

		resp, ok := storage[number]
		if !ok {
			resp = response{Order: number, Attempt: 1}
		}
		attempt := resp.Attempt
		switch {
		case number == "":
			resp.Status = "INVALID"
		case attempt == 1:
			resp.Status = "REGISTERED"
		case attempt == 2:
			resp.Status = "PROCESSING"
		case attempt == 3:
			resp.Status = "PROCESSED"
			resp.Accrual = decimal.NewFromFloat(100.33)
		}
		resp.Attempt++
		storage[number] = resp

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(s.T(), err, "failed to encode response")
	})

	server := httpserver.New(
		router,
		httpserver.Addr(cfg.AccrualAPIAddr),
		httpserver.ShutdownTimeout(cfg.ServerShutdownTimeout))
	s.logger.Debug("server started")

	return server
}

func newAppAPI(cfg *config.Config) *appAPI {
	addr := cfg.ServerAddr
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	client := resty.New()
	client.SetBaseURL(addr)

	return &appAPI{
		client: client,
	}
}

func (api *appAPI) register(t *testing.T, user userModel) (statusCode int) {
	request := struct {
		Login string `json:"login"`
		Pass  string `json:"password"`
	}{
		Login: user.login,
		Pass:  user.pass,
	}

	resp, err := api.client.
		R().
		SetBody(request).
		Post("api/user/register")

	require.NoError(t, err)

	code := resp.StatusCode()
	if code == http.StatusOK {
		token := resp.Header().Get("Authorization")
		require.NotEmpty(t, token, "auth header should not be empty")
		api.client.Header.Set("Authorization", token)
	}

	return code
}

func (api *appAPI) login(t *testing.T, user userModel) (statusCode int) {
	request := struct {
		Login string `json:"login"`
		Pass  string `json:"password"`
	}{
		Login: user.login,
		Pass:  user.pass,
	}

	resp, err := api.client.
		R().
		SetBody(request).
		Post("api/user/login")

	require.NoError(t, err)

	code := resp.StatusCode()
	if code == http.StatusOK {
		token := resp.Header().Get("Authorization")
		require.NotEmpty(t, token, "auth header should not be empty")
		api.client.Header.Set("Authorization", token)
	}

	return code
}

func (api *appAPI) createOrder(t *testing.T, number string) (statusCode int) {
	resp, err := api.client.
		R().
		SetBody(number).
		Post("api/user/orders")

	require.NoError(t, err)

	return resp.StatusCode()
}

func (api *appAPI) getOrders(t *testing.T) (statusCode int, orders []orderModel) {
	resp, err := api.client.
		R().
		SetResult(&orders).
		Get("api/user/orders")

	require.NoError(t, err)

	if resp.StatusCode() != http.StatusOK {
		return resp.StatusCode(), nil
	}
	return resp.StatusCode(), orders
}

func (api *appAPI) getBalance(t *testing.T) (statusCode int, balance balanceModel) {
	resp, err := api.client.
		R().
		SetResult(balanceModel{}).
		Get("api/user/balance")

	require.NoError(t, err)

	if resp.StatusCode() != http.StatusOK {
		return resp.StatusCode(), balanceModel{}
	}

	return resp.StatusCode(), *resp.Result().(*balanceModel)
}

func (api *appAPI) withdraw(t *testing.T, order string, sum decimal.Decimal) (statusCode int) {
	request := struct {
		Order string          `json:"order"`
		Sum   decimal.Decimal `json:"sum"`
	}{
		Order: order,
		Sum:   sum,
	}

	resp, err := api.client.
		R().
		SetBody(request).
		Post("api/user/balance/withdraw")

	require.NoError(t, err)

	return resp.StatusCode()
}

func (api *appAPI) getWithdrawals(t *testing.T) (statusCode int, withdrawals []withdrawalModel) {
	resp, err := api.client.
		R().
		SetResult(&withdrawals).
		Get("api/user/withdrawals")

	require.NoError(t, err)

	if resp.StatusCode() != http.StatusOK {
		return resp.StatusCode(), nil
	}
	return resp.StatusCode(), withdrawals
}

func createPosgres(t *testing.T, cfg config.Config) (*postgres.PostgresContainer, config.Config) {
	values := strings.Split(cfg.DatabaseURI, " ")
	require.NotEmpty(t, values, "failed to parse database uri")
	kmap := make(map[int]string, len(values))
	vmap := make(map[string]string, len(values))
	for i, v := range values {
		kv := strings.Split(v, "=")
		require.Len(t, kv, 2, "failed to parse database uri value")
		kmap[i] = kv[0]
		vmap[kv[0]] = kv[1]
	}
	port, ok := vmap["port"]
	require.True(t, ok, "failed to get database port")
	username, ok := vmap["user"]
	require.True(t, ok, "failed to get database user")
	password, ok := vmap["password"]
	require.True(t, ok, "failed to get database password")
	dbname, ok := vmap["dbname"]
	require.True(t, ok, "failed to get database name")

	ctx := context.Background()
	pgc, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("docker.io/postgres:latest"),
		postgres.WithDatabase(dbname),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		postgres.WithInitScripts(),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(posgresStartup),
		),
	)
	require.NoError(t, err, "failed to start postgres container")

	newHost, err := pgc.Host(ctx)
	require.NoError(t, err, "failed to get postgres container host")
	newPort, err := pgc.MappedPort(ctx, nat.Port(port))
	require.NoError(t, err, "failed to get postgres container port")

	var sb strings.Builder
	for i := 0; i < len(values); i++ {
		k := kmap[i]

		_, _ = sb.WriteString(k)
		_ = sb.WriteByte('=')
		switch {
		case k == "host":
			_, _ = sb.WriteString(newHost)
		case k == "port":
			_, _ = sb.WriteString(strconv.Itoa(newPort.Int()))
		default:
			_, _ = sb.WriteString(vmap[k])
		}
		_ = sb.WriteByte(' ')
	}

	cfg.DatabaseURI = strings.TrimRight(sb.String(), " ")
	return pgc, cfg
}
