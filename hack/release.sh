#!/usr/bin/env bash
# Copyright 2020 IBM Corp.
# SPDX-License-Identifier: Apache-2.0

: ${RELEASE:=latest}
: ${TOOLBIN:=./hack/tools/bin}

${TOOLBIN}/yq eval --inplace ".version = \"$RELEASE\"" ./charts/fybrik/Chart.yaml
${TOOLBIN}/yq eval --inplace ".appVersion = \"$RELEASE\"" ./charts/fybrik/Chart.yaml
${TOOLBIN}/yq eval --inplace ".version = \"$RELEASE\"" ./charts/fybrik-crd/Chart.yaml
${TOOLBIN}/yq eval --inplace ".appVersion = \"$RELEASE\"" ./charts/fybrik-crd/Chart.yaml
${TOOLBIN}/yq eval --inplace ".version = \"$RELEASE\"" ./charts/vault/Chart.yaml
