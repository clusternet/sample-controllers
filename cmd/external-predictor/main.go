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

package main

import (
	"math/rand"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/clusternet/sample-controller/pkg/predictor"
)

var options predictor.PredictorOptions

var rootCmd = &cobra.Command{
	Use:   "predictor",
	Short: "Predictor for clusternet",
	Long:  "Predictor is a service for scheduler predict replicas according to cluster resources",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Flags().Parse(args); err != nil {
			klog.Exit(err)
		}

		p, err := predictor.NewPredictorServer(options)
		if err != nil {
			klog.Exit(err)
		}

		if err = p.Run(); err != nil {
			klog.Exit(err)
		}
	},
}

func main() {
	rand.Seed(time.Now().UnixNano())
	if err := rootCmd.Execute(); err != nil {
		klog.Exit(err)
	}
}

func init() {
	rootCmd.Flags().UintVar(&options.Port, "port", 80, "port of predictor listen")
	rootCmd.Flags().StringVar(&options.MasterURL, "master", "", "kubernetes master url")
	rootCmd.Flags().StringVar(&options.KubeconfigPath, "kubeconfig", "", "kubernetes cluster config path")
}
