create table users_tokens (
	id SERIAL PRIMARY KEY,
	user_id INTEGER,
	access_token TEXT,
	refresh_token TEXT,
	FOREIGN KEY (user_id) REFERENCES users (id)
);