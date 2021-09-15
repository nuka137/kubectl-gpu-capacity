package main


import (
    "context"
    "os"
    "errors"
    "fmt"

    "github.com/spf13/cobra"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/cli-runtime/pkg/genericclioptions"
    "k8s.io/client-go/kubernetes"

    "github.com/nuka137/pkg/gpu"
)

type CommandMode int
const (
    ShowCapacity CommandMode = iota
    ShowAllocatedPods
)

type CommandOptions struct {
    configFlags *genericclioptions.ConfigFlags

    Args []string

    Namespace string

    Mode CommandMode

    ShowAllocatedPods bool
}

func (options *CommandOptions) Complete(cmd *cobra.Command, args []string) (err error) {
    options.Args = args
    options.Namespace, _, err = options.configFlags.ToRawKubeConfigLoader().Namespace()

    if options.ShowAllocatedPods {
        options.Mode = ShowAllocatedPods;
    } else {
        options.Mode = ShowCapacity;
    }

    return err
}

func (options *CommandOptions) Validate() error {
    cases := []struct {
        want bool
        msg string
    }{
    }

    for _, c := range cases {
        if !c.want {
            return errors.New(c.msg)
        }
    }

    return nil
}

func (options *CommandOptions) Run() error {

    config, err := options.configFlags.ToRESTConfig()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    if options.Mode == ShowCapacity {
    }


    return nil
}

func main() {

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
        os.Exit(1)
    }
}
