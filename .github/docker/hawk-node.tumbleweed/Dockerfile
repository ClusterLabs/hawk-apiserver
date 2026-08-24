FROM docker.io/opensuse/tumbleweed

# install deps (hawk2 will pull all the gems, but uglifier)
# libglue-devel is for stonith:external/ssh params hostlist
RUN zypper -n refresh \
 && zypper -n install --no-recommends \
      systemd openssh-server \
      make gcc go pam-devel libpacemaker-devel libqb-devel libxml2-devel hawk2 nodejs git \
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

ENV GEM_HOME=/usr/libexec/hawk/vendor/

USER nobody

CMD ["/usr/lib/systemd/systemd", "--system"]