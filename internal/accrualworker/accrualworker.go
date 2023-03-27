package accrualworker

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gambruh/gophermart/internal/auth"
	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/database"
)

const workersmax = 1
const pingtime = 5

var (
	ErrTooManyReqs = errors.New("too many requests")
	ErrNoNewOrders = errors.New("no orders for accrual")
)

type SQLdb struct {
	db *sql.DB
}

type Agent struct {
	Client      *http.Client
	Server      string
	Storage     database.Storage
	AuthStorage auth.AuthStorage
	Mu          *sync.Mutex
}

func (a *Agent) CheckAccrual() error {
	pingTime := time.NewTicker(pingtime * time.Second)
	defer pingTime.Stop()
	for {
		<-pingTime.C
		err := a.PingAccrual()
		if err != nil {
			log.Println("error while checking accrual:", err)
			return err
		}
	}
}
func NewAgent(st database.Storage) *Agent {
	return &Agent{
		Client:  &http.Client{},
		Server:  config.Cfg.Accrual,
		Storage: st,
		Mu:      &sync.Mutex{},
	}
}

func (a *Agent) askAccrual() ([]database.ProcessedOrder, error) {
	var results []database.ProcessedOrder
	ordsArr, err := a.Storage.GetOrdersForAccrual()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNoNewOrders
		}
		log.Println("error when trying to get orders from storage to ask accrual:", err)
		return nil, err
	}

	for i := 0; i < len(ordsArr); i++ {
		res, err := a.makeGetRequest(ordsArr[i])
		if err != nil {
			log.Println("error when sending order to accrual:", err)
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

func (a *Agent) worker(wg *sync.WaitGroup, jobs <-chan string, results chan<- database.ProcessedOrder) {
	for j := range jobs {
		result, err := a.makeGetRequest(j)
		if err != nil {
			log.Println("error when sending order to accrual:", err)
			return
		}
		results <- result
	}
	wg.Done()
}

func (a *Agent) makeGetRequest(ordernumber string) (database.ProcessedOrder, error) {
	var processed database.ProcessedOrder
	url := fmt.Sprintf("%s/api/orders/%s", a.Server, ordernumber)
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}

	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println("error 1 in makeGetRequest:", err)
		return database.ProcessedOrder{}, err
	}

	res, err := a.Client.Do(r)
	if err != nil {
		log.Println("error 2 in makeGetRequest:", err)
		return database.ProcessedOrder{}, err
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode == 200:
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Println("error in reading resbody in makeGetRequest:", err)
			return database.ProcessedOrder{}, err
		}
		defer res.Body.Close()
		err = json.Unmarshal(body, &processed)
		if err != nil {
			log.Println("error when decoding processed orders from accrual:", err)
			log.Println("body is:", string(body))
			return database.ProcessedOrder{Number: ordernumber, Status: "INVALID"}, nil
		}
		return processed, nil
	case res.StatusCode == 204:
		return database.ProcessedOrder{Number: ordernumber, Status: "NEW"}, nil
	case res.StatusCode == 429:
		return database.ProcessedOrder{Number: ordernumber}, ErrTooManyReqs
	default:
		log.Println("unexpected response from accrual api. StatusCode:", res.StatusCode)
		return database.ProcessedOrder{}, errors.New("unexpected response from accrual api")
	}
}

func (a *Agent) PingAccrual() error {
	input, err := a.askAccrual()
	if err != nil {
		log.Println("error in PingAccrual() func of accrualworker:", err)
		return err
	}
	err = a.Storage.UpdateAccrual(input)
	if err != nil {
		log.Println("error in updatedatabase func of accrualworker:", err)
		return err
	}

	return nil
}
