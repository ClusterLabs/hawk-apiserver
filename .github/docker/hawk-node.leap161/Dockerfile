FROM docker.io/opensuse/leap:16.1

# Build and runtime dependencies for the standalone Go server and cluster tests.
# Replace the minimal curl packages in one transaction to avoid overlapping
# library files, then sync the base image with the enabled Leap repositories.
RUN zypper -n refresh \
 && if rpm -q libcurl-mini4 >/dev/null; then \
      set -- '!libcurl-mini4'; \
      if rpm -q curl-mini >/dev/null; then set -- "$@" '!curl-mini'; fi; \
      zypper -n install --no-recommends --force curl libcurl4 "$@"; \
    fi \
 && zypper -n dist-upgrade --allow-downgrade --no-allow-vendor-change --no-recommends \
 && zypper -n install --no-recommends \
      systemd openssh-server iproute2 \
      make gcc go pam-devel pacemaker-devel libqb-devel libxml2-devel git \
      pacemaker corosync crmsh resource-agents openssl sudo sbd graphviz \
 && zypper -n clean -a

# allow root password login (specifically for the tests)
RUN sed -i \
      -e 's/^#\?PermitRootLogin .*/PermitRootLogin yes/' \
      -e 's/^#\?PasswordAuthentication .*/PasswordAuthentication yes/' \
      /usr/etc/ssh/sshd_config

RUN ssh-keygen -t rsa -N "" -f /root/.ssh/id_rsa \
 && cat /root/.ssh/id_rsa.pub >> /root/.ssh/authorized_keys

# unlock root and set password to "linux"
RUN passwd -u root || true && echo -e "linux\nlinux" | passwd root

# start sshd under systemd (`crm cluster init` will start it later anyway)
RUN systemctl enable sshd

CMD ["/usr/lib/systemd/systemd", "--system"]
