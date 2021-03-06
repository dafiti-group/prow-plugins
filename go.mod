module github.com/dafiti-group/prow-plugins

go 1.14

require (
	github.com/creasty/defaults v1.4.0
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/k0kubun/pp v3.0.1+incompatible
	github.com/sirupsen/logrus v1.6.0
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/apimachinery v0.17.3
	k8s.io/test-infra v0.0.0-20200710181349-57259ab4e5ed
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	k8s.io/client-go => k8s.io/client-go v0.17.3
)
