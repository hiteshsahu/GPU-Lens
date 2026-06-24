# Copyright 2026 Hitesh Kumar Sahu — https://hiteshsahu.com
# SPDX-License-Identifier: Apache-2.0

# Rendered from prometheus.yml.tpl by ./go using values in .env.
# Targets are injected so users point gpulens at THEIR existing exporters.
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - /etc/prometheus/alerts.rules.yml

scrape_configs:
  - job_name: dcgm
    static_configs:
      - targets: [${DCGM_TARGETS}]   # e.g. gpu-node-1:9400,gpu-node-2:9400

  - job_name: slurm
    static_configs:
      - targets: [${SLURM_EXPORTER_TARGET}]   # e.g. controller:8080

  - job_name: node
    static_configs:
      - targets: [${NODE_TARGETS}]   # optional node-exporter targets
