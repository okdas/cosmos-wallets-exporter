{{/*
Expand the name of the chart.
*/}}
{{- define "cosmos-wallets-exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cosmos-wallets-exporter.fullname" -}}
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
{{- define "cosmos-wallets-exporter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cosmos-wallets-exporter.labels" -}}
helm.sh/chart: {{ include "cosmos-wallets-exporter.chart" . }}
{{ include "cosmos-wallets-exporter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "cosmos-wallets-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cosmos-wallets-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "cosmos-wallets-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "cosmos-wallets-exporter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create image name with tag
*/}}
{{- define "cosmos-wallets-exporter.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Process config to ensure proper data types
*/}}
{{- define "cosmos-wallets-exporter.processedConfig" -}}
{{- $config := deepCopy .Values.config -}}
{{- if $config.chains -}}
  {{- range $chainIndex, $chain := $config.chains -}}
    {{- if $chain.denoms -}}
      {{- range $denomIndex, $denom := $chain.denoms -}}
        {{- if hasKey $denom "denom-exponent" -}}
          {{- $_ := set $denom "denom-exponent" (int (index $denom "denom-exponent")) -}}
        {{- end -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- $config | toYaml -}}
{{- end }} 