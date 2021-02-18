# k8sutil

Kubernetes util

## Features

- [x] Apply resources.

## Usage

### Apply

This feature support apply multiple resources to the Kubernetes cluster. It's like `kubectl apply`, support `server-side` and `non server-side`.

The example code in [apply](./examples/apply).

### History view

This feature support view the workload resource revision history. It's like `kubectl rollout history`.

Now only support `Deployment`.

The example code in [history](./examples/history).

### Exec in http

This package support create the container putty console shell when use http.

The example code in [exec](./examples/exec).