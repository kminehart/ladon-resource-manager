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

**This will run the migrations for the Ladon database. If you do not have tables built for ladon, this applicaiton will do it as soon as it starts.**

# Usage

An example `Policy` resource is provided at `k8s/examples/policy.yaml`.

Creating `Policy` resources will create `Policy` entries in your `ladon` database.

```
kubectl apply -f k8s/examples/policy.yaml
```

Example:

```yaml
apiVersion: ladoncontroller.k8s.io/v1alpha1
kind: Policy
metadata:
  name: consent-policy-users
spec:
  description: "Allow Consent app to read user information."
  subjects:
  - "services:consent"
  - "groups:app-internal"
  actions:
  - "read"
  resources:
  - "users:<.*>"
  effect: "allow"
```
