# ladon-resource-manager

This project contains a Kubernetes controller and Custom Resource Definition for managing [Ladon](https://github.com/ory/ladon) policies.

This repository is a fork of `https://github.com/kubernetes/sample-controller`, and a lot of code was borrowed from that project.

# Install

Edit the file located at `k8s/deployment.yaml`

```
vim k8s/deployment.yaml
```

and edit the `POSTGRES_URL` to point to your postgres server.

Once you're done, run:

```
kubectl apply -f k8s/crd.yaml -f k8s/deployment.yaml
```

# Usage

An example `Policy` resource is provided at `k8s/examples/policy.yaml`.

Creating `Policy` resources will create `Policy` entries in your `ladon` database.
