package auth

// auth queries
var CheckUsernameQuery = `
	SELECT id
	FROM users 
	WHERE username = $1;
`

var CheckPasswordQuery = `
	SELECT password
	FROM passwords 
	WHERE id = $1;
`

var AddNewUserQuery = `
	WITH new_user AS (
		INSERT INTO users (username)
		VALUES ($1)
		RETURNING id
	)
	INSERT INTO passwords (id, password)
	VALUES ((SELECT id FROM new_user), $2);
`

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

var getPassQuery = `
		SELECT password
		FROM passwords
		JOIN users ON users.id = passwords.user_id
		WHERE users.username = $1
	`
