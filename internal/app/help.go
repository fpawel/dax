package app

import (
	"github.com/ansel1/merry"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func cleanTmpDir() {
	if err := os.RemoveAll(tmpDir); err != nil {
		log.PrintErr(merry.Append(err, "os.RemoveAll(tmpDir)"))
	}
}

// mustMarshalYaml is a wrapper for toml.Marshal.
func mustMarshalYaml(v interface{}) []byte {
	data, err := yaml.Marshal(v)
	panicIf(err)
	return data
}

// mustWriteFile is a wrapper for ioutil.WriteFile.
func mustWriteFile(name string, buf []byte, perm os.FileMode) {
	err := ioutil.WriteFile(name, buf, perm)
	panicIf(err)
}
