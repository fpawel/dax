package app

import (
	"github.com/ansel1/merry"
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
