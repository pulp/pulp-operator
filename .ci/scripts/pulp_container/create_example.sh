#!/usr/bin/env bash

echo "Create a text file and upload it to Pulp"

echo 'Hello world!' > example.txt

ARTIFACT_HREF=$(http --form POST http://localhost/pulp/api/v3/artifacts/ \
    file@./example.txt \
    | jq -r '.pulp_href')

echo "Inspecting new artifact."
http $BASE_ADDR$ARTIFACT_HREF
