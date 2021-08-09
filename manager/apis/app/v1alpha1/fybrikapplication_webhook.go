// Copyright 2020 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"encoding/json"
	log "log"

	validate "fybrik.io/fybrik/pkg/taxonomy/validate"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func (r *FybrikApplication) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,admissionReviewVersions=v1;v1beta1,sideEffects=None,path=/validate-app-fybrik-io-v1alpha1-fybrikapplication,mutating=false,failurePolicy=fail,groups=app.fybrik.io,resources=fybrikapplications,versions=v1alpha1,name=vfybrikapplication.kb.io

var _ webhook.Validator = &FybrikApplication{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *FybrikApplication) ValidateCreate() error {
	log.Printf("Validating fybrikapplication %s for creation", r.Name)
	taxonomyFile := "/tmp/taxonomy/application.values.schema.json"
	return r.ValidateFybrikApplication(taxonomyFile)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *FybrikApplication) ValidateUpdate(old runtime.Object) error {
	log.Printf("Validating fybrikapplication %s for update", r.Name)
	taxonomyFile := "/tmp/taxonomy/application.values.schema.json"
	return r.ValidateFybrikApplication(taxonomyFile)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *FybrikApplication) ValidateDelete() error {
	return nil
}

func (r *FybrikApplication) ValidateFybrikApplication(taxonomyFile string) error {
	var allErrs []*field.Error

	// Convert Fybrik application Go struct to JSON
	applicationJSON, err := json.Marshal(r)
	if err != nil {
		return err
	}

	// Validate Fybrik application against taxonomy
	allErrs = validate.TaxonomyCheck(applicationJSON, taxonomyFile, "Fybrik application")

	// Return any error
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "app.fybrik.io", Kind: "FybrikApplication"},
		r.Name, allErrs)
}
