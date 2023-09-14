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

Selector labels for nexus-proxy
*/}}
{{- define "nexus-proxy.selectorLabels" }}
app.kubernetes.io/name: nexus-proxy
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Define Nexus URL
*/}}
{{- define "nexus-operator.nexusBaseUrl" -}}
{{- if .Values.nexus.basePath }}
{{- .Values.global.dnsWildCard }}
{{- else }}
{{- printf "nexus-%s.%s" .Release.Namespace .Values.global.dnsWildCard  }}
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

Return the appropriate apiVersion for ingress.
*/}}
{{- define "nexus.ingress.apiVersion" -}}
  {{- if and (.Capabilities.APIVersions.Has "networking.k8s.io/v1") (semverCompare ">= 1.19-0" .Capabilities.KubeVersion.Version) -}}
      {{- print "networking.k8s.io/v1" -}}
  {{- else if .Capabilities.APIVersions.Has "networking.k8s.io/v1beta1" -}}
    {{- print "networking.k8s.io/v1beta1" -}}
  {{- else -}}
    {{- print "extensions/v1beta1" -}}
  {{- end -}}
{{- end -}}

{{/*
Return if ingress is stable.
*/}}
{{- define "nexus.ingress.isStable" -}}
  {{- eq (include "nexus.ingress.apiVersion" .) "networking.k8s.io/v1" -}}
{{- end -}}

{{/*
Return if ingress supports ingressClassName.
*/}}
{{- define "nexus.ingress.supportsIngressClassName" -}}
  {{- or (eq (include "nexus.ingress.isStable" .) "true") (and (eq (include "nexus.ingress.apiVersion" .) "networking.k8s.io/v1beta1") (semverCompare ">= 1.18-0" .Capabilities.KubeVersion.Version)) -}}
{{- end -}}

{{/*
Return if ingress supports pathType.
*/}}
{{- define "nexus.ingress.supportsPathType" -}}
  {{- or (eq (include "nexus.ingress.isStable" .) "true") (and (eq (include "nexus.ingress.apiVersion" .) "networking.k8s.io/v1beta1") (semverCompare ">= 1.18-0" .Capabilities.KubeVersion.Version)) -}}
{{- end -}}
