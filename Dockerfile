FROM quay.io/operator-framework/ansible-operator:v1.23.0

ENV ANSIBLE_FORCE_COLOR=true
ENV ANSIBLE_SHOW_TASK_PATH_ON_FAILURE=true

USER root
RUN dnf update --security --bugfix -y && \
    dnf install -y openssl

USER ${USER_UID}

COPY requirements.yml ${HOME}/requirements.yml
RUN ansible-galaxy collection install --force -r ${HOME}/requirements.yml \
 && chmod -R ug+rwx ${HOME}/.ansible

COPY watches.yaml ${HOME}/watches.yaml
COPY roles/ ${HOME}/roles/
COPY playbooks/ ${HOME}/playbooks/

ENTRYPOINT ["/tini", "--", "/usr/local/bin/ansible-operator", "run", "--watches-file=./watches.yaml", "--reconcile-period=0s"]
