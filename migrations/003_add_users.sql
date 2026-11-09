CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);

-- expires_at храним как unix-время (секунды), так проще сравнивать
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL
);

-- у старых трат user_id будет NULL, их никто не увидит
ALTER TABLE expenses ADD COLUMN user_id INTEGER REFERENCES users(id);
