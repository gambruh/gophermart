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

	"github.com/gambruh/gophermart/internal/database"
)

func TestAgent_makeGetRequest(t *testing.T) {
	var testval float32 = 500

	testAccrualStorage := map[string]database.ProcessedOrder{
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
					order = database.ProcessedOrder{Number: orderNum, Status: "NEW"}
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
		Storage: &database.MemStorage{
			Data: map[string]string{"Vasya": "secret", "Petya": "secretsecret", "Jenya": "123"},
			Umap: map[string]string{"1234567897": "Vasya", "1234532313": "Vasya", "1234532339": "Petya"},
			Orders: map[string][]database.Order{
				"Vasya": {
					database.Order{
						Number:     "1234567897",
						Status:     "NEW",
						UploadedAt: time.Now(),
					},
					database.Order{
						Number:     "1234532313",
						Status:     "NEW",
						UploadedAt: time.Now(),
					},
				},
				"Petya": {
					database.Order{
						Number:     "1234532339",
						Status:     "NEW",
						UploadedAt: time.Now(),
					},
				},
			},
			Operations: make(map[string][]database.Operation),
			Mu:         &sync.Mutex{},
		},
	}

	tests := []struct {
		name       string
		a          *Agent
		order      string
		want       database.ProcessedOrder
		wantStatus int
	}{
		{
			name:  "get order 1",
			a:     &testAgent,
			order: "1234567897",
			want: database.ProcessedOrder{
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
			want: database.ProcessedOrder{
				Number: "1234532313",
				Status: "PROCESSING",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "get order 3",
			a:     &testAgent,
			order: "1234532339",
			want: database.ProcessedOrder{
				Number: "1234532339",
				Status: "INVALID",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "get order 4 - but not in accrual",
			a:     &testAgent,
			order: "12312344",
			want: database.ProcessedOrder{
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
