CREATE TABLE zones (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        VARCHAR(63) NOT NULL UNIQUE,   -- zonename. (с точкой)
    ttl         INTEGER DEFAULT 300
);
CREATE UNIQUE INDEX uniq_zone_name ON zones (name)