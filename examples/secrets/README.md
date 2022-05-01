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
    --secret-volume runtainer-test-files:mountPath=/my-secrets:item=bar-secret.txt \
    alpine sh
```

Example adding local SSH credentials:

```bash
kubectl create secret generic runtainer-ssh --from-file=${HOME}/.ssh/id_rsa --from-file=${HOME}/.ssh/id_rsa.pub
rm -f ${HOME}/.ssh/{id_rsa,id_rsa.pub} # you don't need them in plain text on the disc anymore, right?
runtainer --secret-volume runtainer-ssh:mountPath=/root/.my_ssh:item=id_rsa:item:id_rsa.pub alpine/git sh
> cat ~/.my_ssh/id_rsa > ~/.ssh/my_id_rsa # because we use fsGroup - kubernetes will force g+r and SSH will reject it
> chmod 600 ~/.ssh/my_id_rsa
> ssh -i ~/.ssh/my_id_rsa -T git@github.com
```
