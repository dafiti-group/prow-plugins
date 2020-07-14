module github.com/dafiti-group/prow-plugins

go 1.14

require (
	github.com/sirupsen/logrus v1.6.0
	k8s.io/test-infra v0.0.0-20200710181349-57259ab4e5ed
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	k8s.io/client-go => k8s.io/client-go v0.17.3
)
