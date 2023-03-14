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

	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/orders"
	"github.com/gambruh/gophermart/internal/storage"
)

const workersmax = 1

var ErrTooManyReqs = errors.New("too many requests")

type SQLdb struct {
	db      *sql.DB
	address string
}

type Agent struct {
	Client  *http.Client
	Server  string
	Storage storage.Storage
	Mu      *sync.Mutex
}

func NewAgent(st storage.Storage) *Agent {
	return &Agent{
		Client:  &http.Client{},
		Server:  config.Cfg.Accrual,
		Storage: st,
		Mu:      &sync.Mutex{},
	}
}

func (a *Agent) askAccrual() ([]orders.ProcessedOrder, error) {
	var results []orders.ProcessedOrder
	ordsArr, err := a.Storage.GetOrdersForAccrual()
	if err != nil {
		log.Println("error when trying to get orders from storage to ask accrual:", err)
		return []orders.ProcessedOrder{}, err
	}

	for i := 0; i < len(ordsArr); i++ {
		res, err := a.makeGetRequest(ordsArr[i])
		if err != nil {
			if err == ErrTooManyReqs {

			} else {
				log.Println("error when sending order to accrual:", err)
				return nil, err
			}
		}
		results = append(results, res)
	}
	fmt.Println("returning results array:", results)
	return results, nil
}

func (a *Agent) worker(wg *sync.WaitGroup, jobs <-chan string, results chan<- orders.ProcessedOrder) {
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

func (a *Agent) makeGetRequest(ordernumber string) (orders.ProcessedOrder, error) {
	var processed orders.ProcessedOrder
	url := fmt.Sprintf("%s/api/orders/%s", a.Server, ordernumber)
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
	log.Println("url is:", url)
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("error 1 in makeGetRequest:", err)
		return orders.ProcessedOrder{}, err
	}
	fmt.Printf("sending order number %s to ___%s___\n", ordernumber, url)
	res, err := a.Client.Do(r)
	if err != nil {
		fmt.Println("error 2 in makeGetRequest:", err)
		return orders.ProcessedOrder{}, err
	}
	defer res.Body.Close()
	//status code check
	switch {
	case res.StatusCode == 200:
		log.Println("OK UP TO HERE")
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Println("error in reading resbody in makeGetRequest:", err)
			return orders.ProcessedOrder{}, err
		}
		defer res.Body.Close()
		err = json.Unmarshal(body, &processed)
		if err != nil {
			log.Println("error when decoding processed orders from accrual:", err)
			log.Println("body is:", string(body))
			return orders.ProcessedOrder{Number: ordernumber, Status: "INVALID"}, nil
		}
		return processed, nil
	case res.StatusCode == 204:
		return orders.ProcessedOrder{Number: ordernumber, Status: "NEW"}, nil
	case res.StatusCode == 429:
		return orders.ProcessedOrder{Number: ordernumber}, ErrTooManyReqs
	default:
		log.Println("unexpected response from accrual api. StatusCode:", res.StatusCode)
		return orders.ProcessedOrder{}, errors.New("unexpected response from accrual api")
	}
}

func (a *Agent) PingAccrual() error {
	input, err := a.askAccrual()
	if err != nil {
		return err
	}
	err = a.Storage.UpdateAccrual(input)
	if err != nil {
		log.Println("error in updatedatabase func of accrualworker:", err)
		return err
	}

	return nil
}
