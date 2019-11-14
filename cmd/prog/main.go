package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/fpawel/comm"
	"github.com/fpawel/comm/comport"
	"github.com/fpawel/comm/modbus"
	"github.com/fpawel/dax/internal/dax"
	"github.com/pelletier/go-toml"
	"github.com/powerman/structlog"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func main() {
	log := structlog.New()

	type Conf struct {
		Comport string       `toml:"comport" comment:"СОМ порт, к которому подключен стенд"`
		Addr    modbus.Addr  `toml:"addr" comment:"адрес MODBUS стенда"`
		Chip    dax.ChipType `toml:"chip" comment:"тип микросхемы датчика ДАХ(0 – 24LC16|1 – 24LC64|2 – 24W256)"`
		Product dax.Product  `toml:"product" comment:"параметры датчика ДАХ"`
	}

	conf := Conf{
		Comport: "COM1",
		Addr:    101,
		Chip:    2,
	}

	saveConf := func() {
		b := mustMarshalToml(&conf)
		file := mustCreate("config.toml")
		mustWrite(file, b)
		panicIf(file.Close())
		log.Info("config.toml сохранён")
	}

	{
		b, err := ioutil.ReadFile("config.toml")
		if err == nil {
			err = toml.Unmarshal(b, &conf)
		}
		if err != nil {
			log.PrintErr(err)
		}
	}

	saveConf()

	comPort := comport.NewPort(comport.Config{
		Baud:        115200,
		ReadTimeout: time.Millisecond,
		Name:        conf.Comport,
	})

	comPortReader := comPort.NewResponseReader(context.Background(), comm.Config{
		TimeoutEndResponse: 50 * time.Millisecond,
		TimeoutGetResponse: time.Second,
		MaxAttemptsRead:    3,
	})

	action := flag.String("a", "", `что нужно сделать: 
 - read : считать память микросхемы датчика 
 - write : записать память микросхемы датчика
 - fon_minus20 - фоновый ток Iф при -20°С, нА
 - fon0 - считать фоновый ток Iф при 0°С, нА
 - fon20 - считать фоновый ток Iф при +20°С, нА"
 - fon50 - считать фоновый ток Iф при +50°С, нА
 - sens_minus20 - считать ток ПГС3 Iч при -20°С, нА
 - sens0 - считать фоновый ток ПГС3 Iч при 0°С, нА
 - sens20 - считать фоновый ток ПГС3 Iч при +20°С, нА
 - sens50 считать фоновый ток ПГС3 Iч при +50°С, нА`)

	place := flag.Int("place", 1, "номер места в плате стенда, к которому подключен датчик ДАХ")

	flag.Parse()

	log = logPrependSuffixKeys(log, "action", *action, "place", *place)

	defer saveConf()

	readValue := func(p *float64) {
		Var := (*place - 1) * 2
		log.Info("считывание параметра", "var", Var)
		value, err := modbus.Read3BCD(log, comPortReader, conf.Addr, modbus.Var(Var))
		if err != nil {
			log.PrintErr(err)
			return
		}
		*p = value
	}

	switch *action {

	case "write":
		if err := dax.WriteFirmware(log, comPortReader, conf.Addr, *place, conf.Chip, conf.Product.ToFirmwareBytes()); err != nil {
			log.PrintErr(err)
		}

	case "read":
		b, err := dax.ReadFirmware(log, comPortReader, conf.Addr, *place, conf.Chip)
		if err != nil {
			log.PrintErr(err)
			return
		}
		conf.Product.PutFirmwareBytes(b)

	case "fon_minus20":
		readValue(&conf.Product.FonMinus20)
	case "fon0":
		readValue(&conf.Product.Fon0)
	case "fon20":
		readValue(&conf.Product.Fon20)
	case "fon50":
		readValue(&conf.Product.Fon50)
	case "sens_minus20":
		readValue(&conf.Product.SensMinus20)
	case "sens0":
		readValue(&conf.Product.Sens0)
	case "sens20":
		readValue(&conf.Product.Sens20)
	case "sens50":
		readValue(&conf.Product.Sens50)

	default:
		log.PrintErr(fmt.Sprintf("не правильный параметр: -a=%q", *action))
		flag.PrintDefaults()
	}
}

func init() {
	structlog.DefaultLogger.
		// Wrong log.level is not fatal, it will be reported and set to "debug".
		SetLogLevel(structlog.DBG).
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
			"config":            " %+[2]v",
		}).SetTimeFormat("15:04:05")
}

func mustCreate(name string) *os.File {
	f, err := os.Create(name)
	panicIf(err)
	return f
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func mustWrite(f io.Writer, b []byte) int {
	n, err := f.Write(b)
	panicIf(err)
	return n
}

func mustMarshalToml(p interface{}) []byte {
	b, err := toml.Marshal(p)
	panicIf(err)
	return b
}

func logPrependSuffixKeys(log *structlog.Logger, args ...interface{}) *structlog.Logger {
	var keys []string
	for i, arg := range args {
		if i%2 == 0 {
			k, ok := arg.(string)
			if !ok {
				panic("key must be string")
			}
			keys = append(keys, k)
		}
	}
	return log.New(args...).PrependSuffixKeys(keys...)
}
