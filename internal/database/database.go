package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gambruh/gophermart/cmd/argon2id"
	"github.com/gambruh/gophermart/internal/auth"
	"github.com/gambruh/gophermart/internal/balance"
	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/orders"

	_ "github.com/lib/pq"
)

type SQLdb struct {
	DB      *sql.DB
	Address string
}

type Storage interface {
	Register(login string, password string) error
	VerifyCredentials(login string, password string) error
	SetOrder(string, string) error
	GetOrders(ctx context.Context) ([]orders.Order, error)
	GetOrdersForAccrual() ([]string, error)
	UpdateAccrual([]orders.ProcessedOrder) error
	AddAccrualOperation([]orders.ProcessedOrder) error
	GetBalance(context.Context) (balance.Balance, error)
	GetWithdrawals(context.Context) ([]balance.Operation, error)
	Withdraw(context.Context, balance.WithdrawQ) error
}

// SQL queries
// database init queries
var checkTableExistsQuery = `
	SELECT EXISTS (
		SELECT 	1 
		FROM 	information_schema.tables
		WHERE 	table_name = $1
	);
`

var createUsersTableQuery = `
	CREATE TABLE users (
		id SERIAL,
		username text NOT NULL UNIQUE,
		PRIMARY KEY (id)
	);
`

var createPasswordsTableQuery = `
	CREATE TABLE passwords (
		id integer PRIMARY KEY,
		password TEXT NOT NULL,
		CONSTRAINT fk_users
			FOREIGN KEY (id) 
				REFERENCES users(id)
				ON DELETE CASCADE
	);
`

var dropPasswordsTableQuery = `
	DROP TABLE passwords CASCADE;
`

var dropUsersTableQuery = `
	DROP TABLE users CASCADE;
`
var dropOrdersTableQuery = `
	DROP TABLE orders CASCADE;
`

var dropOperationsTableQuery = `
	DROP TABLE operations CASCADE;
`

var createOrdersTableQuery = `
	CREATE TABLE orders (
		number TEXT UNIQUE NOT NULL PRIMARY KEY,
		user_id integer NOT NULL,
		status text NOT NULL,
		uploaded_at TIMESTAMP WITH TIME ZONE,
		CONSTRAINT fk_ousers
			FOREIGN KEY (user_id)
				REFERENCES users(id)
				ON DELETE CASCADE
	);
`

var createOperationsTableQuery = `
	CREATE TABLE operations (
		id SERIAL,
		user_id integer NOT NULL,
		number TEXT NOT NULL,
		accrual double precision,
		processed_at TIMESTAMP WITH TIME ZONE,
		PRIMARY KEY (id),
		CONSTRAINT fk_oorders
			FOREIGN KEY (number)
				REFERENCES orders(number)
				ON DELETE CASCADE
	);
`

// orders queries
var insertNewOrderQuery = `
	INSERT INTO orders(number, user_id, status, uploaded_at) 
	VALUES ($1,$2,$3,TO_TIMESTAMP($4,'YYYY-MM-DD"T"HH24:MI:SS"Z"TZH:TZM'));
`

var getOrdersQuery = `
	SELECT orders.number, orders.status, orders.uploaded_at, users.username
	FROM orders
	JOIN users ON orders.user_id = users.id;
`

var getOrdersByUserQuery = `
	SELECT orders.number, orders.status, orders.uploaded_at
	FROM orders
	JOIN users ON orders.user_id = users.id
	WHERE users.username = $1;
`

var getOrderAccrualQuery = `
	SELECT accrual 
	FROM operations
	WHERE number = $1`

var getUsernameByNumberQuery = `
	SELECT users.username
	FROM orders
	JOIN users ON orders.user_id = users.id
	WHERE orders.number = $1;
`

var CheckIDbyUsernameQuery = `
	SELECT id 
	FROM users 
	WHERE username = $1;
`

// accrual worker queries

var AccrualAddQuery = `
	WITH new_order AS (
		UPDATE orders
		SET status = $1
		WHERE number = $2
		RETURNING user_id
	)
	INSERT INTO operations (user_id, number, accrual, processed_at)
	VALUES ((SELECT user_id FROM new_order),
		$2, 
		$3, 
		TO_TIMESTAMP($4,'YYYY-MM-DD"T"HH24:MI:SS"Z"TZH:TZM') 
	);
`
var UpdateStatusQuery = `
	UPDATE orders
	SET status = $1
	WHERE number = $2;
`

