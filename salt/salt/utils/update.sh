#!/bin/sh

zypper ref
zypper up  -y --replacefiles sbd  resource-agents   pacemaker ocfs2-tools libqb	 libdlm	 hawk2	 hawk-apiserver	 ha-cluster-bootstrap  fence-agents  drbd-utils	 drbd	 csync2	 crmsh	 corosync