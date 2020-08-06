package app

import (
	"github.com/fpawel/comm"
	"github.com/fpawel/dax/internal/dax"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Comport            string        `toml:"comport" comment:"СОМ порт, к которому подключен стенд"`
	LogComm            bool          `yaml:"log_comm"`
	Chip               string        `toml:"chip" comment:"тип микросхемы датчика ДАХ(0 – 24LC16|1 – 24LC64|2 – 24W256)"`
	TimeoutGetResponse time.Duration `yaml:"timeout_get_response"`
	TimeoutEndResponse time.Duration `yaml:"timeout_end_response"`
	MaxAttemptsRead    int           `yaml:"max_attempts_read"`
	Pause              time.Duration `yaml:"pause"`
	Rf                 float64       `yaml:"Rf" comment:"сопротивление обратной связи датчика, кОм"`
}

var (
	config = Config{
		Comport:            "COM1",
		LogComm:            false,
		Chip:               "24W256",
		TimeoutGetResponse: time.Second,
		TimeoutEndResponse: 50 * time.Millisecond,
		Rf:                 17.4,
	}
)

func configFileName() string {
	return filepath.Join(filepath.Dir(os.Args[0]), "config.yaml")
}

func saveConfig() {
	mustWriteFile(configFileName(), mustMarshalYaml(config), 0666)
}

func openConfig() {
	filename := configFileName()

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		saveConfig()
	}

	data, err := ioutil.ReadFile(filename)
	panicIf(err)

	err = yaml.Unmarshal(data, &config)

	config.Chip = formatChipType(parseChipType(config.Chip))

	if err != nil {
		log.Println("config:", err)
		saveConfig()
	}

	comm.SetEnableLog(config.LogComm)
}

func formatChipType(x dax.ChipType) string {
	switch x {
	case dax.Chip16:
		return "24LC16"
	case dax.Chip64:
		return "24LC64"
	case dax.Chip256:
		return "24W256"
	}
	return "24W256"
}

func parseChipType(s string) dax.ChipType {
	switch s {
	case "24LC16":
		return dax.Chip16
	case "24LC64":
		return dax.Chip64
	case "24W256":
		return dax.Chip256
	}
	return dax.Chip256
}
