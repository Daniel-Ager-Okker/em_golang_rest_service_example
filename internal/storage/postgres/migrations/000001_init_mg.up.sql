CREATE TABLE IF NOT EXISTS subscription(
    id SERIAL PRIMARY KEY,
    service_name TEXT NOT NULL,
    price INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    CONSTRAINT unique_subscription UNIQUE (service_name, user_id),
    CONSTRAINT check_end_after_start CHECK (end_date > start_date)
);