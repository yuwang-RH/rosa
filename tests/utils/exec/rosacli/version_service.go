package rosacli

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/Masterminds/semver"

	"github.com/openshift/rosa/tests/utils/log"
)

const VersionChannelGroupStable = "stable"
const VersionChannelGroupNightly = "nightly"
const VersionChannelGroupCandidate = "candidate"

type VersionService interface {
	ResourcesCleaner

	ReflectVersions(result bytes.Buffer) (*OpenShiftVersionTableList, error)
	ListVersions(channelGroup string, hostedCP bool, flags ...string) (bytes.Buffer, error)
	ListAndReflectVersions(channelGroup string, hostedCP bool, flags ...string) (*OpenShiftVersionTableList, error)
}

type versionService struct {
	ResourcesService
}

func NewVersionService(client *Client) VersionService {
	return &versionService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

type OpenShiftVersionTableOutput struct {
	Version           string `json:"VERSION,omitempty"`
	Default           string `json:"DEFAULT,omitempty"`
	AvailableUpgrades string `json:"AVAILABLE UPGRADES,omitempty"`
}

type OpenShiftVersionJsonOutput struct {
	ID                string   `json:"VERSION,omitempty"`
	RAWID             string   `json:"raw_id,omitempty"`
	ChannelGroup      string   `json:"channel_group,omitempty"`
	HCPDefault        bool     `json:"hosted_control_plane_default,omitempty"`
	HCPEnabled        bool     `json:"hosted_control_plane_enabled,omitempty"`
	Default           bool     `json:"default,omitempty"`
	AvailableUpgrades []string `json:"available_upgrades,omitempty"`
}

type OpenShiftVersionTableList struct {
	OpenShiftVersions []*OpenShiftVersionTableOutput `json:"OpenShiftVersions,omitempty"`
}

// Reflect versions
func (v *versionService) ReflectVersions(result bytes.Buffer) (versionList *OpenShiftVersionTableList, err error) {
	versionList = &OpenShiftVersionTableList{}
	theMap := v.client.Parser.TableData.Input(result).Parse().Output()
	for _, item := range theMap {
		version := &OpenShiftVersionTableOutput{}
		err = MapStructure(item, version)
		if err != nil {
			return versionList, err
		}
		versionList.OpenShiftVersions = append(versionList.OpenShiftVersions, version)
	}
	return versionList, err
}

// Reflect versions json
func (v *versionService) ReflectJsonVersions(result bytes.Buffer) (versionList []*OpenShiftVersionJsonOutput, err error) {
	versionList = []*OpenShiftVersionJsonOutput{}
	parser := v.client.Parser.JsonData.Input(result).Parse()
	theMap := parser.Output().([]interface{})
	for index, _ := range theMap {
		version := &OpenShiftVersionJsonOutput{}
		vDetail := parser.DigObject(index).(map[string]interface{})
		err := MapStructure(vDetail, version)
		if err != nil {
			return versionList, err
		}
		versionList = append(versionList, version)
	}
	return versionList, err
}

// list version `rosa list version` or `rosa list version --hosted-cp`
func (v *versionService) ListVersions(channelGroup string, hostedCP bool, flags ...string) (bytes.Buffer, error) {
	listVersion := v.client.Runner.
		Cmd("list", "versions").
		CmdFlags(flags...)

	if hostedCP {
		listVersion.AddCmdFlags("--hosted-cp")
	}

	if channelGroup != "" {
		listVersion.AddCmdFlags("--channel-group", channelGroup)
	}

	return listVersion.Run()
}

func (v *versionService) ListAndReflectVersions(channelGroup string, hostedCP bool, flags ...string) (versionList *OpenShiftVersionTableList, err error) {
	var output bytes.Buffer
	output, err = v.ListVersions(channelGroup, hostedCP, flags...)
	if err != nil {
		return versionList, err
	}

	versionList, err = v.ReflectVersions(output)
	return versionList, err
}
func (v *versionService) ListAndReflectJsonVersions(channelGroup string, hostedCP bool, flags ...string) (versionList *OpenShiftVersionTableList, err error) {
	var output bytes.Buffer
	output, err = v.ListVersions(channelGroup, hostedCP, flags...)
	if err != nil {
		return versionList, err
	}

	versionList, err = v.ReflectVersions(output)
	return versionList, err
}

func (v *versionService) CleanResources(clusterID string) (errors []error) {
	log.Logger.Debugf("Nothing to clean in Version Service")
	return
}

// This function will find the nearest lower OCP version which version is under `Major.{minor-sub}`.
// `strict` will find only the `Major.{minor-sub}` ones
func (vl *OpenShiftVersionTableList) FindNearestBackwardMinorVersion(version string, minorSub int64, strict bool) (vs *OpenShiftVersionTableOutput, err error) {
	var baseVersionSemVer *semver.Version
	baseVersionSemVer, err = semver.NewVersion(version)
	if err != nil {
		return
	}
	nvl, err := vl.FilterVersionsSameMajorAndEqualOrLowerThanMinor(baseVersionSemVer.Major(), baseVersionSemVer.Minor()-minorSub, strict)
	if err != nil {
		return
	}
	if nvl, err = nvl.Sort(true); err == nil && nvl.Len() > 0 {
		vs = nvl.OpenShiftVersions[0]
	}
	return

}

// This function will find the nearest lower OCP version which version is under `Major.minor.{optional-sub}`.
// `strict` will find only the `Major.monior,{optional-sub}` ones
func (vl *OpenShiftVersionTableList) FindNearestBackwardOptionalVersion(version string, optionalsub int, strict bool) (vs *OpenShiftVersionTableOutput, err error) {
	var baseVersionSemVer *semver.Version
	log.Logger.Debugf("Filter versions according to %s", version)
	baseVersionSemVer, err = semver.NewVersion(version)
	if err != nil {
		return
	}
	nvl, err := vl.FilterVersionsSameMajorAndEqualOrLowerThanMinor(baseVersionSemVer.Major(), baseVersionSemVer.Minor(), strict)
	if err != nil {
		return
	}
	if nvl, err = nvl.Sort(true); err == nil && nvl.Len() >= optionalsub {
		vs = nvl.OpenShiftVersions[optionalsub]
	}
	return

}

// Sort sort the version list from lower to higher (or reverse)
func (vl *OpenShiftVersionTableList) Sort(reverse bool) (nvl *OpenShiftVersionTableList, err error) {
	versionListIndexMap := make(map[string]*OpenShiftVersionTableOutput)
	var semVerList []*semver.Version
	var vSemVer *semver.Version
	for _, version := range vl.OpenShiftVersions {
		versionListIndexMap[version.Version] = version
		if vSemVer, err = semver.NewVersion(version.Version); err != nil {
			return
		} else {
			semVerList = append(semVerList, vSemVer)
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(semver.Collection(semVerList)))
	} else {
		sort.Sort(semver.Collection(semVerList))
	}

	var sortedImageVersionList []*OpenShiftVersionTableOutput
	for _, semverVersion := range semVerList {
		sortedImageVersionList = append(sortedImageVersionList, versionListIndexMap[semverVersion.Original()])
	}

	nvl = &OpenShiftVersionTableList{
		OpenShiftVersions: sortedImageVersionList,
	}

	return
}

// FilterVersionsByMajorMinor filter the version list for all major/minor corresponding and returns a new `OpenShiftVersionList` struct
// `strict` will find only the `Major.minor` ones
func (vl *OpenShiftVersionTableList) FilterVersionsSameMajorAndEqualOrLowerThanMinor(major int64, minor int64, strict bool) (nvl *OpenShiftVersionTableList, err error) {
	var filteredVersions []*OpenShiftVersionTableOutput
	var semverVersion *semver.Version
	for _, version := range vl.OpenShiftVersions {
		if semverVersion, err = semver.NewVersion(version.Version); err != nil {
			return
		} else if semverVersion.Major() == major &&
			((strict && semverVersion.Minor() == minor) || (!strict && semverVersion.Minor() <= minor)) {
			filteredVersions = append(filteredVersions, version)
		}
	}

	nvl = &OpenShiftVersionTableList{
		OpenShiftVersions: filteredVersions,
	}

	return
}

// FilterVersionsByMajorMinor filter the version list for all lower versions than the given one
func (vl *OpenShiftVersionTableList) FilterVersionsLowerThan(version string) (nvl *OpenShiftVersionTableList, err error) {
	var givenSemVer *semver.Version
	givenSemVer, err = semver.NewVersion(version)

	var filteredVersions []*OpenShiftVersionTableOutput
	var semverVersion *semver.Version
	for _, version := range vl.OpenShiftVersions {
		if semverVersion, err = semver.NewVersion(version.Version); err != nil {
			return
		} else if semverVersion.LessThan(givenSemVer) {
			filteredVersions = append(filteredVersions, version)
		}
	}

	nvl = &OpenShiftVersionTableList{
		OpenShiftVersions: filteredVersions,
	}

	return
}

func (vl *OpenShiftVersionTableList) DefaultVersion() (defaultVersion *OpenShiftVersionTableOutput) {
	for _, version := range vl.OpenShiftVersions {
		if version.Default == "yes" {
			return version
		}
	}
	return vl.OpenShiftVersions[0]
}

func (vl *OpenShiftVersionTableOutput) MajorMinor() (major int64, minor int64, majorMinorVersion string, err error) {
	var semverVersion *semver.Version
	if semverVersion, err = semver.NewVersion(vl.Version); err != nil {
		return
	}
	major = semverVersion.Major()
	minor = semverVersion.Minor()
	majorMinorVersion = fmt.Sprintf("%d.%d", major, minor)
	return
}

func (vl *OpenShiftVersionTableList) Len() int {
	return len(vl.OpenShiftVersions)
}

func (vl *OpenShiftVersionTableList) Latest() (*OpenShiftVersionTableOutput, error) {
	vl, err := vl.Sort(true)
	if err != nil {
		return nil, err
	}
	return vl.OpenShiftVersions[0], err

}
