# Distributions are created asynchronously. Create one, and specify the repository version that will
# be served at the base path specified.
pulp ansible distribution create --name "bar" --base-path "some_content" --repository "foo" --version 0
