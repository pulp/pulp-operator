#!/usr/bin/env bash

# remove the repository and its content from the current filesystem
pulp container distribution destroy --name "test/fixture"
pulp orphan cleanup --protection-time 0
