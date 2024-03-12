### Building

Local build: `go mod tidy && go build -o app` should suffice, then you can run the app like:
```
API_KEY=demo CURRENCY=BTC MARKET=USD ./app
```

Docker build: `docker build -t data-api .` then run like:
```
docker run -it --rm -p 8080:8080 -e API_KEY=demo -e CURRENCY=BTC -e MARKET=USD data-api
```
The dockerfile supports multi-arch builds.

### Deployment

As we don't want to commit sensitive data to the source code repository, the secret containing the API key should be provisioned separately before the deployment. For example:
```
export API_KEY="...redacted..."
kubectl create secret generic data-api --from-literal=API_KEY="$API_KEY"
```

Alternatively a secret can be encrypted with `sealed-secrets` or SOPS and commited here with the manifests in `kube/`.

The `kube/` subdirectory contains a bundle of Kustomize manifests that can be hydrated with `kustomize build kube`. It then can be either applied to the target cluster directly (e.g. `kustomize build kube | kubectl apply -f-`) or used with a GitOps controller (e.g. ArgoCD of FluxCD), which is preferrable as the ConfigMap generated has a random suffix hash to facilitate the rollout on config changes and previous versions should be garbage collected in some way. Updating the image tag can be achieved with `kustomize image set` or via CI pipeline scripts.

Alternatively a Helm chart could be implemented, but this is more time consuming.
