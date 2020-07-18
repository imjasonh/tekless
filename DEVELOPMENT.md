# Tekton without Kubernetes

https://gist.github.com/ImJasonH/9ff61be88c1365b9f0bbde11885a3c42

## Running

```
go run ./ \
  --project=$(gcloud config get-value project) \
  --tok=$(gcloud auth print-access-token)
```

## Debugging

SSH into the VM:

```
sudo journalctl -u cloud-init.service
sudo journalctl -u kubelet.service
systemctl status kubelet.service
```

kubelet currently fails to start pod with:

```
failed to find plugin "loopback" in path [/opt/cni/bin]
```
