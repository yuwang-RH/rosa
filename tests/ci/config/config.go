package config

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/openshift/rosa/tests/utils/helper"
	. "github.com/openshift/rosa/tests/utils/log"
)

var Test *TestConfig

// TestConfig contains platforms info for the rosacli testing
type TestConfig struct {
	// Env is the OpenShift Cluster Management environment used to provision clusters.
	ENV                 string `env:"OCM_LOGIN_ENV" default:""`
	TestProfile         string `env:"TEST_PROFILE" default:""`
	ResourcesDir        string `env:"RESOURCES_DIR" default:""`
	OutputDir           string `env:"OUTPUT_DIR" default:""`
	YAMLProfilesDir     string `env:"TEST_PROFILE_DIR" default:""`
	RootDir             string `env:"WORKSPACE" default:""`
	ClusterConfigFile   string
	ArtifactDir         string `env:"ARTIFACT_DIR" default:""`
	UserDataFile        string
	KonfluxUserDataFile string
	CreateCommandFile   string
	// Temporary file to compatible to current CI jobs. Will remove once all CI jobs migration finished
	ClusterIDFile     string
	APIURLFile        string
	ClusterNameFile   string
	ClusterTypeFile   string
	ConsoleUrlFile    string
	InfraIDFile       string
	VPCIDFile         string
	PublicSubnetsFile string
	// End of temporary
	ClusterDetailFile               string
	ClusterInstallLogArtifactFile   string
	ClusterAdminFile                string
	ClusterIDPAdminUsernamePassword string
	TestFocusFile                   string
	TestLabelFilterFile             string
	ProxySSHPemFile                 string
	ProxyCABundleFile               string
	GlobalENV                       *GlobalENVVariables
	ClusterENV                      *ClusterENVVariables
	Day2ConfENV                     *Day2ConfENVVariables
}
type GlobalENVVariables struct {
	ChannelGroup          string `env:"CHANNEL_GROUP" default:""`
	Version               string `env:"VERSION" default:""`
	OpenshiftVersion      string `env:"OPENSHIFT_VERSION" default:""`
	Region                string `env:"REGION" default:""`
	ProvisionShard        string `env:"PROVISION_SHARD" default:""`
	NamePrefix            string `env:"NAME_PREFIX"`
	ClusterWaitingTime    int    `env:"CLUSTER_TIMEOUT" default:"90"`
	WaitSetupClusterReady bool   `env:"WAIT_SETUP_CLUSTER_READY" default:"true"`
	AWSCredetialsFile     string `env:"AWS_SHARED_CREDENTIALS_FILE" default:""`
	SVPC_CREDENTIALS_FILE string `env:"SHARED_VPC_AWS_SHARED_CREDENTIALS_FILE" default:""`
	OCM_LOGIN_ENV         string `env:"OCM_LOGIN_ENV" default:""`
}
type ClusterENVVariables struct {
	ComputeMachineType  string `env:"COMPUTE_MACHINE_TYPE" default:""`
	BYOVPC              string `env:"BYOVPC" default:""`
	Private             string `env:"PRIVATE" default:""`
	Autoscale           string `env:"AUTOSCALE" default:""`
	ProxyEnabled        string `env:"PROXY_ENABLED" default:""`
	FipsEnabled         string `env:"FIPS_ENABLED" default:""`
	VolumeSize          string `env:"VOLUME_SIZE" default:""`
	Replicas            string `env:"REPLICAS" default:""`
	MultiAZ             string `env:"MULTI_AZ" default:""`
	AllowRegistries     string `env:"ALLOW_REGISTRIES" default:""`
	Add_UnManaged_Tag   string `env:"ADD_UNMANAGED_TAG" default:"false"`
	UseLocalCredentials bool   `env:"USE_LOCAL_CREDENTIALS" default:"false"`
}

type Day2ConfENVVariables struct {
	LocalZoneMP       bool `env:"LOCAL_ZONE_MP" default:"false"`
	TuningConfig      bool `env:"TUNING_CONFIG" default:"false"`
	TuningConfigMP    bool `env:"TUNING_CONFIG_MP" default:"false"`
	ClusterAutoScaler bool `env:"CLUSTER_AUTOSCALER" default:"false"`
}

