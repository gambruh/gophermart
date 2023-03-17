package accrualworker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gambruh/gophermart/internal/balance"
	"github.com/gambruh/gophermart/internal/orders"
	"github.com/gambruh/gophermart/internal/storage"
)

func TestAgent_askAccrual(t *testing.T) {
	testAgent := Agent{
		Storage: &storage.MemStorage{
			Data: map[string]string{"Vasya": "secret", "Petya": "secretsecret", "Jenya": "123"},
			Umap: map[string]string{"1234567897": "Vasya", "1234532313": "Vasya", "1234532339": "Petya"},
			Orders: map[string][]orders.Order{
				"Vasya": {
					orders.Order{
						Number:     "1234567897",
						Status:     "NEW",
						UploadedAt: time.Now(),
					},
					orders.Order{
						Number:     "1234532313",
						Status:     "NEW",
						UploadedAt: time.Now(),
					},
				},
				"Petya": {
					orders.Order{
						Number:     "1234532339",
						Status:     "NEW",
						UploadedAt: time.Now(),
					},
				},
			},
			Operations: make(map[string][]balance.Operation),
			Mu:         &sync.Mutex{},
		},
	}

	tests := []struct {
		name string
		a    *Agent
		want []orders.ProcessedOrder
	}{
		{
			name: "accrual test 1 - new orders with different statuses",
			a:    &testAgent,
			want: []orders.ProcessedOrder{},
		},
		{
			name: "accrual test 2 - no data in accrual",
			a:    &testAgent,
			want: []orders.ProcessedOrder{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.a.askAccrual()
			if err != nil {
				t.Errorf("Agent.askAccrual() error = %v, ", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Agent.askAccrual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgent_makeGetRequest(t *testing.T) {
	var testval float32 = 500

	testAccrualStorage := map[string]orders.ProcessedOrder{
		"1234567897": {
			Number:  "1234567897",
			Status:  "PROCESSED",
			Accrual: &testval,
		},
		"1234532313": {
			Number: "1234532313",
			Status: "PROCESSING",
		},
		"1234532339": {
			Number: "1234532339",
			Status: "INVALID",
		},
	}

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				// Extract the order number from the URL
				arr := strings.Split(r.URL.Path, "/api/orders/")
				orderNum := arr[1]
				// Generate a new URL using the test server's URL and the order number

				order, ok := testAccrualStorage[orderNum]
				if !ok {
					order = orders.ProcessedOrder{Number: orderNum, Status: "NEW"}
				}
				fmt.Println("TEST RESPONSE:", order)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(order)
			} else {
				http.Error(w, "Invalid request", http.StatusBadRequest)
			}

		}))

	defer ts.Close()

	testAgent := Agent{
		Client: &http.Client{
			Transport: ts.Client().Transport,
		},
		Server: ts.URL,
		Storage: &storage.MemStorage{
			Data: map[string]string{"Vasya": "secret", "Petya": "secretsecret", "Jenya": "123"},
			Umap: map[string]string{"1234567897": "Vasya", "1234532313": "Vasya", "1234532339": "Petya"},
			Orders: map[string][]orders.Order{
				"Vasya": {
					orders.Order{
						Number:     "1234567897",
						Status:     "NEW",
						UploadedAt: time.Now(),
					},
					orders.Order{
						Number:     "1234532313",
						Status:     "NEW",
						UploadedAt: time.Now(),
					},
				},
				"Petya": {
					orders.Order{
						Number:     "1234532339",
						Status:     "NEW",
						UploadedAt: time.Now(),
					},
				},
			},
			Operations: make(map[string][]balance.Operation),
			Mu:         &sync.Mutex{},
		},
	}

	tests := []struct {
		name       string
		a          *Agent
		order      string
		want       orders.ProcessedOrder
		wantStatus int
	}{
		{
			name:  "get order 1",
			a:     &testAgent,
			order: "1234567897",
			want: orders.ProcessedOrder{
				Number:  "1234567897",
				Status:  "PROCESSED",
				Accrual: &testval,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "get order 2",
			a:     &testAgent,
			order: "1234532313",
			want: orders.ProcessedOrder{
				Number: "1234532313",
				Status: "PROCESSING",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "get order 3",
			a:     &testAgent,
			order: "1234532339",
			want: orders.ProcessedOrder{
				Number: "1234532339",
				Status: "INVALID",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "get order 4 - but not in accrual",
			a:     &testAgent,
			order: "12312344",
			want: orders.ProcessedOrder{
				Number: "12312344",
				Status: "NEW",
			},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.a.makeGetRequest(tt.order)
			if err != nil {
				t.Errorf("Agent.askAccrual() error = %v, ", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Agent.askAccrual() = %v, want %v", got, tt.want)
			}
		})
	}
}
