#!/bin/bash -e
#!/usr/bin/env bash

if [[ "$COMPONENT_TYPE" == "azure" ]]; then
  docker run -d -p 10000:10000 --name pulp-azurite mcr.microsoft.com/azure-storage/azurite azurite-blob --blobHost 0.0.0.0 --skipApiVersionCheck
  sleep 5
  AZURE_CONNECTION_STRING="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://pulp-azurite:10000/devstoreaccount1;"
  echo $(minikube ip)   pulp-azurite | sudo tee -a /etc/hosts
  az storage container create --name pulp-test --connection-string $AZURE_CONNECTION_STRING
elif [[ "$COMPONENT_TYPE" == "s3" ]]; then
  export RUSTFS_ACCESS_KEY=AKIAIT2Z5TDYPX3ARJBA
  export RUSTFS_SECRET_KEY=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
  docker run -d -p 0.0.0.0:9000:9000 --name pulp_rustfs -e RUSTFS_ACCESS_KEY=$RUSTFS_ACCESS_KEY -e RUSTFS_SECRET_KEY=$RUSTFS_SECRET_KEY rustfs/rustfs server /data
  while ! nc -z $(minikube ip) 9000; do echo 'Wait rustfs to startup...' && sleep 0.1; done;
  echo $(minikube ip) pulp_rustfs | sudo tee -a /etc/hosts
  sed -i "s/pulp_rustfs/$(minikube ip)/g" config/samples/simple.s3.ci.yaml
  pip install boto3
  python3 -c "
import boto3
from botocore.exceptions import ClientError
client = boto3.client(
    's3',
    aws_access_key_id='${RUSTFS_ACCESS_KEY}',
    aws_secret_access_key='${RUSTFS_SECRET_KEY}',
    endpoint_url='http://$(minikube ip):9000',
    region_name='us-east-1')
try:
    client.create_bucket(Bucket='pulp3', CreateBucketConfiguration={'LocationConstraint': 'us-east-1'})
except ClientError as exc:
    if exc.response['Error']['Code'] not in ('BucketAlreadyOwnedByYou', 'BucketAlreadyExists'):
        raise
"
fi
