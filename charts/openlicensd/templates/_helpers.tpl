{{- define "openlicensd.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "openlicensd.fullname" -}}
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

{{- define "openlicensd.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "openlicensd.labels" -}}
helm.sh/chart: {{ include "openlicensd.chart" . }}
{{ include "openlicensd.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "openlicensd.selectorLabels" -}}
app.kubernetes.io/name: {{ include "openlicensd.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "openlicensd.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "openlicensd.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "openlicensd.secretName" -}}
{{- if eq .Values.secret.mode "existing" }}
{{- required "secret.existingSecret is required when secret.mode is existing" .Values.secret.existingSecret }}
{{- else }}
{{- default (printf "%s-secret" (include "openlicensd.fullname" .)) .Values.secret.name }}
{{- end }}
{{- end }}