func init() {
	Test = new(TestConfig)
	currentDir, _ := os.Getwd()
	project := "rosa"

	Test.TestProfile = helper.ReadENVWithDefaultValue("TEST_PROFILE", "")
	Test.RootDir = helper.ReadENVWithDefaultValue("WORKSPACE", strings.SplitAfter(currentDir, project)[0])
	Test.YAMLProfilesDir = helper.ReadENVWithDefaultValue("TEST_PROFILE_DIR",
		path.Join(Test.RootDir, "tests", "ci", "data", "profiles"))
	Test.OutputDir = helper.ReadENVWithDefaultValue("SHARED_DIR",
		path.Join(Test.RootDir, "tests", "output", Test.TestProfile))
	Test.ResourcesDir = helper.ReadENVWithDefaultValue("RESOURCES_DIR",
		path.Join(Test.RootDir, "tests", "ci", "data", "resources"))
	Test.ArtifactDir = helper.ReadENVWithDefaultValue("ARTIFACT_DIR", Test.OutputDir)
	err := os.MkdirAll(Test.OutputDir, 0777)
	if err != nil {
		Logger.Errorf("Meet error %s when create output dirs", err.Error())
	}
	Test.ClusterConfigFile = path.Join(Test.OutputDir, "cluster-config")
	Test.UserDataFile = path.Join(Test.OutputDir, "resources.json")
	Test.KonfluxUserDataFile = path.Join(Test.OutputDir, "konflux_resources.json")
	Test.APIURLFile = path.Join(Test.OutputDir, "api.url")

	// Temporary files to compatible to current CI jobs. Will remove once all CI jobs migration finished
	Test.ClusterIDFile = path.Join(Test.OutputDir, "cluster-id")
	Test.ClusterNameFile = path.Join(Test.OutputDir, "cluster-name")
	Test.ClusterTypeFile = path.Join(Test.OutputDir, "cluster-type")
	Test.ConsoleUrlFile = path.Join(Test.OutputDir, "console.url")
	Test.InfraIDFile = path.Join(Test.OutputDir, "infra_id")
	Test.VPCIDFile = path.Join(Test.OutputDir, "vpc_id")
	Test.PublicSubnetsFile = path.Join(Test.OutputDir, "public_subnet_ids")
	// End of temporary

	Test.CreateCommandFile = path.Join(Test.OutputDir, "create_cluster.sh")
	Test.ClusterDetailFile = path.Join(Test.OutputDir, "cluster-detail.json")
	Test.ClusterInstallLogArtifactFile = path.Join(Test.ArtifactDir, ".install.log")
	Test.ClusterAdminFile = path.Join(Test.ArtifactDir, ".admin")
	Test.ClusterIDPAdminUsernamePassword = path.Join(Test.ArtifactDir, ".idp-admin-username-password")
	Test.TestFocusFile = path.Join(Test.RootDir, "tests", "ci", "data", "commit-focus")
	Test.TestLabelFilterFile = path.Join(Test.RootDir, "tests", "ci", "data", "label-filter")
	Test.ProxySSHPemFile = "ocm-test-proxy"
	Test.ProxyCABundleFile = path.Join(Test.OutputDir, "proxy-bundle.ca")

	waitingTime, err := strconv.Atoi(helper.ReadENVWithDefaultValue("CLUSTER_TIMEOUT", "60"))
	if err != nil {
		panic(fmt.Errorf("env variable CLUSTER_TIMEOUT must be set to an integer"))
	}
	waitSetupClusterReady, _ := strconv.ParseBool(helper.ReadENVWithDefaultValue("WAIT_SETUP_CLUSTER_READY", "true"))
	useLocalCredentials, _ := strconv.ParseBool(helper.ReadENVWithDefaultValue("USE_LOCAL_CREDENTIALS", "false"))
	Test.GlobalENV = &GlobalENVVariables{
		ChannelGroup:          os.Getenv("CHANNEL_GROUP"),
		Version:               os.Getenv("VERSION"),
		OpenshiftVersion:      os.Getenv("OPENSHIFT_VERSION"),
		Region:                os.Getenv("REGION"),
		ProvisionShard:        os.Getenv("PROVISION_SHARD"),
		NamePrefix:            os.Getenv("NAME_PREFIX"),
		SVPC_CREDENTIALS_FILE: os.Getenv("SHARED_VPC_AWS_SHARED_CREDENTIALS_FILE"),
		AWSCredetialsFile:     os.Getenv("AWS_SHARED_CREDENTIALS_FILE"),
		OCM_LOGIN_ENV:         os.Getenv("OCM_LOGIN_ENV"),
		ClusterWaitingTime:    waitingTime,
		WaitSetupClusterReady: waitSetupClusterReady,
	}
	Test.ClusterENV = &ClusterENVVariables{
		ComputeMachineType:  os.Getenv("COMPUTE_MACHINE_TYPE"),
		BYOVPC:              os.Getenv("BYOVPC"),
		Private:             os.Getenv("PRIVATE"),
		Autoscale:           os.Getenv("AUTOSCALE"),
		ProxyEnabled:        os.Getenv("PROXY_ENABLED"),
		FipsEnabled:         os.Getenv("FIPS_ENABLED"),
		VolumeSize:          os.Getenv("VOLUME_SIZE"),
		Replicas:            os.Getenv("REPLICAS"),
		MultiAZ:             os.Getenv("MULTI_AZ"),
		AllowRegistries:     os.Getenv("ALLOW_REGISTRIES"),
		Add_UnManaged_Tag:   os.Getenv("ADD_UNMANAGED_TAG"),
		UseLocalCredentials: useLocalCredentials,
	}

	local_zone_mp, _ := strconv.ParseBool(helper.ReadENVWithDefaultValue("LOCAL_ZONE_MP", "false"))
	tuning_config, _ := strconv.ParseBool(helper.ReadENVWithDefaultValue("TUNING_CONFIG", "false"))
	tuning_config_mp, _ := strconv.ParseBool(helper.ReadENVWithDefaultValue("TUNING_CONFIG_MP", "false"))
	cluster_autoscaler, _ := strconv.ParseBool(helper.ReadENVWithDefaultValue("CLUSTER_AUTOSCALER", "false"))
	Test.Day2ConfENV = &Day2ConfENVVariables{
		LocalZoneMP:       local_zone_mp,
		TuningConfig:      tuning_config,
		TuningConfigMP:    tuning_config_mp,
		ClusterAutoScaler: cluster_autoscaler,
	}

}
