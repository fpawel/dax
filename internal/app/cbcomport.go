package app

import (
	"github.com/fpawel/comm/comport"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var (
	ComboBoxComport = func() ComboBox {

		var cb *walk.ComboBox

		getComports := func() []string {
			ports, _ := comport.Ports()
			return ports
		}

		comportIndex := func(portName string) int {
			ports, _ := comport.Ports()
			for i, s := range ports {
				if s == portName {
					return i
				}
			}
			return -1
		}

		return ComboBox{
			AssignTo:     &cb,
			MaxSize:      Size{100, 0},
			Model:        getComports(),
			CurrentIndex: comportIndex(setsComportName()),
			OnMouseDown: func(_, _ int, _ walk.MouseButton) {
				n := cb.CurrentIndex()
				_ = cb.SetModel(getComports())
				_ = cb.SetCurrentIndex(n)
			},
			OnCurrentIndexChanged: func() {
				panicIf(sets.Put("comport", cb.Text()))
			},
		}
	}()
)
