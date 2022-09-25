# Unofficial FlashBlade NFS Provisioner

This provisioner allows you to automatically provision File Systems on a FlashBlade Appliance from a `PersistentVolumeClaim` in your Kubernetes cluster by using a `StorageClass`.

This provisioner is using the [kubernetes-sigs/sig-storage-lib-external-provisioner](https://github.com/kubernetes-sigs/sig-storage-lib-external-provisioner) to abstract all Kubernetes related controller stuff and is only called for creating and deleting `PersistentVolume`s with already specified names.

## Development

### Requirements

- golang 1.18 (for build without docker, e.g. `brew install golang`)

### Setup

```bash
go mod download
```

## Build

### As Binary

You can build the provisioner to a binary by simply running:

```bash
go build
```

### As Container

To make the provisioner deployable, we have to build it into a container that we can run in our kubernetes cluster.
Build the container in an internet facing network simply as follows:

```bash
IMAGE_NAME=my-container-registry.example/flashblade-provisioner # TODO replace this with your image URL
docker build . -t $IMAGE_NAME
docker push $IMAGE_NAME
```

In a network that requires proxies you can build the container by e.g. making use of a proxy that runs on your build host:

```bash
IMAGE_NAME=my-container-registry.example/flashblade-provisioner # TODO replace this with your image URL
docker build . --network host --build-arg http_proxy=$HTTP_PROXY --build-arg https_proxy=$HTTP_PROXY -t $IMAGE_NAME
docker push $IMAGE_NAME

```

## Deployment

You can use the sample resources in `deploy/` after adjusting all `TODO`s to deploy the provisioner in your kubernetes cluster.
The Kubernetes roles have the minimal permissions required for the controller to work.
Deploy it with:

```bash
NAMESPACE=your-namespace
kubectl apply -n $NAMESPACE -f deploy/flashblade-provisioner.yml
```

A sample application could look like `deploy/sample-application.yml` and can be deployed in any namespace:

```bash
kubectl apply -f deploy/sample-application.yml
```

## Tooling

### Tests

1. Create a tunnel to the appliance as described below
2. Copy the `.env.example` file to `.env.test` in this repo
3. Get an API Key and place it in the `.env.test` file, which is ignored by git
4. Run the tests with `go test`

### SSH Tunnel to Storage Appliance API

In case you are in a network that has a jump server separating you and the FlashBlade API, create a tunnel via SSH e.g. as follows:

```bash
APPLIANCE_HOST=flashblade-api.example
APPLIANCE_PORT=443
JUMP_SERVER_ADDRESS=jump-server.example

ssh -L 6443:$APPLIANCE_HOST:$APPLIANCE_PORT $JUMP_SERVER_ADDRESS
# tunnel is now open
```

You can now e.g. configure the .env file as described above to run tests directly against the appliance.

## Open Issues

This provisioner was built for a simple use case with minimal requirements.
To make it more usable in other environments the following issues could be resolved.

- add LICENSE file
- implement more API versions, not only version 2.2
- allow parametrization via `StorageClass` parmeters
  - this would allow to e.g. use the same provisioner for multiple FlashBlades with 1 `StorageClass` per FlashBlade
  - it would also allow setting different prefixes or storage configs via these params
