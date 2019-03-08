FROM opensuse/leap:15
MAINTAINER Kristoffer Gronlund version: 0.1

ENV container docker

RUN zypper -n install systemd; zypper clean ; \
(cd /usr/lib/systemd/system/sysinit.target.wants/; for i in *; do [ $i == systemd-tmpfiles-setup.service ] || rm -f $i; done); \
rm -f /usr/lib/systemd/system/multi-user.target.wants/*;\
rm -f /etc/systemd/system/*.wants/*;\
rm -f /usr/lib/systemd/system/local-fs.target.wants/*; \
rm -f /usr/lib/systemd/system/sockets.target.wants/*udev*; \
rm -f /usr/lib/systemd/system/sockets.target.wants/*initctl*; \
rm -f /usr/lib/systemd/system/basic.target.wants/*;\
rm -f /usr/lib/systemd/system/anaconda.target.wants/*;

VOLUME [ "/sys/fs/cgroup" ]

RUN zypper -n --gpg-auto-import-keys ar obs://network:ha-clustering:Factory network:ha-clustering:Factory
RUN zypper -n --gpg-auto-import-keys ref && zypper -n --gpg-auto-import-keys in libpacemaker-devel pacemaker-cli go git

CMD ["/usr/lib/systemd/systemd", "--system"]

