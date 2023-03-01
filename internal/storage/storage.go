package storage

import (
	"errors"
	"sync"
)

var ErrLoginAlreadyExists = errors.New("username is taken")

type Storage interface {
	Register(login string, password string) error
}

type AuthDataStorage struct {
	Data map[string]string
	Mu   *sync.Mutex
}

func NewStorage() AuthDataStorage {
	return AuthDataStorage{
		Data: make(map[string]string),
		Mu:   &sync.Mutex{},
	}
}

func (s *AuthDataStorage) Register(login string, password string) error {
	_, contains := s.Data[login]
	if contains {
		return ErrLoginAlreadyExists
	} else {
		s.Data[login] = password
	}
	return nil
}
