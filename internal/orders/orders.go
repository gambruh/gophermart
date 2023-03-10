package orders

import (
	"errors"
	"time"
)

type Order struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    *int      `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded at"`
}

type ProcessedOrder struct {
	Number  string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

var (
	ErrOrderLoadedThisUser    = errors.New("order has been already loaded by this user")
	ErrOrderLoadedAnotherUser = errors.New("order has been already loaded by another user")
	ErrWrongOrderNumberFormat = errors.New("order number is wrong - can't pass Luhn algorithm")
	ErrNoOrders               = errors.New("orders not found for the user")
)

func LuhnCheck(ordernumber string) bool {

	var sum int
	var digit int
	var even = false

	// iterate over digits from right to left
	for i := len(ordernumber) - 1; i >= 0; i-- {
		digit = int(ordernumber[i] - '0')
		if even {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit

		even = !even
	}

	// return true if sum is divisible by 10
	return sum%10 == 0
}
