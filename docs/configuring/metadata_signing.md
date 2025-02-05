# Metadata Signing

It is possible to sign Pulp's metadata so that users can verify the authenticity of an object.
This is done by enabling the Signing Services feature. The steps to enable it are:

* create a gpg key
* create a Secret with a gpg key
* create a Secret with the signing script(s)
* configure Pulp CR

See pulpcore documentation for details on Content Signing: [https://docs.pulpproject.org/pulpcore/workflows/signed-metadata.html#metadata-signing](https://docs.pulpproject.org/pulpcore/workflows/signed-metadata.html#metadata-signing)  
See pulp_container documentation for details on Container Image Signing: [https://docs.pulpproject.org/pulp_container/workflows/sign-images.html](https://docs.pulpproject.org/pulp_container/workflows/sign-images.html)


## Creating a gpg key

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

## Creating a Secret with the gpg key

!!! WARNING
    Make sure to set `signing_service.gpg` as the key name for the `Secret` (using a different name will fail operator's execution)

```bash
$ gpg --export-secret-keys -a pulp@example.com  > /tmp/gpg_private_key.gpg
$ kubectl create secret generic signing-secret --from-file=signing_service.gpg=/tmp/gpg_private_key.gpg
```

## Creating a Secret with the signing scripts

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

* example of an APT signing script
```bash
$ SIGNING_SCRIPT_PATH=/tmp
$ APT_SIGNING_SCRIPT=apt_script.sh
$ cat<<EOF> "$SIGNING_SCRIPT_PATH/$APT_SIGNING_SCRIPT"
#!/bin/bash

set -e

RELEASE_FILE="\$(/usr/bin/readlink -f \$1)"
OUTPUT_DIR="\$(/usr/bin/mktemp -d)"
DETACHED_SIGNATURE_PATH="\${OUTPUT_DIR}/Release.gpg"
INLINE_SIGNATURE_PATH="\${OUTPUT_DIR}/InRelease"
COMMON_GPG_OPTS="--batch --armor --digest-algo SHA256 --default-key \$PULP_SIGNING_KEY_FINGERPRINT"

# Create a detached signature
/usr/bin/gpg \${COMMON_GPG_OPTS} \
  --detach-sign \
  --output "\${DETACHED_SIGNATURE_PATH}" \
  "\${RELEASE_FILE}"

# Create an inline signature
/usr/bin/gpg \${COMMON_GPG_OPTS} \
  --clearsign \
  --output "\${INLINE_SIGNATURE_PATH}" \
  "\${RELEASE_FILE}"

echo { \
       \"signatures\": { \
         \"inline\": \"\${INLINE_SIGNATURE_PATH}\", \
         \"detached\": \"\${DETACHED_SIGNATURE_PATH}\" \
       } \
     }

EOF
```

* example of an RPM signing script
```bash
$ SIGNING_SCRIPT_PATH=/tmp
$ APT_SIGNING_SCRIPT=rpm_script.sh
$ cat<<EOF> "$SIGNING_SCRIPT_PATH/$RPM_SIGNING_SCRIPT"
#!/bin/bash

set -e

FILE_PATH=\$1
GPG_FINGERPRINT="\$PULP_SIGNING_KEY_FINGERPRINT"
GPG_HOME=/var/lib/pulp/.gnupg/
GPG_BIN=/usr/bin/gpg

# Make sure the gpg public key has been imported
gpg --export -a \$GPG_FINGERPRINT > /tmp/RPM-GPG-KEY
rpm --import /tmp/RPM-GPG-KEY

rpm \
    --define "_signature gpg" \
    --define "_gpg_path \$GPG_HOME" \
    --define "_gpg_name \$GPG_FINGERPRINT" \
    --define "_gpgbin \$GPG_BIN" \
    --define "__gpg_sign_cmd %{__gpg} gpg --force-v3-sigs --batch --verbose --no-armor --no-secmem-warning -u %{_gpg_name} -sbo %{__signature_filename} --digest-algo sha256 -v --pinentry-mode loopback %{__plaintext_filename}" \
    --addsign "\$FILE_PATH" 1> /dev/null

STATUS=\$?
if [[ \$STATUS -eq 0 ]]; then
   echo {\"rpm_package\": \"\$FILE_PATH\"}
else
   exit \$STATUS
fi
EOF
```

!!! WARNING
    Make sure to set `collection_script.sh`, `container_script.sh`, `apt_script.sh`, and/or `rpm_script.sh` as key names (using different names would fail operator's execution)

```bash
$ kubectl create secret generic signing-scripts --from-file=collection_script.sh=/tmp/collection_script.sh --from-file=container_script.sh=/tmp/container_script.sh --from-file=apt_script.sh=/tmp/apt_script.sh --from-file=rpm_script.sh=/tmp/rpm_script.sh
```

## Configuring Pulp CR

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
Signing service 'apt-signing-service' has been successfully removed.
Successfully added signing service apt-signing-service for key 66BBFE010CF70CC92826D9AB71684D7912B09BC1.
Signing service 'rpm-signing-service' has been successfully removed.
Successfully added signing service rpm-signing-service for key 66BBFE010CF70CC92826D9AB71684D7912B09BC1.
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
      "pulp_href": "/pulp/api/v3/signing-services/0191e929-31f4-77d1-841e-2b545cf45da3/",
      "pulp_created": "2024-09-13T02:14:36.846612Z",
      "pulp_last_updated": "2024-09-13T02:14:36.846627Z",
      "name": "apt-signing-service",
      "public_key": "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQGiBGbjgnIRBACc7VbJTNbDRja...",
      "pubkey_fingerprint": "66BBFE010CF70CC92826D9AB71684D7912B09BC1",
      "script": "/var/lib/pulp/scripts/apt_script.sh"
    },
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
    },
    {
      "pulp_href": "/pulp/api/v3/signing-services/0194a988-684c-7dda-9b16-2bb614a8e1ba/",
      "pulp_created": "2025-01-27T20:51:17.323038Z",
      "name": "rpm-signing-service",
      "public_key": "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQGNBGeSYcYBDADaKR4OZ+y...",
      "pubkey_fingerprint": "66BBFE010CF70CC92826D9AB71684D7912B09BC1",
      "script": "/var/lib/pulp/scripts/rpm_script.sh"
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
