/*
Copyright 20212The Clusternet Authors.
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

package deployment

import (
	"encoding/json"

	appsapi "github.com/clusternet/clusternet/pkg/apis/apps/v1alpha1"
	"github.com/clusternet/clusternet/pkg/controllers/apps/feedinventory/utils"

	k8sappsv1 "k8s.io/api/apps/v1"
)

const deployment = "Deployment"

type Plugin struct {
	name string
}

// NewPlugin return a plugin
func NewPlugin() *Plugin {
	return &Plugin{
		name: deployment,
	}
}

// Parser will parse workload spec replicas
func (pl *Plugin) Parser(rawData []byte) (*int32, appsapi.ReplicaRequirements, string, error) {
	var deploy k8sappsv1.Deployment
	if err := json.Unmarshal(rawData, &deploy); err != nil {
		return nil, appsapi.ReplicaRequirements{}, "", err
	}

	return deploy.Spec.Replicas, utils.GetReplicaRequirements(deploy.Spec.Template.Spec), "/spec/replicas", nil
}

// Name return plugin name
func (pl *Plugin) Name() string {
	return pl.name
}

// Kind return resource kind name for plugin
func (pl *Plugin) Kind() string {
	return deployment
}
