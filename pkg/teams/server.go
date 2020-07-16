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
	"net/http"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.TokenGenerator)
	if !ok {
		s.Log.Error("validate webhook failed")
		return
	}

	fmt.Println("================")
	fmt.Println(string(payload))
	fmt.Println("================")
	// Respond with
	if err := s.serverHandler(eventType, eventGUID, payload); err != nil {
		s.Log.WithError(err).Error("Error parsing event.")
		fmt.Fprint(w, "Something went wrong")
		return
	}

	//
	s.Log.Info("handle event ok")
	fmt.Fprint(w, "Event received. Have a nice day.")
}

func (s *Server) serverHandler(eventType, eventGUID string, payload []byte) (err error) {
	//
	l := s.Log.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)
	l.Info("Event received")

	switch eventType {
	case "issue_comment":
		var e github.IssueCommentEvent

		if err := json.Unmarshal(payload, &e); err != nil {
			return err
		}

		go func() {
			if err := s.handleCommentEvent(l, &e); err != nil {
				l.WithError(err).WithFields(l.Data).Info("issue comment failed.")
			}
		}()
	case "pull_request_review":
		var e github.ReviewEvent

		if err := json.Unmarshal(payload, &e); err != nil {
			return err
		}

		go func() {
			if err := s.handleReviewEvent(l, &e); err != nil {
				l.WithError(err).WithFields(l.Data).Info("pull request review  failed.")
			}
		}()
	case "pull_request":
		var p github.PullRequestEvent

		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}

		go func() {
			if err := s.handlePR(l, &p); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Info("Pull request event failed.")
			}
		}()
	default:
		s.Log.Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func HelpProvider() pluginhelp.Command {
	return pluginhelp.Command{
		Usage:       "/sync-teams",
		Description: "Syncs TEAMS file declaration with github teams",
		WhoCanUse:   "Anyone",
		Examples:    []string{"/sync-teams"},
	}
}
