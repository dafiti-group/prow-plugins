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
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/k0kubun/pp"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/interrupts"

	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
)

type options struct {
	port int

	configPath string
	dryRun     bool
	github     prowflagutil.GitHubOptions
	prowURL    string

	webhookSecretFile string
}

func (o *options) Validate() error {
	for _, group := range []flagutil.OptionGroup{&o.github} {
		if err := group.Validate(o.dryRun); err != nil {
			return err
		}
	}

	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&o.port, "port", 8888, "Port to listen on.")
	fs.StringVar(&o.configPath, "config-path", "/etc/config/config.yaml", "Path to config.yaml.")
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.StringVar(&o.webhookSecretFile, "hmac-secret-file", "/etc/webhook/hmac", "Path to the file containing the GitHub HMAC secret.")
	for _, group := range []flagutil.OptionGroup{&o.github} {
		group.AddFlags(fs)
	}
	fs.Parse(os.Args[1:])
	return o
}

func main() {
	o := gatherOptions()
	if err := o.Validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	logrus.SetFormatter(&logrus.JSONFormatter{})
	// TODO: Use global option from the prow config.
	logrus.SetLevel(logrus.DebugLevel)
	log := logrus.StandardLogger().WithField("plugin", "refresh")

	configAgent := &config.Agent{}
	if err := configAgent.Start(o.configPath, ""); err != nil {
		log.Errorf("Error starting config agent. %v", err)
	}

	secretAgent := &secret.Agent{}
	pp.Println("webhookSecretFile", o.webhookSecretFile)
	pp.Println("o.github.TokenPath", o.github.TokenPath)
	if err := secretAgent.Start([]string{o.github.TokenPath, o.webhookSecretFile}); err != nil {
		log.Errorf("Error starting secrets agent. %v", err)
	}

	githubClient, err := o.github.GitHubClient(secretAgent, o.dryRun)
	if err != nil {
		log.Errorf("Error getting GitHub client. %v", err)
	}

	serv := &Server{
		tokenGenerator: secretAgent.GetTokenGenerator(o.webhookSecretFile),
		configAgent:    configAgent,
		ghc:            githubClient,
		log:            log,
	}

	mux := http.NewServeMux()
	mux.Handle("/jira-checker", serv)
	externalplugins.ServeExternalPluginHelp(mux, log, HelpProvider)
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: mux}
	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)
}
