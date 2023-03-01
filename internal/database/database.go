package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gambruh/gophermart/internal/config"
	_ "github.com/lib/pq"
)

type Table struct {
	Name    string
	Column  string
	Coltype string
}

type SQLdb struct {
	db *sql.DB
}

var Tables []Table

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
var dropTableQuery = `
	DROP TABLE $1 CASCADE;
`
var dropPasswordsTableQuery = `
	DROP TABLE passwords CASCADE;
`
var dropUsersTableQuery = `
	DROP TABLE users CASCADE;
`

var createOrdersTableQuery = `
	CREATE TABLE orders (
		id SERIAL PRIMARY KEY,
		number bigint UNIQUE,
		user_id integer,
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

// типы ошибок
var (
	ErrTableDoesntExist = errors.New("table doesn't exist")
	ErrUsernameIsTaken  = errors.New("username is taken")
)

func NewSQLdb(postgresStr string) *SQLdb {
	db, _ := sql.Open("postgres", postgresStr)
	return &SQLdb{db: db}
}

func CheckConn(dbAddress string) error {
	db, err := sql.Open("postgres", dbAddress)
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
	s.CreateTables()
}

func CheckTableExists(tablename string) error {
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
	db, err := sql.Open("postgres", config.Cfg.Database)
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
		_, err = db.Exec(dropUsersTableQuery)
	}
	if err != nil {
		fmt.Printf("error dropping a table: %v", err)
		return err
	}

	return nil
}

func (s *SQLdb) CreateTables() error {
	db, err := sql.Open("postgres", config.Cfg.Database)
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
	//_, err = db.Exec(createOrdersTableQuery)
	//if err != nil {
	//	return err
	//}
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

func (s *SQLdb) VerifyCredentials(login string, password string) (bool, error) {
	var (
		id   int
		pass string
	)

	db, err := sql.Open("postgres", config.Cfg.Database)
	if err != nil {
		fmt.Printf("error opening database: %v", err)
		return false, err
	}
	defer db.Close()

	err = s.db.QueryRow(checkUsernameQuery, login).Scan(&id)
	switch err {
	case sql.ErrNoRows:
		return false, nil
	case nil:
	default:
		fmt.Println("Unexpected case in checking user's credentials in database:", err)
		return false, err
	}

	err = s.db.QueryRow(checkPasswordQuery, id).Scan(&pass)
	if err != nil {
		fmt.Println("Unexpected case in checking user's password in database:", err)
		return false, err
	}

	return pass == password, nil
}
