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
	"regexp"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
	"k8s.io/test-infra/prow/repoowners"
)

type Server struct {
	Oc             *repoowners.Client
	Pa             *plugins.ConfigAgent
	TokenGenerator func() []byte
	ConfigAgent    *config.Agent
	Ghc            github.Client
	Log            *logrus.Entry
}

const (
	InvalidLabel = "do-not-merge/no-jira-issue-on-title"
)

var (
	titleRegex = regexp.MustCompile(`[A-Z]{1,}-\d+`)
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.TokenGenerator)
	if !ok {
		s.Log.Error("validate webhook failed")
		return
	}

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

	switch eventType {
	case "pull_request":
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
		// msg    = "This pull request does not have a jira tag on the title"
	)

	// Setup Logger
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   number,
		"title":             title,
	})

	l.Info("Handle PR")

	// Only consider newly merged PRs
	if action == github.PullRequestActionClosed {
		l.Infof("Pull Request Action '%v' not aplicable", p.Action)
		return nil
	}

	if orgs, repos := s.Pa.Config().EnabledReposForExternalPlugin("jira-checker"); orgs != nil || repos != nil {
		found := false
		repoOrg := fmt.Sprintf("%v/%v", org, repo)
		for _, v := range repos {
			if v == repoOrg {
				found = true
			}
		}

		if found {
			l.Infof("%v is elegible", repoOrg)
		} else {
			err = fmt.Errorf("Org Repo '%v' not allowed", repoOrg)
			l.Error(err)
			return err
		}
	}

	jiraTag := titleRegex.FindString(title)

	if jiraTag == "" {
		err = s.Ghc.AddLabel(org, repo, number, InvalidLabel)
		if err != nil {
			l.WithError(err).Error("failed to add label")
			return err
		}

		// s.ghc.CreateComment(org, repo, number, msg)
		// if err != nil {
		// 	l.WithError(err).Error("failed to add label")
		// 	return err
		// }
		return nil
	}

	// @TODO: Check Jira

	err = s.Ghc.RemoveLabel(org, repo, number, InvalidLabel)
	if err != nil {
		l.WithError(err).Error("failed to remove label")
		return err
	}
	// cp.PruneComments(func(ic github.IssueComment) bool {
	// 	return strings.Contains(ic.Body, blockedPathsBody)
	// })
	return err
}

func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return &pluginhelp.PluginHelp{
		Description: "The Jira checker plugin checks your PR name",
	}, nil
}
