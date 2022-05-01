# Create secrets

```bash
kubectl create secret generic runtainer-test-env --from-literal=FOO=bar
kubectl create secret generic runtainer-test-files --from-file=examples/secrets/bar-secret.txt
```

# Run

```bash
runtainer --secret-env runtainer-test-env --secret-volume runtainer-test-files alpine sh
# echo $FOO
# cat /rt-secrets/runtainer-test-files/bar-secret.txt
```

Alternatively, examine `.runtainer.yaml` in this directory and run:

```bash
(cd examples/secrets && runtainer alpine sh)
```

You can also use additional parameters for the injection:

```bash
runtainer \
    --secret-env runtainer-test-env:prefix=PREFIX_ \
    --secret-volume runtainer-test-files:mountPath=/my-secrets:items=bar-secret.txt \
    alpine sh
```
