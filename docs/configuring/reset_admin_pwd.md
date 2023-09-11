# Reseting Pulp Admin Password

The password from Pulp admin user is managed by Pulp Operator. If a custom `Secret` is not provided
during operator's installation, a [random password](/pulp_operator/configuring/secrets/#pulp-admin-password) will be provided.

To change the admin password, the first thing to do is to get the name of the `admin_password_secret` `Secret`:
```sh
$ kubectl get pulp <PULP CR NAME>  -ojsonpath='{.spec.admin_password_secret}'

## for example:
$ kubectl get pulp example-pulp  -ojsonpath='{.spec.admin_password_secret}'
example-pulp-admin-password
```

Now, update the `Secret` with a new password:
```sh 
$ kubectl apply -f-<<EOF
apiVersion: v1
kind: Secret
metadata:
 name: '<SECRET NAME>'
stringData:
 password: '<NEW PASSWORD>'
EOF


## for example:
$ kubectl apply -f-<<EOF
apiVersion: v1
kind: Secret
metadata:
 name: 'example-pulp-admin-password'
stringData:
 password: 'mysupersecretpassword'
EOF
```

The operator should notice the `Secret` change and create a new `Job` to update the password:
```sh
$ kubectl get jobs
NAME                              COMPLETIONS   DURATION   AGE
reset-admin-password-3140016014   1/1           11s        100s
```

Checking the logs from `reset-admin-password` `Job`:
```sh
$ kubectl logs reset-admin-password-3140016014-k5d4b
Waiting on postgresql to start...     <------ waiting for DB connection before proceed
Checking postgres host 10.0.0.1
Checking postgres port 5432
Postgres started!
Checking for database migrations   <------ waiting for any running migration
Database migrated!
pulp admin can be initialized.
Successfully set password for "admin" user.   <------- password updated!
```


!!! WARNING
    We don't recommend running the `pulpcore-manager reset-admin-password` command nor updating the password via `/api/v3/users` endpoint.  
    Any modification in `admin_password_secret` `Secret` will override Pulp admin password.
