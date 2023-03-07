package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/orders"
	_ "github.com/lib/pq"
)

type SQLdb struct {
	db      *sql.DB
	address string
}

// SQL queries
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

var createOrdersTableQuery = `
	CREATE TABLE orders (
		number TEXT UNIQUE NOT NULL PRIMARY KEY,
		user_id integer NOT NULL,
		status text NOT NULL,
		accrual INTEGER,
		uploaded_at TIMESTAMP WITH TIME ZONE,
		CONSTRAINT fk_ousers
			FOREIGN KEY (user_id)
				REFERENCES users(id)
				ON DELETE CASCADE
	);
`

var checkUsernameQuery = `
	SELECT id
	FROM users 
	WHERE username = $1;
`

var checkPasswordQuery = `
	SELECT password
	FROM passwords 
	WHERE id = $1;
`

var addNewUserQuery = `
	WITH new_user AS (
		INSERT INTO users (username)
		VALUES ($1)
		RETURNING id
	)
	INSERT INTO passwords (id, password)
	VALUES ((SELECT id FROM new_user), $2);
`

var checkOrderNumberQuery = `
	WITH new_order AS (
	SELECT orders (id)
	WHERE ($1);
)
`

var insertNewOrderQuery = `
	INSERT INTO orders(number, user_id, status, uploaded_at) 
	VALUES ($1,$2,$3,TO_TIMESTAMP($4,'YYYY-MM-DD"T"HH24:MI:SS"Z"TZH:TZM'));
`

var getOrdersQuery = `
	SELECT orders.number, orders.status, orders.accrual, orders.uploaded_at, users.username
	FROM orders
	JOIN users ON orders.user_id = users.id;
`

var getOrdersByUserQuery = `
	SELECT orders.number, orders.status, orders.accrual, orders.uploaded_at
	FROM orders
	JOIN users ON orders.user_id = users.id
	WHERE users.username = $1;
`

var getUsernameByNumberQuery = `
	SELECT users.username
	FROM orders
	JOIN users ON orders.user_id = users.id
	WHERE orders.number = $1;
`

var CheckIDbyUsernameQuery = `
"SELECT id FROM users WHERE username = $1"
`

// типы ошибок
var (
	ErrTableDoesntExist = errors.New("table doesn't exist")
	ErrUsernameIsTaken  = errors.New("username is taken")
	ErrWrongPassword    = errors.New("wrong password")
)

func NewSQLdb(postgresStr string) *SQLdb {
	db, _ := sql.Open("postgres", postgresStr)
	return &SQLdb{
		db:      db,
		address: postgresStr,
	}
}

func (s *SQLdb) CheckConn(dbAddress string) error {
	db, err := sql.Open("postgres", s.address)
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
	s.CreateUserTables()
	s.CreateOrdersTable()
}

func (s *SQLdb) CheckTableExists(tablename string) error {
	var check bool
	db, err := sql.Open("postgres", s.address)
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
	db, err := sql.Open("postgres", s.address)
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

	return nil
}

func (s *SQLdb) CreateUserTables() error {
	db, err := sql.Open("postgres", s.address)
	if err != nil {
		fmt.Printf("error opening database: %v", err)
		return err
	}
	defer db.Close()
	_, err = db.Exec(createUsersTableQuery)
	if err != nil {
		return err
	}
	_, err = db.Exec(createPasswordsTableQuery)
	if err != nil {
		return err
	}
	_, err = db.Exec(createOrdersTableQuery)
	if err != nil {
		return err
	}
	return nil
}
func (s *SQLdb) CreateOrdersTable() error {
	db, err := sql.Open("postgres", s.address)
	if err != nil {
		fmt.Printf("error opening database: %v", err)
		return err
	}
	defer db.Close()

	_, err = db.Exec(createOrdersTableQuery)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLdb) Register(login string, password string) error {
	var username string
	e := s.db.QueryRow(checkUsernameQuery, login).Scan(&username)
	switch e {
	case sql.ErrNoRows:
		_, err := s.db.Exec(addNewUserQuery, login, password)
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

	db, err := sql.Open("postgres", s.address)
	if err != nil {
		fmt.Printf("error opening database: %v", err)
		return err
	}
	defer db.Close()

	err = s.db.QueryRow(checkUsernameQuery, login).Scan(&id)
	switch err {
	case sql.ErrNoRows:
		return nil
	case nil:
	default:
		fmt.Println("Unexpected case in checking user's credentials in database:", err)
		return err
	}

	err = s.db.QueryRow(checkPasswordQuery, id).Scan(&pass)
	if err != nil {
		fmt.Println("Unexpected case in checking user's password in database:", err)
		return err
	}

	if pass != password {
		return ErrWrongPassword
	} else {
		return nil
	}
}

func (s *SQLdb) GetStorage() map[string]string {
	//заглушка для применения интерфейса storage.Storage
	return make(map[string]string)
}

//orders

func (s *SQLdb) CheckOrder() {

}

func (s *SQLdb) SetOrder(ordernumber string, username string) error {
	var userq string
	var id string
	err := s.db.QueryRow(CheckIDbyUsernameQuery, username).Scan(&id)
	if err != nil {
		log.Println("error when trying to connect to database in SetOrder method:", err)
		return err
	}
	err = s.db.QueryRow(getUsernameByNumberQuery, ordernumber).Scan(&userq)
	switch {
	case err == sql.ErrNoRows:
		t := time.Now()
		formattedTime := t.Format(time.RFC3339)
		_, e := s.db.Exec(insertNewOrderQuery, ordernumber, id, "NEW", formattedTime)
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
	rows, err := s.db.Query(getOrdersByUserQuery, username)
	if err != nil {
		fmt.Println("error when getting orders:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ord orders.Order
		err = rows.Scan(&ord.Number, &ord.Status, &ord.Accrual, &ord.UploadedAt)
		if err != nil {
			return nil, err
		}
		ords = append(ords, ord)
	}
	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return ords, nil
}
