CREATE TABLE records (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    zone_id     INTEGER NOT NULL,
    domain      VARCHAR(255) NOT NULL,         -- полное имя, например appname.v95.
    type     VARCHAR(10) NOT NULL,          -- A, AAAA, CNAME, MX, TXT, NS и т.д.
    rdata       TEXT NOT NULL,                  -- значение: IP, имя, строка
    ttl         INTEGER,                        -- NULL = использовать TTL зоны
    FOREIGN KEY (zone_id) REFERENCES zones(id)
);

CREATE INDEX idx_records_domain_type ON records(domain, type);
