{{- define "rstream-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "rstream-operator.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "rstream-operator.labels" -}}
app.kubernetes.io/name: {{ include "rstream-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: rstream
{{- end -}}

{{- define "rstream-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "rstream-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: manager
{{- end -}}

{{- define "rstream-operator.image" -}}
{{- printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) -}}
{{- end -}}

{{- define "rstream-operator.agentImage" -}}
{{- if .Values.agent.image.repository -}}
{{- printf "%s:%s" .Values.agent.image.repository (.Values.agent.image.tag | default (.Values.image.tag | default .Chart.AppVersion)) -}}
{{- else -}}
{{- include "rstream-operator.image" . -}}
{{- end -}}
{{- end -}}
