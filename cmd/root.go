package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:          "inet_peercred",
	SilenceUsage: true,
	Version:      "0.1.0",
	Short:        "A simple server to provide the peer credentials of an inet socket",
	Long:         "A simple server to provide the peer credentials of an inet socket",
}

var log = zerolog.New(os.Stdout).With().Logger()

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.AutomaticEnv()
	// zerolog.TimeFieldFormat = ""
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Msg(err.Error())
	}
}
