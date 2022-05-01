# Disable discovery

You can optionally disable automatic discovery. To disable them all, you would use `--disable-discovery all` or `.runtainer.yaml`:

```yaml
discovery:
  disabled:
    - all
```

The full list what is possible to disable:

- `all`
- `system`
- `system.local`
- `system.cache`
- `system.ssh`
- `system.gnupg`
- `system.ssh-auth-sock`
- `aws`
- `golang`
- `helm`
- `java`
- `kube`
- `tf`
