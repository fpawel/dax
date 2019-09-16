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

func WriteFirmware(log *structlog.Logger, ctx context.Context, responseReader modbus.ResponseReader,
	addr modbus.Addr,
	place int, chip ChipType, bytes []byte) error {
	log = gohelp.LogPrependSuffixKeys(log,
		"chip", chip,
		"addr", addr,
		"place", place,

	)
	for _, c := range FirmwareAddresses {
		data := bytes[c.addr1:c.addr2+1]
		log := gohelp.LogPrependSuffixKeys(log,
			"range", fmt.Sprintf("%X...%X", c.addr1, c.addr2),
			"bytes_count", len(data),
			"data", fmt.Sprintf("`% X`", data),
		)
		req := modbus.Request{
			Addr:     addr,
			ProtoCmd: 0x48,
			Data: append( []byte{
				byte(place),
				byte(chip),
				byte(c.addr1 >> 8),
				byte(c.addr1),
				byte(len(data) >> 8),
				byte(len(data)),
			}, data...),
		}
		if _,err := req.GetResponse(log, ctx, responseReader, func(_, response []byte) (s string, e error) {
			if len(response)!=5{
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

func ReadFirmware(log *structlog.Logger, ctx context.Context, responseReader modbus.ResponseReader,
	addr modbus.Addr,
	place int, chip ChipType) ([]byte, error) {

	log = gohelp.NewLogWithSuffixKeys(
		"chip", chip,
		"addr", addr,
		"place", place,
	)

	if chip > Chip256 {
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

		log := gohelp.LogPrependSuffixKeys(log,
			"range", fmt.Sprintf("%X...%X", c.addr1, c.addr2),
			"bytes_count", count,
		)

		resp, err := req.GetResponse(log, ctx, responseReader, func(request, response []byte) (string, error) {
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
		{0, 0x37},
		{0x600, 0x605},
	}
	FirmwareSize = int(FirmwareAddresses[len(FirmwareAddresses)-1].addr2 + 1)
)
