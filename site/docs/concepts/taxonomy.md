# Taxonomy

The Mesh for Data control plane orchestrate together multiple components that needs to communicate and share information. 
For example, when a policy decision calls for an action on the data, Mesh for Data needs to match that action with a module that is able to apply it. Another example is a workload that needs to use a specific API requires the control plane to deploy an appropriate module that accepts request in the given API.

For this purpose Mesh for Data has a common taxonomy, a glossary, that is shared between the control plane components. The taxonomy defines the verbs and terms that are used in `M4DApplication`, `M4DModule`, policy decision actions, and catalog metadata. Thus enabling the control plane to understand how to optimize and protect the use of data.

Mesh for Data open approach to 3rd party data catalog and policy managers requires that the verbs and terms that are used by the catalog or policy manager be translated into equivalent verbs and terms that are used by the control plane. In Mesh for Data architecture this functions is performed by the connectors.

![Taxonomy](../static/m4d-taxonomy.svg)

## Specifying taxonomy

Taxonomy definition are coded as [OpenAPI v3 schemas](https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.0.md#schemaObject). The schemas are deployed as Kubernetes config maps and can be updated at any time.
Note that only the values that appears in the schema can be changed, and not the schema structure.

## Taxonomy for the control plane

### Workload (`M4DApplication`)

The taxonomy determines the valid values that 

### Data plane modules (`M4DModule')

Modules are the active components that interacts with the workload and apply the actions derived from the policies.

### Data catalog

TBD

### Policy decisions (actions)

The evaluation of the policies is driven by input that may be available from multiple sources.

## Taxonomy for Open Policy Agent (OPA)


