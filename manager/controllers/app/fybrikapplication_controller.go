// Copyright 2020 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	connectors "fybrik.io/fybrik/pkg/connectors/clients"
	pb "fybrik.io/fybrik/pkg/connectors/protobuf"

	"emperror.dev/errors"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	app "fybrik.io/fybrik/manager/apis/app/v1alpha1"
	"fybrik.io/fybrik/manager/controllers/app/modules"
	"fybrik.io/fybrik/manager/controllers/utils"
	"fybrik.io/fybrik/pkg/multicluster"
	"fybrik.io/fybrik/pkg/serde"
	"fybrik.io/fybrik/pkg/storage"
	"fybrik.io/fybrik/pkg/vault"
)

// FybrikApplicationReconciler reconciles a FybrikApplication object
type FybrikApplicationReconciler struct {
	client.Client
	Name              string
	Log               logr.Logger
	Scheme            *runtime.Scheme
	PolicyManager     connectors.PolicyManager
	DataCatalog       connectors.DataCatalog
	ResourceInterface ContextInterface
	ClusterManager    multicluster.ClusterLister
	Provision         storage.ProvisionInterface
}

// Reconcile reconciles FybrikApplication CRD
// It receives FybrikApplication CRD and selects the appropriate modules that will run
// The outcome is either a single Blueprint running on the same cluster or a Plotter containing multiple Blueprints that may run on different clusters
func (r *FybrikApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("fybrikapplication", req.NamespacedName)
	// obtain FybrikApplication resource
	applicationContext := &app.FybrikApplication{}
	if err := r.Get(ctx, req.NamespacedName, applicationContext); err != nil {
		log.V(0).Info("The reconciled object was not found")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err := r.reconcileFinalizers(applicationContext); err != nil {
		log.V(0).Info("Could not reconcile finalizers " + err.Error())
		return ctrl.Result{}, err
	}

	// If the object has a scheduled deletion time, update status and return
	if !applicationContext.DeletionTimestamp.IsZero() {
		// The object is being deleted
		return ctrl.Result{}, nil
	}

	observedStatus := applicationContext.Status.DeepCopy()
	appVersion := applicationContext.GetGeneration()

	// check if reconcile is required
	// reconcile is required if the spec has been changed, or the previous reconcile has failed to allocate a Plotter resource
	generationComplete := r.ResourceInterface.ResourceExists(observedStatus.Generated) && (observedStatus.Generated.AppVersion == appVersion)
	if (!generationComplete) || (observedStatus.ObservedGeneration != appVersion) {
		if result, err := r.reconcile(applicationContext); err != nil {
			// another attempt will be done
			// users should be informed in case of errors
			if !equality.Semantic.DeepEqual(&applicationContext.Status, observedStatus) {
				// ignore an update error, a new reconcile will be made in any case
				_ = r.Client.Status().Update(ctx, applicationContext)
			}
			return result, err
		}
		applicationContext.Status.ObservedGeneration = appVersion
	} else {
		resourceStatus, err := r.ResourceInterface.GetResourceStatus(applicationContext.Status.Generated)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err = r.checkReadiness(applicationContext, resourceStatus); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update CRD status in case of change (other than deletion, which was handled separately)
	if !equality.Semantic.DeepEqual(&applicationContext.Status, observedStatus) && applicationContext.DeletionTimestamp.IsZero() {
		log.V(0).Info("Reconcile: Updating status for desired generation " + fmt.Sprint(applicationContext.GetGeneration()))
		if err := r.Client.Status().Update(ctx, applicationContext); err != nil {
			return ctrl.Result{}, err
		}
	}
	if errorOrDeny(applicationContext) {
		log.Info("Reconciled with errors: " + getErrorMessages(applicationContext))
	}

	// trigger a new reconcile if required (the fybrikapplication is not ready)
	if !inFinalState(applicationContext) {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func getBucketResourceRef(name string) *types.NamespacedName {
	return &types.NamespacedName{Name: name, Namespace: utils.GetSystemNamespace()}
}

func (r *FybrikApplicationReconciler) checkReadiness(applicationContext *app.FybrikApplication, status app.ObservedState) error {
	applicationContext.Status.DataAccessInstructions = ""
	resetConditions(applicationContext)
	if applicationContext.Status.CatalogedAssets == nil {
		applicationContext.Status.CatalogedAssets = make(map[string]string)
	}

	if status.Error != "" {
		setErrorCondition(applicationContext, "", status.Error)
		return nil
	}
	if !status.Ready {
		return nil
	}
	// Plotter is ready - update the FybrikApplication status

	// register assets if necessary if the ready state has been received
	for _, dataCtx := range applicationContext.Spec.Data {
		if dataCtx.Requirements.Copy.Catalog.CatalogID != "" {
			if _, cataloged := applicationContext.Status.CatalogedAssets[dataCtx.DataSetID]; cataloged {
				// the asset has been already cataloged
				continue
			}
			// mark the bucket as persistent and register the asset
			provisionedBucketRef, found := applicationContext.Status.ProvisionedStorage[dataCtx.DataSetID]
			if !found {
				message := "No copy has been created for the asset " + dataCtx.DataSetID + " required to be registered"
				r.Log.V(0).Info(message)
				return errors.New(message)
			}
			if err := r.Provision.SetPersistent(getBucketResourceRef(provisionedBucketRef.DatasetRef), true); err != nil {
				return err
			}
			// register the asset: experimental feature
			if newAssetID, err := r.RegisterAsset(dataCtx.Requirements.Copy.Catalog.CatalogID, &provisionedBucketRef, applicationContext); err == nil {
				applicationContext.Status.CatalogedAssets[dataCtx.DataSetID] = newAssetID
			} else {
				// log an error and make a new attempt to register the asset
				r.Log.V(0).Info("Error while registering an asset: " + err.Error())
				return nil
			}
		}
	}
	setReadyCondition(applicationContext, "")
	applicationContext.Status.DataAccessInstructions = status.DataAccessInstructions
	return nil
}

// reconcileFinalizers reconciles finalizers for FybrikApplication
func (r *FybrikApplicationReconciler) reconcileFinalizers(applicationContext *app.FybrikApplication) error {
	// finalizer
	finalizerName := r.Name + ".finalizer"
	hasFinalizer := ctrlutil.ContainsFinalizer(applicationContext, finalizerName)

	// If the object has a scheduled deletion time, delete it and all resources it has created
	if !applicationContext.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if hasFinalizer { // Finalizer was created when the object was created
			// the finalizer is present - delete the allocated resources
			if err := r.deleteExternalResources(applicationContext); err != nil {
				return err
			}

			// remove the finalizer from the list and update it, because it needs to be deleted together with the object
			ctrlutil.RemoveFinalizer(applicationContext, finalizerName)

			if err := r.Update(context.Background(), applicationContext); err != nil {
				return err
			}
		}
		return nil
	}
	// Make sure this CRD instance has a finalizer
	if !hasFinalizer {
		ctrlutil.AddFinalizer(applicationContext, finalizerName)
		if err := r.Update(context.Background(), applicationContext); err != nil {
			return err
		}
	}
	return nil
}

func (r *FybrikApplicationReconciler) deleteExternalResources(applicationContext *app.FybrikApplication) error {
	// clear provisioned storage
	// References to buckets (Dataset resources) are deleted. Buckets that are persistent will not be removed upon Dataset deletion.
	var deletedKeys []string
	var errMsgs []string
	for datasetID, datasetDetails := range applicationContext.Status.ProvisionedStorage {
		if err := r.Provision.DeleteDataset(getBucketResourceRef(datasetDetails.DatasetRef)); err != nil {
			errMsgs = append(errMsgs, err.Error())
		} else {
			deletedKeys = append(deletedKeys, datasetID)
		}
	}
	for _, datasetID := range deletedKeys {
		delete(applicationContext.Status.ProvisionedStorage, datasetID)
	}
	if len(errMsgs) != 0 {
		return errors.New(strings.Join(errMsgs, ";"))
	}
	// delete the generated resource
	if applicationContext.Status.Generated == nil {
		return nil
	}

	r.Log.V(0).Info("Reconcile: FybrikApplication is deleting the generated " + applicationContext.Status.Generated.Kind)
	if err := r.ResourceInterface.DeleteResource(applicationContext.Status.Generated); err != nil {
		return err
	}
	applicationContext.Status.Generated = nil
	return nil
}

// setReadModulesEndpoints populates the ReadEndpointsMap map in the status of the fybrikapplication
// Current implementation assumes there is only one cluster with read modules (which is the same cluster the user's workload)
func setReadModulesEndpoints(applicationContext *app.FybrikApplication, blueprintsMap map[string]app.BlueprintSpec, moduleMap map[string]*app.FybrikModule) {
	var foundReadEndpoints = false
	for _, blueprintSpec := range blueprintsMap {
		for _, step := range blueprintSpec.Flow.Steps {
			if step.Arguments.Read != nil {
				// We found a read module
				foundReadEndpoints = true
				releaseName := utils.GetReleaseName(applicationContext.ObjectMeta.Name, applicationContext.ObjectMeta.Namespace, step)
				moduleName := step.Template

				// Find the read capability section in the module
				// TODO: What if there are more than one read capability sections?  How do we know which endpoint
				// to choose?  They could in theory be different, although that's not likely
				// Currently the last one on the list is used.
				if hasRead, caps := utils.GetModuleCapabilities(moduleMap[moduleName], app.Read); hasRead {
					for _, cap := range caps {
						originalEndpointSpec := cap.API.Endpoint
						fqdn := utils.GenerateModuleEndpointFQDN(releaseName, BlueprintNamespace)
						for _, arg := range step.Arguments.Read {
							applicationContext.Status.ReadEndpointsMap[arg.AssetID] = app.EndpointSpec{
								Hostname: fqdn,
								Port:     originalEndpointSpec.Port,
								Scheme:   originalEndpointSpec.Scheme,
							}
						}
					}
				}
			}
		}
		// We found a blueprint with read modules
		if foundReadEndpoints {
			return
		}
	}
}

// reconcile receives either FybrikApplication CRD
// or a status update from the generated resource
func (r *FybrikApplicationReconciler) reconcile(applicationContext *app.FybrikApplication) (ctrl.Result, error) {
	utils.PrintStructure(applicationContext.Spec, r.Log, "FybrikApplication")
	// Data User created or updated the FybrikApplication

	// clear status
	resetConditions(applicationContext)
	applicationContext.Status.DataAccessInstructions = ""
	if applicationContext.Status.ProvisionedStorage == nil {
		applicationContext.Status.ProvisionedStorage = make(map[string]app.DatasetDetails)
	}
	applicationContext.Status.ReadEndpointsMap = make(map[string]app.EndpointSpec)

	if len(applicationContext.Spec.Data) == 0 {
		if err := r.deleteExternalResources(applicationContext); err != nil {
			return ctrl.Result{}, err
		}
		r.Log.V(0).Info("no blueprint will be generated since no datasets are specified")
		setReadyCondition(applicationContext, "")
		return ctrl.Result{}, nil
	}

	clusters, err := r.ClusterManager.GetClusters()
	if err != nil {
		return ctrl.Result{}, err
	}
	// create a list of requirements for creating a data flow (actions, interface to app, data format) per a single data set
	var requirements []modules.DataInfo
	for _, dataset := range applicationContext.Spec.Data {
		req := modules.DataInfo{
			Context: dataset.DeepCopy(),
		}
		if err := r.constructDataInfo(&req, applicationContext, clusters); err != nil {
			AnalyzeError(applicationContext, req.Context.DataSetID, err)
			continue
		}
		requirements = append(requirements, req)
	}
	// check if can proceed
	if errorOrDeny(applicationContext) {
		return ctrl.Result{}, nil
	}

	// create a module manager that will select modules to be orchestrated based on user requirements and module capabilities
	moduleMap, err := r.GetAllModules()
	if err != nil {
		return ctrl.Result{}, err
	}
	objectKey := client.ObjectKeyFromObject(applicationContext)
	moduleManager := &ModuleManager{
		Client:             r.Client,
		Log:                r.Log,
		Modules:            moduleMap,
		Clusters:           clusters,
		Owner:              objectKey,
		PolicyManager:      r.PolicyManager,
		Provision:          r.Provision,
		ProvisionedStorage: make(map[string]NewAssetInfo),
	}
	instances := make([]modules.ModuleInstanceSpec, 0)
	for _, item := range requirements {
		instancesPerDataset, err := moduleManager.SelectModuleInstances(item, applicationContext)
		if err != nil {
			AnalyzeError(applicationContext, item.Context.DataSetID, err)
			continue
		}
		instances = append(instances, instancesPerDataset...)
	}
	// check if can proceed
	if errorOrDeny(applicationContext) {
		return ctrl.Result{}, nil
	}
	// update allocated storage in the status
	// clean irrelevant buckets
	for datasetID, details := range applicationContext.Status.ProvisionedStorage {
		if _, found := moduleManager.ProvisionedStorage[datasetID]; !found {
			_ = r.Provision.DeleteDataset(getBucketResourceRef(details.DatasetRef))
			delete(applicationContext.Status.ProvisionedStorage, datasetID)
		}
	}
	// add or update new buckets
	for datasetID, info := range moduleManager.ProvisionedStorage {
		raw := serde.NewArbitrary(info.Details)
		applicationContext.Status.ProvisionedStorage[datasetID] = app.DatasetDetails{
			DatasetRef: info.Storage.Name,
			SecretRef:  info.Storage.SecretRef.Name,
			Details:    *raw,
		}
	}
	ready := true
	var allocErr error
	// check that the buckets have been created successfully using Dataset status
	for id, details := range applicationContext.Status.ProvisionedStorage {
		res, err := r.Provision.GetDatasetStatus(getBucketResourceRef(details.DatasetRef))
		if err != nil {
			ready = false
			break
		}
		if !res.Provisioned {
			ready = false
			r.Log.V(0).Info("No bucket has been provisioned for " + id)
			// TODO(shlomitk1): analyze the error
			if res.ErrorMsg != "" {
				allocErr = errors.New(res.ErrorMsg)
			}
			break
		}
	}
	if !ready {
		return ctrl.Result{RequeueAfter: 2 * time.Second}, allocErr
	}
	// generate blueprint specifications (per cluster)
	blueprintPerClusterMap := r.GenerateBlueprints(instances, applicationContext)
	setReadModulesEndpoints(applicationContext, blueprintPerClusterMap, moduleMap)
	ownerRef := &app.ResourceReference{Name: applicationContext.Name, Namespace: applicationContext.Namespace, AppVersion: applicationContext.GetGeneration()}
	resourceRef := r.ResourceInterface.CreateResourceReference(ownerRef)
	if err := r.ResourceInterface.CreateOrUpdateResource(ownerRef, resourceRef, blueprintPerClusterMap); err != nil {
		r.Log.V(0).Info("Error creating " + resourceRef.Kind + " : " + err.Error())
		if err.Error() == app.InvalidClusterConfiguration {
			setErrorCondition(applicationContext, "", app.InvalidClusterConfiguration)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	applicationContext.Status.Generated = resourceRef
	r.Log.V(0).Info("Created " + resourceRef.Kind + " successfully!")
	return ctrl.Result{}, nil
}

func (r *FybrikApplicationReconciler) constructDataInfo(req *modules.DataInfo, input *app.FybrikApplication, clusters []multicluster.Cluster) error {
	var err error

	// Call the DataCatalog service to get info about the dataset
	var response *pb.CatalogDatasetInfo
	var credentialPath string
	if input.Spec.SecretRef != "" {
		credentialPath = utils.GetVaultAddress() + vault.PathForReadingKubeSecret(input.Namespace, input.Spec.SecretRef)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if response, err = r.DataCatalog.GetDatasetInfo(ctx, &pb.CatalogDatasetRequest{
		CredentialPath: credentialPath,
		DatasetId:      req.Context.DataSetID,
	}); err != nil {
		return err
	}

	details := response.GetDetails()
	dataDetails, err := modules.CatalogDatasetToDataDetails(response)
	if err != nil {
		return err
	}
	req.DataDetails = dataDetails
	req.VaultSecretPath = ""
	if details.CredentialsInfo != nil {
		req.VaultSecretPath = details.CredentialsInfo.VaultSecretPath
	}

	return nil
}

// NewFybrikApplicationReconciler creates a new reconciler for FybrikApplications
func NewFybrikApplicationReconciler(mgr ctrl.Manager, name string,
	policyManager connectors.PolicyManager, catalog connectors.DataCatalog, cm multicluster.ClusterLister, provision storage.ProvisionInterface) *FybrikApplicationReconciler {
	return &FybrikApplicationReconciler{
		Client:            mgr.GetClient(),
		Name:              name,
		Log:               ctrl.Log.WithName("controllers").WithName(name),
		Scheme:            mgr.GetScheme(),
		PolicyManager:     policyManager,
		ResourceInterface: NewPlotterInterface(mgr.GetClient()),
		ClusterManager:    cm,
		Provision:         provision,
		DataCatalog:       catalog,
	}
}

// SetupWithManager registers FybrikApplication controller
func (r *FybrikApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mapFn := func(a client.Object) []reconcile.Request {
		labels := a.GetLabels()
		if labels == nil {
			return []reconcile.Request{}
		}
		namespace, foundNamespace := labels[app.ApplicationNamespaceLabel]
		name, foundName := labels[app.ApplicationNameLabel]
		if !foundNamespace || !foundName {
			return []reconcile.Request{}
		}
		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}},
		}
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&app.FybrikApplication{}).
		Watches(&source.Kind{
			Type: &app.Plotter{},
		}, handler.EnqueueRequestsFromMapFunc(mapFn)).Complete(r)
}

// AnalyzeError analyzes whether the given error is fatal, or a retrial attempt can be made.
// Reasons for retrial can be either communication problems with external services, or kubernetes problems to perform some action on a resource.
// A retrial is achieved by returning an error to the reconcile method
func AnalyzeError(application *app.FybrikApplication, assetID string, err error) {
	if err == nil {
		return
	}
	switch err.Error() {
	case app.InvalidAssetID, app.ReadAccessDenied, app.CopyNotAllowed, app.WriteNotAllowed:
		setDenyCondition(application, assetID, err.Error())
	default:
		setErrorCondition(application, assetID, err.Error())
	}
}

func ownerLabels(id types.NamespacedName) map[string]string {
	return map[string]string{
		app.ApplicationNamespaceLabel: id.Namespace,
		app.ApplicationNameLabel:      id.Name,
	}
}

// GetAllModules returns all CRDs of the kind FybrikModule mapped by their name
func (r *FybrikApplicationReconciler) GetAllModules() (map[string]*app.FybrikModule, error) {
	ctx := context.Background()

	moduleMap := make(map[string]*app.FybrikModule)
	var moduleList app.FybrikModuleList
	if err := r.List(ctx, &moduleList, client.InNamespace(utils.GetSystemNamespace())); err != nil {
		r.Log.V(0).Info("Error while listing modules: " + err.Error())
		return moduleMap, err
	}
	r.Log.Info("Listing all modules")
	for _, module := range moduleList.Items {
		r.Log.Info(module.GetName())
		moduleMap[module.Name] = module.DeepCopy()
	}
	return moduleMap, nil
}
