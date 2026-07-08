/*
Copyright © 2026 Eduard Larionov <vesh95.17@ya.ru>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command for the DNSO application.
var rootCmd = &cobra.Command{
	Use:   "dnso",
	Short: "Local DNS server with web interface for zone management",
	Long: `DNSO is a local DNS server with a web interface for managing DNS zones and records.

It supports creating custom DNS zones, adding A, AAAA, CNAME, MX, TXT, NS, SOA records,
and proxying other queries to an upstream DNS server.

Built with Go, data is stored in SQLite.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
