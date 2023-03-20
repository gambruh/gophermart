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
	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/helpers"

	_ "github.com/lib/pq"
)

type SQLdb struct {
	DB *sql.DB
}

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

type Order struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    *float32  `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded at"`
}

type ProcessedOrder struct {
	Number  string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float32 `json:"accrual,omitempty"`
}

type Storage interface {
	Register(login string, password string) error
	VerifyCredentials(login string, password string) error
	SetOrder(string, string) error
	GetOrders(ctx context.Context) ([]Order, error)
	GetOrdersForAccrual() ([]string, error)
	GetPass(username string) (string, error)
	UpdateAccrual([]ProcessedOrder) error
	AddAccrualOperation([]ProcessedOrder) error
	GetBalance(context.Context) (Balance, error)
	GetWithdrawals(context.Context) ([]Operation, error)
	Withdraw(context.Context, WithdrawQ) error
}

// типы ошибок
var (
	ErrUserNotFound           = errors.New("user not found in database")
	ErrTableDoesntExist       = errors.New("table doesn't exist")
	ErrUsernameIsTaken        = errors.New("username is taken")
	ErrWrongPassword          = errors.New("wrong password")
	ErrWrongCredentials       = errors.New("wrong login credentials")
	ErrNoOperations           = errors.New("no records found")
	ErrInsufficientFunds      = errors.New("not enough accrual to withdraw")
	ErrWrongOrder             = errors.New("wrong order")
	ErrOrderLoadedThisUser    = errors.New("order has been already loaded by this user")
	ErrOrderLoadedAnotherUser = errors.New("order has been already loaded by another user")
	ErrWrongOrderNumberFormat = errors.New("order number is wrong - can't pass Luhn algorithm")
	ErrNoOrders               = errors.New("orders not found for the user")
)

func NewSQLdb(postgresStr string) *SQLdb {
	DB, _ := sql.Open("postgres", postgresStr)
	return &SQLdb{
		DB: DB,
	}
}

func GetDB() (defstorage Storage) {
	if config.Cfg.Storage {
		defstorage = NewStorage()
	} else {
		db := NewSQLdb(config.Cfg.Database)
		db.InitDatabase()
		defstorage = db
	}
	return defstorage
}

func (s *SQLdb) CheckConn(dbAddress string) error {
	db, err := sql.Open("postgres", config.Cfg.Database)
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

func (s *SQLdb) InitDatabase() error {
	err := s.CheckNDropTables()
	if err != nil {
		return err
	}
	err = s.CreateOrdersTable()
	if err != nil {
		return err
	}

	err = s.CreateOperationsTable()
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLdb) CheckTableExists(tablename string) error {
	var check bool
	db, err := sql.Open("postgres", config.Cfg.Database)
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

	var check bool

	err := s.DB.QueryRow(checkTableExistsQuery, "orders").Scan(&check)
	if err != nil {
		fmt.Printf("error checking if table exists: %v", err)
		return err
	}
	if check {
		_, err = s.DB.Exec(dropOrdersTableQuery)
		if err != nil {
			return err
		}
	}
	if err != nil {
		fmt.Printf("error dropping a table: %v", err)
		return err
	}

	err = s.DB.QueryRow(checkTableExistsQuery, "operations").Scan(&check)
	if err != nil {
		fmt.Printf("error checking if table exists: %v", err)
		return err
	}
	if check {
		_, err = s.DB.Exec(dropOperationsTableQuery)
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

func (s *SQLdb) CreateOrdersTable() error {
	err := s.CheckTableExists("orders")
	if err == ErrTableDoesntExist {
		_, err := s.DB.Exec(createOrdersTableQuery)
		return err
	}
	return nil

}

func (s *SQLdb) CreateOperationsTable() error {
	err := s.CheckTableExists("operations")
	if err == ErrTableDoesntExist {
		_, err := s.DB.Exec(createOperationsTableQuery)
		return err
	}
	return nil
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

	err := s.DB.QueryRow(auth.CheckUsernameQuery, login).Scan(&id)
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

// orders
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
		return ErrOrderLoadedThisUser
	case userq != username:
		return ErrOrderLoadedAnotherUser
	default:
		log.Println("Unexpected error in SetOrder method:", err)
		return err
	}
}

func (s *SQLdb) GetOrders(ctx context.Context) ([]Order, error) {
	var ords []Order
	username := ctx.Value(config.UserID("userID"))
	rows, err := s.DB.QueryContext(ctx, getOrdersByUserQuery, username)
	if err != nil {
		log.Println("error when getting orders:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ord Order
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
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	if len(ords) == 0 {
		return ords, ErrNoOrders
	}
	return ords, nil
}

func (s *SQLdb) GetOrdersForAccrual() (results []string, err error) {
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
	return results, nil
}

func (s *SQLdb) UpdateAccrual(ords []ProcessedOrder) error {
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

func (s *SQLdb) AddAccrualOperation(ords []ProcessedOrder) error {
	balanceAddQ, _ := s.DB.Prepare(InsertOperationQuery)
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

func (s *SQLdb) GetBalance(ctx context.Context) (Balance, error) {
	var b Balance
	username := ctx.Value(config.UserID("userID"))
	err := s.DB.QueryRowContext(ctx, GetBalanceQuery, username).Scan(&b.Current)
	if err != nil {
		return Balance{}, err
	}

	err = s.DB.QueryRowContext(ctx, GetWithdrawnQuery, username).Scan(&b.Withdrawn)
	if err != nil {
		log.Println("error when trying to connect to database in GetBalance method:", err)
		return Balance{}, err
	}
	if b.Withdrawn != 0 {
		b.Withdrawn *= -1
	}

	return b, nil
}

func (s *SQLdb) GetWithdrawals(ctx context.Context) ([]Operation, error) {
	var ops []Operation
	username := ctx.Value(config.UserID("userID"))
	rows, err := s.DB.QueryContext(ctx, GetWithdrawalsQuery, username)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var op Operation
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

func (s *SQLdb) Withdraw(ctx context.Context, withdrawq WithdrawQ) error {
	username := ctx.Value(config.UserID("userID"))
	currentbalance, err := s.GetBalance(ctx)
	if err != nil {
		return err
	}
	pass := helpers.LuhnCheck(withdrawq.Order)
	if !pass {
		return ErrWrongOrder
	}

	if currentbalance.Current < withdrawq.Sum {
		return ErrInsufficientFunds
	}
	t := time.Now()
	formattedtime := t.Format(time.RFC3339)

	_, err = s.DB.ExecContext(ctx, InsertWithdrawOperation, username, withdrawq.Order, withdrawq.Sum*(-1), formattedtime)
	return err
}

func (s *SQLdb) GetPass(username string) (string, error) {
	var password string

	err := s.DB.QueryRow(getPassQuery, username).Scan(&password)
	if err != nil {
		return "", err
	}

	return password, nil
}
