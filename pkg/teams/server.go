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
	"strings"

	"github.com/creasty/defaults"
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

type Base struct {
	ApiVersion string              `yaml:"apiVersion"`
	Teams      []map[string][]Team `yaml:"teams"`
}

type Team struct {
	Login      string `yaml:"login"`
	Maintainer bool   `default:"true" yaml:"maintainer"`
}

var (
	fileName      = "TEAMS"
	succesMessage = "Teams were synced"
	failMessage   = "Failed to sync Teams: `%v`"
	usersDiffMsg  = "Some users are on the organization but are not declared on the file, please remove them manualy or update the file: %v"
	PRNotApproved = "Your pull request is not approved yet"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//
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

	//
	s.Log.Info("handle event ok")
	fmt.Fprint(w, "Event received. Have a nice day.")
}

func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) (err error) {
	//
	l := s.Log.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)
	l.Info("Event received")

	pp.Println("=======")
	fmt.Println(string(payload))
	pp.Println("=======")

	switch eventType {
	case "pull_request_review_comment":
		var e github.ReviewEvent

		if err := json.Unmarshal(payload, &e); err != nil {
			return err
		}

		go func() {
			if err := s.handleReviewEvent(l, &e); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Info("Refreshing github statuses failed.")
			}
		}()
	default:
		s.Log.Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *Server) handleReviewEvent(l *logrus.Entry, e *github.ReviewEvent) (err error) {
	var (
		org    = e.Repo.Owner.Login
		repo   = e.PullRequest.Base.Repo.Name
		number = e.PullRequest.Number
		state  = github.ReviewState(strings.ToUpper(string(e.Review.State)))
		commit = e.PullRequest.Head.Ref
	)

	// Setup Logger
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   number,
		"state":             state,
		"commit":            commit,
	})

	//
	l.Info("Handle PR")

	//
	botName, err := s.Ghc.BotName()
	if err != nil {
		s.Log.WithError(err).Error("failed getting botName")
		return err
	}

	//
	if err = s.Ghc.DeleteStaleComments(org, repo, number, nil, shouldPrune(botName)); err != nil {
		s.Log.WithError(err).Error("failed to prune comments")
		return err
	}

	//
	if state != github.ReviewStateApproved {
		s.Log.Info(PRNotApproved)
		return nil
	}

	//
	if err = s.handle(org, repo, commit); err != nil {
		s.Log.Error(err)
		//
		if err = s.Ghc.CreateComment(org, repo, number, fmt.Sprintf(failMessage, err.Error())); err != nil {
			s.Log.WithError(err).Error("failed to create comment on handle")
			return err
		}
		return err
	}

	//
	if err = s.Ghc.CreateComment(org, repo, number, succesMessage); err != nil {
		s.Log.WithError(err).Error("failed to create comment after handle")
		return err
	}

	return err
}

func shouldPrune(botName string) func(github.IssueComment) bool {
	return func(ic github.IssueComment) bool {
		return github.NormLogin(botName) == github.NormLogin(ic.User.Login) &&
			(strings.Contains(ic.Body, succesMessage) || strings.ContainsAny(ic.Body, failMessage))
	}
}

func (s *Server) handle(org, repo, commit string) error {
	// Clone the repo, checkout the target branch.
	r, err := s.Gc.ClientFor(org, repo)
	if err != nil {
		return err
	}

	//
	defer func() {
		if err := r.Clean(); err != nil {
			s.Log.WithError(err).Error("Error cleaning up repo.")
		}
	}()

	//
	if err := r.Checkout(commit); err != nil {
		s.Log.WithError(err).Warningf("cannot checkout %s", commit)
		return err
	}

	//
	base := &Base{}
	path := filepath.Join(r.Directory(), fileName)

	//
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		s.Log.Error(err)
		return err
	}

	//
	err = yaml.Unmarshal(yamlFile, base)
	if err != nil {
		s.Log.Error(err)
		return err
	}

	diffMembersList := make(map[string][]string)
	//
	for _, a := range base.Teams {
		for teamName, teamMembers := range a {
			var teamMembersList []Team
			//
			team, err := s.Ghc.GetTeamBySlug(teamName, org)
			if err != nil {
				s.Log.WithError(err).Errorf("team %v not found", teamName)
				return err
			}

			//
			for _, teamMember := range teamMembers {
				defaults.Set(teamMember)
				teamMembersList = append(teamMembersList, teamMember)
				if err := s.updateMembership(team, org, teamName, teamMember); err != nil {
					s.Log.WithError(err).Error("failed to sync")
					return err
				}
			}

			//
			actualMembers, err := s.Ghc.ListTeamMembers(team.ID, "")
			if err != nil {
				s.Log.WithError(err).Error("failed geting team members")
				return err
			}

			//
			diff := getDiff(actualMembers, teamMembersList)
			diffMembersList[team.Slug] = diff
		}
	}

	if len(diffMembersList) != 0 {
		err = fmt.Errorf(usersDiffMsg, diffMembersList)
		s.Log.Error(err)
		return err
	}

	return nil
}
func getDiff(currentUsers []github.TeamMember, fileUsers []Team) []string {
	mb := make(map[string]struct{}, len(fileUsers))
	for _, x := range fileUsers {
		mb[x.Login] = struct{}{}
	}

	var diff []string
	for _, x := range currentUsers {
		if _, found := mb[x.Login]; !found {
			diff = append(diff, x.Login)
		}
	}
	return diff
}

//
func (s *Server) updateMembership(team *github.Team, org string, teamName string, teamMember Team) (err error) {
	isMember, err := s.Ghc.IsMember(org, teamMember.Login)
	if err != nil {
		return err
	}

	//
	if isMember {
		s.Log.Infof("%v is already member of %v", teamMember.Login, team.Name)
		return err
	}

	// Add Member
	_, err = s.Ghc.UpdateTeamMembership(team.ID, teamMember.Login, teamMember.Maintainer)

	return err
}

func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return &pluginhelp.PluginHelp{
		Description: "Syncs TEAMS file declaration with github teams",
	}, nil
}
