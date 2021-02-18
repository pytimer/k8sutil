# View Workload History

This example shows you how to view the `Deployment` revision history.

## Running this example

Make sure your kubectl and ~/.kube/config is configured. Run kubectl get nodes to confirm.

View the `Deployment` nginx in `default` namespace revision history.

Use `kubectl rollout history` view the history:

```bash
$ kubectl rollout history deployment/nginx
deployment.apps/nginx
REVISION  CHANGE-CAUSE
1         <none>
2         <none>
```

Use `kubectl get deploy nginx -oyaml` show the current revision is `2`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "2"
  creationTimestamp: "2021-02-17T09:05:09Z"
  generation: 2
  labels:
    history: nginx
  name: nginx
  namespace: default
  resourceVersion: "836"
  selfLink: /apis/apps/v1/namespaces/default/deployments/nginx
  uid: 4efa8de9-96a9-4e44-99b7-7c1d8233358e
```

Running this application use the kubeconfig file to authenticate the cluster, and view the revision history.

Run this application with:

```bash
$ go build -o viewer .
$ ./viewer -namespace default -kind Deployment -name nginx
New Revision: 2
Old Revision: 1
```

The results the same as `kubectl rollout history`.