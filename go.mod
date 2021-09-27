module github.com/plumber-cd/runtainer

go 1.16

require (
	github.com/docker/cli v0.0.0-20200130152716-5d0cf8839492
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.3.3
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6
	github.com/spf13/afero v1.3.4
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	gopkg.in/yaml.v2 v2.3.0 // indirect
	helm.sh/helm/v3 v3.3.0
	k8s.io/api v0.18.4
	k8s.io/apimachinery v0.18.4
	k8s.io/client-go v0.18.4
	k8s.io/kubectl v0.18.4
)
