## Dashboards

gpulens does **not** ship hand-written Grafana JSON. Untested dashboard JSON is
the most common thing that's subtly broken in monitoring repos, so instead the
installer fetches official, community-proven dashboards by ID at install time.

| Dashboard            | grafana.com ID | Shows                                            |
|----------------------|----------------|--------------------------------------------------|
| NVIDIA DCGM Exporter | **12239**      | GPU utilization, memory, temperature, power, ECC |
| Node Exporter Full   | **1860**       | Host CPU, memory, disk, network                  |

The installer pins the `${DS_PROMETHEUS}` datasource input to `Prometheus`
automatically.

- **Compose:** `./go compose` downloads both into
  `compose/grafana/dashboards/`, which Grafana provisions on startup.
- **Kubernetes:** `./go dashboards-k8s --namespace <ns>` creates a
  ConfigMap labelled `grafana_dashboard=1` so the Grafana sidecar imports it.

## Adding a SLURM dashboard

Scheduler dashboards depend on which SLURM exporter you run, so pick the one
that matches your exporter (e.g. the companion board for
`vpenso/prometheus-slurm-exporter`) and add its ID the same way. Metric names
differ between exporters — verify the panels resolve before relying on them.
