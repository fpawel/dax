package main

import (
	"context"
	"flag"
	"github.com/fpawel/comm"
	"github.com/fpawel/comm/comport"
	"github.com/fpawel/comm/modbus"
	"github.com/fpawel/dax/internal/dax"
	"github.com/fpawel/gohelp"
	"github.com/pelletier/go-toml"
	"github.com/powerman/must"
	"github.com/powerman/structlog"
	"os"
	"path/filepath"
	"time"
)

func main() {
	flag.Parse()

	log = gohelp.NewLogWithSuffixKeys(
		"chip", *chip,
		"addr", *addr,
		"place", *place,
		"action", *action,
		"comport", *commPortName,
	)
	switch *action {
	case "write":
		b := must.ReadFile("product.toml")
		var p dax.Product
		if err := toml.Unmarshal(b, &p); err != nil {
			log.Fatal(err)
		}
		if err := dax.WriteFirmware(log,ctx, commPort, modbus.Addr(*addr), *place, dax.ChipType(*chip), p.ToFirmwareBytes()); err != nil {
			log.Fatal(err)
		}

	case "read":
		b, err := dax.ReadFirmware(log, ctx, commPort, modbus.Addr(*addr), *place, dax.ChipType(*chip))
		if err != nil {
			log.Fatal(err)
		}
		var p dax.Product
		p.PutFirmwareBytes(b)

		b, err = toml.Marshal(&p)
		if err != nil {
			log.Fatal(err)
		}
		file := must.Create("product.toml")
		defer log.ErrIfFail(file.Close)
		if _, err := file.Write(b); err != nil {
			log.Fatal(err)
		}

	default:
		log.Fatal("не правильный параметр: action")
	}
	if err := commPort.Open(log, ctx); err != nil {
		log.Fatal(err)
	}
}

var (
	log          *structlog.Logger
	ctx          = context.Background()
	commPortName = flag.String("comport", "COM1", "имя СОМ порта")
	action       = flag.String("action", "read", "что нужно сделать (read|write)")
	addr         = flag.Int("addr", 101, "адрес MODBUS")
	place        = flag.Int("place", 1, "номер места")
	chip         = flag.Int("chip", 2, "тип микросхемы (0 – 24LC16|1 – 24LC64|2 – 24W256)")
	commPort     = comport.NewReadWriter(func() comport.Config {
		return comport.Config{
			Baud:        115200,
			ReadTimeout: time.Millisecond,
			Name:        *commPortName,
		}
	}, func() comm.Config {
		return comm.Config{
			ReadByteTimeoutMillis: 50,
			ReadTimeoutMillis:     1000,
			MaxAttemptsRead:       3,
		}
	})
)

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
