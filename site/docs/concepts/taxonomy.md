# Taxonomy

Fybrik control plane orchestrate together multiple components, including 3rd party ones, that needs to communicate and share information. 
For example, when a policy decision calls for an action on the data, Fybrik needs to match that action with a module that is able to apply it. Another example is a workload that needs to use a specific API requires the control plane to deploy an appropriate module that accepts requests in the given API format.

For this purpose Fybrik has a common taxonomy, a glossary of common terms, that is shared between the control plane components.
The taxonomy describes the information that Fybrik receives from `FybrikApplication`, `FybrikModule`, policy decisions, catalog metadata and the configuration information that Fybrik specify when orchestrating data access modules in the data path.
The verbs and terms are used to select the set of data access modules that are needed to build the data plane from the application to the data, matching APIs, protocols, and applying the required policies (e.g., transformation).

Fybrik taxonomy is coded as [OpenAPI v3 schemas](https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.0.md#schemaObject) and is designed with [base taxonomy definitions](https://github.com/fybrik/fybrik/blob/master/config/taxonomy/objects/taxonomy.json) that describes the information the control plane needs in order to deploy data access modules, and is extensible with [layers](https://github.com/fybrik/fybrik/tree/master/config/taxonomy/example) which can be added to enable validation and cross check of the values across the component that Fybrik is employing.

### Working with 3rd-party metatdata catalog and policy manager
Fybrik open approach to 3rd party data catalog and policy managers requires that the verbs and terms that are used in a 3rd-party catalog or policy manager be either recognized by the control plane or translated into equivalent verbs and terms that are used by the control plane. 
When translation of terms are needed to conform with defined taxonomy in Fybrik, this functions is performed by the connectors.

![Taxonomy](../static/m4d-taxonomy.svg)

## Using taxonomy

The control plane uses the verbs in the taxonomy to build and orchestrate the data plane, and to communicate with the 3rd-party tools.

The definitions in `taxonomy.json` describe the customizable verbs and values that can be included in the taxonomy. The default taxonomy in Fybrik is specifying the required verbs for the control plane, without additional additional requirements.
> One can extend the default taxonomy by adding verbs to `taxonomy.json`. This is done by adding taxonomy layers and compiling the base taxonomy and layers together to generate a custom `taxonomy.json` that will be deployed with Fybrik. 

### Workload (`FybrikApplication`)

The workload definitions for Fybrik are described in the `FybrikApplication` that is deployed.
The `FybrikApplication` should specify `appInfo` and `data`.

### Data plane modules (`FybrikModule')

Modules are the active components that interacts with the workload and apply the actions derived from the policies.

### Policy manager

Fybrik communicates with a policy manager, sending requests for policy decisions for accessing data assets, and received requests that contains the policy decisions (actions) that should be enforced.

The information required to be in the requests and responses is details as follows:

Policy manager request | [policy_manager_request.json](https://github.com/fybrik/fybrik/blob/master/config/taxonomy/objects/policy_manager_request.json)
Policy manager response | [policy_manager_response.json](https://github.com/fybrik/fybrik/blob/master/config/taxonomy/objects/policy_manager_response.json)

#### Policy decisions (actions)

The policy decision is documented in the [response json](https://github.com/fybrik/fybrik/blob/master/config/taxonomy/objects/policy_manager_response.json).
Concretely it include the action that should be enforced. The default taxonomy does not include any action.
> For reference we have include [sample actions](https://github.com/fybrik/fybrik/blob/master/config/taxonomy/example/module/actions.yaml)

### Data catalog

TBD



