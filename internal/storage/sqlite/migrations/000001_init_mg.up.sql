CREATE TABLE IF NOT EXISTS subscription(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_name TEXT NOT NULL,
    price INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    start_date TEXT NOT NULL CHECK (
        -- Check ISO date format YYYY-MM-DD
        start_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND
        
        -- Year
        CAST(substr(start_date, 1, 4) AS INTEGER) BETWEEN 2000 AND 2100 AND
        
        -- Month
        CAST(substr(start_date, 6, 2) AS INTEGER) BETWEEN 1 AND 12 AND
        
        -- Day
        CAST(substr(start_date, 9, 2) AS INTEGER) BETWEEN 1 AND 31
    ),
    end_date TEXT NOT NULL CHECK (
        -- Check ISO date format YYYY-MM-DD
        end_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND
        CAST(substr(end_date, 1, 4) AS INTEGER) BETWEEN 2000 AND 2100 AND
        CAST(substr(end_date, 6, 2) AS INTEGER) BETWEEN 1 AND 12 AND
        CAST(substr(end_date, 9, 2) AS INTEGER) BETWEEN 1 AND 31
    ),
    CONSTRAINT unique_subscription UNIQUE (service_name, user_id),
    CONSTRAINT check_end_after_start CHECK (end_date > start_date)
);