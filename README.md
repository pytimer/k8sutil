# k8sutil

[![Go Reference](https://pkg.go.dev/badge/github.com/pytimer/k8sutil.svg)](https://pkg.go.dev/github.com/pytimer/k8sutil)

The repository provides some toolkits package to make it easier to call [client-go](https://github.com/kubernetes/client-go) to operate Kubernetes cluster.

## Features

- [x] Apply resources.
- [x] View workload resource revision history.
- [x] Create the container putty console shell when use http(s).
- [x] Pod util.
- [x] Node util.

## Usage

### Apply

This package support apply multiple resources to the Kubernetes cluster. It's like `kubectl apply`, support `server-side` and `non server-side`.

The example code in [apply](./examples/apply).

### History view

This package support view the workload resource revision history. It's like `kubectl rollout history`.

Now only support `Deployment`.

The example code in [history](./examples/history).

### Exec in http

This package support create the container putty console shell when use http.

The example code in [exec](./examples/exec).