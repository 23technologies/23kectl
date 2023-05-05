/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"github.com/23technologies/23kectl/pkg/check"
	"github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// installCmd represents the install command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the status of a current 23ke installation",
	Long: `This command will print status messages for flux resources.

If e.g. a HelmRelease failed, the error message message including a hint
will be printed.
`,
	RunE: func(cmd *cobra.Command, args []string) error {

		doctor()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.PersistentFlags().String("kubeconfig", "", "The KUBECONFIG of your base cluster")
}

func doctor() {
	var checks []check.Check

	hrList := &v2beta1.HelmReleaseList{}
	_ = check.KubeClient.List(context.TODO(), hrList, &client.ListOptions{Namespace: "flux-system"})

	for _, hr := range hrList.Items {
		checks = append(checks, &check.HelmReleaseCheck{Name: hr.Name, Namespace: hr.Namespace})
	}

	ksList := &v1beta2.KustomizationList{}
	_ = check.KubeClient.List(context.TODO(), ksList, &client.ListOptions{Namespace: "flux-system"})

	for _, ks := range ksList.Items {
		checks = append(checks, &check.KustomizationCheck{Name: ks.Name, Namespace: ks.Namespace})
	}

	fmt.Print("\033[H\033[2J")

	for _, c := range checks {
		result := c.Run()

		emoji := "⌛"

		if result.IsError {
			emoji = "❌"
		} else if result.IsOkay {
			emoji = "✔️"
		}

		fmt.Printf("%s %s status: %s\n", emoji, c.GetName(), result.Status)
	}

}
