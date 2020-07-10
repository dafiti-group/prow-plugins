/*
Copyright 2017 The Kubernetes Authors.

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

// Refresh retries GitHub status updates for stale PR statuses.
package main

import (
	"github.com/k0kubun/pp"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/github"
	"sigs.k8s.io/yaml"
)

type Server struct {
	tokenGenerator func() []byte
}

func main() {
	tokenPath := "/home/yuri/Keybase/private/yurifrl/envs/GITHUB_ACCESS_TOKEN"
	hmacPlain := "/home/yuri/Workdir/src/github.com/dafiti-group/prow-plugins/examples/hmac-plain"
	hmacYaml := "/home/yuri/Workdir/src/github.com/dafiti-group/prow-plugins/examples/hmac-yaml"

	plainSecretAgent := &secret.Agent{}
	if err := plainSecretAgent.Start([]string{tokenPath, hmacPlain}); err != nil {
		logrus.Errorf("Error starting plain secrets agent. %v", err)
	}

	yamlSecretAgent := &secret.Agent{}
	if err := yamlSecretAgent.Start([]string{tokenPath, hmacYaml}); err != nil {
		logrus.Errorf("Error starting plain secrets agent. %v", err)
	}

	plainS := &Server{
		tokenGenerator: plainSecretAgent.GetTokenGenerator(hmacPlain),
	}

	yamlS := &Server{
		tokenGenerator: yamlSecretAgent.GetTokenGenerator(hmacYaml),
	}

	pt := plainS.tokenGenerator()
	yt := yamlS.tokenGenerator()
	pp.Println("plain", string(pt))
	pp.Println("yaml", string(yt))

	plainRepoToTokenMap := map[string]github.HMACsForRepo{}
	yamlRepoToTokenMap := map[string]github.HMACsForRepo{}

	err := yaml.Unmarshal(pt, &plainRepoToTokenMap)
	if err != nil {
		logrus.WithError(err).Error("Fail to unmarshal yaml")
	}

	err = yaml.Unmarshal(yt, &yamlRepoToTokenMap)
	if err != nil {
		logrus.WithError(err).Error("Fail to unmarshal plain")
	}
}
