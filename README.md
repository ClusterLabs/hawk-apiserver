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

Go and Rails separation
=======================
The Go server authenticates directly through `/usr/sbin/hawk_chkpwd passwd <user>`.
It does not contact Puma, use Rails session cookies/secrets, or fall back to Rails.
The PAM helper remains a shared system executable; no PAM or cluster settings are
changed by this separation.

`/login`, `/login.json`, `/logout`, and `/logout.json` are native Go endpoints.
The HTML login keeps the existing `session[username]` and `session[password]`
field names. `/login.json` also accepts these form fields for dashboard clients.
Sessions last ten days, survive server restarts, and are invalidated on logout.
Passwords are sent to the helper through stdin and are never stored in sessions.

Go owns the `hawk_go_session` cookie and a separate private session directory,
`/var/lib/hawk/go-sessions` by default. The Go service account must be able to
own and create/write this directory with mode 0700. Override it with `-session-dir`.
Separate deployments can use separate directories. Existing Rails logins are
not imported; sign into Go again.

Listener, TLS and session settings come from command-line arguments. The Go
server no longer accepts `-config` or reads `/etc/hawk/server.json`. `/monitor`
and `/monitor.json` are handled in Go. Templates and static assets are read
relative to the working directory: use the repository root for development,
or `/usr/share/hawk` when running the installed binary. Remove `-config` from
existing service commands before deploying this version.
No service units, `/etc/sysconfig/hawk`, certificates, or firewall settings are
changed by this code.

Unported endpoints return 404. This includes legacy table JSON/rename forms used
by Edit Configuration, Command Log's `commands.json`, the report transition
viewer, and unported batch pages. Their replacement is outside this separation.
Native editor APIs and the existing Go batch simulation logic remain in place.

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
./hawk-apiserver -key /etc/hawk/hawk.key -cert /etc/hawk/hawk.pem -port 7631
```
The TLS files `/etc/hawk/hawk.key` and `/etc/hawk/hawk.pem` are created by `crm cluster init ...`.
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
                "-port", "7631"
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