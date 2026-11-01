CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

INSERT INTO categories (name) VALUES
    ('Еда'),
    ('Транспорт'),
    ('Жильё'),
    ('Развлечения'),
    ('Здоровье'),
    ('Другое');

ALTER TABLE expenses ADD COLUMN category_id INTEGER REFERENCES categories(id);

UPDATE expenses SET category_id = (SELECT id FROM categories WHERE name = 'Другое');
