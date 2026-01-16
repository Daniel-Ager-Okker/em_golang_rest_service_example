CREATE TABLE IF NOT EXISTS subscription(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_name TEXT NOT NULL,
    price INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    start_date TEXT NOT NULL CHECK (
        start_date GLOB '[0-9][0-9]-[0-9][0-9][0-9][0-9]' AND
        CAST(substr(start_date, 1, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    end_date TEXT NOT NULL CHECK (
        start_date GLOB '[0-9][0-9]-[0-9][0-9][0-9][0-9]' AND
        CAST(substr(start_date, 1, 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    CONSTRAINT unique_subscription UNIQUE (service_name, user_id),
    CONSTRAINT check_end_after_start 
        CHECK (
            -- sneaky trick (convert 'MM-YYYY' to 'YYYYMM' and compare integers)
            (
                CAST(substr(end_date, 4, 4) AS INTEGER) * 100 + 
                CAST(substr(end_date, 1, 2) AS INTEGER)
            ) >
            (
                CAST(substr(start_date, 4, 4) AS INTEGER) * 100 + 
                CAST(substr(start_date, 1, 2) AS INTEGER)
            )
        )
);