// типы ошибок
var (
	ErrTableDoesntExist = errors.New("table doesn't exist")
	ErrUsernameIsTaken  = errors.New("username is taken")
	ErrWrongPassword    = errors.New("wrong password")
)

func NewSQLdb(postgresStr string) *SQLdb {
	DB, _ := sql.Open("postgres", postgresStr)
	return &SQLdb{
		DB:      DB,
		Address: postgresStr,
	}
}

func (s *SQLdb) CheckConn(dbAddress string) error {
	db, err := sql.Open("postgres", s.Address)
	if err != nil {
		fmt.Printf("error while opening DB:%v\n", err)
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		fmt.Printf("error while pinging: %v\n", err)
		return err
	}
	return nil
}

func (s *SQLdb) InitDatabase() {
	s.CheckNDropTables()
	s.CreateUsersTable()
	s.CreatePasswordsTable()
	s.CreateOrdersTable()
	s.CreateOperationsTable()
}

func (s *SQLdb) CheckTableExists(tablename string) error {
	var check bool
	db, err := sql.Open("postgres", s.Address)
	if err != nil {
		fmt.Printf("error opening database: %v", err)
		return err
	}
	defer db.Close()

	err = db.QueryRow(checkTableExistsQuery, tablename).Scan(&check)
	if err != nil {
		fmt.Printf("error checking if table exists: %v", err)
		return err
	}
	if !check {
		return ErrTableDoesntExist
	}
	return nil
}

func (s *SQLdb) CheckNDropTables() error {
	db, err := sql.Open("postgres", s.Address)
	if err != nil {
		fmt.Printf("error opening database: %v", err)
		return err
	}
	defer db.Close()

	var check bool

	err = db.QueryRow(checkTableExistsQuery, "users").Scan(&check)
	if err != nil {
		fmt.Printf("error checking if table exists: %v", err)
		return err
	}
	if check {
		_, err = db.Exec(dropPasswordsTableQuery)
		if err != nil {
			return err
		}
		_, err = db.Exec(dropUsersTableQuery)
		if err != nil {
			return err
		}
	}
	if err != nil {
		fmt.Printf("error dropping a table: %v", err)
		return err
	}
	err = db.QueryRow(checkTableExistsQuery, "orders").Scan(&check)
	if err != nil {
		fmt.Printf("error checking if table exists: %v", err)
		return err
	}
	if check {
		_, err = db.Exec(dropOrdersTableQuery)
		if err != nil {
			return err
		}
	}
	if err != nil {
		fmt.Printf("error dropping a table: %v", err)
		return err
	}

	err = db.QueryRow(checkTableExistsQuery, "operations").Scan(&check)
	if err != nil {
		fmt.Printf("error checking if table exists: %v", err)
		return err
	}
	if check {
		_, err = db.Exec(dropOperationsTableQuery)
		if err != nil {
			fmt.Println("Error when dropping ops table:", err)
			return err
		}
	}
	if err != nil {
		fmt.Printf("error dropping a table: %v", err)
		return err
	}

	return nil
}

func (s *SQLdb) CreateUsersTable() error {
	err := s.CheckTableExists("users")
	if err == ErrTableDoesntExist {
		_, err := s.DB.Exec(createUsersTableQuery)
		if err != nil {
			return err
		}
	}
	return err
}
func (s *SQLdb) CreatePasswordsTable() error {
	err := s.CheckTableExists("passwords")
	if err == ErrTableDoesntExist {
		_, err = s.DB.Exec(createPasswordsTableQuery)
		if err != nil {
			return err
		}
	}
	return err
}
func (s *SQLdb) CreateOrdersTable() error {
	err := s.CheckTableExists("orders")
	if err == ErrTableDoesntExist {
		_, err := s.DB.Exec(createOrdersTableQuery)
		return err
	}
	return err

}

func (s *SQLdb) CreateOperationsTable() error {
	err := s.CheckTableExists("operations")
	if err == ErrTableDoesntExist {
		_, err := s.DB.Exec(createOperationsTableQuery)
		return err
	}
	return err
}

