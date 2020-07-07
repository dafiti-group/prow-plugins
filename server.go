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
	"regexp"

	"github.com/k0kubun/pp"
	"github.com/sirupsen/logrus"

	gogh "github.com/google/go-github/github"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

const pluginName = "refresh"

var refreshRe = regexp.MustCompile(`(?mi)^/refresh\s*$`)

func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The refresh plugin is used for refreshing status contexts in PRs. Useful in case GitHub breaks down.`,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/refresh",
		Description: "Refresh status contexts on a PR.",
		WhoCanUse:   "Anyone",
		Examples:    []string{"/refresh"},
	})
	return pluginHelp, nil
}

type Server struct {
	tokenGenerator func() []byte
	prowURL        string
	configAgent    *config.Agent
	ghc            github.Client
	log            *logrus.Entry
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.tokenGenerator)
	if !ok {
		return
	}
	fmt.Fprint(w, "Event received. Have a nice day.")

	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) (err error) {
	l := logrus.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)

	pp.Println(eventType)
	switch eventType {
	case "pull_request":
		var pe github.PullRequestEvent
		var pr github.PullRequest
		var gpe gogh.PullRequestEvent
		var gpr gogh.PullRequest
		err = json.Unmarshal(payload, &pr)
		pp.Println(pr)
		pp.Println(err)
		err = json.Unmarshal(payload, &pe)
		pp.Println(pe)
		pp.Println(err)
		err = json.Unmarshal(payload, &gpe)
		pp.Println(gpe)
		pp.Println(err)
		err = json.Unmarshal(payload, &gpr)
		pp.Println(gpr)
		pp.Println(err)

		if err := json.Unmarshal(payload, &pr); err != nil {
			return err
		}
		return nil
		go func() {
			if err := s.handlePR(l, &pr); err != nil {
				s.log.WithError(err).WithFields(l.Data).Info("Refreshing github statuses failed.")
			}
		}()
	default:
		logrus.Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *Server) handlePR(l *logrus.Entry, pr *github.PullRequest) (err error) {
	pp.Println(pr)

	var (
		org    = pr.Base.Repo.Owner.Login
		repo   = pr.Base.Repo.Name
		number = pr.Number
		title  = pr.Title
		draft  = pr.Draft
	)

	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   number,
		"title":             title,
		"draft":             draft,
	})
	pp.Println("title", title, "org", org, "repo", repo, "number", number)

	if title == "test" {
		err = s.ghc.AddLabel(org, repo, number, "missing-jira-tag")
	}

	return err
}
