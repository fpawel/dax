package dax

import (
	"context"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/fpawel/comm"
	"github.com/fpawel/comm/modbus"
	"github.com/fpawel/gohelp"
	"github.com/powerman/structlog"
)

type ChipType byte

const (
	Chip16 ChipType = iota
	Chip64
	Chip256
)

func WriteFirmware(log *structlog.Logger, ctx context.Context, cm comm.T,
	addr modbus.Addr,
	place int, chip ChipType, bytes []byte) error {
	log = logPrependSuffixKeys(log,
		"chip", chip,
		"addr", addr,
		"place", place,
	)
	for _, c := range FirmwareAddresses {
		data := bytes[c.addr1 : c.addr2+1]
		log := logPrependSuffixKeys(log,
			"range", fmt.Sprintf("%X...%X", c.addr1, c.addr2),
			"bytes_count", len(data),
			"data", fmt.Sprintf("`% X`", data),
		)
		req := modbus.Request{
			Addr:     addr,
			ProtoCmd: 0x48,
			Data: append([]byte{
				byte(place),
				byte(chip),
				byte(c.addr1 >> 8),
				byte(c.addr1),
				byte(len(data) >> 8),
				byte(len(data)),
			}, data...),
		}
		response, err := req.GetResponse(log, ctx, cm)
		if err != nil {
			return merry.Wrap(err)
		}
		if len(response) != 5 {
			return merry.New("длина ответа должна быть 5")
		}
		if response[2] != 0 {
			return merry.Errorf("код ошибки %d", response[2])
		}
	}
	return nil
}

func ReadFirmware(log *structlog.Logger, ctx context.Context, cm comm.T,
	addr modbus.Addr,
	place int, chip ChipType) ([]byte, error) {

	log = gohelp.NewLogWithSuffixKeys(
		"chip", chip,
		"addr", addr,
		"place", place,
	)

	if chip > Chip256 || chip < Chip16 {
		log.Fatal("не правильный тип микросхеммы")
	}

	b := make([]byte, FirmwareSize)
	for i := range b {
		b[i] = 0xff
	}
	for _, c := range FirmwareAddresses {
		count := c.addr2 - c.addr1 + 1
		req := modbus.Request{
			Addr:     addr,
			ProtoCmd: 0x44,
			Data: []byte{
				byte(place),
				byte(chip),
				byte(c.addr1 >> 8),
				byte(c.addr1),
				byte(count >> 8),
				byte(count),
			},
		}

		log := logPrependSuffixKeys(log,
			"range", fmt.Sprintf("%X...%X", c.addr1, c.addr2),
			"bytes_count", count,
		)

		response, err := req.GetResponse(log, ctx, cm)

		if err != nil {
			return nil, merry.Wrap(err)
		}

		if len(response) != 10+int(count) {
			return nil, merry.Errorf("ожидалось %d байт ответа, получено %d",
				10+int(count), len(response))
		}

		copy(b[c.addr1:c.addr1+count], response[8:8+count])
	}
	return b, nil
}

var (
	FirmwareAddresses = []struct{ addr1, addr2 uint16 }{
		{0, 0x3B},
		{0x600, 0x609 + 2},
	}
)

const FirmwareSize = 0x609 + 2 + 1

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
