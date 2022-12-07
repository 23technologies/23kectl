# 23kectl

## Requirements
1. A Kubernetes cluster (also called base cluster) running in the cloud
2. A DNS provider e.g. azure-dns, aws-route53, openstack-designate
3. A domain delegated to the DNS provider of choice
4. A remote git repository which is accessible (read and write) via ssh
5. Knowledge about Flux, Helm and Kustomize

## Quickstart
Make sure that your host has all github.com hosts keys in the ~/.ssh/known_hosts file:
```shell
ssh-keyscan github.com >> ~/.ssh/known_hosts
```

Run 23kectl
```shell
go mod tidy && go mod vendor
go run main.go install --kubeconfig KUBECONFIG_FOR_BASE_CLUSTER 
```

The wizard will guide you through the configuration process.
Once finished, you will find the configuration files in your configuration git repository.
This is meant to be the main entry point for further configuration, as 23ke comes as a gitops driven Gardener distribution.
Therefore, the preferred way for configuration is to change values/add resources/ whatnot in the configuration repository.

If you want to watch the installation process, you can watch the flux resources, such as helm releases:
```shell
kubectl get -n flux-system hr --watch
```

## Bigger Picture

If you are interested in 23ke details you can checkout our documentation, which is yet to be written.
