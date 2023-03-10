package accrualworker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/orders"
	"github.com/gambruh/gophermart/internal/storage"
)

const workersmax = 1

type SQLdb struct {
	db      *sql.DB
	address string
}

var UpdateOrdersFromAccrualQuery = `
		UPDATE orders
		SET status = $1
		WHERE number = $2;
`

type Agent struct {
	Client  *http.Client
	Server  string
	Storage storage.Storage
	Mu      *sync.Mutex
}

func NewAgent(st storage.Storage) *Agent {
	return &Agent{
		Client:  &http.Client{},
		Server:  config.Cfg.Address,
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
	var resArr []orders.ProcessedOrder

	for _, j := range ordsArr {
		res, err := a.makeGetRequest(j)
		if err != nil {
			log.Println("error when sending order to accrual:", err)
			return []orders.ProcessedOrder{}, err
		}
		results = append(results, res)
	}
	fmt.Println("returning resArr:", resArr)
	return resArr, nil
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
	url := fmt.Sprintf("http://%s/api/orders/%s", a.Server, ordernumber)
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
	//defer res.Body.Close()
	log.Println("OK UP TO HERE")
	err = json.NewDecoder(res.Body).Decode(&processed)
	if err != nil {
		log.Println("error when decoding processed orders from accrual:", err)
		return orders.ProcessedOrder{}, err
	}
	log.Println("processed order:", processed)
	return processed, nil
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
