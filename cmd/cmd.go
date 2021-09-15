package cmd


import (
    "errors"

    "github.com/spf13/cobra"

    "k8s.io/cli-runtime/pkg/genericclioptions"
    "k8s.io/client-go/kubernetes"

    gpu "github.com/nuka137/kubectl-gpu-capacity/pkg/gpu"
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
        return err
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return err
    }

    if options.Mode == ShowCapacity {
        info, err := gpu.GetNodeGpuInfo(clientset)
        if err != nil {
            return err
        }
        gpu.PrintNodeGpuInfo(info)
    } else {
        info, err := gpu.GetPodGpuInfo(clientset)
        if err != nil {
            return err
        }
        gpu.PrintPodGpuInfo(info)
    }

    return nil
}

