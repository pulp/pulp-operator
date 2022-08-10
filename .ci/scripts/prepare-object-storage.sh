#!/bin/bash -e
#!/usr/bin/env bash

if [[ "$CI_TEST_STORAGE" == "azure" ]]; then
  docker run -d -p 10000:10000 --name pulp-azurite mcr.microsoft.com/azure-storage/azurite azurite-blob --blobHost 0.0.0.0
  sleep 5
  AZURE_CONNECTION_STRING="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://pulp-azurite:10000/devstoreaccount1;"
  echo $(minikube ip)   pulp-azurite | sudo tee -a /etc/hosts
  az storage container create --name pulp-test --connection-string $AZURE_CONNECTION_STRING
elif [[ "$CI_TEST_STORAGE" == "s3" ]]; then
  export MINIO_ACCESS_KEY=AKIAIT2Z5TDYPX3ARJBA
  export MINIO_SECRET_KEY=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
  docker run -d -p 0.0.0.0:9000:9000 --name pulp_minio -e MINIO_ACCESS_KEY=$MINIO_ACCESS_KEY -e MINIO_SECRET_KEY=$MINIO_SECRET_KEY minio/minio server /data
  wget https://dl.min.io/client/mc/release/linux-amd64/mc
  sudo mv mc /usr/local/bin/
  sudo chmod +x /usr/local/bin/mc
  while ! nc -z $(minikube ip) 9000; do echo 'Wait minio to startup...' && sleep 0.1; done;
  echo $(minikube ip)   pulp_minio | sudo tee -a /etc/hosts
  sed -i "s/pulp_minio/$(minikube ip)/g" config/samples/galaxy.s3.ci.yaml
  mc config host add s3 http://$(minikube ip):9000 AKIAIT2Z5TDYPX3ARJBA fqRvjWaPU5o0fCqQuUWbj9Fainj2pVZtBCiDiieS --api S3v4
  mc config host rm local
  mc mb s3/pulp3 --region us-east-1
fi
