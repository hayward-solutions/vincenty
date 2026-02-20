{{/*
Common labels applied to all resources.
*/}}
{{- define "sitaware.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end }}

{{/*
Selector labels for a specific component.
*/}}
{{- define "sitaware.selectorLabels" -}}
app: {{ .component }}
app.kubernetes.io/name: {{ .chart.Name }}
app.kubernetes.io/instance: {{ .release.Name }}
{{- end }}

{{/*
Database host — internal StatefulSet or external.
*/}}
{{- define "sitaware.dbHost" -}}
{{- if .Values.postgresql.internal -}}
postgres
{{- else -}}
{{ .Values.postgresql.external.host }}
{{- end -}}
{{- end }}

{{/*
Redis host — internal Deployment or external.
*/}}
{{- define "sitaware.redisHost" -}}
{{- if .Values.redis.internal -}}
redis
{{- else -}}
{{ .Values.redis.external.host }}
{{- end -}}
{{- end }}
