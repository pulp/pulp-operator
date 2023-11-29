#!/bin/bash

set -eu

function deploy_ldap_server {
kubectl apply -f-<<EOF
---
apiVersion: v1
kind: Pod
metadata:
  name: ldap-server
  labels:
    app.kubernetes.io/name: ldap
spec:
  containers:
  - name: ldap
    image: docker.io/osixia/openldap:1.3.0
    ports:
    - containerPort: 389
    - containerPort: 636
    env:
    - name: LDAP_TLS_VERIFY_CLIENT
      value: try
---
apiVersion: v1
kind: Service
metadata:
  name: ldap
spec:
  selector:
    app.kubernetes.io/name: ldap
  ports:
    - name: ldap-389
      protocol: TCP
      port: 389
      targetPort: 389
    - name: ldap-636
      protocol: TCP
      port: 636
      targetPort: 636
EOF
}

function add_users_and_groups {
kubectl exec -i ldap-server -- bash << COMMANDS
cat<<EOF>/tmp/a
dn: ou=users,dc=example,dc=org
objectClass: organizationalUnit
ou: users

dn: ou=groups,dc=example,dc=org
objectClass: organizationalUnit
ou: groups
EOF

cat<<EOF>/tmp/b
dn: uid=alice,ou=users,dc=example,dc=org
changetype: add
objectClass: inetOrgPerson
givenName: Alice
sn: Smith
mail: alice@example.com
cn: Alice Smith
uid: alice

dn: uid=bob,ou=users,dc=example,dc=org
changetype: add
objectClass: inetOrgPerson
givenName: Bob
sn: Traveller
mail: bob@example.com
cn: Bob Traveller
uid: bob

dn: uid=eve,ou=users,dc=example,dc=org
changetype: add
objectClass: inetOrgPerson
givenName: Eve
sn: Evil
mail: eve@example.com
cn: Eve Evil
uid: eve
EOF

cat<<EOF>/tmp/c
dn: cn=fileGlobalAdmin,ou=groups,dc=example,dc=org
cn: fileGlobalAdmin
gidnumber: 10004
memberuid: alice
objectclass: posixGroup
objectclass: top
EOF


ldapadd -x -H ldap://localhost -D "cn=admin,dc=example,dc=org" -w admin -f /tmp/a
ldapadd -x -H ldap://localhost -D "cn=admin,dc=example,dc=org" -w admin -f /tmp/b
ldapadd -x -H ldap://localhost -D "cn=admin,dc=example,dc=org" -w admin -f /tmp/c

ldappasswd -s alice -D "cn=admin,dc=example,dc=org" -x "uid=alice,ou=users,dc=example,dc=org" -w admin
ldappasswd -s bob -D "cn=admin,dc=example,dc=org" -x "uid=bob,ou=users,dc=example,dc=org" -w admin
ldappasswd -s eve -D "cn=admin,dc=example,dc=org" -x "uid=eve,ou=users,dc=example,dc=org" -w admin

COMMANDS
}

function build_pulp_minimal_image {
cat<<EOF>/tmp/Dockerfile
FROM quay.io/pulp/pulp-minimal:stable
RUN pip3 install django-auth-ldap==4.6.0
RUN sed -i '126i \            if options != None:' /usr/local/lib/python3.8/site-packages/django_auth_ldap/backend.py
RUN sed -i '127i \                options = {int(k):v for k,v in options.items()}' /usr/local/lib/python3.8/site-packages/django_auth_ldap/backend.py
RUN sed -i '859i \                optInt = int(opt)' /usr/local/lib/python3.8/site-packages/django_auth_ldap/backend.py
RUN sed -i '860s/opt, value/optInt, value/' /usr/local/lib/python3.8/site-packages/django_auth_ldap/backend.py
EOF

docker build --no-cache -t localhost/pulp-minimal:stable -f /tmp/Dockerfile /tmp

# for reference, if deploying in a kind cluster with a local registry
#docker build --no-cache -t localhost:5001/pulp-minimal:stable -f /tmp/Dockerfile /tmp
#docker push localhost:5001/pulp-minimal:stable
}


echo "Deploying ldap server as a pod ..."
deploy_ldap_server
kubectl wait --for=condition=Ready pod/ldap-server
sleep 5
kubectl exec ldap-server -- ldapsearch -x -H ldap://localhost -b dc=example,dc=org -D "cn=admin,dc=example,dc=org" -w admin

echo "Creating ldap users and groups ..."
add_users_and_groups

echo "Checking users ..."
kubectl exec ldap-server -- ldapsearch -x -H ldap://localhost -b dc=example,dc=org -D "cn=admin,dc=example,dc=org" -w admin

echo "Building pulp-minimal image with django-auth-ldap support ..."
build_pulp_minimal_image
