//todo настройки приёмопередачи - редактировать и сохранять в файде настроек приложения
//todo добавит комбобок с выбором типа микросхемы
//todo ограничить высоту таблицы
//todo считывание параметров ДАХ из показаний и сохранение в БД

package app

import (
	"context"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/fpawel/comm"
	"github.com/fpawel/comm/comport"
	"github.com/fpawel/comm/modbus"
	"github.com/fpawel/dax/internal/data"
	"github.com/fpawel/dax/internal/dax"
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
	"reflect"
	"strings"
	"sync"
	"time"
)

var (
	log          = structlog.New()
	mainWnd      *walk.MainWindow
	prodsVm      *prodsTblVm
	db           *sqlx.DB
	tmpDir       = filepath.Join(filepath.Dir(os.Args[0]), "tmp")
	gbPartyTitle *walk.GroupBox
)

func Main() {

	log.Debug(sets.FilePath())

	defer func() {
		panicIf(sets.Save())
	}()

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

	prodsVm = new(prodsTblVm)

	party, err := data.GetCurrentParty(db)
	panicIf(err)
	prodsVm.SetProducts(party.Products)

	var prodCols1 []TableViewColumn
	for _, x := range prodCols {
		prodCols1 = append(prodCols1, x.C)
	}

	var archProdCols1 []TableViewColumn
	for _, x := range archProdCols {
		archProdCols1 = append(archProdCols1, x.C)
	}

	var (
		prodsTblView         *walk.TableView
		pbRunInterrogate     *walk.PushButton
		interruptInterrogate = func() {}
		wgInterrogate        sync.WaitGroup
		lblStatus            *walk.LineEdit
	)

	setStatus := func(s string, c walk.Color) {
		lblStatus.SetTextColor(c)
		panicIf(lblStatus.SetText(fmt.Sprintf("%s: %s", time.Now().Format("15:04:05"), s)))
	}
	setStatusError := func(err error) {
		setStatus(err.Error(), walk.RGB(255, 0, 0))
	}
	setStatusOk := func(s string) {
		setStatus(s, walk.RGB(0, 0, 0))
	}

	withComportReader := func(workName string, work func(modbus.ResponseReader, context.Context) error) {
		comPort := comport.NewPort(comport.Config{
			Baud:        115200,
			ReadTimeout: time.Millisecond,
			Name:        setsComportName(),
		})
		if err := comPort.Open(); err != nil {
			setStatusError(merry.Append(err, workName))
			return
		}

		setStatusOk(fmt.Sprintf("%s: %s: выполняется", workName, setsComportName()))
		panicIf(pbRunInterrogate.SetText("Стоп"))

		wgInterrogate.Add(1)

		var ctx context.Context
		ctx, interruptInterrogate = context.WithCancel(context.Background())

		comPortReader := comPort.NewResponseReader(ctx, comm.Config{
			TimeoutEndResponse: 50 * time.Millisecond,
			TimeoutGetResponse: time.Second,
			MaxAttemptsRead:    3,
		})

		go func() {
			err := work(comPortReader, ctx)
			wgInterrogate.Done()
			log.ErrIfFail(comPort.Close)
			mainWnd.Synchronize(func() {
				panicIf(pbRunInterrogate.SetText("Старт"))
				if err != nil {
					setStatusError(merry.New(workName).WithCause(err))
				} else {
					setStatusOk(fmt.Sprintf("%s: выполнено", workName))
				}
			})
		}()
	}

	runWindowMaximized(MainWindow{
		Title:    "Настройка ДАХ-М",
		MinSize:  Size{600, 400},
		Font:     Font{Family: "Arial", PointSize: 9},
		AssignTo: &mainWnd,

		MenuItems: []MenuItem{
			Menu{
				Text: "Загрузка",
				Items: []MenuItem{
					Action{
						Text: "Создать новую",
						OnTriggered: func() {
							_, err := newPartyDialog.Run(mainWnd)
							panicIf(err)
						},
					},
					Action{
						Text:        "Конфигурация",
						OnTriggered: runEditPartyConfig,
					},
					Action{
						Text: "Архив",
						OnTriggered: func() {
							mainWnd.SetVisible(false)
							runWindowMaximized(MainWindow{
								Title:  "Архив ДАХ",
								Font:   Font{Family: "Arial", PointSize: 9},
								Layout: Grid{},
								Children: []Widget{
									TableView{
										Model:   newArchProdsTblVm(),
										Columns: archProdCols1,
									},
								},
							})

							mainWnd.SetVisible(true)
						},
					},
				},
			},
			Menu{
				Text: "Память",
				Items: []MenuItem{
					Action{
						Text: "Записать",
						OnTriggered: func() {
							withComportReader("запись", writeFirmware)
						},
					},
					Action{
						Text: "Считать",
						OnTriggered: func() {
							withComportReader("считывание", readFirmware)
						},
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
					ComboBoxComport,
					PushButton{
						AssignTo: &pbRunInterrogate,
						Text:     "Старт",

						OnClicked: func() {
							if pbRunInterrogate.Text() == "Стоп" {
								interruptInterrogate()
								wgInterrogate.Wait()
								return
							}
							withComportReader("Опрос", interrogate)
						},
					},
					LineEdit{Text: " ", AssignTo: &lblStatus, ReadOnly: true},
				},
			},
			GroupBox{
				AssignTo: &gbPartyTitle,
				Title:    fmt.Sprintf("Текущая загрузка: №%d %s", party.PartyID, party.CreatedAt.Format("02.01.06 15:04")),
				Layout:   Grid{},
				Children: []Widget{
					TableView{
						AssignTo: &prodsTblView,
						Model:    prodsVm,
						Columns:  prodCols1,
						OnItemActivated: func() {
							if n := prodsTblView.CurrentIndex(); n >= 0 && n < len(prodsVm.xs) {
								runEditProductConfig(prodsVm.xs[n].Product)
							}
						},
					},
				},
			},
		},
	})
}

func uploadLastParty() {
	p, err := data.GetCurrentParty(db)
	panicIf(err)
	panicIf(gbPartyTitle.SetTitle(fmt.Sprintf("Текущая загрузка: №%d %s", p.PartyID, p.CreatedAt.Format("02.01.06 15:04"))))
	prodsVm.SetProducts(p.Products)
}

func setProductOk(x *prodsTblVmProduct, s string) {
	x.Indication = s
	x.ErrOccurred = false
	mainWnd.Synchronize(prodsVm.PublishRowsReset)
}
func setProductErr(x *prodsTblVmProduct, err error) {
	x.Indication = err.Error()
	x.ErrOccurred = true
	mainWnd.Synchronize(prodsVm.PublishRowsReset)
}

func readFirmware(reader modbus.ResponseReader, _ context.Context) error {
	for i := range prodsVm.xs {

		x := &prodsVm.xs[i]
		setProductOk(x, "считывание...")

		b, err := dax.ReadFirmware(log, reader, 101, i+1, dax.Chip256)
		if err == nil {
			x.Product.PutFirmwareBytes(b)
			setProductOk(x, "считано")
			continue
		}
		if merry.Is(err, context.Canceled) {
			setProductOk(x, "прервано")
			return nil
		}
		setProductErr(x, err)
	}
	return nil
}

func writeFirmware(reader modbus.ResponseReader, _ context.Context) error {
	for i := range prodsVm.xs {

		x := &prodsVm.xs[i]

		setProductOk(x, "запись...")

		err := dax.WriteFirmware(log, reader, 101, i+1, dax.Chip256, x.Product.ToFirmwareBytes())
		if err == nil {
			setProductOk(x, "записано")
			continue
		}
		if merry.Is(err, context.Canceled) {
			setProductOk(x, "прервано")
			return nil
		}
		setProductErr(x, err)
	}
	return nil
}

func interrogate(reader modbus.ResponseReader, ctx context.Context) error {
	for {
		xs, err := modbus.Read3BCDs(log, reader, 101, 0, 10)
		if err == context.Canceled {
			return nil
		}
		if err != nil {
			return err
		}
		for i, x := range xs {
			prodsVm.xs[i].Indication = fmt.Sprintf("%v", x)
		}
		mainWnd.Synchronize(prodsVm.PublishRowsReset)
	}
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
		defer mainWnd.Synchronize(uploadLastParty)
		panicIf(cmd.Wait())
		b, err = ioutil.ReadFile(filename)
		panicIf(err)
		if err := yaml.Unmarshal(b, &prodsVm.xs); err != nil {
			showConfigErr(err)
			return
		}
		for _, p := range prodsVm.xs {
			if err := data.UpdateProduct(db, p.Product); err != nil {
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
		defer mainWnd.Synchronize(uploadLastParty)
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

func showArchive(db *sqlx.DB) {
	var xs []data.Product
	panicIf(db.Select(&xs, `
SELECT created_at, product.* FROM product
INNER JOIN party USING (party_id)
ORDER BY created_at DESC, place`))

	filename := filepath.Join(tmpDir, "архив.txt")
	file, err := os.Create(filename)
	panicIf(err)
	writeStr := func(s string) {
		_, err := file.WriteString(s)
		panicIf(err)
	}
	writeStr(strings.Join([]string{
		"product_id",
		"product_type",
		"serial1",
		"serial2",
		"serial3",
		"year",
		"quarter",
		"place",
		"fon_minus20",
		"fon0",
		"fon_minus20",
		"fon0",
		"fon20",
		"fon50",
		"sens_minus20",
		"sens0",
		"sens20",
		"sens50",
		"temp_minus20",
		"temp0",
		"party_id",
		"created_at",
	}, "\t") + "\n")

	typeProduct := reflect.TypeOf(data.Product{})
	// Iterate over all available fields and read the tag value
	for i := 0; i < typeProduct.NumField(); i++ {
		writeStr(typeProduct.Field(i).Name + "\t")
	}
	writeStr("\n")
	for _, x := range xs {
		v := reflect.ValueOf(x)
		for i := 0; i < typeProduct.NumField(); i++ {
			writeStr(v.Field(i).String() + "\t")
		}
		writeStr("\n")
	}
	panicIf(file.Close())
	panicIf(exec.Command("./npp/notepad++.exe", filename).Start())
}

var newPartyDialog = func() Dialog {
	var (
		ed  *walk.NumberEdit
		dlg *walk.Dialog
	)
	return Dialog{
		AssignTo: &dlg,
		Title:    "Создать новую партию",
		MinSize:  Size{300, 100},
		MaxSize:  Size{300, 100},
		Size:     Size{300, 100},
		Layout:   HBox{},
		Children: []Widget{
			TextLabel{Text: "Исполнение новой партии:"},
			NumberEdit{Value: 240, MinValue: 0, AssignTo: &ed, Decimals: 0},
			PushButton{
				Text: "Создать",
				OnClicked: func() {
					dlg.Accept()
					panicIf(data.CreateNewParty(db, int(ed.Value())))
					uploadLastParty()
				},
			},
		},
	}
}()

func showConfigErr(err error) {
	mainWnd.Synchronize(func() {
		walk.MsgBox(mainWnd, "Ошибка ввода конфигурации", err.Error(), walk.MsgBoxIconError|walk.MsgBoxOK)
	})
}
