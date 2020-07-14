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

package jira

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

type Server struct {
	tokenGenerator func() []byte
	configAgent    *config.Agent
	ghc            github.Client
	log            *logrus.Entry
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.tokenGenerator)
	if !ok {
		s.log.Error("validate webhook failed")
		return
	}

	// Respond with
	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		s.log.WithError(err).Error("Error parsing event.")
		fmt.Fprint(w, "Something went wrong")
		return
	}

	s.log.Info("handle event ok")
	fmt.Fprint(w, "Event received. Have a nice day.")
}

func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) (err error) {
	l := s.log.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)

	switch eventType {
	case "pull_request":
		var p github.PullRequestEvent

		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}

		go func() {
			if err := s.handlePR(l, &p); err != nil {
				s.log.WithError(err).WithFields(l.Data).Info("Refreshing github statuses failed.")
			}
		}()
	default:
		s.log.Debugf("skipping event of type %q", eventType)
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

	return err
}

func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return &pluginhelp.PluginHelp{
		Description: "Syncs TEAMS file declaration with github teams",
	}, nil
}
