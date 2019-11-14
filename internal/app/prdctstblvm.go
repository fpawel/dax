package app

import (
	"fmt"
	"github.com/fpawel/dax/internal/data"
	"github.com/jmoiron/sqlx"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var _ walk.TableModel = new(prodsTblVm)

type prodsTblVm struct {
	walk.TableModelBase
	db *sqlx.DB
	xs []data.Product
	vs []string
}

func (x *prodsTblVm) Upload() {
	p, err := data.GetCurrentParty(x.db)
	panicIf(err)
	x.xs = p.Products
	x.vs = make([]string, 10)
}

func (x *prodsTblVm) RowCount() int {
	return 10
}
func (x *prodsTblVm) Value(row, col int) interface{} {
	p := x.xs[row]
	return prodCols[col].F(p)
}

var prodCols = []prodCol{
	{
		C: TableViewColumn{Name: "№", Width: 40},
		F: func(p data.Product) interface{} {
			return p.Place
		},
	},
	{
		C: TableViewColumn{Name: "ДАХ", Width: 40},
		F: func(p data.Product) interface{} {
			return p.ProductID
		},
	},
	{
		C: TableViewColumn{Name: "сер.№"},
		F: func(p data.Product) interface{} {
			return fmt.Sprintf("%d%d%d", p.Serial1, p.Serial2, p.Serial3)
		},
	},
	{
		C: TableViewColumn{Name: "Год", Width: 40},
		F: func(p data.Product) interface{} {
			return p.Year
		},
	},
	{
		C: TableViewColumn{Name: "Квартал", Width: 40},
		F: func(p data.Product) interface{} {
			return p.Quarter
		},
	},
	{
		C: TableViewColumn{Name: "Тип"},
		F: func(p data.Product) interface{} {
			return p.ProductType
		},
	},
	{
		C: TableViewColumn{Name: "Фон -20°С"},
		F: func(p data.Product) interface{} {
			return p.FonMinus20
		},
	},
	{
		C: TableViewColumn{Name: "Фон 0°С"},
		F: func(p data.Product) interface{} {
			return p.Fon0
		},
	},
	{
		C: TableViewColumn{Name: "Фон +20°С"},
		F: func(p data.Product) interface{} {
			return p.Fon20
		},
	},
	{
		C: TableViewColumn{Name: "Фон +50°С"},
		F: func(p data.Product) interface{} {
			return p.Fon50
		},
	},
	{
		C: TableViewColumn{Name: "Ч -20°С"},
		F: func(p data.Product) interface{} {
			return p.SensMinus20
		},
	},
	{
		C: TableViewColumn{Name: "Ч 0°С"},
		F: func(p data.Product) interface{} {
			return p.Sens0
		},
	},
	{
		C: TableViewColumn{Name: "Ч +20°С"},
		F: func(p data.Product) interface{} {
			return p.Sens20
		},
	},
	{
		C: TableViewColumn{Name: "Ч +50°С"},
		F: func(p data.Product) interface{} {
			return p.Sens50
		},
	},
	{
		C: TableViewColumn{Name: "Т -20°С"},
		F: func(p data.Product) interface{} {
			return p.TempMinus20
		},
	},
	{
		C: TableViewColumn{Name: "Т 0°С"},
		F: func(p data.Product) interface{} {
			return p.Temp0
		},
	},
	{
		C: TableViewColumn{Name: "Т +20°С"},
		F: func(p data.Product) interface{} {
			return p.Temp20
		},
	},
	{
		C: TableViewColumn{Name: "Т +50°С"},
		F: func(p data.Product) interface{} {
			return p.Temp50
		},
	},
}

type prodCol struct {
	C TableViewColumn
	F func(p data.Product) interface{}
}
