package data

import (
	"database/sql"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/fpawel/dax/internal/dax"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

//go:generate go run github.com/fpawel/gotools/cmd/sqlstr/...

func Open(filename string) (*sqlx.DB, error) {
	db, err := openSqliteDBx(filename)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(SQLCreate); err != nil {
		return nil, err
	}
	if _, err := GetCurrentPartyID(db); err != nil {
		return nil, err
	}
	return db, nil
}

type Party struct {
	PartyInfo
	Products []Product
}

type PartyInfo struct {
	PartyID   int64     `db:"party_id"`
	CreatedAt time.Time `db:"created_at"`
}

type Product struct {
	CreatedAt time.Time `db:"created_at" yaml:"created_at"`
	ProductID int64     `db:"product_id" yaml:"product_id"`
	PartyID   int64     `db:"party_id" yaml:"party_id"`
	Place     int       `db:"place" yaml:"place"`
	Active    bool      `db:"active" yaml:"active"`
	dax.Product
}

func UpdateProduct(db *sqlx.DB, p Product) error {
	r, err := db.NamedExec(`
UPDATE product 
 SET serial1=:serial1,
     serial2=:serial2,
     serial3=:serial3,
     product_type=:product_type,
     year=:year,
     quarter=:quarter,
     fon_minus20=:fon_minus20,
     fon0=:fon0,
     fon20=:fon20,
     fon50=:fon50,
     sens_minus20=:sens_minus20,
     sens0=:sens0,
     sens20=:sens20,
     sens50=:sens50,
     temp_minus20=:temp_minus20,
     temp0=:temp0,
     temp20=:temp20,
     temp50=:temp50
WHERE product_id=:product_id`, p)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return merry.Errorf("ожидалось изменение одной записи: %d: %+v", n, p)
	}
	return nil
}

func GetCurrentPartyID(db *sqlx.DB) (int64, error) {
	var partyID int64
	err := db.Get(&partyID, `SELECT party_id FROM last_party`)
	if err == nil {
		return partyID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	if err := CreateNewParty(db, 240); err != nil {
		return 0, err
	}
	err = db.Get(&partyID, `SELECT party_id FROM last_party`)
	return partyID, err
}

func GetCurrentParty(db *sqlx.DB) (Party, error) {
	partyID, err := GetCurrentPartyID(db)
	if err != nil {
		return Party{}, err
	}
	return GetParty(db, partyID)
}

func CreateNewParty(db *sqlx.DB, productType int) error {
	t := time.Now()
	r, err := db.Exec(`INSERT INTO party (created_at) VALUES (?)`, t)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("excpected 1 rows affected, got %d", n)
	}
	newPartyID, err := getNewInsertedID(r)
	if err != nil {
		return err
	}
	quarter := int(t.Month())/3 + 1
	for i := 0; i < 10; i++ {
		if r, err = db.Exec(`INSERT INTO product(party_id, place, year, quarter, product_type) VALUES (?, ?, ?, ?, ?);`,
			newPartyID, i+1, t.Year()-2000, quarter, productType); err != nil {
			return err
		}
		if _, err = getNewInsertedID(r); err != nil {
			return err
		}
	}
	return nil
}

func GetParty(db *sqlx.DB, partyID int64) (Party, error) {
	var party Party
	err := db.Get(&party, `SELECT * FROM party WHERE party_id=?`, partyID)
	if err != nil {
		return Party{}, err
	}
	if party.Products, err = ListProducts(db, party.PartyID); err != nil {
		return Party{}, err
	}
	return party, err
}

func ListProducts(db *sqlx.DB, partyID int64) (products []Product, err error) {
	err = db.Select(&products, `
SELECT product.*, created_at FROM product
INNER JOIN party USING (party_id)
WHERE party_id=? 
ORDER BY product_id`, partyID)
	return
}

func openSqliteDB(fileName string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite3", fileName)
	if err != nil {
		return nil, err
	}
	conn.SetMaxIdleConns(1)
	conn.SetMaxOpenConns(1)
	conn.SetConnMaxLifetime(0)
	return conn, err
}

func openSqliteDBx(fileName string) (*sqlx.DB, error) {
	conn, err := openSqliteDB(fileName)
	if err != nil {
		return nil, err
	}
	return sqlx.NewDb(conn, "sqlite3"), nil
}

func getNewInsertedID(r sql.Result) (int64, error) {
	id, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, merry.New("was not inserted")
	}
	return id, nil
}
