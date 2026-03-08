What is it?
===========
This is a new branch of hawk-apiserver aimed at reproducing some of the functionality
from the original hawk2.

Why is it good?
===============
The goal is to migrate all functionality from the original hawk2 RoR project
to the Go-based hawk-apiserver project, so that we can eventually remove hawk entirely.
This means we will no longer need to maintain the Hawk ruby packages.

What is already implemented?
============================
As of 17.11.2025, the following entry point exists:
```
cib/live/primitives/{resouce-id}/edit
```
it is currently in alpha state. You can access it via:
```
MONITORING -> Status -> [{Resource-ID} dwopdown menu ▼] -> ✏️ Edit
```

There are also custom JavaScript classes in static/js/classes designed to simplify the development.

What about hawk2?
=================
The idea is to migrate the functionality from Ruby to Go without big changes to the Ruby part.
However, some changes are unavoidable. For example, when redirecting from one web page to another, Ruby passes the arguments through its internal cookies, which are not accessible from Go. Therefore, Ruby must to provide the arguments explicitely through the URL.
For now, it's recommented to simply ignore this issue. If you want all tests to pass, you can use
the patched hawk from https://build.suse.de/package/show/home:aburlakov:branches:SUSE:SLE-15-SP6:Update/hawk2

How to install?
===============
Scenario: you have a fresh Leap15.6 virtual machine.

The easy way is to install it from https://build.suse.de/package/show/home:aburlakov:branches:SUSE:SLE-15-SP6:Update/hawk-apiserver

```
zypper ar -p1 https://download.suse.de/ibs/home:/aburlakov:/branches:/SUSE:/SLE-15-SP6:/Update/standard/?ssl_verify=no myrepo
zypper ref
zypper in hawk2
zypper se --details hawk2 # make sure the hawk2 is installed from myrepo

crm cluster init -s /dev/vdb -y
systemctl status hawk hawk-backend # make sure both are green
```

This script will also install hawk-apiserver, but that's fine even if you don't need it.

How to build?
=============
1) install hawk2 either from the official repo, or from the link above.

2) Install necessary dependencies:
```
zypper in make git go golang-packaging libpacemaker-devel libqb-devel libxml2-devel
```

3) Build:
```
git clone https://github.com/aleksei-burlakov/hawk-apiserver.git
cd hawk-apiserver
git checkout ver0.1.0
make
```

4) Start:
```
./hawk-apiserver -key /etc/hawk/hawk.key -cert /etc/hawk/hawk.pem -port 7631 -config /etc/hawk/server.json
```
The `/etc/hawk/hawk.key`, `/etc/hawk/hawk.pem`, and `/etc/hawk/server.json` are created by `crm cluster init ...`
Don't use the port `7630`, because it's already bound to the default `hawk.service` .

5) Open `https://127.0.0.1:7631`, use the default credentials. User: `hacluster`, Password: `linux`.

How to debug?
=============

1) Install VS Code:
```
rpm --import https://packages.microsoft.com/keys/microsoft.asc
zypper ar https://packages.microsoft.com/yumrepos/vscode vscode
zypper refresh
zypper install -y code

go install github.com/go-delve/delve/cmd/dlv@latest

cd hawk-apiserver
code . --user-data-dir=".vscode" --no-sandbox
```

2) Install Go extension.

3) Use the folowing launch.json
```
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}",
            "dlvToolPath": "/usr/bin/dlv",
            "args": [
                "-key", "/etc/hawk/hawk.key",
                "-cert", "/etc/hawk/hawk.pem",
                "-port", "7631",
                "-config", "/etc/hawk/server.json"
            ]
        }
    ]
}
```


Tests
=====
The hawk/e2e_tests were copied here. Additionally to existing tests, there were added

* test_copy_primitive: cool_primitive --> cool_primitive + hot_primitive
* test_rename_primitive: hot_primitive --> dummy_primitive
* test_delete_primitive: Delete the dummy_primitive

All of them test the same entry point cib/live/primitives/{resouce-id}/edit.