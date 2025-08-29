#!/usr/bin/env bash
set -e

export BASE_ADDR=${BASE_ADDR:-http://pulp:80}

if [ -z "$(pip freeze | grep pulp-cli)" ]; then
  echo "Installing pulp-cli"
  pip install pulp-cli[pygments]
fi

# Set up CLI config file
echo "Configuring pulp-cli"
mkdir -p ~/.config/pulp
cat > ~/.config/pulp/cli.toml << EOF
[cli]
base_url = "$BASE_ADDR"
verify_ssl = false
format = "json"
username = "admin"
password = "password"
EOF
