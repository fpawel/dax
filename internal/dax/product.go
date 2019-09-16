package dax

import (
	"github.com/fpawel/comm/modbus"
	"math"
	"reflect"
	"strconv"
)

type Product struct {
	FonMinus20 float64 `flash_addr:"00" flash_format:"bcd" toml:"fon_minus20" comment:"фоновый ток Iф при -20°С, нА"`
	Fon0 float64 `flash_addr:"04" flash_format:"bcd" toml:"fon0" comment:"фоновый ток Iф при 0°С, нА"`
	Fon20 float64 `flash_addr:"08" flash_format:"bcd" toml:"fon20" comment:"фоновый ток Iф при +20°С, нА"`
	Fon50 float64 `flash_addr:"0C" flash_format:"bcd" toml:"fon50" comment:"фоновый ток Iф при +50°С, нА"`

	SensMinus20 float64 `flash_addr:"14" flash_format:"bcd" toml:"sens_minus20" comment:"ток ПГС3 Iч при -20°С, нА"`
	Sens0 float64 `flash_addr:"18" flash_format:"bcd" toml:"sens0" comment:"фоновый ток ПГС3 Iч при 0°С, нА"`
	Sens20 float64 `flash_addr:"1C" flash_format:"bcd" toml:"sens20" comment:"фоновый ток ПГС3 Iч при +20°С, нА"`
	Sens50 float64 `flash_addr:"20" flash_format:"bcd" toml:"sens50" comment:"фоновый ток ПГС3 Iч при +50°С, нА"`

	TempMinus20 float64 `flash_addr:"28" flash_format:"bcd" toml:"temp_minus20" comment:"температура °С, при которой происходит измерение Iф-20°С, Iч-20°С"`
	Temp0 float64 `flash_addr:"2C" flash_format:"bcd" toml:"temp0" comment:"температура °С, при которой происходит измерение Iф 0°С, Iч 0°С"`
	Temp20 float64 `flash_addr:"30" flash_format:"bcd" toml:"temp20" comment:"температура °С, при которой происходит измерение Iф 20°С, Iч 20°С"`
	Temp50 float64 `flash_addr:"34" flash_format:"bcd" toml:"temp50" comment:"температура °С, при которой происходит измерение Iф 50°С, Iч 50°С"`

	ProductType int `flash_addr:"0600" flash_format:"uint16" toml:"product_type" comment:"Тип датчика: CH3OH – 240, CH2O – 241, C2H4 – 242, C2H4O – 243"`
	SerialNumber int `flash_addr:"0602" flash_format:"uint16" toml:"serial_number" comment:"заводской номер датчика"`
	Year int `flash_addr:"0604" flash_format:"byte" toml:"year" comment:"Год выпуска, последние две цифры"`
	Quarter int `flash_addr:"0605" flash_format:"byte" toml:"quarter" comment:"Квартал изготовления датчика"`
}

func (p *Product) PutFirmwareBytes(b []byte)  {
	v := reflect.ValueOf(p).Elem()
	// Iterate over all available fields and read the tag value
	for i := 0; i < typeProduct.NumField(); i++ {
		addr := getProductFieldAddr(i)
		field := typeProduct.Field(i)
		format := field.Tag.Get("flash_format")
		switch format {
		case "bcd":
			n,ok := modbus.ParseBCD6(b[addr:addr+4])
			if !ok {
				n = math.NaN()
			}
			v.Field(i).SetFloat(n)
		case "uint16":
			n := int64(b[addr+1]) * 256 + int64(b[addr])
			v.Field(i).SetInt(n)
		case "byte":
			n := int64(b[addr])
			v.Field(i).SetInt(n)
		default:
			panic(format)
		}
	}
}

func (p Product) ToFirmwareBytes() []byte {
	b := make([]byte, FirmwareSize)

	v := reflect.ValueOf(p)

	// Iterate over all available fields and read the tag value
	for i := 0; i < typeProduct.NumField(); i++ {
		addr := getProductFieldAddr(i)
		field := typeProduct.Field(i)
		format := field.Tag.Get("flash_format")
		switch format {
		case "bcd":
			modbus.PutBCD6(b[addr:addr+4], v.Field(i).Float())
		case "uint16":
			n := v.Field(i).Int()
			b[addr] = byte(n)
			b[addr+1] = byte(n >> 8)
		case "byte":
			n := v.Field(i).Int()
			b[addr] = byte(n)
		default:
			panic(format)
		}
	}
	return b
}

func getProductFieldAddr(i int) uint16{
	x,err := strconv.ParseInt(typeProduct.Field(i).Tag.Get("flash_addr"), 16, 16)
	if err != nil {
		panic(err)
	}
	return uint16(x)
}

var (
	typeProduct = reflect.TypeOf(Product{})
)