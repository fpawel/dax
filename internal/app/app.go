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
	"sync"
	"time"
)

var (
	log           = structlog.New()
	mainWnd       *walk.MainWindow
	prodsVm       *prodsTblVm
	db            *sqlx.DB
	tmpDir        = filepath.Join(filepath.Dir(os.Args[0]), "tmp")
	gbPartyTitle  *walk.GroupBox
	comportReader comm.T
)

func Main() {
	defer saveConfig()
	defer cleanTmpDir()

	initLog()
	openConfig()

	cleanTmpDir()
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		log.PrintErr(merry.Append(err, "os.RemoveAll(tmpDir)"))
	}

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

	withComportReader := func(workName string, work func(ctx context.Context) error) {
		comPort := comport.NewPort(comport.Config{
			Baud:        115200,
			ReadTimeout: time.Millisecond,
			Name:        config.Comport,
		})
		if err := comPort.Open(); err != nil {
			setStatusError(merry.Append(err, workName))
			return
		}

		setStatusOk(fmt.Sprintf("%s: %s: выполняется", workName, config.Comport))
		panicIf(pbRunInterrogate.SetText("Стоп"))

		wgInterrogate.Add(1)

		var ctx context.Context
		ctx, interruptInterrogate = context.WithCancel(context.Background())

		comportReader = comm.New(comPort, comm.Config{
			TimeoutEndResponse: config.TimeoutEndResponse,
			TimeoutGetResponse: config.TimeoutGetResponse,
			MaxAttemptsRead:    config.MaxAttemptsRead,
			Pause:              config.Pause,
		})

		go func() {
			err := work(ctx)
			wgInterrogate.Done()
			log.ErrIfFail(comPort.Close)
			mainWnd.Synchronize(func() {
				panicIf(pbRunInterrogate.SetText("Опрос"))
				if err != nil {
					setStatusError(merry.New(workName).WithCause(err))
				} else {
					setStatusOk(fmt.Sprintf("%s: выполнено", workName))
				}
			})
		}()
	}

	var menuReadParam []MenuItem

	for _, x := range readParams {
		x := x
		menuReadParam = append(menuReadParam, Action{
			Text: x.S,
			OnTriggered: func() {
				withComportReader(x.S, func(ctx context.Context) error {
					return readParam(x.S, ctx, x.F)
				})

			},
		})
	}

	runWindowMaximized(MainWindow{
		Title:    "Настройка ДАХ-М",
		Font:     Font{Family: "Arial", PointSize: 9},
		AssignTo: &mainWnd,
		MenuItems: []MenuItem{
			Action{
				Text:        "Конфигурация",
				OnTriggered: runEditPartyConfig,
			},
			Action{
				Text: "Новая партия",
				OnTriggered: func() {
					_, err := newPartyDialog.Run(mainWnd)
					panicIf(err)
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
				Text:  "Считать параметр",
				Items: menuReadParam,
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
		Layout:  VBox{},
		MaxSize: Size{0, 380},
		MinSize: Size{0, 380},
		Size:    Size{0, 380},
		Children: []Widget{
			ScrollView{
				VerticalFixed: true,
				Layout:        HBox{},
				Children: []Widget{
					TextLabel{Text: "СОМ порт", MaxSize: Size{80, 0}},
					ComboBoxComport(),
					PushButton{
						AssignTo: &pbRunInterrogate,
						Text:     "Опрос",

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
				MaxSize:  Size{0, 280},
				MinSize:  Size{0, 280},
				Title:    fmt.Sprintf("Текущая загрузка: №%d %s", party.PartyID, party.CreatedAt.Format("02.01.06 15:04")),
				Layout:   Grid{},
				Children: []Widget{
					TableView{
						AssignTo:   &prodsTblView,
						Model:      prodsVm,
						Columns:    prodCols1,
						CheckBoxes: true,
						OnItemActivated: func() {
							if n := prodsTblView.CurrentIndex(); n >= 0 && n < len(prodsVm.xs) {
								runEditProductConfig(prodsVm.xs[n].Product)
							}
						},
					},
				},
			},
			TableView{},
		},
	})
	panicIf(err)
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

func readParam(workName string, ctx context.Context, f func(*prodsTblVmProduct) *float64) error {

	defer mainWnd.Synchronize(uploadLastParty)

	xs, err := modbus.Read3Values(log, ctx, comportReader, config.Addr, 0, 10, modbus.BCD)
	if err == context.Canceled {
		return nil
	}
	if err != nil {
		return err
	}
	for i := range prodsVm.xs {
		x := &prodsVm.xs[i]

		*f(x) = xs[i] * 1000. / config.Rf

		setProductOk(x, fmt.Sprintf("%v: %s", xs[i], workName))
		panicIf(data.UpdateProduct(db, x.Product))
	}
	return nil
}

func readFirmware(ctx context.Context) error {
	for i := range prodsVm.xs {

		x := &prodsVm.xs[i]
		setProductOk(x, "считывание...")

		b, err := dax.ReadFirmware(log, ctx, comportReader, config.Addr, i+1, parseChipType(config.Chip))
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

func writeFirmware(ctx context.Context) error {
	for i := range prodsVm.xs {

		x := &prodsVm.xs[i]

		setProductOk(x, "запись...")

		err := dax.WriteFirmware(log, ctx, comportReader, config.Addr, i+1, parseChipType(config.Chip), x.Product.ToFirmwareBytes())
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

func interrogate(ctx context.Context) error {
	for {
		xs, err := modbus.Read3Values(log, ctx, comportReader, config.Addr, 6, 10, modbus.BCD)
		if merry.Is(err, context.Canceled) {
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

	type appConfig struct {
		App      Config         `yaml:"app"`
		Products []data.Product `yaml:"products"`
	}

	var c appConfig
	c.App = config

	for _, p := range prodsVm.xs {
		c.Products = append(c.Products, p.Product)
	}

	b, err := yaml.Marshal(c)
	panicIf(err)

	filename := filepath.Join(tmpDir, "app-config.yaml")

	panicIf(ioutil.WriteFile(filename, b, 0644))
	cmd := exec.Command("./npp/notepad++.exe", filename)
	panicIf(cmd.Start())

	go func() {
		defer mainWnd.Synchronize(uploadLastParty)
		panicIf(cmd.Wait())
		b, err = ioutil.ReadFile(filename)
		panicIf(err)
		if err := yaml.Unmarshal(b, &c); err != nil {
			showConfigErr(err)
			return
		}

		config = c.App
		comm.SetEnableLog(config.LogComm)

		for i, p := range c.Products {
			prodsVm.xs[i].Product = p
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

func initLog() {
	structlog.DefaultLogger.
		SetPrefixKeys(
			structlog.KeyApp, structlog.KeyPID, structlog.KeyLevel, structlog.KeyUnit, structlog.KeyTime,
		).
		SetDefaultKeyvals(
			structlog.KeyApp, filepath.Base(os.Args[0]),
			structlog.KeySource, structlog.Auto,
		).
		SetSuffixKeys(
			structlog.KeyStack,
		).
		SetSuffixKeys(structlog.KeySource).
		SetKeysFormat(map[string]string{
			structlog.KeyTime:   " %[2]s",
			structlog.KeySource: " %6[2]s",
			structlog.KeyUnit:   " %6[2]s",
		})
}

var readParams = []struct {
	S string
	F func(*prodsTblVmProduct) *float64
}{
	{
		"фоновый ток при -20°С",
		func(p *prodsTblVmProduct) *float64 {
			return &p.FonMinus20
		},
	},
	{
		"фоновый ток при 0°С",
		func(p *prodsTblVmProduct) *float64 {
			return &p.Fon0
		},
	},
	{
		"фоновый ток при +20°С",
		func(p *prodsTblVmProduct) *float64 {
			return &p.Fon20
		},
	},
	{
		"фоновый ток при +50°С",
		func(p *prodsTblVmProduct) *float64 {
			return &p.Fon50
		},
	},
	{
		"чувствительность при -20°С",
		func(p *prodsTblVmProduct) *float64 {
			return &p.SensMinus20
		},
	},
	{
		"чувствительность при 0°С",
		func(p *prodsTblVmProduct) *float64 {
			return &p.Sens0
		},
	},
	{
		"чувствительность при +20°С",
		func(p *prodsTblVmProduct) *float64 {
			return &p.Sens20
		},
	},
	{
		"чувствительность при +50°С",
		func(p *prodsTblVmProduct) *float64 {
			return &p.Sens50
		},
	},
}
