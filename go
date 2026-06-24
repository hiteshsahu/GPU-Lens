#!/usr/bin/env bash
# Copyright 2026 Hitesh Kumar Sahu — https://hiteshsahu.com
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail

# -----------------------------
# Config
# -----------------------------
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DCGM_DASHBOARD_ID=12239            # NVIDIA DCGM Exporter dashboard
NODE_DASHBOARD_ID=1860             # Node Exporter Full
K8S_NAMESPACE="${K8S_NAMESPACE:-monitoring}"
# CONTAINER_RUNTIME=docker|podman  -- override auto-detection

# -----------------------------
# Helpers
# -----------------------------
c_red() { printf '\033[31m%s\033[0m\n' "$*"; }
c_grn() { printf '\033[32m%s\033[0m\n' "$*"; }
die()   { c_red "error: $*" >&2; exit 1; }
need()  { command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not installed."; }

detect_container_runtime() {
  if [ -n "${CONTAINER_RUNTIME:-}" ]; then need "$CONTAINER_RUNTIME"; echo "$CONTAINER_RUNTIME"; return; fi
  command -v docker >/dev/null 2>&1 && { echo docker; return; }
  command -v podman >/dev/null 2>&1 && { echo podman; return; }
  die "docker or podman is required"
}

fetch_dashboard() {
  # $1 = grafana.com dashboard id, $2 = output file
  curl -fsSL "https://grafana.com/api/dashboards/$1/revisions/latest/download" -o "$2" \
    || die "failed to fetch dashboard $1 (need network access to grafana.com)"
  # -i.bak (not bare -i) so this works on both GNU and BSD/macOS sed.
  sed -i.bak 's/${DS_PROMETHEUS}/Prometheus/g' "$2"
  rm -f "$2.bak"
  c_grn "fetched dashboard $1 -> $2"
}

# Extracts --namespace/--namespace=VALUE from "$@" into $NS (falling back to
# $K8S_NAMESPACE), and leaves the rest in the $REMAINING array.
parse_namespace() {
  NS="$K8S_NAMESPACE"
  REMAINING=()
  while [ $# -gt 0 ]; do
    case "$1" in
      --namespace) NS="${2:?--namespace requires a value}"; shift 2 ;;
      --namespace=*) NS="${1#*=}"; shift ;;
      *) REMAINING+=("$1"); shift ;;
    esac
  done
}

# -----------------------------
# 0. PREREQUISITES
# -----------------------------
check() {
  echo "=== 🛠  PREREQUISITES-CHECK ==="
  for t in curl envsubst; do
    command -v "$t" >/dev/null 2>&1 && c_grn "✅ $t" || c_red "❌ $t (required)"
  done
  for t in docker podman helm kubectl promtool shellcheck; do
    command -v "$t" >/dev/null 2>&1 && c_grn "✅ $t" || echo "➖ $t (optional, path-dependent)"
  done
}

# -----------------------------
# 1. KUBERNETES
# -----------------------------
helm_install() {
  need helm; need kubectl; shift || true
  parse_namespace "$@"
  helm upgrade --install gpulens "$REPO_ROOT/helm/gpulens" \
    --namespace "$NS" --create-namespace "${REMAINING[@]}"
  c_grn "Helm release installed in '$NS'."
  echo "Next: ./go dashboards-k8s --namespace $NS"
}

helm_down() {
  need helm; shift || true
  parse_namespace "$@"
  helm uninstall gpulens --namespace "$NS"
  c_grn "Helm release removed from '$NS'."
}

dashboards_k8s() {
  need kubectl; need curl; shift || true
  parse_namespace "$@"
  local tmp; tmp="$(mktemp)"
  fetch_dashboard "$DCGM_DASHBOARD_ID" "$tmp"
  kubectl create configmap gpulens-dcgm-dashboard \
    --namespace "$NS" --from-file=dcgm.json="$tmp" \
    --dry-run=client -o yaml \
    | kubectl label --local -f - grafana_dashboard=1 -o yaml \
    | kubectl apply -f -
  rm -f "$tmp"
  c_grn "DCGM dashboard ConfigMap created in '$NS' (Grafana sidecar will load it)."
}

