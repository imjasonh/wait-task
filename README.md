# 🚨 This work has moved 🚨

This functionality is now available in Tekton's experimental repo: https://github.com/tektoncd/experimental/tree/main/wait-task

---

This repo provided an experimental [Tekton Custom
Task](https://tekton.dev/docs/pipelines/runs/) that, when run, simply waits a
given amount of time before succeeding, specified by an input parameter named
`duration`.

### Motivation

Some users might want their Pipeline to wait some amount of time between two
tasks, for example, to deploy to one cluster then wait an hour before deploying
to other clusters.

This is possible without this Custom Task using a Task that sleeps (e.g.,
`script: sleep 600`), but this results in Pods running on the cluster that are
just sleeping. This wastes cluster resources, and if nodes on the cluster go
down then those sleeps will fail, leading to flaky deployments.

Instead, this Custom Task centralizes all wait operations in the Custom Task
controller, which is more efficient and more fault tolerant.

This also acts as a simple Custom Task that can be modified to perform other
more complex actions.

## Install

Install and configure `ko`.

```
ko apply -f controller.yaml
```

This will build and install the controller on your cluster, in the namespace
`wait-task`.

## Example

Create and watch an example `Run` that waits for 10 seconds:

```
$ kubectl create -f example-run.yaml 
run.tekton.dev/wait-run-5pnzz created
$ kubectl get runs -w
NAME             SUCCEEDED   REASON    STARTTIME   COMPLETIONTIME
wait-run-5pnzz   Unknown     Waiting   2s          
wait-run-5pnzz   True        DurationElapsed   10s         0s
```
