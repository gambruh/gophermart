package database

// SQL queries
// database init queries
const checkTableExistsQuery = `
	SELECT EXISTS (
		SELECT 	1 
		FROM 	information_schema.tables
		WHERE 	table_name = $1
	);
`

const dropOrdersTableQuery = `
	DROP TABLE orders CASCADE;
`

const dropOperationsTableQuery = `
	DROP TABLE operations CASCADE;
`

const createOrdersTableQuery = `
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

const createOperationsTableQuery = `
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
const insertNewOrderQuery = `
	INSERT INTO orders(number, user_id, status, uploaded_at) 
	VALUES ($1,$2,$3,TO_TIMESTAMP($4,'YYYY-MM-DD"T"HH24:MI:SS"Z"TZH:TZM'));
`

const getOrdersQuery = `
	SELECT orders.number, orders.status, orders.uploaded_at, users.username
	FROM orders
	JOIN users ON orders.user_id = users.id;
`

const getOrdersByUserQuery = `
	SELECT orders.number, orders.status, orders.uploaded_at
	FROM orders
	JOIN users ON orders.user_id = users.id
	WHERE users.username = $1;
`

const getOrderAccrualQuery = `
	SELECT accrual 
	FROM operations
	WHERE number = $1`

const getUsernameByNumberQuery = `
	SELECT users.username
	FROM orders
	JOIN users ON orders.user_id = users.id
	WHERE orders.number = $1;
`

const CheckIDbyUsernameQuery = `
	SELECT id 
	FROM users 
	WHERE username = $1;
`

// accrual worker queries

const AccrualAddQuery = `
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
const UpdateStatusQuery = `
	UPDATE orders
	SET status = $1
	WHERE number = $2;
`

// SQL queries
const GetBalanceQuery = `
	SELECT COALESCE(SUM(accrual),0)
	FROM operations
	WHERE user_id = (
		SELECT id 
		FROM users 
		WHERE username = $1
		); 
`

const GetWithdrawnQuery = `
	SELECT COALESCE(SUM(accrual),0)
	FROM operations
	WHERE user_id = (
		SELECT id 
		FROM users 
		WHERE username = $1
		)
	AND	accrual < 0; 
`

const GetWithdrawalsQuery = `
	SELECT number, accrual, processed_at
	FROM operations
	WHERE user_id = (
		SELECT id
		FROM users
		WHERE username = $1
		) 
	AND	accrual < 0;
`

const InsertOperationQuery = `
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

const InsertWithdrawOperation = `
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

const getPassQuery = `
		SELECT password
		FROM passwords
		JOIN users ON users.id = passwords.user_id
		WHERE users.username = $1
	`

const getOrdersAccrualStatusUpdQuery = `
	SELECT number
	FROM orders
	WHERE status='NEW' 
		OR status='PROCESSING';
`
