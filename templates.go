package main

import (
	"embed"
	"html/template"
)

//go:embed template
var templateFiles embed.FS

var evenOddFuncMap = template.FuncMap{
	"even": func(i int) bool {
		return i%2 == 0
	},
}
