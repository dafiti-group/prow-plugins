package file

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"
)

type Base struct {
	ApiVersion string `yaml:"apiVersion"`
	Teams      []Team `yaml:"teams"`
	log        *logrus.Entry
	ghc        github.Client
	gc         git.ClientFactory
	org        string
}

type Team struct {
	ID      int      `yaml:"id"`
	Name    string   `yaml:"name"`
	Members []Member `yaml:"members"`
}

type Member struct {
	Login      string `yaml:"login"`
	Maintainer bool   `default:"true" yaml:"maintainer"`
}

var (
	fileName = "TEAMS"
)

func New(l *logrus.Entry, ghc github.Client, gc git.ClientFactory, org string) *Base {
	return &Base{
		ghc: ghc,
		log: l,
		org: org,
		gc:  gc,
	}
}

func (s *Base) Clone(repo, commit string) (err error) {
	// Clone the repo, checkout the target branch.
	r, err := s.gc.ClientFor(s.org, repo)
	if err != nil {
		return err
	}

	//
	defer func() {
		if err := r.Clean(); err != nil {
			s.log.WithError(err).Error("Error cleaning up repo.")
		}
	}()

	//
	if err := r.Checkout(commit); err != nil {
		s.log.WithError(err).Warningf("cannot checkout %s", commit)
		return err
	}

	//
	path := filepath.Join(r.Directory(), fileName)

	//
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		s.log.Error(err)
		return err
	}

	//
	err = yaml.Unmarshal(yamlFile, s)
	if err != nil {
		s.log.Error(err)
		return err
	}

	return nil
}

//
func (s *Base) Sync() (err error) {
	for _, team := range s.Teams {
		for _, member := range team.Members {
			if _, err = s.ghc.UpdateTeamMembership(team.ID, member.Login, member.Maintainer); err != nil {
				s.log.WithError(err).Error("failed to sync")
				return err
			}
		}
	}
	return nil
}

func (s *Base) Fetch() (err error) {
	diffMembersList := make(map[string][]string)

	for key, team := range s.Teams {
		//
		t, err := s.ghc.GetTeamBySlug(team.Name, s.org)
		if err != nil {
			s.log.WithError(err).Errorf("team %v not found", team.Name)
			return err
		}

		//
		actualMembers, err := s.ghc.ListTeamMembers(t.ID, "")
		if err != nil {
			s.log.WithError(err).Error("failed geting team members")
			return err
		}

		// Update team ID
		s.Teams[key].ID = t.ID

		//
		diffMembersList[t.Slug] = diff(actualMembers, team.Members)
	}

	//
	if len(diffMembersList) != 0 {
		err = fmt.Errorf("%v", diffMembersList)
		s.log.Error(err)
		return err
	}

	return nil
}

//
func diff(currentUsers []github.TeamMember, fileUsers []Member) []string {
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
