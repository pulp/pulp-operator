#!/usr/bin/env bash

echo "Create a Containerfile that expects foo/bar/example.txt inside /pulp_working_directory."

echo 'FROM centos:7

# Copy a file using COPY statement. Use the relative path specified in the 'artifacts' parameter.
COPY foo/bar/example.txt /inside-image.txt

# Print the content of the file when the container starts
CMD ["cat", "/inside-image.txt"]' >> Containerfile