package dax

import (
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
	_
	Chip256
)

func WriteFirmware(log *structlog.Logger, responseReader modbus.ResponseReader,
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
		if _, err := req.GetResponse(log, responseReader, func(_, response []byte) (s string, e error) {
			if len(response) != 5 {
				return "", comm.Err.Here().Append("длина ответа должна быть 5")
			}
			if response[2] != 0 {
				return "", comm.Err.Here().Appendf("код ошибки %d", response[2])
			}
			return "", nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func ReadFirmware(log *structlog.Logger, responseReader modbus.ResponseReader,
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

		resp, err := req.GetResponse(log, responseReader, func(request, response []byte) (string, error) {
			if len(response) != 10+int(count) {
				return "", comm.Err.Here().Appendf("ожидалось %d байт ответа, получено %d",
					10+int(count), len(response))
			}
			return "", nil
		})
		if err != nil {
			return nil, merry.Wrap(err)
		}
		copy(b[c.addr1:c.addr1+count], resp[8:8+count])
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
