/*
Copyright © 2026 Eduard Larionov <vesh95.17@ya.ru>
*/
package main

import (
	"dnso/cmd"
	"embed"
)

//go:embed migrations
var migrationsFS embed.FS

func main() {
	cmd.MigrationsFS = migrationsFS
	cmd.Execute()
}
