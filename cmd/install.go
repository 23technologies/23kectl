/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/23technologies/23kectl/pkg/install"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		viper.Debug()

		config := install.KeConfig{}
		configByte, err := os.ReadFile(viper.ConfigFileUsed())

		if errors.Is(err, os.ErrNotExist) {
			configByte = make([]byte, 0)
		} else if err != nil {
			panic(err)
		}

		err = yaml.Unmarshal(configByte, &config)
		if err != nil {
			panic(err)
		}

		kubeConfig := viper.GetString("KUBECONFIG")
		if kubeConfig == "" {
			kubeConfig = viper.GetString("kubeconfig")
		}
		if kubeConfig == "" {
			fmt.Println("A kubeconfig has to be set")
			return
		}

		install.Install(kubeConfig, &config)
		data, err := yaml.Marshal(&config)
		err = os.WriteFile(viper.GetViper().ConfigFileUsed(), data, 0)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.
	installCmd.PersistentFlags().String("kubeconfig", "", "The KUBECONFIG of your base cluster")
	viper.BindPFlag("kubeconfig", installCmd.PersistentFlags().Lookup("kubeconfig"))

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
