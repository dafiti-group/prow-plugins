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
	"strings"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/github"
)

func (s *Server) handleCommentEvent(l *logrus.Entry, e *github.IssueCommentEvent) (err error) {
	var (
		org    = e.Repo.Owner.Login
		repo   = e.Repo.Name
		number = e.Issue.Number
		body   = e.Comment.Body
	)

	pr, err := s.Ghc.GetPullRequest(org, repo, number)
	if err != nil {
		l.WithError(err).Error("failed to get pull request")
		return err
	}
	commit := pr.Head.Ref
	state := github.ReviewState(strings.ToUpper(string(pr.State)))

	err = s.handle(l, org, repo, commit, body, true, number, state)
	if err != nil {
		l.WithError(err).Error("failed do handle request on handle comment")
		return err
	}
	return err
}

func (s *Server) handleReviewEvent(l *logrus.Entry, e *github.ReviewEvent) (err error) {
	var (
		org    = e.Repo.Owner.Login
		repo   = e.PullRequest.Base.Repo.Name
		number = e.PullRequest.Number
		state  = github.ReviewState(strings.ToUpper(string(e.PullRequest.State)))
		commit = e.PullRequest.Head.Ref
	)
	err = s.handle(l, org, repo, commit, "", false, number, state)
	if err != nil {
		l.WithError(err).Error("failed do handle request on pull request review")
		return err
	}
	return err
}

func (s *Server) handlePR(l *logrus.Entry, e *github.PullRequestEvent) (err error) {
	var (
		org    = e.Repo.Owner.Login
		repo   = e.PullRequest.Base.Repo.Name
		number = e.PullRequest.Number
		state  = github.ReviewState(strings.ToUpper(string(e.PullRequest.State)))
		commit = e.PullRequest.Head.Ref
	)
	err = s.handle(l, org, repo, commit, "", false, number, state)
	if err != nil {
		l.WithError(err).Error("failed do handle request on pull request")
		return err
	}
	return err
}
