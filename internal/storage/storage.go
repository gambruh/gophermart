package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/orders"
)

var (
	ErrUsernameIsTaken  = errors.New("username is taken")
	ErrWrongCredentials = errors.New("wrong login/password")
)

type Storage interface {
	Register(login string, password string) error
	VerifyCredentials(login string, password string) error
	GetStorage() map[string]string
	SetOrder(string, string) error
	GetOrders(ctx context.Context) ([]orders.Order, error)
}

type MemStorage struct {
	Data   map[string]string
	Umap   map[string]string
	Orders map[string][]orders.Order
	Mu     *sync.Mutex
}

func NewStorage() MemStorage {
	return MemStorage{
		Data:   make(map[string]string),
		Umap:   make(map[string]string),
		Orders: make(map[string][]orders.Order),
		Mu:     &sync.Mutex{},
	}
}

func (s *MemStorage) GetStorage() map[string]string {
	return s.Data
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
		return ErrWrongCredentials
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
		s.Orders[ordernumber] = append(s.Orders[ordernumber],
			orders.Order{
				Number:     ordernumber,
				Status:     "NEW",
				UploadedAt: t,
			})
		return nil
	case contains && uname == username:
		fmt.Println("Order has been loaded by the user already:", username)
		return orders.ErrOrderLoadedThisUser
	case contains && uname != username:
		fmt.Println("order has been loaded by another user")
		return orders.ErrOrderLoadedAnotherUser
	default:
		return errors.New("unexpected case when trying to load order into storage")
	}
}

func (s *MemStorage) GetOrders(ctx context.Context) ([]orders.Order, error) {
	username := ctx.Value(config.UserID("userID"))
	return s.Orders[username.(string)], nil
}
