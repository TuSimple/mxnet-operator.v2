# K8s Custom Resource and Operator For MXNet jobs (v1alpha2)

## Overview

MXJob provides a Kubernetes custom resource that makes it easy to
run distributed or non-distributed MXNet jobs on Kubernetes.

Using a Custom Resource Definition (CRD) gives users the ability to create and manage MX Jobs just like builtin K8s resources. For example to create a job

```
kubectl create -f examples/mx_job_dist_gpu.yaml 
```

To list jobs

```bash
kubectl get mxjobs

NAME          CREATED AT
example-dist-job   3m
```

### Requirements

kubelet : v1.11.1

kubeadm : v1.11.1

Docker： 
```
Client:
 Version:      17.03.2-ce
 API version:  1.27
 Go version:   go1.6.2
 Git commit:   f5ec1e2
 Built:        Thu Jul  5 23:07:48 2018
 OS/Arch:      linux/amd64

Server:
 Version:      17.03.2-ce
 API version:  1.27 (minimum version 1.12)
 Go version:   go1.6.2
 Git commit:   f5ec1e2
 Built:        Thu Jul  5 23:07:48 2018
 OS/Arch:      linux/amd64
 Experimental: false
```

kubectl :

```
Client Version: version.Info{Major:"1", Minor:"11", GitVersion:"v1.11.1", GitCommit:"b1b29978270dc22fecc592ac55d903350454310a", GitTreeState:"clean", BuildDate:"2018-07-17T18:53:20Z", GoVersion:"go1.10.3", Compiler:"gc", Platform:"linux/amd64"}
Server Version: version.Info{Major:"1", Minor:"10", GitVersion:"v1.10.5", GitCommit:"32ac1c9073b132b8ba18aa830f46b77dcceb0723", GitTreeState:"clean", BuildDate:"2018-06-21T11:34:22Z", GoVersion:"go1.9.3", Compiler:"gc", Platform:"linux/amd64"}
```

kubernetes : branch release-1.11

incubator-mxnet : v1.2.0

## Installing the MXJob CRD and operator on your k8s cluster

### Create crds

kubectl create -f examples/crd-v1alpha2.yaml

### Verify MXNet support 

Check that the MXNet custom resource is installed

```
kubectl get crd
```

The output should include `mxjobs.kubeflow.org`

```
NAME                                           AGE
...
mxjobs.kubeflow.org                       4d
...
```

### Creating a job

You create a job by defining a MXJob and then creating it with.

```
kubectl create -f examples/mx_job_dist_gpu.yaml 
```

Each replicaSpec defines a set of MXNet processes.
The mxReplicaType defines the semantics for the set of processes.
The semantics are as follows

**scheduler**
  * A job must have 1 and only 1 scheduler
  * The pod must contain a container named mxnet
  * The overall status of the MXJob is determined by the exit code of the
    mxnet container
      * 0 = success
      * 1 || 2 || 126 || 127 || 128 || 139 = permanent errors:
          * 1: general errors
          * 2: misuse of shell builtins
          * 126: command invoked cannot execute
          * 127: command not found
          * 128: invalid argument to exit
          * 139: container terminated by SIGSEGV(Invalid memory reference)
      * 130 || 137 || 143 = retryable error for unexpected system signals:
          * 130: container terminated by Control-C
          * 137: container received a SIGKILL
          * 143: container received a SIGTERM
      * 138 = reserved in tf-operator for user specified retryable errors
      * others = undefined and no guarantee

**worker**
  * A job can have 0 to N workers
  * The pod must contain a container named mxnet
  * Workers are automatically restarted if they exit

**server**
  * A job can have 0 to N servers
  * parameter servers are automatically restarted if they exit


For each replica you define a **template** which is a K8s
[PodTemplateSpec](https://kubernetes.io/docs/api-reference/v1.8/#podtemplatespec-v1-core).
The template allows you to specify the containers, volumes, etc... that
should be created for each replica.

### Using GPUs

Mxnet-operator is now supporting the gpu training .

Please verify your image is available for gpu distributed training .

For example ,

```
command: ["python"]
args: ["/incubator-mxnet/example/image-classification/train_mnist.py","--num-epochs","1","--num-layers","2","--kv-store","dist_device_sync","--gpus","0"]
resources:
  limits:
    nvidia.com/gpu: 1
```

Mxnet-operator will arrange the pod to nodes which satisfied the gpu limit .

## Monitoring your job

To get the status of your job

```bash
kubectl get -o yaml mxjobs $JOB
```

## Contributing

Please refer to the [developer_guide](./docs/developer_guide.md)

## Community

This is a part of Kubeflow, so please see [readme in kubeflow/kubeflow](https://github.com/kubeflow/kubeflow#get-involved) to get in touch with the community.
