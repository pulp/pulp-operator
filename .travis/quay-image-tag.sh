#!/bin/bash
#!/usr/bin/env bash

if [[ "$TRAVIS_BRANCH" == "main" ]]; then
  export QUAY_IMAGE_TAG="latest"
fi

if [[ "$TRAVIS_PULL_REQUEST" != "false" ]]
  export QUAY_IMAGE_TAG="pr_$TRAVIS_PULL_REQUEST"
fi
