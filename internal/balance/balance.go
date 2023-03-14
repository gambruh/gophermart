package balance

import (
	"errors"
	"time"
)

type Operation struct {
	Order       string    `json:"order"`
	Accrual     float32   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type Balance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

type WithdrawQ struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

// типы ошибок
var ErrNoOperations = errors.New("no records found")
var ErrInsufficientFunds = errors.New("not enough accrual to withdraw")
var ErrWrongOrder = errors.New("wrong order")

// SQL queries
var GetBalanceQuery = `
	SELECT COALESCE(SUM(accrual),0)
	FROM operations
	WHERE user_id = (
		SELECT id 
		FROM users 
		WHERE username = $1
		); 
`

var GetWithdrawnQuery = `
	SELECT COALESCE(SUM(accrual),0)
	FROM operations
	WHERE user_id = (
		SELECT id 
		FROM users 
		WHERE username = $1
		)
	AND	accrual < 0; 
`

var GetWithdrawalsQuery = `
	SELECT number, accrual, processed_at
	FROM operations
	WHERE user_id = (
		SELECT id
		FROM users
		WHERE username = $1
		) 
	AND	accrual < 0;
`

var InsertOperationQuery = `
	INSERT INTO operations (user_id, number, accrual, processed_at)
	VALUES (
		(SELECT user_id 
		FROM orders 
		WHERE number = $1),
		$1,
		$2, 
		TO_TIMESTAMP($3,'YYYY-MM-DD"T"HH24:MI:SS"Z"TZH:TZM')
	);
`

var InsertWithdrawOperation = `
	WITH new_order AS (
		INSERT INTO orders(number, user_id, status, uploaded_at) 
		VALUES ($2,
			(SELECT id 
			FROM users 
			WHERE username = $1),
			'NEW',
			TO_TIMESTAMP($4,'YYYY-MM-DD"T"HH24:MI:SS"Z"TZH:TZM'))
		RETURNING user_id)
	INSERT INTO operations (user_id, number, accrual, processed_at)
	VALUES (
		(SELECT user_id 
		FROM new_order),
		$2,
		$3, 
		TO_TIMESTAMP($4,'YYYY-MM-DD"T"HH24:MI:SS"Z"TZH:TZM')
	);
`
