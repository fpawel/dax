package data

const SQLCreate = `
PRAGMA foreign_keys = ON;
PRAGMA encoding = 'UTF-8';

CREATE TABLE IF NOT EXISTS party
(
    party_id   INTEGER PRIMARY KEY NOT NULL,
    created_at TIMESTAMP           NOT NULL DEFAULT (DATETIME('now', '+3 hours'))
);

CREATE TABLE IF NOT EXISTS product
(
    product_id   INTEGER PRIMARY KEY NOT NULL,
    party_id     INTEGER             NOT NULL,
    place        SMALLINT            NOT NULL,

    UNIQUE (party_id, place),
    UNIQUE (party_id, serial),
    FOREIGN KEY (party_id) REFERENCES party (party_id) ON DELETE CASCADE,

    serial       SMALLINT            NOT NULL,
    product_type SMALLINT            NOT NULL,
    year         SMALLINT            NOT NULL,
    quoter       SMALLINT            NOT NULL,
    fonMinus20   REAL                NOT NULL,
    fon0         REAL                NOT NULL,
    fon20        REAL                NOT NULL,
    fon50        REAL                NOT NULL,
    sensMinus20  REAL                NOT NULL,
    sens0        REAL                NOT NULL,
    sens20       REAL                NOT NULL,
    sens50       REAL                NOT NULL,
    tempMinus20  REAL                NOT NULL,
    temp0        REAL                NOT NULL,
    temp20       REAL                NOT NULL,
    temp50       REAL                NOT NULL
);
`
