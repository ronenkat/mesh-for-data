{{/*
Expand the name of the chart.
*/}}
{{- define "fybrik.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "fybrik.fullname" -}}
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
{{- define "fybrik.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "fybrik.labels" -}}
helm.sh/chart: {{ include "fybrik.chart" . }}
{{ include "fybrik.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "fybrik.selectorLabels" -}}
app.kubernetes.io/name: {{ include "fybrik.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the value of an image field from hub, image and tag
*/}}
{{- define "fybrik.image" -}}
{{- $root := first . -}}
{{- $ctx := last . -}}
{{- if contains "/" $ctx.image }}
{{- printf "%s" $ctx.image }}
{{- else }}
{{- printf "%s/%s:%s" ( $ctx.hub | default $root.Values.global.hub ) $ctx.image ( $ctx.tag | default $root.Values.global.tag | default $root.Chart.AppVersion ) }}
{{- end }}
{{- end }}

{{/*
isEnabled evaluates an enabled flag that might be set to "auto".
Returns true if one of the following is true:
The return value when using `include` is always a String.
1. The flag is set to "true"
2. The flag is set to true
3. The flag is set to "auto" and the second parameter to this function is true 
*/}}
{{- define "fybrik.isEnabled" -}}
{{- $flag := toString (first .) -}}
{{- $condition := last . -}}
{{- if or (eq $flag "true") (and (eq $flag "auto") $condition) }}
true
{{- end -}}
{{- end }}

{{/*
isRazeeEnabled checks if razee configuration is enabled
*/}}
{{- define "fybrik.isRazeeEnabled" -}}
{{- if or .Values.coordinator.razee.user .Values.coordinator.razee.apiKey .Values.coordinator.razee.iamKey -}}
true
{{- end -}}
{{- end }}

{{/*
Detect the version of cert manager crd that is installed
Defaults to cert-manager.io/v1alpha2 
*/}}
{{- define "fybrik.certManagerApiVersion" -}}
{{- if (.Capabilities.APIVersions.Has "certmanager.k8s.io/v1alpha1") -}}
certmanager.k8s.io/v1alpha1
{{- else  -}}
cert-manager.io/v1alpha2
{{- end -}}
{{- end -}}
