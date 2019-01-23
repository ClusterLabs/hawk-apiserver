/etc/prometheus/prometheus.yml:
  file.managed:
    - source: salt://files/prometheus.yml
    - template: jinja
    - user: root
    - group: root
    - mode: 644

prometheus:
  service.running:
    - enable: True
    - require:
      - file: /etc/prometheus/prometheus.yml

prometheus-node_exporter:
  service.running:
    - enable: True

salt://utils/pacemaker_exporter.sh:
  cmd.script:
    - runas: root

/etc/systemd/system/pacemaker-exporter.service:
  file.managed:
    - source: salt://files/pacemaker-exporter.service
    - user: root
    - group: root
    - mode: 644

pacemaker-exporter:
  service.running:
    - require:
      - file: /etc/systemd/system/pacemaker-exporter.service
    - watch:
      - /etc/systemd/system/pacemaker-exporter.service
    - enable: True

change_grafana_http_port:
  cmd.run:
    - name: sed -i 's/;http_port = 3000/http_port = 3999/g' /etc/grafana/grafana.ini

change_grafana_root_url:
  cmd.run:
    - name: sed -i 's@;root_url = http://localhost:3000@root_url = http://localhost:3999@g' /etc/grafana/grafana.ini

grafana-server:
  service.running:
    - enable: True
    - require:
      - cmd: change_grafana_http_port
      - cmd: change_grafana_root_url

