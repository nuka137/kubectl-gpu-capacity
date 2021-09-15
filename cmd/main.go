package main


import (
    "context"
    "os"
    "errors"
    "fmt"
    "text/tabwriter"

    "github.com/spf13/cobra"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/cli-runtime/pkg/genericclioptions"
    "k8s.io/client-go/kubernetes"
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
        nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
        if err != nil {
            fmt.Println(err)
            os.Exit(1)
        }

        type NodeInfo struct {
            nodeName string
            gpuAllocatable int64
            gpuRequests int64
            gpuLimits int64
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
            info.nodeName = node.Name
            gpuAlloc := node.Status.Allocatable["nvidia.com/gpu"]
            info.gpuAllocatable += gpuAlloc.Value()

            info.gpuRequests = 0
            info.gpuLimits = 0
            for _, pod := range pods.Items {
                for _, container := range pod.Spec.Containers {
                    gpuReq := container.Resources.Requests["nvidia.com/gpu"]
                    gpuLim := container.Resources.Limits["nvidia.com/gpu"]
                    info.gpuRequests += gpuReq.Value()
                    info.gpuLimits += gpuLim.Value()
                }
            }

            nodeInfo = append(nodeInfo, *info)
        }

        writer := new(tabwriter.Writer)
        writer.Init(os.Stdout, 0, 8, 0, '\t', 0)
        fmt.Fprintf(writer, "NODE NAME\tGPU (Requests/Total)\tGPU (Limits/Total)\n")
        var gpuAllocatableTotal int64 = 0
        var gpuRequestsTotal int64 = 0
        var gpuLimitsTotal int64 = 0
        for _, info := range nodeInfo {
            fmt.Fprintf(writer, "%s\t%d/%d\t%d/%d\n",
                        info.nodeName,
                        info.gpuRequests,
                        info.gpuAllocatable,
                        info.gpuLimits,
                        info.gpuAllocatable)
            gpuAllocatableTotal += info.gpuAllocatable
            gpuRequestsTotal += info.gpuRequests
            gpuLimitsTotal += info.gpuLimits
        }
        fmt.Fprintf(writer, "\t\t\n")
        fmt.Fprintf(writer, "TOTAL\t%d/%d\t%d/%d\n", gpuRequestsTotal, gpuAllocatableTotal, gpuLimitsTotal, gpuAllocatableTotal)
        writer.Flush()
    } else {
        pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
        if err != nil {
            fmt.Println(err)
            os.Exit(1)
        }

        type PodInfo struct {
            podName string
            gpuRequest int64
            gpuLimit int64
        }

        var podInfo []PodInfo
        for _, pod := range pods.Items {
            info := &PodInfo{}
            info.podName = pod.Name
            for _, container := range pod.Spec.Containers {
                gpuReq := container.Resources.Requests["nvidia.com/gpu"]
                gpuLim := container.Resources.Limits["nvidia.com/gpu"]
                info.gpuRequest += gpuReq.Value()
                info.gpuLimit += gpuLim.Value()
            }
            podInfo = append(podInfo, *info)
        }

        writer := new(tabwriter.Writer)
        writer.Init(os.Stdout, 0, 8, 0, '\t', 0)
        fmt.Fprintf(writer, "POD NAME\tGPU (Request)\tGPU (Limit)\n")
        for _, info := range podInfo {
            if info.gpuRequest > 0 && info.gpuLimit > 0 {
                fmt.Fprintf(writer, "%s\t%d\t%d\n",
                            info.podName,
                            info.gpuRequest,
                            info.gpuLimit)
            }
        }
        writer.Flush()
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

    options.configFlags.AddFlags(rootCmd.Flags())

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
