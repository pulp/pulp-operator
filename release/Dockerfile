FROM  registry.fedoraproject.org/fedora:38

ARG TOWNCRIER_VERSION="${TOWNCRIER_VERSION:-19.9.0}"
ARG GOLANG_VERSION="${GOLANG_VERSION:-1.20.11}"
ARG CONTROLLER_TOOLS_VERSION="${CONTROLLER_TOOLS_VERSION:-v0.9.2}"

ENV PATH="${PATH}:/usr/local/go/bin"
ENV GIT_RELEASE_BRANCH="${GIT_RELEASE_BRANCH:-release-beta-4}"
ENV GIT_UPSTREAM_REMOTE_NAME="${GIT_UPSTREAM_REMOTE_NAME:-upstream}"
ENV GIT_REMOTE_NAME="${GIT_REMOTE_NAME:-origin}"

ENV PULP_OPERATOR_SOURCE_PATH="${PULP_OPERATOR_SOURCE_PATH:-/app/pulp-operator}"
ENV PULP_OPERATOR_REPLACE_VERSION="${PULP_OPERATOR_REPLACE_VERSION:-1.0.0-beta.3}"
ENV PULP_OPERATOR_RELEASE_VERSION="${PULP_OPERATOR_RELEASE_VERSION:-1.0.0-beta.4}"
ENV PULP_OPERATOR_DEV_VERSION="${PULP_OPERATOR_DEV_VERSION:-1.0.0-beta.5}"

ENV OPERATORHUB_REPO_PATH="${OPERATORHUB_REPO_PATH:-/app/community-operators}"
ENV GIT_OPERATORHUB_RELEASE_BRANCH="${GIT_OPERATORHUB_RELEASE_BRANCH:-pulp-operator-$PULP_OPERATOR_RELEASE_VERSION}"
ENV GIT_OPERATORHUB_UPSTREAM_REMOTE_NAME="${GIT_OPERATORHUB_UPSTREAM_REMOTE_NAME:-upstream}"
ENV GIT_OPERATORHUB_REMOTE_NAME="${GIT_UPSTREAM_REMOTE_NAME:-origin}"

ENV REDHAT_CATALOG_REPO_PATH="${REDHAT_CATALOG_REPO_PATH:-/app/community-operators-prod}"
ENV GIT_REDHAT_CATALOG_RELEASE_BRANCH="${GIT_REDHAT_CATALOG_RELEASE_BRANCH:-pulp-operator-$PULP_OPERATOR_RELEASE_VERSION}"
ENV GIT_REDHAT_CATALOG_UPSTREAM_REMOTE_NAME="${GIT_REDHAT_CATALOG_UPSTREAM_REMOTE_NAME:-upstream}"
ENV GIT_REDHAT_CATALOG_REMOTE_NAME="${GIT_REDHAT_CATALOG_REMOTE_NAME:-origin}"

RUN dnf install -y \
  git \
  make \
  pip \
  which \
  && dnf clean all && rm -rf /var/cache/dnf

RUN pip install --no-cache-dir towncrier==${TOWNCRIER_VERSION}

RUN curl -sSLo golang.tar.gz "https://go.dev/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz" && \
	tar -C /usr/local -xzf golang.tar.gz && \
  rm -f golang.tar.gz

RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_TOOLS_VERSION}

RUN mkdir /app
WORKDIR /app

COPY release.sh /app
COPY ssh_config /root/.ssh/config


ENTRYPOINT ["./release.sh"]