package app

import "github.com/lxn/walk"

var sets = func() *walk.IniFileSettings {
	app := walk.App()
	app.SetOrganizationName("analitpribor")
	app.SetProductName("dax")
	sets := walk.NewIniFileSettings("settings.ini")
	panicIf(sets.Load())
	app.SetSettings(sets)
	return sets
}()

func setsGet(key string) string {
	s, _ := sets.Get(key)
	return s
}

func setsComportName() string {
	return setsGet("comport")
}
