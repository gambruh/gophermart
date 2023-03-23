package auth

// auth queries
const CheckUsernameQuery = `
	SELECT id
	FROM users 
	WHERE username = $1;
`

const CheckPasswordQuery = `
	SELECT password
	FROM passwords 
	WHERE id = $1;
`

const AddNewUserQuery = `
	WITH new_user AS (
		INSERT INTO users (username)
		VALUES ($1)
		RETURNING id
	)
	INSERT INTO passwords (id, password)
	VALUES ((SELECT id FROM new_user), $2);
`

const checkTableExistsQuery = `
	SELECT EXISTS (
		SELECT 	1 
		FROM 	information_schema.tables
		WHERE 	table_name = $1
	);
`

const createUsersTableQuery = `
	CREATE TABLE users (
		id SERIAL,
		username text NOT NULL UNIQUE,
		PRIMARY KEY (id)
	);
`

const createPasswordsTableQuery = `
	CREATE TABLE passwords (
		id integer PRIMARY KEY,
		password TEXT NOT NULL,
		CONSTRAINT fk_users
			FOREIGN KEY (id) 
				REFERENCES users(id)
				ON DELETE CASCADE
	);
`

const dropPasswordsTableQuery = `
	DROP TABLE passwords CASCADE;
`

const dropUsersTableQuery = `
	DROP TABLE users CASCADE;
`

const getPassQuery = `
		SELECT password
		FROM passwords
		JOIN users ON users.id = passwords.user_id
		WHERE users.username = $1
	`
