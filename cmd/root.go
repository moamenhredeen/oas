/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "oas",
	Short: "Rest API Testing Tool based on OpenAPI Specification",
	Long: `oas is a tool for testing REST APIs based on OpenAPI Specification.
It allows you to test your APIs using a simple and intuitive interface.

You can use oas to test your APIs by providing the OpenAPI Specification file and the endpoints to test.`,
}

func Execute() {
	cobra.OnInitialize(func() {
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(".")
		viper.ReadInConfig()
	})
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Removed placeholder toggle flag
}
