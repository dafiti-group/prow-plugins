package file

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/repoowners"
)

type Base struct {
	ApiVersion string `yaml:"apiVersion"`
	Teams      []Team `yaml:"teams"`
	log        *logrus.Entry
	ghc        github.Client
	gc         git.ClientFactory
	oc         *repoowners.Client
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
	fileName = "OWNERS_ALIASES"
)

func New(l *logrus.Entry, ghc github.Client, gc git.ClientFactory, oc *repoowners.Client, org string) *Base {
	return &Base{
		ghc: ghc,
		log: l,
		org: org,
		gc:  gc,
		oc:  oc,
	}
}

func (s *Base) Clone(repo, commit string) (err error) {
	ra, err := s.oc.LoadRepoAliases(s.org, repo, commit)
	for teamName, members := range ra {
		team := Team{
			Name: teamName,
		}
		for _, member := range members.List() {
			team.Members = append(team.Members, Member{
				Login: member,
			})
		}
		s.Teams = append(s.Teams, team)
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
		diff := diff(actualMembers, team.Members)
		if len(diff) != 0 {
			diffMembersList[t.Slug] = diff
		}
	}

	//
	if len(diffMembersList) != 0 {
		err = fmt.Errorf("%v", diffMembersList)
		return err
	}

	return nil
}

//
func diff(currentUsers []github.TeamMember, fileUsers []Member) []string {
	mb := make(map[string]struct{}, len(fileUsers))
	for _, x := range fileUsers {
		mb[strings.ToLower(x.Login)] = struct{}{}
	}

	var diff []string
	for _, x := range currentUsers {
		if _, found := mb[strings.ToLower(x.Login)]; !found {
			diff = append(diff, x.Login)
		}
	}
	return diff
}
