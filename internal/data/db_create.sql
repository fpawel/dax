PRAGMA foreign_keys = ON;
PRAGMA encoding = 'UTF-8';

CREATE TABLE IF NOT EXISTS party
(
    party_id   INTEGER PRIMARY KEY NOT NULL,
    created_at TIMESTAMP           NOT NULL DEFAULT (DATETIME('now'))
);

CREATE VIEW IF NOT EXISTS last_party AS
SELECT *
FROM party
ORDER BY created_at DESC
LIMIT 1;

CREATE TABLE IF NOT EXISTS product
(
    product_id   INTEGER PRIMARY KEY NOT NULL,
    party_id     INTEGER             NOT NULL,
    place        SMALLINT            NOT NULL,
    serial1      SMALLINT            NOT NULL DEFAULT 0,
    serial2      SMALLINT            NOT NULL DEFAULT 0,
    serial3      SMALLINT            NOT NULL DEFAULT 0,
    product_type SMALLINT            NOT NULL DEFAULT 0,
    year         SMALLINT            NOT NULL DEFAULT 0,
    quarter      SMALLINT            NOT NULL DEFAULT 0,
    fon_minus20  REAL                NOT NULL DEFAULT 0,
    fon0         REAL                NOT NULL DEFAULT 0,
    fon20        REAL                NOT NULL DEFAULT 0,
    fon50        REAL                NOT NULL DEFAULT 0,
    sens_minus20 REAL                NOT NULL DEFAULT 0,
    sens0        REAL                NOT NULL DEFAULT 0,
    sens20       REAL                NOT NULL DEFAULT 0,
    sens50       REAL                NOT NULL DEFAULT 0,
    temp_minus20 REAL                NOT NULL DEFAULT 0,
    temp0        REAL                NOT NULL DEFAULT 0,
    temp20       REAL                NOT NULL DEFAULT 0,
    temp50       REAL                NOT NULL DEFAULT 0,
    CHECK ( place BETWEEN 1 AND 10),
    UNIQUE (party_id, place),
    FOREIGN KEY (party_id) REFERENCES party (party_id) ON DELETE CASCADE
);
