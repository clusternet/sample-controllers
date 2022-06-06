/*
Copyright 2022 The Clusternet Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package registry

import (
	appsapi "github.com/clusternet/clusternet/pkg/apis/apps/v1alpha1"
	"github.com/clusternet/sample-controller/pkg/registry/deployment"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Registry map[schema.GroupVersionKind]PluginFactory

type PluginFactory interface {
	Parser(rawData []byte) (*int32, appsapi.ReplicaRequirements, string, error)
	Name() string
	Kind() string
}

// NewRegitsty return a plugin registry with workload gvk
func NewRegistry() Registry {
	// Add plugin for your custom define workload
	deployPlugin := deployment.NewPlugin()
	//TODO add statefulse

	return map[schema.GroupVersionKind]PluginFactory{
		// Deployment
		{Group: "apps", Version: "v1", Kind: deployPlugin.Kind()}:            deployPlugin,
		{Group: "apps", Version: "v1beta1", Kind: deployPlugin.Kind()}:       deployPlugin,
		{Group: "apps", Version: "v1beta2", Kind: deployPlugin.Kind()}:       deployPlugin,
		{Group: "extensions", Version: "v1beta1", Kind: deployPlugin.Kind()}: deployPlugin,
	}
}
