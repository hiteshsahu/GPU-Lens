{{/*
Copyright 2026 Hitesh Kumar Sahu — https://hiteshsahu.com
SPDX-License-Identifier: Apache-2.0
*/}}
{{- define "gpulens.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "gpulens.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "gpulens.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "gpulens.labels" -}}
app.kubernetes.io/name: {{ include "gpulens.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{- define "gpulens.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gpulens.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
