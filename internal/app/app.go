//todo Добавить дату прошивки
//todo Просмотр таблицы архива в отдельной вкладке
//todo Фильтр таблицы архива по серийному номеру

package app

import (
	"fmt"
	"github.com/ansel1/merry"
	"github.com/fpawel/dax/internal/data"
	"github.com/jmoiron/sqlx"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"github.com/powerman/structlog"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	log     = structlog.New()
	mainWnd *walk.MainWindow
	prodsVm *prodsTblVm
	db      *sqlx.DB
	tmpDir  = filepath.Join(filepath.Dir(os.Args[0]), "tmp")
)

func Main() {

	cleanTmpDir()
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		log.PrintErr(merry.Append(err, "os.RemoveAll(tmpDir)"))
	}
	defer cleanTmpDir()

	dbFilename := filepath.Join(filepath.Dir(os.Args[0]), "dax.sqlite")
	log.Debug("open database: " + dbFilename)
	var err error
	db, err = data.Open(dbFilename)
	panicIf(err)
	defer log.ErrIfFail(db.Close)

	prodsVm = &prodsTblVm{db: db}
	prodsVm.Upload()

	var prodCols1 []TableViewColumn
	for _, x := range prodCols {
		prodCols1 = append(prodCols1, x.C)
	}

	var prodsTblView *walk.TableView

	runWindowMaximized(MainWindow{
		Title:    "Настройка ДАХ-М",
		MinSize:  Size{600, 400},
		Font:     Font{Family: "Arial", PointSize: 9},
		AssignTo: &mainWnd,

		MenuItems: []MenuItem{
			Menu{
				Text: "Партия",
				Items: []MenuItem{
					Action{
						Text:        "Конфигурация",
						OnTriggered: runEditPartyConfig,
					},
				},
			},
			Menu{
				Text: "Память",
				Items: []MenuItem{
					Action{
						Text: "Записать",
					},
					Action{
						Text: "Считать",
					},
				},
			},
			Menu{
				Text: "Считать параметр",
				Items: []MenuItem{
					Action{
						Text: "фоновый ток при -20°С",
					},
					Action{
						Text: "фоновый ток при 0°С",
					},
					Action{
						Text: "фоновый ток при +20°С",
					},
					Action{
						Text: "фоновый ток при +50°С",
					},
					Action{
						Text: "чувствительность при -20°С",
					},
					Action{
						Text: "чувствительность при 0°С",
					},
					Action{
						Text: "чувствительность при +20°С",
					},
					Action{
						Text: "чувствительность при +50°С",
					},
				},
			},
		},
		Layout: VBox{},
		Children: []Widget{
			ScrollView{
				VerticalFixed: true,
				Layout:        HBox{},
				Children: []Widget{
					TextLabel{Text: "СОМ порт", MaxSize: Size{80, 0}},
					ComboBox{
						MaxSize: Size{100, 0},
					},
				},
			},
			TableView{
				AssignTo: &prodsTblView,
				Model:    prodsVm,
				Columns:  prodCols1,
				OnItemActivated: func() {
					if n := prodsTblView.CurrentIndex(); n >= 0 && n < len(prodsVm.xs) {
						runEditProductConfig(prodsVm.xs[n])
					}
				},
			},
		},
	})
}

func runWindowMaximized(aw MainWindow) {
	if aw.AssignTo == nil {
		var x *walk.MainWindow
		aw.AssignTo = &x
	}
	panicIf(aw.Create())
	w := *aw.AssignTo
	if !win.ShowWindow(w.Handle(), win.SW_SHOWMAXIMIZED) {
		panic("can`t show window")
	}
	w.Run()
}

func runEditPartyConfig() {
	b, err := yaml.Marshal(prodsVm.xs)
	panicIf(err)

	filename := filepath.Join(tmpDir, fmt.Sprintf("products%d.yaml", prodsVm.xs[0].PartyID))

	panicIf(ioutil.WriteFile(filename, b, 0644))
	cmd := exec.Command("./npp/notepad++.exe", filename)
	panicIf(cmd.Start())

	go func() {
		defer mainWnd.Synchronize(prodsVm.Upload)
		panicIf(cmd.Wait())
		b, err = ioutil.ReadFile(filename)
		panicIf(err)
		if err := yaml.Unmarshal(b, &prodsVm.xs); err != nil {
			showConfigErr(err)
			return
		}
		for _, p := range prodsVm.xs {
			if err := data.UpdateProduct(db, p); err != nil {
				showConfigErr(err)
				return
			}
		}
	}()
}

func runEditProductConfig(p data.Product) {

	b, err := yaml.Marshal(p.Product)
	panicIf(err)

	filename := filepath.Join(tmpDir, fmt.Sprintf("product%d.yaml", p.ProductID))

	panicIf(ioutil.WriteFile(filename, b, 0644))
	cmd := exec.Command("./npp/notepad++.exe", filename)
	panicIf(cmd.Start())

	go func() {
		defer mainWnd.Synchronize(prodsVm.Upload)
		panicIf(cmd.Wait())
		b, err = ioutil.ReadFile(filename)
		panicIf(err)
		if err := yaml.Unmarshal(b, &p.Product); err != nil {
			showConfigErr(err)
			return
		}
		if err := data.UpdateProduct(db, p); err != nil {
			showConfigErr(err)
			return
		}
	}()
}
func showConfigErr(err error) {
	mainWnd.Synchronize(func() {
		walk.MsgBox(mainWnd, "Ошибка ввода конфигурации", err.Error(), walk.MsgBoxIconError|walk.MsgBoxOK)
	})
}

func cleanTmpDir() {
	if err := os.RemoveAll(tmpDir); err != nil {
		log.PrintErr(merry.Append(err, "os.RemoveAll(tmpDir)"))
	}
}
