CREATE TABLE IF NOT EXISTS deliveries (
                                          id BIGSERIAL PRIMARY KEY,
                                          address TEXT NOT NULL CHECK (btrim(address) <> ''),
    status INTEGER NOT NULL CHECK (status BETWEEN 1 AND 4)
    );