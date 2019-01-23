webui_packages:
  pkg.installed:
    - names:
      - pam-devel
      - libglue-devel

salt://utils/init_cluster.sh:
  cmd.script:
    - env:
{% if 'vdb' in grains['disks'] %}
      - PDEV: /dev/vdb
      - PDEV2: /dev/vdb2
{% else %}
      - PDEV: /dev/sdb
      - PDEV2: /dev/sdb2
{% endif %}

salt://utils/configure_drbd.sh:
  cmd.script:
    - require:
      - file: /etc/drbd.d/global_common.conf
      - file: /etc/drbd.d/r0.res
      - cmd: "salt://utils/init_cluster.sh"

/root/initial.crm:
  file.managed:
    - source: salt://files/crm-initial.conf
    - template: jinja

apply_initial_configuration:
  cmd.run:
    - name: crm configure load update /root/initial.crm
    - require:
      - file: /root/initial.crm
      - cmd: "salt://utils/configure_drbd.sh"
