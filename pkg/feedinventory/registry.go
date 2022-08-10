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

package feedinventory

import (
	feedinv "github.com/clusternet/clusternet/pkg/controllers/apps/feedinventory"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewRegistry return a plugin registry with workload gvk
func NewRegistry() feedinv.Registry {
	myFeedRegistry := feedinv.NewInTreeRegistry()

	// TODO: we can replace with our own plugins here and replace default in-tree plugins
	deployPlugin := NewPlugin() // here is an example
	myFeedRegistry[schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: deployPlugin.Kind()}] = deployPlugin
	myFeedRegistry[schema.GroupVersionKind{Group: "apps", Version: "v1beta1", Kind: deployPlugin.Kind()}] = deployPlugin
	myFeedRegistry[schema.GroupVersionKind{Group: "apps", Version: "v1beta2", Kind: deployPlugin.Kind()}] = deployPlugin
	myFeedRegistry[schema.GroupVersionKind{Group: "extensions", Version: "v1beta1", Kind: deployPlugin.Kind()}] = deployPlugin

	return myFeedRegistry
}
