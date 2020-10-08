![Kind Calico](https://github.com/mchirico/kind/workflows/Kind%20Calico/badge.svg)
![Kind CertManager](https://github.com/mchirico/kind/workflows/Kind%20CertManager/badge.svg)
![Kind Kudo](https://github.com/mchirico/kind/workflows/Kind%20Kudo/badge.svg)
![Kind CertManager v1.19](https://github.com/mchirico/kind/workflows/Kind%20CertManager%20v1.19/badge.svg)
# kind

Example k8s clusters using [`kind`](https://kind.sigs.k8s.io/), put into a Makefile.

## Kind with Calico and Nginx Ingress

```
make calico
```


## Example compiling kubernetes source with cert manager

```
make cert-manager-v1.19
```


```
kubectl exec -it node-starter-deploy-7d7b887466-gvrb7 -- /bin/sh

```

## Trouble Shooting

```
docker exec -it kind-worker  /bin/bash
docker exec -it kind-control-plane  /bin/bash

```

