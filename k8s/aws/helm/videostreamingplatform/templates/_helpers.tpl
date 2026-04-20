{{- define "vsp.fullname" -}}
{{- .Chart.Name }}
{{- end }}

{{- define "vsp.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: videostreamingplatform
environment: {{ .Values.global.environment }}
{{- end }}