func (s *SQLdb) Register(login string, password string) error {
	var username string
	hashedpassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Println("error when trying to hash password:", err)
		return err
	}
	e := s.DB.QueryRow(auth.CheckUsernameQuery, login).Scan(&username)

	switch e {
	case sql.ErrNoRows:
		_, err := s.DB.Exec(auth.AddNewUserQuery, login, hashedpassword)
		return err
	case nil:
		err := ErrUsernameIsTaken
		return err
	default:
		fmt.Printf("Something wrong when adding a user in database: %v\n", e)
		return e
	}
}

func (s *SQLdb) VerifyCredentials(login string, password string) error {
	var (
		id   int
		pass string
	)

	db, err := sql.Open("postgres", s.Address)
	if err != nil {
		log.Println("error opening database:", err)
		return err
	}
	defer db.Close()

	err = s.DB.QueryRow(auth.CheckUsernameQuery, login).Scan(&id)
	switch err {
	case sql.ErrNoRows:
		return nil
	case nil:
	default:
		log.Println("Unexpected case in checking user's credentials in database:", err)
		return err
	}

	err = s.DB.QueryRow(auth.CheckPasswordQuery, id).Scan(&pass)
	if err != nil {
		log.Println("Unexpected case in checking user's password in database:", err)
		return err
	}

	check, err := argon2id.ComparePasswordAndHash(password, pass)
	if err != nil {
		log.Println("error when trying to compare password and hash:", err)
		return err
	}

	if !check {
		return ErrWrongPassword
	}

	return nil
}

func (s *SQLdb) GetStorage() map[string]string {
	//mock to implement interface storage.Storage
	return make(map[string]string)
}

//orders

func (s *SQLdb) SetOrder(ordernumber string, username string) error {
	var userq string
	var id string
	err := s.DB.QueryRow(CheckIDbyUsernameQuery, username).Scan(&id)
	if err != nil {
		log.Println("error when trying to connect to database in SetOrder method:", err)
		return err
	}
	err = s.DB.QueryRow(getUsernameByNumberQuery, ordernumber).Scan(&userq)
	switch {
	case err == sql.ErrNoRows:
		t := time.Now()
		formattedTime := t.Format(time.RFC3339)
		_, e := s.DB.Exec(insertNewOrderQuery, ordernumber, id, "NEW", formattedTime)
		if e != nil {
			return e
		}
		return nil
	case userq == username:
		return orders.ErrOrderLoadedThisUser
	case userq != username:
		return orders.ErrOrderLoadedAnotherUser
	default:
		log.Println("Unexpected error in SetOrder method:", err)
		return err
	}
}

