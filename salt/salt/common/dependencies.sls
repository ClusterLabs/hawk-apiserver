server:monitoring:
  pkgrepo.managed:
    - humanname: server:monitoring
    - baseurl: https://download.opensuse.org/repositories/server:/monitoring/openSUSE_Leap_15.0/
    - refresh: True
    - gpgautoimport: True

devel:tools:
  pkgrepo.managed:
    - humanname: devel:tools
    - baseurl: http://download.opensuse.org/repositories/devel:/tools/openSUSE_Leap_15.0/
    - refresh: True
    - gpgautoimport: True

dev_packages:
  pkg.installed:
    - names:
      - pam-devel
      - libglue-devel

monitoring_packages:
  pkg.installed:
    - version: 'latest'
    - refresh: True
    - names:
        - git
        - go1.10
        - golang-github-prometheus-prometheus
        - golang-github-prometheus-promu
        - golang-github-prometheus-node_exporter
        - phantomjs
        - grafana
    - require:
        - pkgrepo: server:monitoring
        - pkgrepo: devel:tools

/etc/profile.d/go.sh:
  file.managed:
    - source: salt://files/go.sh

'source /etc/profile.d/go.sh':
  cmd.run

gobin:
   environ.setenv:
     - name: GOBIN
     - value: /usr/bin
     - update_minion: True

install_godep:
  cmd.run:
    - name: curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
    - require:
      - environ: gobin