# -----------------------------
# 2. COMPOSE / BARE-METAL
# -----------------------------
compose() {
  local rt dir; rt="$(detect_container_runtime)"
  "$rt" compose version >/dev/null 2>&1 || die "'$rt compose' is not available"
  need curl; need envsubst
  dir="$REPO_ROOT/compose"
  [ -f "$dir/.env" ] || die "compose/.env not found. Copy compose/.env.example to compose/.env and edit it."
  # shellcheck disable=SC1090
  set -a; source "$dir/.env"; set +a
  [ -n "${DCGM_TARGETS:-}" ] || die "DCGM_TARGETS is empty in .env — point gpulens at your GPU nodes."

  envsubst < "$dir/prometheus/prometheus.yml.tpl" > "$dir/prometheus/prometheus.yml"
  c_grn "rendered prometheus.yml (dcgm: $DCGM_TARGETS)"

  mkdir -p "$dir/grafana/dashboards"
  fetch_dashboard "$DCGM_DASHBOARD_ID" "$dir/grafana/dashboards/dcgm.json"
  fetch_dashboard "$NODE_DASHBOARD_ID" "$dir/grafana/dashboards/node.json"

  ( cd "$dir" && "$rt" compose up -d )
  c_grn "gpulens up"
  echo "Grafana:    http://localhost:${GRAFANA_PORT:-3000}"
  echo "Prometheus: http://localhost:${PROMETHEUS_PORT:-9090}"
}

compose_down() {
  local rt; rt="$(detect_container_runtime)"
  ( cd "$REPO_ROOT/compose" && "$rt" compose down )
  c_grn "compose stack stopped."
}

# -----------------------------
# 3. DASHBOARDS
# -----------------------------
dashboards() {
  need curl
  local dir="$REPO_ROOT/compose/grafana/dashboards"
  mkdir -p "$dir"
  fetch_dashboard "$DCGM_DASHBOARD_ID" "$dir/dcgm.json"
  fetch_dashboard "$NODE_DASHBOARD_ID" "$dir/node.json"
}

# -----------------------------
# 4. VALIDATE
# -----------------------------
lint() {
  need helm
  helm lint "$REPO_ROOT/helm/gpulens"
  helm template "$REPO_ROOT/helm/gpulens" >/dev/null && c_grn "helm template OK"
  if command -v promtool >/dev/null 2>&1; then
    promtool check rules "$REPO_ROOT/compose/prometheus/alerts.rules.yml"
  else echo "➖ promtool not found — skipping rule check (CI runs it)"; fi
  if command -v shellcheck >/dev/null 2>&1; then shellcheck "$REPO_ROOT/go"; fi
}

# -----------------------------
# HELP / HINT (Interactive)
# -----------------------------
help() {
cat <<HEREDOC
Usage: ./go <command> [options]

Commands:
=== 0. 🛠  PREREQUISITES         ===
=== 1. ☸️  KUBERNETES            ===
=== 2. 🐳 COMPOSE / BARE-METAL   ===
=== 3. 📊 DASHBOARDS             ===
=== 4. 🧪 VALIDATE               ===

Enter a number to see details:
HEREDOC

read -rn 1 option
echo ""; echo ""

case ${option} in
  0)
    echo "=== 🛠  PREREQUISITES ==="
    echo "🛠   check               -- Verify required + optional tools are installed"
    ;;
  1)
    echo "=== ☸️  KUBERNETES ==="
    echo "🚀  helm                -- Install/upgrade the Helm chart (DCGM exporter + alerts)"
    echo "🧹  helm-down           -- Uninstall the Helm release"
    echo "📊  dashboards-k8s      -- Create the Grafana dashboard ConfigMap"
    echo "    (override target ns with --namespace <ns> or K8S_NAMESPACE=...)"
    ;;
  2)
    echo "=== 🐳 COMPOSE / BARE-METAL ==="
    echo "▶️   compose             -- Render config, fetch dashboards, bring up Prometheus+Grafana"
    echo "🧹   compose-down        -- Tear down the compose stack"
    echo "    (override runtime with CONTAINER_RUNTIME=docker|podman)"
    ;;
  3)
    echo "=== 📊 DASHBOARDS ==="
    echo "📥  dashboards          -- Fetch official dashboards by ID (DCGM 12239, Node 1860)"
    ;;
  4)
    echo "=== 🧪 VALIDATE ==="
    echo "🧪  lint                -- helm lint + template + promtool rules + shellcheck"
    ;;
  *)
    echo "Section $option does not exist"
    ;;
esac
}

# -----------------------------
# Dispatch
# -----------------------------
case "${1:-help}" in
  check)            check ;;
  helm)             helm_install "$@" ;;
  helm-down)        helm_down "$@" ;;
  dashboards-k8s)   dashboards_k8s "$@" ;;
  compose)          compose ;;
  compose-down)     compose_down ;;
  dashboards)       dashboards ;;
  lint)             lint ;;
  help|-h|--help|"") help ;;
  *)                die "unknown command '$1' (run ./go for the menu)" ;;
esac
