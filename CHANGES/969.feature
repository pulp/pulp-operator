Modified the reconciliation for `pulpcore-content` to wait for `API` pods get
into a READY state before updating the `Deployment` in case of image version change.
