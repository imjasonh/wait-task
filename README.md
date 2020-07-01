# Wait for Tekton

This repo provides an experimental [Tekton Custom
Task](https://github.com/tektoncd/community/pull/128) that, when run, simply
waits a given amount of time, specified by an input parameter.

The intention is to demonstrate the kinds of things a Custom Task can do, and
to demonstrate how to write a Custom Task.

## Install

Install and configure `ko`.

```
ko apply -f controller.yaml
```

This will build and install the controller on your cluster, in the namespace
`wait-task`.

## Run a Wait

Create a `Run` that refers to a `Wait`:

```
$ kubectl create -f wait-run.yaml 
run.tekton.dev/wait-run-5pnzz created
$ kubectl get runs -w
NAME             SUCCEEDED   REASON    STARTTIME   COMPLETIONTIME
wait-run-5pnzz   Unknown     Waiting   2s          
wait-run-5pnzz   True        DurationElapsed   10s         0s
```
