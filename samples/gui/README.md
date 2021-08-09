## Description

Web UI application for use by data users to create an environment for running their indicated workload with the requested data sets, for the declared purpose. Using this application, a data user can create a new environment, or manage an existing one, while monitoring the progress of the environment's construction. As a prerequisite to environment creation, the user must provide credentials to allow the environment to access the requested data asset.

The APIs and examples of their use are as follows:
  
  Create FybrikApplication (to be used by the specific workload):

	curl -X POST -i http://localhost:8080/v1/dma/fybrikapplication --data '{"apiVersion": "app.fybrik.io/v1alpha1","kind": "FybrikApplication","metadata": {"name": "unittest-notebook1", "namespace": "default"},"spec": {"selector": {"clusterName": "thegreendragon","workloadSelector": {"matchLabels": {"app": "unittest-notebook1"}}}, "appInfo": {"intent": "fraud-detection"}, "data": [{ "dataSetID": "whatever", "requirements": {"interface": {"protocol": "s3","dataformat": "parquet"}}}]}}'

  Create FybrikApplication (to copy data and register in the public catalog):

	curl -X POST -i http://localhost:8080/v1/dma/fybrikapplication --data '{"apiVersion": "app.fybrik.io/v1alpha1","kind": "FybrikApplication","metadata": {"name": "unittest-copy", "namespace": "default"},"spec": {"selector": {"workloadSelector": {}, "appInfo": {"intent": "copy data"}, "data": [{ "dataSetID": "whatever", "requirements": {"copy": {"required": true, catalog: {catalogID: "Enterprise Catalog"}}, "interface": {"protocol": "s3","dataformat": "parquet"}}}]}}}'

curl -X POST -i http://localhost:8080/v1/dma/fybrikapplication --data '{"apiVersion": "app.fybrik.io/v1alpha1","kind": "FybrikApplication", }'

	Get list of FybrikApplications
	  curl -X GET -i http://localhost:8080/v1/dma/fybrikapplication
	
	Get specific FybrikApplication:
	  curl -X GET -i http://localhost:8080/v1/dma/fybrikapplication/unittest-notebook1
	
	Delete FybrikApplication:
	  curl -X DELETE -i http://localhost:8080/v1/dma/fybrikapplication/unittest-notebook1
	
	
	Create Credentials
	  curl -X POST -i http://localhost:8080/v1/creds/usercredentials --data '{"SecretName": "user-creds","System": "Egeria", "Credentials": {"username": "admin"}}'
	
	Get Credentials
	  curl -X GET -i http://localhost:8080/v1/creds/usercredentials/user-creds
	  ==> returns: "{\"Egeria_username\":\"admin\"}"
	
	Delete Credentials
	  curl -X DELETE -i http://localhost:8080/v1/creds/usercredentials/user-creds



	Get Environment Info
	  curl -X GET -i http://localhost:8080/v1/env/datauserenv


## Run server locally - run vault as well as REST API server

export KUBECONFIG=$HOME/.kube/config
export GEOGRAPHY=hobbiton
make build
./datauserserver

## Test locally (assuming datauserserver is not running)
From within samples/gui/server
make test

## Working in a cluster
GUI is deployed in the namespace the workload is running in. This should also be your current namespace.

## Creating docker images
Backend image creation
```
make docker-all
```
Frontend image creation

```
cd <root>/samples/gui/front-end
npm install
Ensure that .env has a correct configuration 
export NODE_OPTIONS=--max_old_space_size=4096
rm -rf build
npm run build
make docker-all
```
## Deployment
  ```
cd <root>>/samples/gui
./deploy.sh
```

## Run 

```
kubectl port-forward service/datauserserver 8080:8080&
kubectl port-forward service/datauserclient 3000:3000&

Open a browser

Connect to localhost:3000

