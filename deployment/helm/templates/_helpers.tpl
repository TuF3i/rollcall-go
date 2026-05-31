{{- define "rollcall-go.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "rollcall-go.fullname" -}}
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

{{- define "rollcall-go.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "rollcall-go.labels" -}}
helm.sh/chart: {{ include "rollcall-go.chart" . }}
{{ include "rollcall-go.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "rollcall-go.selectorLabels" -}}
app.kubernetes.io/name: {{ include "rollcall-go.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "rollcall-go.image" -}}
{{- $registry := .registry | default "" }}
{{- $repository := .repository | default "rollcall-go" }}
{{- $tag := .tag | default "latest" }}
{{- if $registry }}{{ $registry }}/{{ end }}{{ $repository }}:{{ $tag }}
{{- end }}

{{- define "rollcall-go.instance.name" -}}
{{- $root := .root -}}
{{- $instance := .instance -}}
{{- $fullname := include "rollcall-go.fullname" $root -}}
{{- printf "%s-%s" $fullname $instance.name | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{- define "rollcall-go.instance.labels" -}}
{{ include "rollcall-go.labels" .root }}
rollcall.auto/edge-instance: {{ .instance.name }}
{{- end }}

{{- define "rollcall-go.instance.selectorLabels" -}}
{{ include "rollcall-go.selectorLabels" .root }}
rollcall.auto/edge-instance: {{ .instance.name }}
{{- end }}
