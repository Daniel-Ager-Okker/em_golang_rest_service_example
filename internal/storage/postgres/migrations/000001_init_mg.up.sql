CREATE TABLE IF NOT EXISTS subscription(
    id SERIAL PRIMARY KEY,
    service_name TEXT NOT NULL,
    price INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    start_date TEXT NOT NULL CHECK (
        start_date ~ '^[0-9]{2}-[0-9]{4}$' AND
        CAST(SUBSTRING(start_date FROM 1 FOR 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    end_date TEXT NOT NULL CHECK (
        end_date ~ '^[0-9]{2}-[0-9]{4}$' AND
        CAST(SUBSTRING(end_date FROM 1 FOR 2) AS INTEGER) BETWEEN 1 AND 12
    ),
    CONSTRAINT unique_subscription UNIQUE (service_name, user_id),
    CONSTRAINT check_end_after_start 
        CHECK (
            -- sneaky trick (convert 'MM-YYYY' to 'YYYYMM' and compare integers)
            (
                CAST(SUBSTRING(end_date FROM 4) AS INTEGER) * 100 + 
                CAST(SUBSTRING(end_date FROM 1 FOR 2) AS INTEGER)
            ) >
            (
                CAST(SUBSTRING(start_date FROM 4) AS INTEGER) * 100 + 
                CAST(SUBSTRING(start_date FROM 1 FOR 2) AS INTEGER)
            )
        )
);