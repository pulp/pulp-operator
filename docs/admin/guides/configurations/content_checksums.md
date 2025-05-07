# Configure Pulp Allowed Content Checksums

During repositories synchronization, Pulp checks the downloaded files against a list
of checksums algorithms. Together with a valid (and trusted) release file signature this will guarantee the integrity of the synchronized repository.

The list of checksums algorithms is defined using the `ALLOWED_CONTENT_CHECKSUMS` setting.  
For more information on how `Pulp` uses the checksums check: [https://pulpproject.org/pulp_deb/docs/user/guides/checksums/#configuring-checksums](https://pulpproject.org/pulp_deb/docs/user/guides/checksums/#configuring-checksums)


To set the `ALLOWED_CONTENT_CHECKSUMS` in Pulp Operator, update Pulp CR with:
```yaml
spec:
  allowed_content_checksums:
  - sha256
```

!!! note
    `sha256` is a mandatory checksum.
    The possible checksums are: `md5`, `sha1`,`sha256`,`sha512`.

After modifying the `allowed_content_checksums` field in Pulp CR, the operator will create a kubernetes job to run the `pulpcore-manager handle-artifact-checksums` command to ensure database consistency.

!!! note
    Missing checksums will need to be recalculated for all your artifacts which can take some time.
