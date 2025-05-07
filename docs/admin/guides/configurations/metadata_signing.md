# Configure Metadata Signing

It is possible to sign Pulp's metadata so that users can verify the authenticity of an object.
This is done by enabling the Signing Services feature. The steps to enable it are:

* create a gpg key
* create a Secret with a gpg key
* create a Secret with the signing script(s)
* configure Pulp CR

See pulpcore documentation for details on Content Signing: [https://docs.pulpproject.org/pulpcore/workflows/signed-metadata.html#metadata-signing](https://docs.pulpproject.org/pulpcore/workflows/signed-metadata.html#metadata-signing)  
See pulp_container documentation for details on Container Image Signing: [https://docs.pulpproject.org/pulp_container/workflows/sign-images.html](https://docs.pulpproject.org/pulp_container/workflows/sign-images.html)


## Create a gpg key

* create the key
```bash
$ GPG_EMAIL=pulp@example.com
$ cat >/tmp/gpg.txt <<EOF
%echo Generating a basic OpenPGP key
Key-Type: DSA
Key-Length: 1024
Subkey-Type: ECDSA
Subkey-Curve: nistp256
Name-Real: Collection Signing Service
Name-Comment: with no passphrase
Name-Email: $GPG_EMAIL
Expire-Date: 0
%no-ask-passphrase
%no-protection
# Do a commit here, so that we can later print "done" :-)
%commit
%echo done
EOF

$ gpg --batch --gen-key /tmp/gpg.txt
```

* verify the list of available keyrings
```bash
$ gpg --list-keys
/var/lib/pulp/.gnupg/pubring.kbx
--------------------------------
pub   rsa4096 2022-12-14 [SC]
      66BBFE010CF70CC92826D9AB71684D7912B09BC1
uid           [ultimate] Collection Signing Service (with no passphrase) <pulp@example.com>
sub   rsa2048 2022-12-14 [E]
```

See the GnuPG official documentation for more information on how to generate a new keypair: [https://www.gnupg.org/gph/en/manual/c14.html](https://www.gnupg.org/gph/en/manual/c14.html)

## Create a Secret with the gpg key

```bash
$ gpg --export-secret-keys -a pulp@example.com  > /tmp/gpg_private_key.gpg
$ kubectl create secret generic signing-secret --from-file=signing_service.gpg=/tmp/gpg_private_key.gpg
```

## Create a Secret with the signing scripts

* example of a collection signing script
```bash
$ SIGNING_SCRIPT_PATH=/tmp
$ COLLECTION_SIGNING_SCRIPT=collection_script.sh
$ cat<<EOF> "$SIGNING_SCRIPT_PATH/$COLLECTION_SIGNING_SCRIPT"
#!/usr/bin/env bash
set -u
FILE_PATH=\$1
SIGNATURE_PATH="\$1.asc"

ADMIN_ID="\$PULP_SIGNING_KEY_FINGERPRINT"
PASSWORD="password"

# Create a detached signature
gpg --quiet --batch --pinentry-mode loopback --yes --passphrase \
   \$PASSWORD --homedir ~/.gnupg/ --detach-sign --default-key \$ADMIN_ID \
   --armor --output \$SIGNATURE_PATH \$FILE_PATH

# Check the exit status
STATUS=\$?
if [ \$STATUS -eq 0 ]; then
   echo {\"file\": \"\$FILE_PATH\", \"signature\": \"\$SIGNATURE_PATH\"}
else
   exit \$STATUS
fi
EOF
```

* example of a container signing script
```bash
$ SIGNING_SCRIPT_PATH=/tmp
$ CONTAINER_SIGNING_SCRIPT=container_script.sh
$ cat<<EOF> "$SIGNING_SCRIPT_PATH/$CONTAINER_SIGNING_SCRIPT"
#!/usr/bin/env bash
set -u

MANIFEST_PATH=\$1
IMAGE_REFERENCE="\$REFERENCE"
SIGNATURE_PATH="\$SIG_PATH"

skopeo standalone-sign \
      \$MANIFEST_PATH \
      \$IMAGE_REFERENCE \
      \$PULP_SIGNING_KEY_FINGERPRINT \
      --output \$SIGNATURE_PATH

# Check the exit status
STATUS=\$?
if [ \$STATUS -eq 0 ]; then
  echo {\"signature_path\": \"\$SIGNATURE_PATH\"}
else
  exit \$STATUS
fi
EOF
```

!!! WARNING
    Make sure to set `collection_script.sh` and/or `container_script.sh` as key names (using different names would fail operator's execution)

```bash
$ kubectl create secret generic signing-scripts --from-file=collection_script.sh=/tmp/collection_script.sh --from-file=container_script.sh=/tmp/container_script.sh
```

## Configure Pulp CR

* configure Pulp CR with the Secrets created in the previous steps
```yaml
$ kubectl edit pulp
...
spec:
  signing_secret: "signing-secret"
  signing_scripts: "signing-scripts"
...
```

After configuring Pulp CR the operator should create a new job to store the new
signing services into the database:
```bash
$ kubectl get jobs
NAME                          COMPLETIONS   DURATION   AGE
pulp-signing-metadata-54mtp   1/1           15s        30s

$ kubectl logs job/pulp-signing-metadata-54mtp
...
Signing service 'collection-signing-service' has been successfully removed.
Successfully added signing service collection-signing-service for key 66BBFE010CF70CC92826D9AB71684D7912B09BC1.
Signing service 'container-signing-service' has been successfully removed.
Successfully added signing service container-signing-service for key 66BBFE010CF70CC92826D9AB71684D7912B09BC1.
```

double-checking if the signing services are stored in the database:
```bash
$ PULP_PWD=$(kubectl get secrets pulp-admin-password -ojsonpath='{.data.password}'|base64 -d)
$ kubectl exec deployment/pulp-api -- curl -suadmin:$PULP_PWD localhost:24817/pulp/api/v3/signing-services/|jq
{
  "count": 2,
  "next": null,
  "previous": null,
  "results": [
    {
      "pulp_href": "/pulp/api/v3/signing-services/018c0126-1f0c-7803-868d-1a1ee7210db1/",
      "pulp_created": "2023-11-22T11:45:25.042451Z",
      "name": "container-signing-service",
      "public_key": "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQINBGJFjREBEACS1aBb6sqz1kfO/Ii...",
      "pubkey_fingerprint": "66BBFE010CF70CC92826D9AB71684D7912B09BC1",
      "script": "/var/lib/pulp/scripts/container_script.sh"
    },
    {
      "pulp_href": "/pulp/api/v3/signing-services/018c0126-1226-7d7d-abae-aebdc040743c/",
      "pulp_created": "2023-11-22T11:45:21.522412Z",
      "name": "collection-signing-service",
      "public_key": "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQINBGJFjREBE...",
      "pubkey_fingerprint": "66BBFE010CF70CC92826D9AB71684D7912B09BC1",
      "script": "/var/lib/pulp/scripts/collection_script.sh"
    }
  ]
}
```

and it should also redeploy pulpcore pods and mount the gpg key:
```bash
$ kubectl exec deployment/pulp-api -- gpg -k
------------------------------
pub   rsa4096 2022-12-14 [SC]
      66BBFE010CF70CC92826D9AB71684D7912B09BC1
uid           [ultimate] Collection Signing Service (with no passphrase) <pulp@example.com>
sub   rsa2048 2022-12-14 [E]


$ kubectl exec deployment/pulp-worker -- gpg -k
------------------------------
pub   rsa4096 2022-12-14 [SC]
      66BBFE010CF70CC92826D9AB71684D7912B09BC1
uid           [ultimate] Collection Signing Service (with no passphrase) <pulp@example.com>
sub   rsa2048 2022-12-14 [E]
```
