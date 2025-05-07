# LDAP Authentication

By default, Pulp authenticates each request with a username and password against its own user database. Requests can also authenticate by using an LDAP service. Pulp-operator can do that using [`django-auth-ldap`](https://django-auth-ldap.readthedocs.io/en/latest/).


## Configure LDAP (without encrypted connection)

The first step to allow LDAP integration with Pulp is to create a `Secret` with the LDAP service information.  
Here is an example of a `Secret` config:
```yaml
kubectl apply -f- <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: pulp-ldap-secret
stringData:
  auth_ldap_server_uri: "ldap://10.0.0.1"
  auth_ldap_bind_dn: "cn=admin,dc=example,dc=org"
  auth_ldap_bind_password: "admin"
  auth_ldap_group_search: LDAPSearch("ou=groups,dc=example,dc=org",ldap.SCOPE_SUBTREE,"(objectClass=posixGroup)")
  auth_ldap_user_search: LDAPSearch("ou=users,dc=example,dc=org", ldap.SCOPE_SUBTREE, "(uid=%(user)s)")
  auth_ldap_group_type: PosixGroupType(name_attr='cn')
EOF
```

after creating the `Secret`, we need to update Pulp CR with it:
```yaml
kubectl edit pulp
...
spec:
  ldap:
    config: pulp-ldap-secret
...
```

pulp-operator will notice the changes and will redeploy `pulpcore` pods with the new settings.  
Check `django-auth-ldap` documentation to see the list of possible configurations: [https://django-auth-ldap.readthedocs.io/en/latest/reference.html#reference](https://django-auth-ldap.readthedocs.io/en/latest/reference.html#reference)


## Configure LDAP with TLS

!!! Info
    LDAP+TLS connection with client cert authentication is not available yet.

!!! WARNING
    This is a tech preview feature! There are some issues with Pulp and `django-auth-ldap` that is under investigation.  
    To workaround some possible exceptions while using LDAP+TLS, we made the tests modifying `pulp-minimal` container image with:
    ```
    FROM quay.io/pulp/pulp-minimal:3.32
    RUN pip3 install django-auth-ldap==4.5.0
    RUN sed -i '126i \            if options != None:' /usr/local/lib/python3.8/site-packages/django_auth_ldap/backend.py
    RUN sed -i '127i \                options = {int(k):v for k,v in options.items()}' /usr/local/lib/python3.8/site-packages/django_auth_ldap/backend.py
    RUN sed -i '859i \                optInt = int(opt)' /usr/local/lib/python3.8/site-packages/django_auth_ldap/backend.py
    RUN sed -i '860s/opt, value/optInt, value/' /usr/local/lib/python3.8/site-packages/django_auth_ldap/backend.py
    ```


#### Ignoring TLS errors

The following configuration will configure ldap+tls connection, but ignoring the certificate validations (self signed or expired certs, non-trusted CAs, etc):
```yaml
kubectl apply -f- <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: pulp-ldap-secret
stringData:
  auth_ldap_server_uri: "ldap://10.0.0.1"
  auth_ldap_start_tls: "True"
  auth_ldap_bind_dn: "cn=admin,dc=example,dc=org"
  auth_ldap_bind_password: "admin"
  auth_ldap_group_search: LDAPSearch("ou=groups,dc=example,dc=org",ldap.SCOPE_SUBTREE,"(objectClass=posixGroup)")
  auth_ldap_user_search: LDAPSearch("ou=users,dc=example,dc=org", ldap.SCOPE_SUBTREE, "(uid=%(user)s)")
  auth_ldap_group_type: PosixGroupType(name_attr='cn')
  auth_ldap_global_options: |-
    { ldap.OPT_X_TLS_REQUIRE_CERT: ldap.OPT_X_TLS_ALLOW }
EOF
```

after creating the `Secret`, we need to update Pulp CR with it:
```yaml
kubectl edit pulp
...
spec:
  ldap:
    config: pulp-ldap-secret
...
```

pulp-operator will notice the changes and will redeploy `pulpcore` pods with the new settings.  
Check `django-auth-ldap` documentation to see the list of possible configurations: [https://django-auth-ldap.readthedocs.io/en/latest/reference.html#reference](https://django-auth-ldap.readthedocs.io/en/latest/reference.html#reference)

---
#### Providing a CA

If the certificate used in LDAP server is signed by a "*custom*" CA, it is possible to configure Pulp to pass it to the LDAP connection.
The first step is to create a `Secret` with the CA chain:
```yaml
oc apply -f-<<EOF
apiVersion: v1
kind: Secret
metadata:
  name: ldap-ca-cert
stringData:
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIC0zCCAlmgAwIBAgIUCfQ+m0pgZ/BjYAJvxrn/bdGNZokwCgYIKoZIzj0EAwMw
    gZYxCzAJBgNVBAYTAlVTMRUwEwYDVQQKEwxBMUEgQ2FyIFdhc2gxJDAiBgNVBAsT
    ...
    -----END CERTIFICATE-----
EOF
```

now, we need to create a new `Secret` with the LDAP settings:
```yaml
kubectl apply -f- <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: pulp-ldap-secret
stringData:
  auth_ldap_server_uri: "ldap://10.0.0.1"
  auth_ldap_start_tls: "True"
  auth_ldap_bind_dn: "cn=admin,dc=example,dc=org"
  auth_ldap_bind_password: "admin"
  auth_ldap_group_search: LDAPSearch("ou=groups,dc=example,dc=org",ldap.SCOPE_SUBTREE,"(objectClass=posixGroup)")
  auth_ldap_user_search: LDAPSearch("ou=users,dc=example,dc=org", ldap.SCOPE_SUBTREE, "(uid=%(user)s)")
  auth_ldap_group_type: PosixGroupType(name_attr='cn')
  auth_ldap_connection_options: |-
    { ldap.OPT_X_TLS_CACERTFILE: AUTH_LDAP_CA_FILE, ldap.OPT_X_TLS_NEWCTX: 0, ldap.OPT_X_TLS_REQUIRE_CERT: ldap.OPT_X_TLS_ALLOW }
  auth_ldap_ca_file: "/tmp/ca.crt"      <---------------- make sure to define filename with the same value as the key defined in ldap-cert Secret (ca.crt in this example)
EOF
```

after creating the `Secret`, we need to update Pulp CR with it:
```yaml
kubectl edit pulp
...
spec:
  ldap:
    config: pulp-ldap-secret
    ca: ldap-ca-cert
...
```

pulp-operator will notice the changes and will redeploy `pulpcore` pods with the new settings.  
Check `django-auth-ldap` documentation to see the list of possible configurations: [https://django-auth-ldap.readthedocs.io/en/latest/reference.html#reference](https://django-auth-ldap.readthedocs.io/en/latest/reference.html#reference)
