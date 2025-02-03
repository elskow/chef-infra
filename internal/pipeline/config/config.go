package config

type PipelineConfig struct {
	BuildDir       string       `mapstructure:"build_dir"`
	ArtifactsDir   string       `mapstructure:"artifacts_dir"`
	CacheDir       string       `mapstructure:"cache_dir"`
	DefaultTimeout int          `mapstructure:"default_timeout"`
	NodeJS         NodeJSConfig `mapstructure:"nodejs"`
	Deploy         DeployConfig `mapstructure:"deploy"`
}

type DeployConfig struct {
	Platform      string `mapstructure:"platform"` // "kubernetes" or "static"
	Namespace     string `mapstructure:"namespace"`
	IngressDomain string `mapstructure:"ingress_domain"`
	Registry      string `mapstructure:"registry"`
	PullSecret    string `mapstructure:"pull_secret"`
	ReplicaCount  int    `mapstructure:"replica_count"`

	// Static deployment specific configuration
	StaticPath    string `mapstructure:"static_path"`     // Path where static files will be deployed
	MaxDeploySize int64  `mapstructure:"max_deploy_size"` // Maximum size of deployable artifacts in bytes
}

type NodeJSConfig struct {
	DefaultVersion string            `mapstructure:"default_version"`
	AllowedEngines []string          `mapstructure:"allowed_engines"`
	MaxBuildTime   int               `mapstructure:"max_build_time"`
	BuildCache     bool              `mapstructure:"build_cache"`
	EnvVars        map[string]string `mapstructure:"env_vars"`
	BuildImage     string            `mapstructure:"build_image"`
	Registry       string            `mapstructure:"registry"`
}
