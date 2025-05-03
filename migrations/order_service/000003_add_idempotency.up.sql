-- Проверяем существование таблицы request_ids и создаем, если не существует

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'request_ids') THEN
        CREATE TABLE request_ids (
            id VARCHAR(255) PRIMARY KEY,
            resource VARCHAR(100) NOT NULL,
            resource_id INTEGER,
            user_id INTEGER NOT NULL,
            created_at TIMESTAMP NOT NULL DEFAULT NOW(),
            expires_at TIMESTAMP NOT NULL
        );

        CREATE INDEX idx_request_ids_resource ON request_ids(resource);
        CREATE INDEX idx_request_ids_resource_id ON request_ids(resource_id);
        CREATE INDEX idx_request_ids_user_id ON request_ids(user_id);
        CREATE INDEX idx_request_ids_expires_at ON request_ids(expires_at);

        RAISE NOTICE 'Таблица request_ids создана';
    ELSE
        RAISE NOTICE 'Таблица request_ids уже существует';
    END IF;
END
$$;