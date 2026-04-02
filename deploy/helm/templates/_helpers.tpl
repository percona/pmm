{{/*
Chart name, truncated to 63 chars.
*/}}
{{- define "pmm-headless.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified app name. If release name contains the chart name it won't be
duplicated. Truncated to 63 chars for Kubernetes label compliance.
*/}}
{{- define "pmm-headless.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart label value.
*/}}
{{- define "pmm-headless.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to every resource.
*/}}
{{- define "pmm-headless.labels" -}}
helm.sh/chart: {{ include "pmm-headless.chart" . }}
{{ include "pmm-headless.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels for the PMM server.
*/}}
{{- define "pmm-headless.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pmm-headless.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Resolve the effective storageClassName for a given component.
Priority: component-level > global > "" (cluster default).
*/}}
{{- define "pmm-headless.storageClass" -}}
{{- $sc := "" -}}
{{- if .global.storageClass }}
{{- $sc = .global.storageClass }}
{{- end }}
{{- if .component.storageClass }}
{{- $sc = .component.storageClass }}
{{- end }}
{{- if $sc }}
storageClassName: {{ $sc | quote }}
{{- end }}
{{- end }}