func (s *SQLdb) GetOrders(ctx context.Context) ([]orders.Order, error) {
	var ords []orders.Order
	username := ctx.Value(config.UserID("userID"))
	rows, err := s.DB.QueryContext(ctx, getOrdersByUserQuery, username)
	if err != nil {
		log.Println("error when getting orders:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ord orders.Order
		err = rows.Scan(&ord.Number, &ord.Status, &ord.UploadedAt)
		if err != nil {
			log.Println("error when scanning rows in GetOrders:", err)
			return nil, err
		}
		if ord.Status == "PROCESSED" {
			err = s.DB.QueryRowContext(ctx, getOrderAccrualQuery, ord.Number).Scan(&ord.Accrual)
			if err != nil {
				log.Println("error when scanning orders accrual in GetOrders:", err)
				return nil, err
			}
		}
		ords = append(ords, ord)
	}
	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	if len(ords) == 0 {
		return ords, orders.ErrNoOrders
	}
	return ords, nil
}

func (s *SQLdb) GetOrdersForAccrual() (results []string, err error) {
	var getOrdersAccrualStatusUpdQuery = `
		SELECT number
		FROM orders
		WHERE status='NEW' 
			OR status='PROCESSING';
	`
	fmt.Println("PINGING ACCRUAL TO UPDATE ORDER STATUS!")

	rows, err := s.DB.Query(getOrdersAccrualStatusUpdQuery)
	if err != nil {
		log.Println("error while trying to get orders for accrual status update:", err)
		return nil, err
	}
	var number string
	for rows.Next() {
		rows.Scan(&number)
		results = append(results, number)
	}

	err = rows.Err()
	if err != nil {
		log.Println("error when trying to query database in GetOrdersForAccrual:", err)
		return nil, err
	}
	fmt.Println("orders to ask accrual:", results)
	return results, nil
}

func (s *SQLdb) UpdateAccrual(ords []orders.ProcessedOrder) error {
	accAddQ, err := s.DB.Prepare(AccrualAddQuery)
	if err != nil {
		log.Println("error in preparing SQL query AccrualAdd:", err)
		return err
	}
	statusChangeQ, err := s.DB.Prepare(UpdateStatusQuery)
	if err != nil {
		log.Println("error in preparing SQL query UPDATESTATUS:", err)
		return err
	}

	// Шаг 1 - объявляем транзакцию
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	// Шаг 1.1 - откат, если ошибка
	defer tx.Rollback()

	// шаг 2
	for _, o := range ords {
		// order status assertion
		switch o.Status {
		case "NEW":
			o.Status = "NEW"
		case "REGISTERED":
			o.Status = "PROCESSING"
		case "PROCESSING":
			o.Status = "PROCESSING"
		case "PROCESSED":
			o.Status = "PROCESSED"
		case "INVALID":
			o.Status = "INVALID"
		default:
			log.Println("unexpected order status from accrual:", o.Status)
			return errors.New("unexpected order status")
		}
		if o.Accrual != nil {
			formattedTime := time.Now().Format(time.RFC3339)
			_, err := accAddQ.Exec(o.Status, o.Number, *o.Accrual, formattedTime)
			if err != nil {
				log.Println("error in executing AccrualAddQuery:", err)
				return err
			}
		} else {
			_, err := statusChangeQ.Exec(o.Status, o.Number)
			if err != nil {
				log.Println("error in executing AccrualAddQuery:", err)
				return err
			}
		}

	}
	return tx.Commit()
}

func (s *SQLdb) AddAccrualOperation(ords []orders.ProcessedOrder) error {

	balanceAddQ, _ := s.DB.Prepare(balance.InsertOperationQuery)
	// Шаг 1 - объявляем транзакцию
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	// Шаг 1.1 - откат, если ошибка
	defer tx.Rollback()

	// Ставим время записи в базу как время выполнения операции (accrual не возвращает время)
	formattedTime := time.Now().Format(time.RFC3339)
	// шаг 2
	for _, o := range ords {
		if *o.Accrual == 0 {
			continue
		}
		_, err := balanceAddQ.Exec(o.Number, o.Accrual, formattedTime)
		if err != nil {
			log.Println("error in executing InsertOperationQuery:", err)
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLdb) GetBalance(ctx context.Context) (balance.Balance, error) {
	var b balance.Balance
	username := ctx.Value(config.UserID("userID"))
	err := s.DB.QueryRowContext(ctx, balance.GetBalanceQuery, username).Scan(&b.Current)
	if err != nil {
		return balance.Balance{}, err
	}

	err = s.DB.QueryRowContext(ctx, balance.GetWithdrawnQuery, username).Scan(&b.Withdrawn)
	if err != nil {
		log.Println("error when trying to connect to database in GetBalance method:", err)
		return balance.Balance{}, err
	}
	if b.Withdrawn != 0 {
		b.Withdrawn *= -1
	}

	return b, nil
}

func (s *SQLdb) GetWithdrawals(ctx context.Context) ([]balance.Operation, error) {
	var ops []balance.Operation
	username := ctx.Value(config.UserID("userID"))
	rows, err := s.DB.QueryContext(ctx, balance.GetWithdrawalsQuery, username)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var op balance.Operation
		err = rows.Scan(&op.Order, &op.Accrual, &op.ProcessedAt)
		if err != nil {
			log.Println("error when scanning rows in getting orders:", err)
			return nil, err
		}
		op.Accrual *= -1
		ops = append(ops, op)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return ops, nil
}

func (s *SQLdb) Withdraw(ctx context.Context, withdrawq balance.WithdrawQ) error {
	username := ctx.Value(config.UserID("userID"))
	currentbalance, err := s.GetBalance(ctx)
	if err != nil {
		return err
	}
	pass := orders.LuhnCheck(withdrawq.Order)
	if !pass {
		return balance.ErrWrongOrder
	}

	if currentbalance.Current < withdrawq.Sum {
		return balance.ErrInsufficientFunds
	}
	t := time.Now()
	formattedtime := t.Format(time.RFC3339)

	_, err = s.DB.ExecContext(ctx, balance.InsertWithdrawOperation, username, withdrawq.Order, withdrawq.Sum*(-1), formattedtime)
	return err
}
