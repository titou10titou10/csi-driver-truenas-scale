{{/* Expand the name of the chart.*/}}
{{- define "tnsplugin.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* labels for helm resources */}}
{{- define "tnsplugin.labels" -}}
labels:
  app.kubernetes.io/instance: "{{ .Release.Name }}"
  app.kubernetes.io/managed-by: "{{ .Release.Service }}"
  app.kubernetes.io/name: "{{ template "tnsplugin.name" . }}"
  app.kubernetes.io/version: "{{ .Chart.AppVersion }}"
  helm.sh/chart: "{{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}"
  {{- if .Values.customLabels }}
{{ toYaml .Values.customLabels | indent 2 -}}
  {{- end }}
{{- end -}}

{{- define "csi.sock.path" -}}
{{- printf "/csi/%s" .Values.driver.name -}}
{{- end -}}
{{- define "csi.sock.name" -}}
{{- printf "/csi/%s/csi.sock" .Values.driver.name -}}
{{- end -}}
