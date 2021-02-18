# Create putty shell in container when http

This example shows you how to create putty container console shell when http.

## Running this example

Make sure your kubectl and `~/.kube/config` is configured.

Get the pod name and container name.

Modify the `url` parameters in `./frontend/index.html` according your actual Kubernetes cluster.

Running this application with:

```bash
go build -o exec .
./exec
```

It will listen on `:8080` port, you can access the `http://127.0.0.1:8080` from browser. It will open the container putty console in browser.
