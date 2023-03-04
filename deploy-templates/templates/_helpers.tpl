{{/*
Expand the name of the chart.
*/}}
{{- define "nexus-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "nexus-operator.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "nexus-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nexus-operator.labels" -}}
helm.sh/chart: {{ include "nexus-operator.chart" . }}
{{ include "nexus-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nexus-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nexus-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Define Nexus URL
*/}}
{{- define "nexus-operator.nexusBaseUrl" -}}
{{- if .Values.nexus.basePath }}
{{- .Values.global.dnsWildCard }}
{{- else }}
{{- printf "nexus-%s.%s" .Values.global.edpName .Values.global.dnsWildCard  }}
{{- end }}
{{- end }}

{{/*
Define Nexus BasePath
*/}}
{{- define "nexus-operator.nexusBasePath" -}}
{{- if .Values.nexus.basePath }}
{{- printf "/%s" .Values.nexus.basePath }}
{{- else }}
{{- printf "/"  }}
{{- end }}
{{- end }}
