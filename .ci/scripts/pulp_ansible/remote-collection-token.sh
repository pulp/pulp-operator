# Create a remote that syncs some versions of django into your repository.
pulp ansible remote -t "collection" create \
    --name "abar" \
    --auth-url "https://sso.qa.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token" \
    --token "$ANSIBLE_TOKEN_AUTH" \
    --tls-validation false \
    --url "https://cloud.redhat.com/api/automation-hub/" \
    --requirements $'collections:\n  - testing.ansible_testing_content'
