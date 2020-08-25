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

package checkmarx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/v2"
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
	Gc             git.ClientFactory
	Ghc            github.Client
	Log            *logrus.Entry
}

const (
	InvalidLabel = "do-not-merge/verify-checkmarx"
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
	//
	botName, err := s.Ghc.BotName()
	if err != nil {
		l.WithError(err).Error("failed getting botName")
		return err
	}

	// Clear comments
	if err = s.Ghc.DeleteStaleComments(org, repo, number, nil, shouldPrune(botName)); err != nil {
		l.WithError(err).Error("failed to prune comments")
		return err
	}

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

	pluginAgent := s.Pa
	if pluginAgent == nil {
		return errors.New("Could not get Plugin Agent in checkmarx")
	}

	config := pluginAgent.Config()
	if config == nil {
		return errors.New("Could not get Configuration in checkmarx")
	}

	if orgs, repos := config.EnabledReposForExternalPlugin("checkmarx"); orgs != nil || repos != nil {
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

	err = s.Ghc.AddLabel(org, repo, number, InvalidLabel)
	if err != nil {
		l.WithError(err).Error("failed to add label")
		return err
	}
	l.Info("Label added %v", InvalidLabel)

	// @TODO: Start checkmarx job
	l.Info("Start prow job")
	// @TODO: Create commentary to tell job status
	// s.ghc.CreateComment(org, repo, number, msg)
	// if err != nil {
	// 	l.WithError(err).Error("failed to add label")
	// 	return err
	// }

	// @TODO: Remove label when job is done
	// err = s.Ghc.RemoveLabel(org, repo, number, InvalidLabel)
	// if err != nil {
	// 	l.WithError(err).Error("failed to remove label")
	// 	return err
	// }

	l.Info("HandlePR end")
	return nil
}

//
func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return &pluginhelp.PluginHelp{
		Description: "The Jira checker plugin checks your PR name",
	}, nil
}

//
func shouldPrune(botName string) func(github.IssueComment) bool {
	return func(ic github.IssueComment) bool {
		hasMsgs := strings.Contains(ic.Body, InvalidLabel)
		return github.NormLogin(botName) == github.NormLogin(ic.User.Login) && hasMsgs
	}
}
