package database

// SQL queries
// database init queries
var checkTableExistsQuery = `
	SELECT EXISTS (
		SELECT 	1 
		FROM 	information_schema.tables
		WHERE 	table_name = $1
	);
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

var getPassQuery = `
		SELECT password
		FROM passwords
		JOIN users ON users.id = passwords.user_id
		WHERE users.username = $1
	`

var getOrdersAccrualStatusUpdQuery = `
	SELECT number
	FROM orders
	WHERE status='NEW' 
		OR status='PROCESSING';
`
