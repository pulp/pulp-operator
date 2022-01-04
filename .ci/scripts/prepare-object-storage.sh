#!/bin/bash -e
#!/usr/bin/env bash

if [[ "$CI_TEST_STORAGE" == "azure" ]]; then
  docker run -d -p 10000:10000 --name pulp-azurite mcr.microsoft.com/azure-storage/azurite azurite-blob --blobHost 0.0.0.0
  sleep 5
  AZURE_CONNECTION_STRING="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://pulp-azurite:10000/devstoreaccount1;"
  echo $(minikube ip)   pulp-azurite | sudo tee -a /etc/hosts
  az storage container create --name pulp-test --connection-string $AZURE_CONNECTION_STRING
elif [[ "$CI_TEST_STORAGE" == "s3" ]]; then
  MINIO_ACCESS_KEY=AKIAIT2Z5TDYPX3ARJBA
  MINIO_SECRET_KEY=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
  docker run -d -p 0.0.0.0:9000:9000 --name pulp_minio -e MINIO_ACCESS_KEY=$MINIO_ACCESS_KEY -e MINIO_SECRET_KEY=$MINIO_SECRET_KEY minio/minio server /data
  while ! nc -z localhost 9000; do echo 'Wait minio to startup...' && sleep 0.1; done;
  mc config host add s3 http://localhost:9000 AKIAIT2Z5TDYPX3ARJBA fqRvjWaPU5o0fCqQuUWbj9Fainj2pVZtBCiDiieS --api S3v4
  mc config host rm local
fi
