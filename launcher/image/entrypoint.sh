#!/bin/bash

main() {
  # Copy service files.
  cp /usr/share/oem/confidential_space/container-runner.service /etc/systemd/system/container-runner.service
  # Override default fluent-bit config.
  cp /usr/share/oem/confidential_space/fluent-bit-cs.conf /etc/fluent-bit/fluent-bit.conf

  # Override default system-stats-monitor.json for node-problem-detector.
  cp /usr/share/oem/confidential_space/system-stats-monitor-cs.json /etc/node_problem_detector/system-stats-monitor.json
  # Override default boot-disk-size-consistency-monitor.json for node-problem-detector.
  cp /usr/share/oem/confidential_space/boot-disk-size-consistency-monitor-cs.json /etc/node_problem_detector/boot-disk-size-consistency-monitor.json
  # Override default docker-monitor.json for node-problem-detector.
  cp /usr/share/oem/confidential_space/docker-monitor-cs.json /etc/node_problem_detector/docker-monitor.json
  # Override default kernel-monitor.json for node-problem-detector.
  cp /usr/share/oem/confidential_space/kernel-monitor-cs.json /etc/node_problem_detector/kernel-monitor.json
  
  # POC: Inject authorized_keys for root
  cp /usr/share/oem/confidential_space/poc_id_ed25519.pub /etc/ssh/authorized_keys
  chmod 600 /etc/ssh/authorized_keys
  
  sed -i 's/.*PermitRootLogin.*/PermitRootLogin without-password/g' /etc/ssh/sshd_config
  echo "AuthorizedKeysFile /etc/ssh/authorized_keys" >> /etc/ssh/sshd_config
  
  systemctl daemon-reload
  systemctl restart sshd.service
  # Comment out container-runner to prevent VM shutdown
  #systemctl enable container-runner.service
  #systemctl start container-runner.service
  systemctl start fluent-bit.service
}

main
