package app

import "C"
import (
	"fmt"
	"github.com/fpawel/dax/internal/data"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"strconv"
)

var _ walk.TableModel = new(prodsTblVm)

type prodsTblVm struct {
	walk.TableModelBase
	xs []prodsTblVmProduct
}

type prodsTblVmProduct struct {
	data.Product
	Indication  string
	ErrOccurred bool
}

func (x *prodsTblVm) SetProducts(xs []data.Product) {
	x.xs = nil
	for _, p := range xs {
		x.xs = append(x.xs, prodsTblVmProduct{Product: p})
	}
	x.PublishRowsReset()
}

func (x *prodsTblVm) Checked(n int) bool {
	return x.xs[n].Active
}

func (x *prodsTblVm) SetChecked(n int, checked bool) error {
	x.xs[n].Active = checked
	db.MustExec(`UPDATE product SET active = ? WHERE product_id = ?`, checked, x.xs[n].ProductID)
	return nil
}

func (x *prodsTblVm) RowCount() int {
	return 10
}
func (x *prodsTblVm) Value(row, col int) interface{} {
	p := x.xs[row]
	return prodCols[col].F(p)
}

func (x *prodsTblVm) StyleCell(s *walk.CellStyle) {
	if s.Col() < 0 || s.Col() >= len(prodCols) {
		return
	}
	//col := prodCols[s.Col()]
	if s.Col() == 2 && x.xs[s.Row()].ErrOccurred {
		s.TextColor = walk.RGB(255, 0, 0)
		s.BackgroundColor = walk.RGB(220, 220, 220)
	}
}

type archProdsTblVm struct {
	walk.TableModelBase
	xs []prodsTblVmProduct
}

func newArchProdsTblVm() *archProdsTblVm {
	x := new(archProdsTblVm)
	var xs []data.Product
	panicIf(db.Select(&xs, `
SELECT created_at, product.* FROM product
INNER JOIN party USING (party_id)
ORDER BY created_at DESC, place`))

	x.xs = nil
	for _, p := range xs {
		x.xs = append(x.xs, prodsTblVmProduct{Product: p})
	}
	return x
}

func (x *archProdsTblVm) RowCount() int {
	return len(x.xs)
}

func (x *archProdsTblVm) Value(row, col int) interface{} {
	p := x.xs[row]
	return archProdCols[col].F(p)
}

var archProdCols = func() []prodCol {
	xs := []prodCol{
		{
			C: TableViewColumn{Name: "Дата", Width: 80, Format: "02.01.06 15:04"},
			F: func(p prodsTblVmProduct) interface{} {
				return p.CreatedAt
			},
		},
		{
			C: TableViewColumn{Name: "Загрузка", Width: 80},
			F: func(p prodsTblVmProduct) interface{} {
				return p.PartyID
			},
		},
	}
	xs = append(xs, prodCols[0:2]...)
	xs = append(xs, prodCols[3:]...)
	return xs
}()

var prodCols = func() []prodCol {
	return []prodCol{
		{
			C: TableViewColumn{Name: "Место", Width: 40},
			F: func(p prodsTblVmProduct) interface{} {
				return p.Place
			},
		},
		{
			C: TableViewColumn{Name: "Сер.№"},
			F: func(p prodsTblVmProduct) interface{} {
				return fmt.Sprintf("%d%d%d", p.Serial1, p.Serial2, p.Serial3)
			},
		},
		{
			C: TableViewColumn{Name: "U,мВ"},
			F: func(p prodsTblVmProduct) interface{} {
				return p.Indication
			},
		},
		{
			C: TableViewColumn{Name: "I,мкА"},
			F: func(p prodsTblVmProduct) interface{} {
				v, err := strconv.ParseFloat(p.Indication, 64)
				if err != nil {
					return ""
				}
				return v / config.Rf
			},
		},
		{
			C: TableViewColumn{Name: "Год", Width: 40},
			F: func(p prodsTblVmProduct) interface{} {
				return p.Year
			},
		},
		{
			C: TableViewColumn{Name: "Квартал", Width: 40},
			F: func(p prodsTblVmProduct) interface{} {
				return p.Quarter
			},
		},
		{
			C: TableViewColumn{Name: "Тип"},
			F: func(p prodsTblVmProduct) interface{} {
				return p.ProductType
			},
		},
		{
			C: TableViewColumn{Name: "Фон -20°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.FonMinus20)
			},
		},
		{
			C: TableViewColumn{Name: "Фон 0°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.Fon0)
			},
		},
		{
			C: TableViewColumn{Name: "Фон +20°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.Fon20)
			},
		},
		{
			C: TableViewColumn{Name: "Фон +50°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.Fon50)
			},
		},
		{
			C: TableViewColumn{Name: "Ч -20°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.SensMinus20)
			},
		},
		{
			C: TableViewColumn{Name: "Ч 0°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.Sens0)
			},
		},
		{
			C: TableViewColumn{Name: "Ч +20°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.Sens20)
			},
		},
		{
			C: TableViewColumn{Name: "Ч +50°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.Sens50)
			},
		},
		{
			C: TableViewColumn{Name: "Т -20°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.TempMinus20)
			},
		},
		{
			C: TableViewColumn{Name: "Т 0°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.Temp0)
			},
		},
		{
			C: TableViewColumn{Name: "Т +20°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.Temp20)
			},
		},
		{
			C: TableViewColumn{Name: "Т +50°С"},
			F: func(p prodsTblVmProduct) interface{} {
				return formatFloat(p.Temp50)
			},
		},
	}
}()

type prodCol struct {
	C TableViewColumn
	F func(p prodsTblVmProduct) interface{}
}
