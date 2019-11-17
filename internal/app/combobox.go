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
			CurrentIndex: comportIndex(config.Comport),
			OnMouseDown: func(_, _ int, _ walk.MouseButton) {
				n := cb.CurrentIndex()
				_ = cb.SetModel(getComports())
				_ = cb.SetCurrentIndex(n)
			},
			OnCurrentIndexChanged: func() {
				config.Comport = cb.Text()
			},
		}
	}()

	ComboBoxChip = func() ComboBox {

		var cb *walk.ComboBox

		model := []string{"24LC16", "24LC64", "24W256"}

		getIdx := func(x string) int {
			for i, s := range model {
				if s == x {
					return i
				}
			}
			return -1
		}

		return ComboBox{
			AssignTo:     &cb,
			MaxSize:      Size{100, 0},
			Model:        model,
			CurrentIndex: getIdx(config.Chip),
			OnCurrentIndexChanged: func() {
				config.Chip = cb.Text()
				config.Chip = cb.Text()
			},
		}
	}()
)
