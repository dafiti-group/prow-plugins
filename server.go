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

package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/k0kubun/pp"
	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

type Server struct {
	tokenGenerator func() []byte
	prowURL        string
	configAgent    *config.Agent
	ghc            github.Client
	log            *logrus.Entry
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.log.Info("will validade webhook")
	pp.Println("token from generator", string(s.tokenGenerator()))
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.tokenGenerator)
	if !ok {
		s.log.Error("validate webhook failed")
		return
	}
	s.log.Info("webhook ok")

	// Respond with
	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		s.log.Errorf("Error parsing event. %v", err)
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

		if err := s.handlePR(l, &p); err != nil {
			s.log.WithFields(l.Data).Errorf("Refreshing github statuses failed. %v", err)
		}

		// go func() {
		// 	if err := s.handlePR(l, &p); err != nil {
		// 		s.log.WithError(err).WithFields(l.Data).Info("Refreshing github statuses failed.")
		// 	}
		// }()
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
	)
	pp.Println("title", title, "org", org, "repo", repo, "number", number)

	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   number,
		"title":             title,
	})

	s.log.Info("get labels")
	labels, err := s.ghc.GetIssueLabels(org, repo, number)
	if err != nil {
		l.Errorf("failed to get labels", err)
		return err
	}
	pp.Println("labels", labels)

	if title == "test" {
		s.log.Info("will add label")
		err = s.ghc.AddLabel(org, repo, number, "invalid")
		if err != nil {
			l.Errorf("Failed to add label %v", err)
			return err
		}
	}

	return err
}

func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The refresh plugin is used for refreshing status contexts in PRs. Useful in case GitHub breaks down.`,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/jira-checker",
		Description: "Check PR Title for Jira Tag",
		WhoCanUse:   "Anyone",
		Examples:    []string{"/jira-checker"},
	})
	return pluginHelp, nil
}
