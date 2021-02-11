# Apply resource to cluster

This example shows you how to apply multiple resources to the Kubernetes cluster.

## Running this example

Make sure your kubectl and `~/.kube/config` is configured. Run `kubectl get nodes` to confirm.

Run this application with:

```bash
go build -o apply .
./apply
```

Running this application will use the kubeconfig file to authenticate the cluster, and apply the `nginx` Deployment to the default namespace.

```bash
$ kubectl get svc
NAME         TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)     AGE
kubernetes   ClusterIP   10.96.0.1        <none>        443/TCP     113d
nginx-svc    ClusterIP   10.109.238.235   <none>        80/TCP      6s

$ kubectl get deploy
NAME      READY   UP-TO-DATE   AVAILABLE   AGE
nginx     0/1     1            0           10s
```
