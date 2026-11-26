FROM docker.io/opensuse/tumbleweed

# Build and runtime dependencies for the standalone Go server and cluster tests.
RUN zypper -n refresh \
 && zypper -n install --no-recommends \
      systemd openssh-server iproute2 \
      make gcc go pam-devel libpacemaker-devel libqb-devel libxml2-devel git \
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
