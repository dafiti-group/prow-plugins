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
	"fmt"
	"regexp"
	"strings"

	"github.com/dafiti-group/prow-plugins/pkg/teams/file"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
)

var (
	succesMessage = "Teams were synced"
	failMessage   = "Failed to sync Teams: `%v`"
	usersDiffMsg  = "Some users are on the organization but are not declared on the file, please remove them manualy or update the file: %v"
	PRNotApproved = "Your pull request is not approved yet"
	syncRe        = regexp.MustCompile(`(?mi)^/sync-teams\s*$`)
)

func (s *Server) handle(l *logrus.Entry, org, repo, commit, body string, isCMD bool, number int, state github.ReviewState) (err error) {
	//
	bodyMatchString := syncRe.MatchString(body)

	// Setup Logger
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   number,
		"state":             state,
		"commit":            commit,
		"body":              body,
		"bodyMatchString ":  bodyMatchString,
	})

	//
	l.Info("Handle")

	// Skip if trigger false
	if !bodyMatchString && isCMD {
		l.Infof("will not trigger for `%v`", body)
		return nil
	}

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

	//
	file := file.New(l, s.Ghc, s.Gc, org)

	// Clone Repo
	if err = file.Clone(repo, commit); err != nil {
		return err
	}

	// Fetch team ID
	if err = file.Fetch(); err != nil {
		// Comment if there something wrong with the file
		// @TODO: Check erro type here, This error might not be about this
		if err = s.Ghc.CreateComment(org, repo, number, fmt.Sprintf(usersDiffMsg, err.Error())); err != nil {
			return err
		}
		return err
	}

	// If is not approved and and is valid command comment on PR
	if state != github.ReviewStateApproved && bodyMatchString {
		if err = s.Ghc.CreateComment(org, repo, number, PRNotApproved); err != nil {
			return err
		}
	}

	//
	if state != github.ReviewStateApproved {
		s.Log.Warn(PRNotApproved)
		return nil
	}

	//
	if err = file.Sync(); err != nil {
		if err = s.Ghc.CreateComment(org, repo, number, failMessage); err != nil {
			return err
		}
		return err
	}

	//
	if err = s.Ghc.CreateComment(org, repo, number, succesMessage); err != nil {
		return err
	}

	return err
}

//
func shouldPrune(botName string) func(github.IssueComment) bool {
	return func(ic github.IssueComment) bool {
		hasMsgs := strings.Contains(ic.Body, succesMessage) ||
			strings.ContainsAny(ic.Body, failMessage) ||
			strings.ContainsAny(ic.Body, usersDiffMsg) ||
			strings.ContainsAny(ic.Body, PRNotApproved)
		return github.NormLogin(botName) == github.NormLogin(ic.User.Login) && hasMsgs
	}
}
