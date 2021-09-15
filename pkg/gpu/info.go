package gpu


import (
    "context"
    "os"
    "errors"
    "fmt"
    "text/tabwriter"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)


type NodeGpuInfo struct {
    NodeName string
    GpuAllocatable int64
    GpuRequests int64
    GpuLimits int64
}


type PodGpuInfo struct {
    PodName string
    GpuRequests int64
    GpuLimits int64
}

func GetNodeGpuInfo(clientset Client) ([]NodeInfo, error) {
    nodes, err := clientset.CoreV1().Nodes().List(
        context.TODO(),
        metav1.ListOptions{}
    )
    if err != nil {
        return nil, err
    }

    var nodeInfo []NodeGpuInfo
    for _, node := range nodes.Items {
        pods, err := clientset.CoreV1().Pods("").List(
            context.TODO(),
            metav1.ListOptions{
                FieldSelector: "spec.nodeName=" + node.Name,
            }
        )
        if err != nil {
            return nil, err
        }

        info := &NodeGpuInfo{}
        info.NodeName = node.Name
        gpuAlloc := node.Status.Allocatable["nvidia.com/gpu"]
        info.GpuAllocatable += gpuAlloc.Value()

        info.GpuRequests = 0
        info.GpuLimits = 0
        for _, pod := range pods.Items {
            for _, container := range pod.Spec.Containers {
                gpuReq := container.Resources.Requests["nvidia.com/gpu"]
                gpuLim := container.Resources.Limits["nvidia.com/gpu"]
                info.GpuRequests += gpuReq.Value()
                info.GpuLimits += gpuLim.Value()
            }
        }

        nodeInfo = append(nodeInfo, *info)
    }

    return nodeInfo, nil
}

func PrintNodeGpuInfo(nodeInfo []NodeGpuInfo) {
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
    fmt.Fprintf(writer, "TOTAL\t%d/%d\t%d/%d\n",
                gpuRequestsTotal, gpuAllocatableTotal,
                gpuLimitsTotal, gpuAllocatableTotal)

    writer.Flush()
}

func GetPodGpuInfo(clientset Client) ([]PodGpuInfo, error) {
    pods, err := clientset.CoreV1().Pods("").List(
        context.TODO(),
        metav1.ListOptions{}
    )
    if err != nil {
        return nil, err
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

    return podInfo, nil
}

func PrintPodGpuInfo(podInfo []PodGpuInfo) {
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

