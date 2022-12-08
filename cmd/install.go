/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/23technologies/23kectl/pkg/install"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the 23KE Gardener distribution",
	Long: `This command will guide you through the installation process of 23KE.

You are required to have access to the 23KE git repository.
Access can be granted by the 23T administrators only.
If you do not have access yet, contact us. You will find contact information on:

https://23technologies.cloud

Other than that you need:
-  A Kubernetes cluster (also called base cluster) running in the cloud
-  A DNS provider e.g. azure-dns, aws-route53, openstack-designate
-  A domain delegated to the DNS provider of choice
-  A remote git repository which is accessible (read and write) via ssh
-  Knowledge about Flux, Helm and Kustomize
for the installation.

Dependent on your relationship with 23T you will be charged for using 23KE.
`,
	Run: func(cmd *cobra.Command, args []string) {
		config := install.KeConfig{}
		viper.ReadInConfig()
		install.UnmarshalKeConfig(&config)

		kubeConfig, err := cmd.Flags().GetString("kubeconfig")
		if err != nil {
			panic(err)
		}
		install.Install(kubeConfig, &config)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.
	installCmd.PersistentFlags().String("kubeconfig", "", "The KUBECONFIG of your base cluster")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// installCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
