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

package teams

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/k0kubun/pp"
	"github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

type Server struct {
	TokenGenerator func() []byte
	Gc             git.ClientFactory
	ConfigAgent    *config.Agent
	Ghc            github.Client
	Log            *logrus.Entry
}

type Teams struct {
	Teams []Team `yaml:"teams"`
}

type Team struct {
	Login string `yaml:"login"`
}

var (
	targetBranch = "master"
	fileName     = "TEAMS"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.TokenGenerator)
	if !ok {
		s.Log.Error("validate webhook failed")
		return
	}

	pp.Println("==========")
	fmt.Println(string(payload))
	pp.Println("==========")
	// Respond with
	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		s.Log.WithError(err).Error("Error parsing event.")
		fmt.Fprint(w, "Something went wrong")
		return
	}

	s.Log.Info("handle event ok")
	fmt.Fprint(w, "Event received. Have a nice day.")
}

func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) (err error) {
	l := s.Log.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)
	l.Info("Event received")

	switch eventType {
	case "push":
		var p github.PullRequestEvent

		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}

		go func() {
			if err := s.handlePR(l, &p); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Info("Refreshing github statuses failed.")
			}
		}()
	default:
		s.Log.Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *Server) handlePR(l *logrus.Entry, p *github.PullRequestEvent) (err error) {
	var (
		org    = p.Repo.Owner.Login
		repo   = p.Repo.Name
		number = p.Number
		title  = p.PullRequest.Title
		action = p.Action
	)

	// Setup Logger
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   number,
		"action":            action,
		"title":             title,
	})

	l.Info("Handle PR")

	if err = s.handle(org, repo, targetBranch); err != nil {
		s.Log.Error(err)
		return err
	}

	return err
}

func (s *Server) handle(org, repo, targetBranch string) error {
	// Clone the repo, checkout the target branch.
	r, err := s.Gc.ClientFor(org, repo)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Clean(); err != nil {
			s.Log.WithError(err).Error("Error cleaning up repo.")
		}
	}()
	if err := r.Checkout(targetBranch); err != nil {
		resp := fmt.Sprintf("cannot checkout %s: %v", targetBranch, err)
		s.Log.Warningf(resp)
		return err
	}
	teams := &Teams{}
	path := filepath.Join(r.Directory(), fileName)
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		s.Log.Error(err)
	}
	err = yaml.Unmarshal(yamlFile, teams)
	if err != nil {
		s.Log.Error(err)
	}

	pp.Println(teams)

	return nil
}

func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return &pluginhelp.PluginHelp{
		Description: "Syncs TEAMS file declaration with github teams",
	}, nil
}
