package main


import (
    "context"
    "os"
    "strings"
    "errors"
    "fmt"
    "text/tabwriter"

    "github.com/spf13/pflag"
    "github.com/spf13/cobra"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/cli-runtime/pkg/genericclioptions"
    "k8s.io/client-go/kubernetes"
)

type SampleOptions struct {
    configFlags *genericclioptions.ConfigFlags

    args []string

    Namespace string

    genericclioptions.IOStreams
}

func (options *SampleOptions) Complete(cmd *cobra.Command, args []string) (err error) {
    options.args = args
    options.Namespace, _, err = options.configFlags.ToRawKubeConfigLoader().Namespace()

    return err
}

func (options *SampleOptions) Validate() error {
    cases := []struct {
        want bool
        msg string
    }{
        {
            want: len(options.args) > 0,
            msg: "Number of arguments must be > 0",
        },
    }

    for _, c := range cases {
        if !c.want {
            return errors.New(c.msg)
        }
    }

    return nil
}

func (options *SampleOptions) Run() error {

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

    nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    type NodeInfo struct {
        NodeName string
        CpuAllocatable int64
        CpuRequested int64
        GpuAllocatable int64
        GpuRequested int64
    }

    var nodeInfo []NodeInfo
    for _, node := range nodes.Items {
        pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
            FieldSelector: "spec.nodeName=" + node.Name,
        })
        if err != nil {
            fmt.Print(err)
            os.Exit(1)
        }

        info := &NodeInfo{}
        info.NodeName = node.Name
        info.CpuAllocatable = node.Status.Allocatable.Cpu().Value() * 1000
        gpuAlloc := node.Status.Allocatable["nvidia.com/gpu"]
        info.GpuAllocatable += gpuAlloc.Value()

        info.CpuRequested = 0
        for _, pod := range pods.Items {
            for _, container := range pod.Spec.Containers {
                gpuReq := container.Resources.Requests["nvidia.com/gpu"]
                info.GpuRequested += gpuReq.Value()
                info.CpuRequested += container.Resources.Requests.Cpu().MilliValue()
            }
        }

        nodeInfo = append(nodeInfo, *info)
    }

    writer := new(tabwriter.Writer)
    writer.Init(os.Stdout, 0, 8, 0, '\t', 0)
    fmt.Fprintf(writer, "NODE NAME\tGPU(Req/Alloc)\tCPU(Req/Alloc)\t\n")
    for _, info := range nodeInfo {
        num_meter_total := int(info.CpuAllocatable / 1000.0)
        num_meter_used := int(info.CpuRequested / 1000.0)

        meter := strings.Repeat("+", num_meter_used) + strings.Repeat("-", num_meter_total - num_meter_used)

        fmt.Fprintf(writer, "[%s]\t%d / %d\t%.2f / %.2f\t%s\n",
                    info.NodeName,
                    info.GpuRequested,
                    info.GpuAllocatable,
                    float32(info.CpuRequested) / 1000.0,
                    float32(info.CpuAllocatable) / 1000.0,
                    meter)
    }
    writer.Flush()


    return nil
}

func main() {
    flags := pflag.NewFlagSet("kubectl-ns", pflag.ExitOnError)
    pflag.CommandLine = flags

    streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

    options := &SampleOptions{
        configFlags: genericclioptions.NewConfigFlags(true),
        IOStreams: streams,
    }

    rootCmd := &cobra.Command{
        Use: "kubectl-sample-plugin",
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

    options.configFlags.AddFlags(rootCmd.Flags())

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
