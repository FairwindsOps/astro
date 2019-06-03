// Copyright 2019 ReactiveOps
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/reactiveops/dd-manager/pkg/controller"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

// RootCmd is the base dd-manager command
func RootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "dd-manager",
		Short: "Kubernetes datadog monitor manager",
		Long:  "A kubernetes agent that manages datadog monitors.",
		Run:   run,
	}
	return root
}

func run(cmd *cobra.Command, args []string) {
	log.SetReportCaller(true)
	log.SetOutput(os.Stdout)
	controller.NewController()
}
