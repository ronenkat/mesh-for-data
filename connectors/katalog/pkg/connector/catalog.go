// Copyright 2021 IBM Corp.
// SPDX-License-Identifier: Apache-2.0
package connector

import (
	"context"
	"encoding/json"

	"log"

	connectors "fybrik.io/fybrik/pkg/connectors/protobuf"
	vault "fybrik.io/fybrik/pkg/vault"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO(roee88): This is a temporary implementation of a catalog connector to
// Katalog. It is here to map between Katalog CRDs to the connectors proto
// definitions. Eventually, the connectors proto definitions won't hardcode so
// much and rely on validating against a configured OpenAPI spec instead, making
// most of the code in this file unnecessary.

type DataCatalogService struct {
	connectors.UnimplementedDataCatalogServiceServer

	client kclient.Client
}

func (s *DataCatalogService) GetDatasetInfo(ctx context.Context, req *connectors.CatalogDatasetRequest) (*connectors.CatalogDatasetInfo, error) {
	namespace, name, err := splitNamespacedName(req.DatasetId)
	if err != nil {
		return nil, err
	}
	log.Printf("In GetDatasetInfo: asset namespace is " + namespace + " asset name is " + name)
	asset, err := getAsset(ctx, s.client, namespace, name)
	if err != nil {
		return nil, err
	}

	datastore, err := buildDataStore(asset)
	if err != nil {
		return nil, err
	}

	log.Printf("In GetDatasetInfo: VaultSecretPath is " + vault.PathForReadingKubeSecret(namespace, asset.Spec.SecretRef.Name))
	return &connectors.CatalogDatasetInfo{
		DatasetId: req.DatasetId,
		Details: &connectors.DatasetDetails{
			Name:       req.DatasetId,
			DataOwner:  emptyIfNil(asset.Spec.AssetMetadata.Owner),
			DataFormat: emptyIfNil(asset.Spec.AssetDetails.DataFormat),
			Geo:        emptyIfNil(asset.Spec.AssetMetadata.Geography),
			DataStore:  datastore,
			CredentialsInfo: &connectors.CredentialsInfo{
				VaultSecretPath: vault.PathForReadingKubeSecret(namespace, asset.Spec.SecretRef.Name),
			},
			Metadata: buildDatasetMetadata(asset),
		},
	}, nil
}

func buildDatasetMetadata(asset *Asset) *connectors.DatasetMetadata {
	assetMetadata := asset.Spec.AssetMetadata

	var namedMetadata map[string]string
	if assetMetadata.NamedMetadata != nil {
		namedMetadata = assetMetadata.NamedMetadata.AdditionalProperties
	}

	componentsMetadata := map[string]*connectors.DataComponentMetadata{}
	for componentName, componentValue := range assetMetadata.ComponentsMetadata.AdditionalProperties {
		var componentNamedMetadata map[string]string
		if componentValue.NamedMetadata != nil {
			componentNamedMetadata = componentValue.NamedMetadata.AdditionalProperties
		}
		componentsMetadata[componentName] = &connectors.DataComponentMetadata{
			ComponentType: "column",
			Tags:          emptyArrayIfNil(componentValue.Tags),
			NamedMetadata: componentNamedMetadata,
		}
	}

	return &connectors.DatasetMetadata{
		DatasetTags:          emptyArrayIfNil(assetMetadata.Tags),
		DatasetNamedMetadata: namedMetadata,
		ComponentsMetadata:   componentsMetadata,
	}
}

func buildDataStore(asset *Asset) (*connectors.DataStore, error) {
	connection := asset.Spec.AssetDetails.Connection
	switch connection.Type {
	case "s3":
		return &connectors.DataStore{
			Type: connectors.DataStore_S3,
			Name: asset.Name,
			S3: &connectors.S3DataStore{
				Endpoint:  connection.S3.Endpoint,
				Bucket:    connection.S3.Bucket,
				ObjectKey: connection.S3.ObjectKey,
				Region:    emptyIfNil(connection.S3.Region),
			},
		}, nil
	case "kafka":
		return &connectors.DataStore{
			Type: connectors.DataStore_KAFKA,
			Name: asset.Name,
			Kafka: &connectors.KafkaDataStore{
				TopicName:             emptyIfNil(connection.Kafka.TopicName),
				BootstrapServers:      emptyIfNil(connection.Kafka.BootstrapServers),
				SchemaRegistry:        emptyIfNil(connection.Kafka.SchemaRegistry),
				KeyDeserializer:       emptyIfNil(connection.Kafka.KeyDeserializer),
				ValueDeserializer:     emptyIfNil(connection.Kafka.ValueDeserializer),
				SecurityProtocol:      emptyIfNil(connection.Kafka.SecurityProtocol),
				SaslMechanism:         emptyIfNil(connection.Kafka.SaslMechanism),
				SslTruststore:         emptyIfNil(connection.Kafka.SslTruststore),
				SslTruststorePassword: emptyIfNil(connection.Kafka.SslTruststorePassword),
			},
		}, nil
	case "db2":
		return &connectors.DataStore{
			Type: connectors.DataStore_DB2,
			Name: asset.Name,
			Db2: &connectors.Db2DataStore{
				Url:      emptyIfNil(connection.Db2.Url),
				Database: emptyIfNil(connection.Db2.Database),
				Table:    emptyIfNil(connection.Db2.Table),
				Port:     emptyIfNil(connection.Db2.Port),
				Ssl:      emptyIfNil(connection.Db2.Ssl),
			},
		}, nil
	default:
		return nil, errors.New("unknown datastore type")
	}
}

func getAsset(ctx context.Context, client kclient.Client, namespace string, name string) (*Asset, error) {
	// Read asset as unstructured
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(schema.GroupVersionKind{Group: GroupVersion.Group, Version: GroupVersion.Version, Kind: "Asset"})
	object.SetNamespace(namespace)
	object.SetName(name)

	objectKey := kclient.ObjectKeyFromObject(object)

	err := client.Get(ctx, objectKey, object)
	if err != nil {
		return nil, err
	}

	// Decode into an Asset object
	asset := &Asset{}
	bytes, err := object.MarshalJSON()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, asset)
	if err != nil {
		return nil, err
	}

	return asset, nil
}
