package mocks

import (
	"context"
	"github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/dlomanov/go-diploma-tpl/internal/entity"
	"github.com/dlomanov/go-diploma-tpl/internal/usecase"
	"sync"
)

var (
	_ usecase.UserRepo    = (*MockUserRepo)(nil)
	_ usecase.OrderRepo   = (*MockOrderRepo)(nil)
	_ usecase.BalanceRepo = (*MockBalanceRepo)(nil)
	_ trm.Manager         = (*MockTrm)(nil)
)

type (
	MockUserRepo struct {
		mu      sync.RWMutex
		storage map[entity.Login]entity.User
	}

	MockOrderRepo struct {
		mu      sync.RWMutex
		storage map[entity.OrderID]entity.Order
	}

	MockBalanceRepo struct {
		mu      sync.RWMutex
		storage map[entity.UserID]entity.Balance
	}

	MockPublisher struct {
	}

	MockTrm struct {
	}
)

func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		mu:      sync.RWMutex{},
		storage: make(map[entity.Login]entity.User),
	}
}

func (r *MockUserRepo) Exists(_ context.Context, login entity.Login) (bool, error) {
	_, ok := r.get(login)
	return ok, nil
}

func (r *MockUserRepo) Get(_ context.Context, login entity.Login) (entity.User, error) {
	user, ok := r.get(login)
	if !ok {
		return entity.User{}, usecase.ErrAuthUserNotFound
	}

	return user, nil
}

func (r *MockUserRepo) Save(_ context.Context, user entity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.storage[user.Login]; ok {
		return usecase.ErrAuthUserExists
	}
	r.storage[user.Login] = user

	return nil
}

func (r *MockUserRepo) get(login entity.Login) (entity.User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.storage[login]
	return user, ok
}

func NewMockOrderRepo() *MockOrderRepo {
	return &MockOrderRepo{
		mu:      sync.RWMutex{},
		storage: make(map[entity.OrderID]entity.Order),
	}
}

func (r *MockOrderRepo) Get(_ context.Context, id entity.OrderID) (entity.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.storage[id]
	if !ok {
		return entity.Order{}, usecase.ErrOrderNotFound
	}

	return order, nil
}

func (r *MockOrderRepo) GetAll(
	_ context.Context,
	filter *usecase.OrderFilter,
) ([]entity.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]entity.Order, 0)
	for _, order := range r.storage {
		if filter != nil {
			if filter.Type != nil && order.Type != *filter.Type {
				continue
			}
			if filter.UserID != nil && order.UserID != *filter.UserID {
				continue
			}
		}
		result = append(result, order)
	}

	return result, nil
}

func (r *MockOrderRepo) Save(
	_ context.Context,
	order entity.Order,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.storage[order.ID]; ok {
		return usecase.ErrOrderExists
	}

	r.storage[order.ID] = order
	return nil
}

func (r *MockOrderRepo) Update(
	_ context.Context,
	order entity.Order,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.storage[order.ID]; !ok {
		return usecase.ErrOrderNotFound
	}

	r.storage[order.ID] = order
	return nil
}

func NewMockBalanceRepo() *MockBalanceRepo {
	return &MockBalanceRepo{
		mu:      sync.RWMutex{},
		storage: make(map[entity.UserID]entity.Balance),
	}
}

func (r *MockBalanceRepo) Save(_ context.Context, balance entity.Balance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.storage[balance.UserID]; ok {
		return usecase.ErrBalanceExists
	}

	r.storage[balance.UserID] = balance

	return nil
}

func (r *MockBalanceRepo) Update(
	_ context.Context,
	balance entity.Balance,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.storage[balance.UserID] = balance
	return nil
}

func (r *MockBalanceRepo) Get(
	_ context.Context,
	userID entity.UserID,
) (entity.Balance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	balance, ok := r.storage[userID]
	if !ok {
		return entity.Balance{}, usecase.ErrBalanceNotFound
	}

	return balance, nil
}

func NewMockTrm() *MockTrm {
	return &MockTrm{}
}

func (m MockTrm) Do(
	ctx context.Context,
	f func(ctx context.Context) error,
) error {
	return f(ctx)
}

func (m MockTrm) DoWithSettings(
	ctx context.Context,
	_ trm.Settings,
	f func(ctx context.Context) error,
) error {
	return f(ctx)
}
