package cmd


import (
    "os"
    "fmt"

    "github.com/spf13/cobra"

    "k8s.io/cli-runtime/pkg/genericclioptions"
)

func Execute() {

    options := &CommandOptions{
        configFlags: genericclioptions.NewConfigFlags(true),
    }

    rootCmd := &cobra.Command{
        Use: "kubectl-gpu-capacity",
        SilenceUsage: true,
        RunE: func(cmd *cobra.Command, args []string) error {

            if err := options.Complete(cmd, args); err != nil {
                return err
            }

            if err := options.Validate(); err != nil {
                return err
            }

            if err := options.Run(); err != nil {
                return err
            }

            return nil
        },
    }

    rootCmd.PersistentFlags().BoolVarP(&options.ShowAllocatedPods, "pods", "p", false, "Show GPU allocated pods")

    //options.configFlags.AddFlags(rootCmd.Flags())

    if err := rootCmd.Execute(); err != nil {
        fmt.Print(err)
        os.Exit(1)
    }
}
