package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gambruh/gophermart/internal/config"
)

var ()

type MemStorage struct {
	// loginpass pairs
	Data map[string]string

	// ordernumber - username key-value pair
	Umap map[string]string

	// map with key-value pair of username - slice of order.Orders
	Orders map[string][]Order

	// map with username - slice of operations pairs
	Operations map[string][]Operation

	// to ensure possible concurrent usage
	Mu *sync.Mutex
}

func NewStorage() *MemStorage {
	return &MemStorage{
		Data:       make(map[string]string),
		Umap:       make(map[string]string),
		Orders:     make(map[string][]Order),
		Operations: make(map[string][]Operation),
		Mu:         &sync.Mutex{},
	}
}

func (s *MemStorage) GetStorage() map[string]string {
	return s.Data
}

func (s *MemStorage) GetPass(username string) (string, error) {
	password, contains := s.Data[username]
	if !contains {
		return "", ErrUserNotFound
	}

	return password, nil
}

func (s *MemStorage) Register(login string, password string) error {
	_, contains := s.Data[login]
	if contains {
		return ErrUsernameIsTaken
	} else {
		s.Data[login] = password
	}
	return nil
}

func (s MemStorage) VerifyCredentials(login string, password string) error {
	check, contains := s.Data[login]
	if contains && check == password {
		return nil
	} else {
		return ErrWrongPassword
	}
}

func (s *MemStorage) SetOrder(ordernumber string, username string) error {
	uname, contains := s.Umap[ordernumber]

	switch {
	case !contains:
		t := time.Now()
		formattedTime := t.Format(time.RFC3339)
		t, err := time.Parse(time.RFC3339, formattedTime)
		if err != nil {
			fmt.Println("return when parsing time in SetOrder handler")
			return err
		}
		s.Orders[username] = append(s.Orders[username],
			Order{
				Number:     ordernumber,
				Status:     "NEW",
				UploadedAt: t,
			})
		s.Umap[ordernumber] = username
		return nil
	case contains && uname == username:
		fmt.Println("Order has been loaded by the user already:", username)
		return ErrOrderLoadedThisUser
	case contains && uname != username:
		fmt.Println("order has been loaded by another user")
		return ErrOrderLoadedAnotherUser
	default:
		return errors.New("unexpected case when trying to load order into storage")
	}
}

func (s *MemStorage) GetOrders(ctx context.Context) ([]Order, error) {
	username := ctx.Value(config.UserID("userID"))
	return s.Orders[username.(string)], nil
}

func (s *MemStorage) GetOrdersForAccrual() ([]string, error) {
	var preparr []string

	for _, v := range s.Orders {
		for _, o := range v {
			if o.Status == "PROCESSING" || o.Status == "NEW" {
				preparr = append(preparr, o.Number)
			}
		}
	}
	return preparr, nil
}

func (s *MemStorage) UpdateAccrual([]ProcessedOrder) error {
	return nil
}
func (s *MemStorage) AddAccrualOperation([]ProcessedOrder) error {
	return nil
}

func (s *MemStorage) GetBalance(ctx context.Context) (Balance, error) {
	var b Balance
	username := ctx.Value(config.UserID("userID"))
	if len(s.Operations[username.(string)]) == 0 {
		return Balance{Current: 0, Withdrawn: 0}, nil
	}

	for _, op := range s.Operations[username.(string)] {
		b.Current += op.Accrual
		if op.Accrual < 0 {
			value := op.Accrual * (-1)
			b.Withdrawn += value
		}
	}

	return b, nil
}

func (s *MemStorage) GetWithdrawals(ctx context.Context) ([]Operation, error) {
	username := ctx.Value(config.UserID("userID"))
	return s.Operations[username.(string)], nil
}

func (s *MemStorage) Withdraw(ctx context.Context, withdrawq WithdrawQ) error {
	username := ctx.Value(config.UserID("userID"))
	currentbalance, err := s.GetBalance(ctx)
	if err != nil {
		log.Println("error in Withdraw:", err)
		return err
	}

	if currentbalance.Current >= withdrawq.Sum {
		var withdrawal Operation

		t := time.Now()
		formattedTime := t.Format(time.RFC3339)
		t, err := time.Parse(time.RFC3339, formattedTime)
		if err != nil {
			log.Println("error when parsing time in Withdraw op:", err)
			return err
		}
		withdrawal.Accrual = withdrawq.Sum
		withdrawal.Order = withdrawq.Order
		withdrawal.ProcessedAt = t
		s.Operations[username.(string)] = append(s.Operations[username.(string)], withdrawal)
	} else {
		return ErrInsufficientFunds
	}
	return nil
}
