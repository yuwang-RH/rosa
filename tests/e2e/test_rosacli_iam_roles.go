package e2e

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Edit IAM",
	labels.Day2,
	labels.FeatureRoles,
	func() {
		defer GinkgoRecover()

		var (
			accountRolePrefixesNeedCleanup  = make([]string, 0)
			operatorRolePrefixedNeedCleanup = make([]string, 0)

			clusterID              string
			rosaClient             *rosacli.Client
			clusterService         rosacli.ClusterService
			ocmResourceService     rosacli.OCMResourceService
			permissionsBoundaryArn string = "arn:aws:iam::aws:policy/AdministratorAccess"
		)
		BeforeEach(func() {
			By("Get the cluster id")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Parse cluster profile")
			var err error
			Expect(err).ToNot(HaveOccurred())

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			ocmResourceService = rosaClient.OCMResource
		})

		AfterEach(func() {
			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("can create/list/delete account-roles - [id:43070]",
			labels.High,
			func() {
				defer func() {
					By("Cleanup created account-roles in high level of the test case")
					if len(accountRolePrefixesNeedCleanup) > 0 {
						for _, v := range accountRolePrefixesNeedCleanup {
							_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
								"--prefix", v,
								"-y")

							Expect(err).To(BeNil())
						}
					}
				}()

				var (
					userRolePrefixB = "prefixB"
					userRolePrefixH = "prefixH"
					userRolePrefixC = "prefixC"
					path            = "/fd/sd/"
					versionH        string
					versionC        string
				)

				By("Get the testing version")
				versionService := rosaClient.Version
				versionListC, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, false)
				Expect(err).To(BeNil())
				defaultVersionC := versionListC.DefaultVersion()
				Expect(defaultVersionC).ToNot(BeNil())
				_, _, versionC, err = defaultVersionC.MajorMinor()
				Expect(err).To(BeNil())

				versionListH, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, true)
				Expect(err).To(BeNil())
				defaultVersionH := versionListH.DefaultVersion()
				Expect(defaultVersionH).ToNot(BeNil())
				_, _, versionH, err = defaultVersionH.MajorMinor()
				Expect(err).To(BeNil())

				By("Create boundary policy")
				rosaClient.Runner.JsonFormat()

				whoamiOutput, err := ocmResourceService.Whoami()
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
				AWSAccountID := whoamiData.AWSAccountID

				By("Create advanced account-roles of both hosted-cp and classic")
				output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", userRolePrefixB,
					"--path", path,
					"--permissions-boundary", permissionsBoundaryArn,
					"-y")
				Expect(err).To(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, userRolePrefixB)
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Creating classic account roles"))
				Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
				Expect(textData).To(ContainSubstring("Created role"))

				By("Create advance account-roles of only hosted-cp")
				output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", userRolePrefixH,
					"--path", path,
					"--permissions-boundary", permissionsBoundaryArn,
					"--version", versionH,
					"--hosted-cp",
					"-y")
				Expect(err).To(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, userRolePrefixH)
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).ToNot(ContainSubstring("Creating classic account roles"))
				Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
				Expect(textData).To(ContainSubstring("Created role"))

				By("Create advance account-roles of only classic")
				output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", userRolePrefixC,
					"--path", path,
					"--permissions-boundary", permissionsBoundaryArn,
					"--version", versionC,
					"--classic",
					"-y")
				Expect(err).To(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, userRolePrefixC)
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Creating classic account roles"))
				Expect(textData).ToNot(ContainSubstring("Creating hosted CP account roles"))
				Expect(textData).To(ContainSubstring("Created role"))

				By("List account-roles and check the result are expected")
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())

				accountRoleSetB := accountRoleList.AccountRoles(userRolePrefixB)
				accountRoleSetH := accountRoleList.AccountRoles(userRolePrefixH)
				accountRoleSetC := accountRoleList.AccountRoles(userRolePrefixC)

				selectedRoleH := accountRoleSetH[rand.Intn(len(accountRoleSetH))]
				selectedRoleC := accountRoleSetC[rand.Intn(len(accountRoleSetC))]

				Expect(len(accountRoleSetB)).To(Equal(7))
				Expect(len(accountRoleSetH)).To(Equal(3))
				Expect(len(accountRoleSetC)).To(Equal(4))

				Expect(selectedRoleH.RoleArn).To(Equal(fmt.Sprintf("arn:aws:iam::%s:role%s%s-HCP-ROSA-%s", AWSAccountID, path, userRolePrefixH, rosacli.RoleTypeSuffixMap[selectedRoleH.RoleType])))
				Expect(selectedRoleH.OpenshiftVersion).To(Equal(versionH))
				Expect(selectedRoleH.AWSManaged).To(Equal("Yes"))
				Expect(selectedRoleC.RoleArn).To(Equal(fmt.Sprintf("arn:aws:iam::%s:role%s%s-%s", AWSAccountID, path, userRolePrefixC, rosacli.RoleTypeSuffixMap[selectedRoleC.RoleType])))
				Expect(selectedRoleC.OpenshiftVersion).To(Equal(versionC))
				Expect(selectedRoleC.AWSManaged).To(Equal("No"))

				By("Delete account-roles")
				output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", userRolePrefixB,
					"-y")

				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the classic account roles"))
				Expect(textData).To(ContainSubstring("Successfully deleted the hosted CP account roles"))

				output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", userRolePrefixH,
					"--hosted-cp",
					"-y",
				)

				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the hosted CP account roles"))

				output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", userRolePrefixC,
					"--classic",
					"-y",
				)

				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the classic account roles"))

				By("List account-roles to check they are deleted")
				accountRoleList, _, err = ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())

				accountRoleSetB = accountRoleList.AccountRoles(userRolePrefixB)
				accountRoleSetH = accountRoleList.AccountRoles(userRolePrefixH)
				accountRoleSetC = accountRoleList.AccountRoles(userRolePrefixC)

				Expect(len(accountRoleSetB)).To(Equal(0))
				Expect(len(accountRoleSetH)).To(Equal(0))
				Expect(len(accountRoleSetC)).To(Equal(0))
			})

		It("can create operator-roles prior to cluster creation - [id:60971]",
			labels.High,
			func() {
				defer func() {
					By("Cleanup created operator-roles in high level of the test case")
					if len(operatorRolePrefixedNeedCleanup) > 0 {
						for _, v := range operatorRolePrefixedNeedCleanup {
							_, err := ocmResourceService.DeleteOperatorRoles(
								"--prefix", v,
								"--mode", "auto",
								"-y",
							)
							Expect(err).To(BeNil())
						}
					}
				}()

				var (
					oidcPrivodeIDFromOutputMessage  string
					oidcPrivodeARNFromOutputMessage string
					notExistedOIDCConfigID          = "asdasdfsdfsdf"
					invalidInstallerRole            = "arn:/qeci-default-accountroles-Installer-Role"
					notExistedInstallerRole         = "arn:aws:iam::301721915996:role/notexisted-accountroles-Installer-Role"
					hostedCPOperatorRolesPrefix     = "hopp60971"
					classicSTSOperatorRolesPrefix   = "sopp60971"
					managedOIDCConfigID             string
					hostedCPInstallerRoleArn        string
					classicInstallerRoleArn         string
					accountRolePrefix               string
				)

				listOperatorRoles := func(prefix string) (rosacli.OperatorRoleList, error) {
					var operatorRoleList rosacli.OperatorRoleList
					output, err := ocmResourceService.ListOperatorRoles(
						"--prefix", prefix,
					)
					if err != nil {
						return operatorRoleList, err
					}
					operatorRoleList, err = ocmResourceService.ReflectOperatorRoleList(output)
					return operatorRoleList, err
				}

				By("Create account-roles for testing")
				accountRolePrefix = fmt.Sprintf("QEAuto-accr60971-%s", time.Now().UTC().Format("20060102"))
				_, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accountRolePrefix,
					"--permissions-boundary", permissionsBoundaryArn,
					"-y")
				Expect(err).To(BeNil())

				defer func() {
					By("Cleanup created account-roles")
					_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
						"--prefix", accountRolePrefix,
						"-y")
					Expect(err).To(BeNil())
				}()

				By("Get the installer role arn")
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				classicInstallerRoleArn = accountRoleList.InstallerRole(accountRolePrefix, false).RoleArn
				hostedCPInstallerRoleArn = accountRoleList.InstallerRole(accountRolePrefix, true).RoleArn

				By("Create managed oidc-config in auto mode")
				output, err := ocmResourceService.CreateOIDCConfig("--mode", "auto", "-y")
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Created OIDC provider with ARN"))
				oidcPrivodeARNFromOutputMessage = common.ExtractOIDCProviderARN(output.String())
				oidcPrivodeIDFromOutputMessage = common.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)

				managedOIDCConfigID, err = ocmResourceService.GetOIDCIdFromList(oidcPrivodeIDFromOutputMessage)
				Expect(err).To(BeNil())
				defer func() {
					output, err := ocmResourceService.DeleteOIDCConfig(
						"--oidc-config-id", managedOIDCConfigID,
						"--mode", "auto",
						"-y",
					)
					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("Successfully deleted the OIDC provider"))
				}()
				By("Create hosted-cp and classic sts Operator-roles pror to cluster spec")
				output, err = ocmResourceService.CreateOperatorRoles(
					"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
					"--installer-role-arn", classicInstallerRoleArn,
					"--mode", "auto",
					"--prefix", classicSTSOperatorRolesPrefix,
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Created role"))
				operatorRolePrefixedNeedCleanup = append(operatorRolePrefixedNeedCleanup, classicSTSOperatorRolesPrefix)

				defer func() {
					output, err := ocmResourceService.DeleteOperatorRoles(
						"--prefix", classicSTSOperatorRolesPrefix,
						"--mode", "auto",
						"-y",
					)
					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("Successfully deleted the operator roles"))

				}()

				roles, err := listOperatorRoles(classicSTSOperatorRolesPrefix)
				Expect(err).To(BeNil())
				Expect(len(roles.OperatorRoleList)).To(Equal(6))

				output, err = ocmResourceService.CreateOperatorRoles(
					"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
					"--installer-role-arn", hostedCPInstallerRoleArn,
					"--mode", "auto",
					"--prefix", hostedCPOperatorRolesPrefix,
					"--hosted-cp",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Created role"))
				operatorRolePrefixedNeedCleanup = append(operatorRolePrefixedNeedCleanup, hostedCPOperatorRolesPrefix)

				roles, err = listOperatorRoles(hostedCPOperatorRolesPrefix)
				Expect(err).To(BeNil())
				Expect(len(roles.OperatorRoleList)).To(Equal(8))
				defer func() {
					output, err := ocmResourceService.DeleteOperatorRoles(
						"--prefix", hostedCPOperatorRolesPrefix,
						"--mode", "auto",
						"-y",
					)
					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("Successfully deleted the operator roles"))
				}()

				By("Create operator roles with not-existed role")
				output, err = ocmResourceService.CreateOperatorRoles(
					"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
					"--installer-role-arn", notExistedInstallerRole,
					"--mode", "auto",
					"--prefix", classicSTSOperatorRolesPrefix,
					"-y",
				)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("cannot be found"))

				By("Create operator roles with role arn in incorrect format")
				output, err = ocmResourceService.CreateOperatorRoles(
					"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
					"--installer-role-arn", invalidInstallerRole,
					"--mode", "auto",
					"--prefix", classicSTSOperatorRolesPrefix,
					"-y",
				)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Invalid ARN"))

				By("Create operator roles with not-existed oidc id")
				output, err = ocmResourceService.CreateOperatorRoles(
					"--oidc-config-id", notExistedOIDCConfigID,
					"--installer-role-arn", classicInstallerRoleArn,
					"--mode", "auto",
					"--prefix", classicSTSOperatorRolesPrefix,
					"-y",
				)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("not found"))

				By("Create operator-role without setting oidc-config-id")
				output, err = ocmResourceService.CreateOperatorRoles(
					"--installer-role-arn", classicInstallerRoleArn,
					"--mode", "auto",
					"--prefix", hostedCPOperatorRolesPrefix,
					"--hosted-cp",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("oidc-config-id is mandatory for prefix param flow"))

				By("Create operator-role without setting installer-role-arn")
				output, err = ocmResourceService.CreateOperatorRoles(
					"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
					"--mode", "auto",
					"--prefix", hostedCPOperatorRolesPrefix,
					"--hosted-cp",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("role-arn is mandatory for prefix param flow"))

				By("Create operator-role without setting id neither prefix")
				output, err = ocmResourceService.CreateOperatorRoles(
					"--oidc-config-id", oidcPrivodeIDFromOutputMessage,
					"--installer-role-arn", classicInstallerRoleArn,
					"--mode", "auto",
					"--hosted-cp",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Either a cluster key for STS cluster or an operator roles prefix must be specified"))
			})

		It("can validate when user create operator-roles to cluster - [id:43051]",
			labels.High,
			func() {
				By("Check if cluster is sts cluster")
				StsCluster, err := clusterService.IsSTSCluster(clusterID)
				Expect(err).To(BeNil())

				By("Check if cluster is using reusable oidc config")
				notExistedClusterID := "notexistedclusterid111"

				switch StsCluster {
				case true:
					By("Create operator-roles on sts cluster which status is not pending")
					output, err := ocmResourceService.CreateOperatorRoles(
						"--mode", "auto",
						"-c", clusterID,
						"-y")
					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("Operator Roles already exists"))
				case false:
					By("Create operator-roles on classic non-sts cluster")
					output, err := ocmResourceService.CreateOIDCProvider(
						"--mode", "auto",
						"-c", clusterID,
						"-y")
					Expect(err).NotTo(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).To(ContainSubstring("is not an STS cluster"))
				}
				By("Create operator-roles on not-existed cluster")
				output, err := ocmResourceService.CreateOIDCProvider(
					"--mode", "auto",
					"-c", notExistedClusterID,
					"-y")
				Expect(err).NotTo(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("There is no cluster with identifier or name"))
			})

		It("can validate that upgrade account-roles with the managed policies should be forbidden - [id:57441]",
			labels.High,
			func() {
				defer func() {
					By("Cleanup created account-roles in high level of the test case")
					if len(accountRolePrefixesNeedCleanup) > 0 {
						for _, v := range accountRolePrefixesNeedCleanup {
							_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
								"--prefix", v,
								"-y")
							Expect(err).To(BeNil())
						}
					}
				}()
				var (
					accrolePrefix = "accrolep57441"
					path          = "/aa/vv/"
					modes         = []string{"auto", "manual"}
				)

				By("Create hosted-cp account-roles")
				output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accrolePrefix,
					"--path", path,
					"--hosted-cp",
					"-y")
				Expect(err).To(BeNil())
				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, accrolePrefix)
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
				Expect(textData).To(ContainSubstring("Created role"))

				By("Upgrade managed account-roles")
				for _, mode := range modes {
					output, err := ocmResourceService.UpgradeAccountRole(
						"--prefix", accrolePrefix,
						"--hosted-cp",
						"--mode", mode,
						"-y",
					)
					Expect(err).To(BeNil())
					Expect(output.String()).To(ContainSubstring("have attached managed policies. An upgrade isn't needed"))
				}

				By("Delete account-roles")
				output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", accrolePrefix,
					"--hosted-cp",
					"-y")

				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the hosted CP account roles"))

				By("List account-roles to check they are deleted")
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				Expect(len(accountRoleList.AccountRoles(accrolePrefix))).To(Equal(0))
			})

		It("can delete account-roles with --hosted-cp and --classic - [id:62083]",
			labels.High,
			func() {
				defer func() {
					By("Cleanup created account-roles in high level of the test case")
					if len(accountRolePrefixesNeedCleanup) > 0 {
						for _, v := range accountRolePrefixesNeedCleanup {
							_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
								"--prefix", v,
								"-y")

							Expect(err).To(BeNil())
						}
					}
				}()

				var accrolePrefix = "accrolep62083"

				By("Create advanced account-roles of both hosted-cp and classic")
				output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accrolePrefix,
					"-y")
				Expect(err).To(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, accrolePrefix)
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Creating classic account roles"))
				Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
				Expect(textData).To(ContainSubstring("Created role"))

				By("Delete account-roles with --classic flag")
				output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", accrolePrefix,
					"--classic",
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the classic account roles"))

				By("Delete account-roles with --hosted-cp flag")
				output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", accrolePrefix,
					"--hosted-cp",
					"-y",
				)
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the hosted CP account roles"))

				By("List account-roles to check they are deleted")
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				Expect(len(accountRoleList.AccountRoles(accrolePrefix))).To(Equal(0))
			})

		It("can validate create/link/unlink user-role - [id:52580]",
			labels.High,
			func() {
				var (
					userRolePrefix                                string
					invalidPermisionBoundary                      string
					notExistedPermissionBoundaryUnderDifferentAWS string
					ocmAccountUsername                            string
					notExistedUserRoleArn                         string
					userRoleArnInWrongFormat                      string
					foundUserRole                                 rosacli.UserRole
				)
				rosaClient.Runner.JsonFormat()
				whoamiOutput, err := ocmResourceService.Whoami()
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
				ocmAccountUsername = whoamiData.OCMAccountUsername
				rand.Seed(time.Now().UnixNano())
				userRolePrefix = fmt.Sprintf("QEAuto-user-%s-OCP-52580", time.Now().UTC().Format("20060102"))

				By("Create an user-role with invalid mode")
				output, err := ocmResourceService.CreateUserRole("--mode", "invalidamode",
					"--prefix", userRolePrefix,
					"-y")
				Expect(err).NotTo(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Invalid mode. Allowed values are [auto manual]"))

				By("Create an user-role with invalid permision boundady")
				invalidPermisionBoundary = "arn-permission-boundary"
				output, err = ocmResourceService.CreateUserRole("--mode", "auto",
					"--permissions-boundary", invalidPermisionBoundary,
					"--prefix", userRolePrefix,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Expected a valid policy ARN for permissions boundary"))

				By("Create an user-role with the permision boundady under another aws account")
				notExistedPermissionBoundaryUnderDifferentAWS = "arn:aws:iam::aws:policy/notexisted"
				output, err = ocmResourceService.CreateUserRole("--mode", "auto",
					"--permissions-boundary", notExistedPermissionBoundaryUnderDifferentAWS,
					"--prefix", userRolePrefix,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("There was an error creating the ocm user role: operation error IAM: CreateRole"))
				Expect(textData).Should(ContainSubstring("api error NoSuchEntity"))

				By("Create an user-role")
				output, err = ocmResourceService.CreateUserRole("--mode", "auto",
					"--prefix", userRolePrefix,
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Created role"))
				Expect(textData).Should(ContainSubstring("Successfully linked role"))

				By("Get the user-role info")
				userRoleList, output, err := ocmResourceService.ListUserRole()
				Expect(err).To(BeNil())
				foundUserRole = userRoleList.UserRole(userRolePrefix, ocmAccountUsername)
				Expect(foundUserRole).ToNot(BeNil())

				defer func() {
					By("Delete user-role")
					output, err = ocmResourceService.DeleteUserRole("--mode", "auto",
						"--role-arn", foundUserRole.RoleArn,
						"-y")

					Expect(err).To(BeNil())
					textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).Should(ContainSubstring("Successfully deleted the user role"))
				}()

				By("Unlink user-role with not-exist role")
				notExistedUserRoleArn = "arn:aws:iam::301721915996:role/notexistuserrolearn"
				output, err = ocmResourceService.UnlinkUserRole("--role-arn", notExistedUserRoleArn, "-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("is not linked with the current account"))

				By("Unlink user-role with the role arn in incorrect format")
				userRoleArnInWrongFormat = "arn301721915996:rolenotexistuserrolearn"
				output, err = ocmResourceService.UnlinkUserRole("--role-arn", userRoleArnInWrongFormat, "-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Expected a valid user role ARN to unlink from the current account"))

				By("Unlink user-role")
				output, err = ocmResourceService.UnlinkUserRole("--role-arn", foundUserRole.RoleArn, "-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Successfully unlinked role"))

				By("Get the user-role info")
				userRoleList, output, err = ocmResourceService.ListUserRole()
				Expect(err).To(BeNil())
				foundUserRole = userRoleList.UserRole(userRolePrefix, ocmAccountUsername)
				Expect(foundUserRole.Linded).To(Equal("No"))

				By("Link user-role with the role arn in incorrect format")
				output, err = ocmResourceService.LinkUserRole("--role-arn", userRoleArnInWrongFormat, "-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Expected a valid user role ARN to link to a current account"))
			})

		It("can create/delete/unlink/link ocm-roles in auto mode - [id:46187]",
			labels.High,
			func() {
				var (
					ocmrolePrefix                                 string
					invalidPermisionBoundary                      string
					notExistedPermissionBoundaryUnderDifferentAWS string
					ocmOrganizationExternalID                     string
					notExistedOcmroleocmRoleArn                   string
					ocmroleArnInWrongFormat                       string
					foundOcmrole                                  rosacli.OCMRole
					path                                          = "/aa/bb/"
					ocmRoleList                                   rosacli.OCMRoleList
					ocmRoleNeedRecoved                            rosacli.OCMRole
				)
				rosaClient.Runner.JsonFormat()
				whoamiOutput, err := ocmResourceService.Whoami()
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
				ocmOrganizationExternalID = whoamiData.OCMOrganizationExternalID
				rand.Seed(time.Now().UnixNano())
				ocmrolePrefix = fmt.Sprintf("QEAuto-ocmr-%s-46187", time.Now().UTC().Format("20060102"))

				By("Check linked ocm role then unlink it")
				ocmRoleList, _, err = ocmResourceService.ListOCMRole()
				ocmRoleNeedRecoved = ocmRoleList.FindLinkedOCMRole()
				Expect(err).To(BeNil())
				if (ocmRoleNeedRecoved != rosacli.OCMRole{}) {
					output, err := ocmResourceService.UnlinkOCMRole("--role-arn", ocmRoleNeedRecoved.RoleArn, "-y")
					Expect(err).To(BeNil())
					Expect(output.String()).Should(ContainSubstring("Successfully unlinked role"))
					defer func() {
						By("Link the ocm-role to recover the original status")
						if (ocmRoleNeedRecoved != rosacli.OCMRole{}) {
							output, err := ocmResourceService.LinkOCMRole("--role-arn", ocmRoleNeedRecoved.RoleArn, "-y")
							Expect(err).To(BeNil())
							Expect(output.String()).Should(ContainSubstring("Successfully linked role"))
						}
					}()
				}
				defer func() {
					By("Delete ocm-role")
					ocmRoleList, _, err := ocmResourceService.ListOCMRole()
					Expect(err).To(BeNil())
					foundOcmrole := ocmRoleList.OCMRole(ocmrolePrefix, ocmOrganizationExternalID)
					output, err := ocmResourceService.DeleteOCMRole("--mode", "auto",
						"--role-arn", foundOcmrole.RoleArn,
						"-y")

					Expect(err).To(BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).Should(ContainSubstring("Successfully deleted the OCM role"))
				}()

				By("Create an ocm-role with invalid mode")
				output, err := ocmResourceService.CreateOCMRole("--mode", "invalidamode",
					"--prefix", ocmrolePrefix,
					"-y")
				Expect(err).NotTo(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Invalid mode. Allowed values are [auto manual]"))

				By("Create an ocm-role with invalid permision boundady")
				invalidPermisionBoundary = "arn-permission-boundary"
				output, err = ocmResourceService.CreateOCMRole("--mode", "auto",
					"--permissions-boundary", invalidPermisionBoundary,
					"--prefix", ocmrolePrefix,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Expected a valid policy ARN for permissions boundary"))

				By("Create ocm-role with the permision boundady under another aws account")
				notExistedPermissionBoundaryUnderDifferentAWS = "arn:aws:iam::aws:policy/notexisted"
				output, err = ocmResourceService.CreateOCMRole("--mode", "auto",
					"--permissions-boundary", notExistedPermissionBoundaryUnderDifferentAWS,
					"--prefix", ocmrolePrefix,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("There was an error creating the ocm role"))
				Expect(textData).Should(ContainSubstring("NoSuchEntity"))

				By("Create an ocm-role")
				output, err = ocmResourceService.CreateOCMRole("--mode", "auto",
					"--prefix", ocmrolePrefix,
					"--path", path,
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Created role"))
				Expect(textData).Should(ContainSubstring("Successfully linked role"))

				By("Get the ocm-role info")
				ocmRoleList, output, err = ocmResourceService.ListOCMRole()
				Expect(output).ToNot(BeNil())
				Expect(err).To(BeNil())
				foundOcmrole = ocmRoleList.OCMRole(ocmrolePrefix, ocmOrganizationExternalID)
				Expect(foundOcmrole).ToNot(BeNil())

				By("Unlink ocm-role with not-exist role")
				notExistedOcmroleocmRoleArn = "arn:aws:iam::301721915996:role/notexistuserrolearn"
				output, err = ocmResourceService.UnlinkOCMRole("--role-arn", notExistedOcmroleocmRoleArn, "-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("is not linked with the organization account"))

				By("Unlink ocm-role with the role arn in incorrect format")
				ocmroleArnInWrongFormat = "arn301721915996:rolenotexistuserrolearn"
				output, err = ocmResourceService.UnlinkOCMRole("--role-arn", ocmroleArnInWrongFormat, "-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Expected a valid ocm role ARN to unlink from the current organization"))

				By("Unlink ocm-role")
				output, err = ocmResourceService.UnlinkOCMRole("--role-arn", foundOcmrole.RoleArn, "-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Successfully unlinked role"))

				By("Get the ocm-role info")
				ocmRoleList, output, err = ocmResourceService.ListOCMRole()
				Expect(output).ToNot(BeNil())
				Expect(err).To(BeNil())
				foundOcmrole = ocmRoleList.OCMRole(ocmrolePrefix, ocmOrganizationExternalID)
				Expect(foundOcmrole.Linded).To(Equal("No"))

				By("Link ocm-role with the role arn in incorrect format")
				output, err = ocmResourceService.LinkOCMRole("--role-arn", ocmroleArnInWrongFormat, "-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Expected a valid ocm role ARN to link to a current organization"))

				By("Delete the role and keep the attached policy then create again")
				awsClient, err := aws_client.CreateAWSClient("", "")
				Expect(err).To(BeNil())
				err = awsClient.DetachRolePolicies(foundOcmrole.RoleName)
				Expect(err).To(BeNil())
				err = awsClient.DeleteRole(foundOcmrole.RoleName)
				Expect(err).To(BeNil())

				output, err = ocmResourceService.CreateOCMRole("--mode", "auto",
					"--prefix", ocmrolePrefix,
					"--path", path,
					"--admin",
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Created role"))
				Expect(textData).Should(ContainSubstring("Successfully linked role"))

				attachedPolicies, err := awsClient.ListAttachedRolePolicies(foundOcmrole.RoleName)
				Expect(err).To(BeNil())
				Expect(len(attachedPolicies)).To(Equal(2))
			})

		It("can create/link/unlink/delete user-role in auto mode - [id:52419]",
			labels.High,
			func() {
				var (
					userrolePrefix     string
					ocmAccountUsername string
					foundUserRole      rosacli.UserRole

					path = "/aa/bb/"
				)

				rosaClient.Runner.JsonFormat()
				whoamiOutput, err := ocmResourceService.Whoami()
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
				ocmAccountUsername = whoamiData.OCMAccountUsername
				userrolePrefix = fmt.Sprintf("QEAuto-userr-%s-52419", time.Now().UTC().Format("20060102"))

				By("Create an user-role")
				output, err := ocmResourceService.CreateUserRole(
					"--mode", "auto",
					"--prefix", userrolePrefix,
					"--path", path,
					"--permissions-boundary", permissionsBoundaryArn,
					"-y")
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Created role"))
				Expect(textData).Should(ContainSubstring("Successfully linked role"))
				defer func() {
					By("Delete user-role")
					output, err = ocmResourceService.DeleteUserRole("--mode", "auto",
						"--role-arn", foundUserRole.RoleArn,
						"-y")

					Expect(err).To(BeNil())
					textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).Should(ContainSubstring("Successfully deleted the user role"))
				}()

				By("Get the ocm-role info")
				userRoleList, output, err := ocmResourceService.ListUserRole()
				Expect(err).To(BeNil())
				foundUserRole = userRoleList.UserRole(userrolePrefix, ocmAccountUsername)
				Expect(foundUserRole).ToNot(BeNil())

				By("Get the user-role info")
				userRoleList, output, err = ocmResourceService.ListUserRole()
				Expect(err).To(BeNil())
				foundUserRole = userRoleList.UserRole(userrolePrefix, ocmAccountUsername)
				Expect(foundUserRole.Linded).To(Equal("Yes"))

				By("Unlink user-role")
				output, err = ocmResourceService.UnlinkUserRole("--role-arn", foundUserRole.RoleArn, "-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Successfully unlinked role"))

				By("Get the user-role info")
				userRoleList, output, err = ocmResourceService.ListUserRole()
				Expect(err).To(BeNil())
				foundUserRole = userRoleList.UserRole(userrolePrefix, ocmAccountUsername)
				Expect(foundUserRole.Linded).To(Equal("No"))
			})

		It("create/delete hypershift account roles with managed policies - [id:61322]",
			labels.Critical,
			func() {
				defer func() {
					By("Cleanup created account-roles in the test case")
					if len(accountRolePrefixesNeedCleanup) > 0 {
						for _, v := range accountRolePrefixesNeedCleanup {
							_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
								"--prefix", v,
								"-y")

							Expect(err).To(BeNil())
						}
					}
				}()

				var (
					rolePrefixStable    = "prefixS"
					rolePrefixCandidate = "prefixC"
					rolePrefixClassic   = "prefixClassic"
					versionStable       string
					versionCandidate    string
					path                = "/fd/sd/"
				)
				By("Prepare verson for testing")
				versionService := rosaClient.Version
				versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, true)
				Expect(err).To(BeNil())
				defaultVersion := versionList.DefaultVersion()
				Expect(defaultVersion).ToNot(BeNil())
				version, err := versionList.FindNearestBackwardMinorVersion(defaultVersion.Version, 1, true)
				Expect(err).To(BeNil())
				Expect(version).NotTo(BeNil())
				_, _, versionStable, err = version.MajorMinor()
				Expect(err).To(BeNil())

				versionList, err = versionService.ListAndReflectVersions(rosacli.VersionChannelGroupCandidate, true)
				Expect(err).To(BeNil())
				defaultVersion = versionList.DefaultVersion()
				Expect(defaultVersion).ToNot(BeNil())
				version, err = versionList.FindNearestBackwardMinorVersion(defaultVersion.Version, 1, true)
				Expect(err).To(BeNil())
				Expect(version).NotTo(BeNil())
				_, _, versionCandidate, err = version.MajorMinor()
				Expect(err).To(BeNil())

				By("Get the AWS Account Id")
				rosaClient.Runner.JsonFormat()
				whoamiOutput, err := ocmResourceService.Whoami()
				Expect(err).To(BeNil())
				rosaClient.Runner.UnsetFormat()
				whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
				AWSAccountID := whoamiData.AWSAccountID

				By("Create account-roles of hosted-cp in stable channel")
				output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", rolePrefixStable,
					"--path", path,
					"--permissions-boundary", permissionsBoundaryArn,
					"--force-policy-creation", "--version", versionStable,
					"--channel-group", "stable",
					"--hosted-cp",
					"-y")
				Expect(err).To(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, rolePrefixStable)
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).ToNot(ContainSubstring("Creating classic account roles"))
				Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
				Expect(textData).To(ContainSubstring("WARN: Setting `version` flag for hosted CP managed policies has no effect, any supported ROSA version can be installed with managed policies"))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Installer-Role'", rolePrefixStable)))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Support-Role'", rolePrefixStable)))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Worker-Role'", rolePrefixStable)))

				By("Create account-roles of hosted-cp in candidate channel")
				output, err = ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", rolePrefixCandidate,
					"--path", path,
					"--permissions-boundary", permissionsBoundaryArn,
					"--version", versionCandidate,
					"--channel-group", "candidate",
					"--hosted-cp",
					"-y")
				Expect(err).To(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, rolePrefixCandidate)
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).ToNot(ContainSubstring("Creating classic account roles"))
				Expect(textData).To(ContainSubstring("Creating hosted CP account roles"))
				Expect(textData).To(ContainSubstring("WARN: Setting `version` flag for hosted CP managed policies has no effect, any supported ROSA version can be installed with managed policies"))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Installer-Role'", rolePrefixCandidate)))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Support-Role'", rolePrefixCandidate)))
				Expect(textData).To(ContainSubstring(fmt.Sprintf("Created role '%s-HCP-ROSA-Worker-Role'", rolePrefixCandidate)))

				By("List the account roles ")
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())

				By("Get the stable/candidate account roles that are created above")
				accountRoleSetStable := accountRoleList.AccountRoles(rolePrefixStable)
				accountRoleSetCandidate := accountRoleList.AccountRoles(rolePrefixCandidate)

				selectedRoleStable := accountRoleSetStable[rand.Intn(len(accountRoleSetStable))]
				selectedRoleCandidate := accountRoleSetCandidate[rand.Intn(len(accountRoleSetCandidate))]

				By("Check 3 roles are created for hosted CP account roles")
				Expect(len(accountRoleSetStable)).To(Equal(3))
				Expect(len(accountRoleSetCandidate)).To(Equal(3))

				By("Check the roles are AWS managed, and path and version flag works correctly")
				Expect(selectedRoleStable.AWSManaged).To(Equal("Yes"))
				Expect(selectedRoleStable.RoleArn).To(Equal(fmt.Sprintf("arn:aws:iam::%s:role%s%s-HCP-ROSA-%s", AWSAccountID, path, rolePrefixStable, rosacli.RoleTypeSuffixMap[selectedRoleStable.RoleType])))
				Expect(selectedRoleStable.OpenshiftVersion).To(Equal(versionStable))
				Expect(selectedRoleCandidate.AWSManaged).To(Equal("Yes"))
				Expect(selectedRoleCandidate.RoleArn).To(Equal(fmt.Sprintf("arn:aws:iam::%s:role%s%s-HCP-ROSA-%s", AWSAccountID, path, rolePrefixCandidate, rosacli.RoleTypeSuffixMap[selectedRoleCandidate.RoleType])))
				Expect(selectedRoleCandidate.OpenshiftVersion).To(Equal(versionCandidate))

				By("Delete the hypershift account roles in auto mode")
				output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", rolePrefixStable,
					"--hosted-cp",
					"-y",
				)

				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted the hosted CP account roles"))

				By("Create a classic account role")
				_, err = ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", rolePrefixClassic,
					"--classic",
					"-y")
				Expect(err).To(BeNil())
				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, rolePrefixClassic)

				By("Try to delete classic account-role with --hosted-cp flag")
				output, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", rolePrefixClassic,
					"--hosted-cp",
					"-y")
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("WARN: There are no hosted CP account roles to be deleted"))
			})

		It("Validation for operator roles and oidc provider will work well - [id:70859]",
			labels.Medium,
			func() {
				By("Check cluster is sts cluster")
				isSTS, err := clusterService.IsSTSCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				By("Check the cluster is using reusable oIDCConfig")
				IsUsingReusableOIDCConfig, err := clusterService.IsUsingReusableOIDCConfig(clusterID)
				Expect(err).ToNot(HaveOccurred())

				if isSTS && IsUsingReusableOIDCConfig {
					By("Create operator roles to the cluster again")
					output, err := ocmResourceService.CreateOperatorRoles("-c", clusterID, "-y", "--mode", "auto")
					Expect(err).ToNot(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).To(ContainSubstring("Operator Roles already exists"))

					By("Create oidc config to the cluster again")
					output, err = ocmResourceService.CreateOIDCProvider("-c", clusterID, "-y", "--mode", "auto")
					Expect(err).ToNot(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).To(ContainSubstring("OIDC provider already exists"))

					By("Delete the oidc-provider to the cluster")
					output, err = ocmResourceService.DeleteOIDCProvider("-c", clusterID, "-y", "--mode", "auto")
					Expect(err).To(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).To(ContainSubstring("ERR: Cluster '%s' is in 'ready' state. OIDC provider can be deleted only for the uninstalled clusters", clusterID))

					By("Delete the operator-roles to the cluster")
					output, err = ocmResourceService.DeleteOperatorRoles("-c", clusterID, "-y", "--mode", "auto")
					Expect(err).To(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).To(ContainSubstring("ERR: Cluster '%s' is in 'ready' state. Operator roles can be deleted only for the uninstalled clusters", clusterID))

					By("Get the --oidc-config-id from the cluster and it's issuer url")
					rosaClient.Runner.JsonFormat()
					jsonOutput, err := clusterService.DescribeCluster(clusterID)
					Expect(err).To(BeNil())
					rosaClient.Runner.UnsetFormat()
					jsonData := rosaClient.Parser.JsonData.Input(jsonOutput).Parse()
					oidcConfigID := jsonData.DigString("aws", "sts", "oidc_config", "id")
					issuerURL := jsonData.DigString("aws", "sts", "oidc_config", "issuer_url")

					By("Try to delete oidc provider with --oidc-config-id")
					output, err = ocmResourceService.DeleteOIDCProvider("--oidc-config-id", oidcConfigID, "-y", "--mode", "auto")
					Expect(err).To(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).To(ContainSubstring("ERR: There are clusters using OIDC config '%s', can't delete the provider", issuerURL))

					By("Try to create oidc provider with --oidc-config-id")
					output, err = ocmResourceService.CreateOIDCProvider("--oidc-config-id", oidcConfigID, "-y", "--mode", "auto")
					Expect(err).ToNot(HaveOccurred())
					Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).To(ContainSubstring("OIDC provider already exists"))
				}
			})

		It("Validation for account-role creation by user - [id:43067]",
			labels.Medium,
			func() {
				var (
					validRolePrefix                          = "valid"
					invalidRolePrefix                        = "^^^^"
					longRolePrefix                           = "accountroleprefixlongerthan32characters"
					validModeAuto                            = "auto"
					validModeManual                          = "manual"
					invalidMode                              = "invalid"
					invalidPermissionsBoundaryArn     string = "invalid"
					nonExistingPermissionsBoundaryArn string = "arn:aws:iam::aws:policy/non-existing"
				)

				By("Try to create account-roles with invalid prefix")
				output, err := ocmResourceService.CreateAccountRole("--mode", validModeAuto,
					"--prefix", invalidRolePrefix,
					"--permissions-boundary", permissionsBoundaryArn,
					"-y")
				Expect(err).NotTo(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, invalidRolePrefix)
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Expected a valid role prefix matching ^[\\w+=,.@-]+$"))

				By("Try to create account-roles with longer than 32 chars prefix")
				output, err = ocmResourceService.CreateAccountRole("--mode", validModeAuto,
					"--prefix", longRolePrefix,
					"--permissions-boundary", permissionsBoundaryArn,
					"--hosted-cp",
					"-y")
				Expect(err).NotTo(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, longRolePrefix)
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Expected a prefix with no more than 32 characters"))

				By("Try to create account-roles with invalid mode")
				output, err = ocmResourceService.CreateAccountRole("--mode", invalidMode,
					"--prefix", validRolePrefix,
					"--permissions-boundary", permissionsBoundaryArn,
					"-y")
				Expect(err).NotTo(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, validRolePrefix)
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Invalid mode. Allowed values are [auto manual]"))

				By("Try to create account-roles with force-policy-creation and manual mode")
				output, err = ocmResourceService.CreateAccountRole("--mode", validModeManual,
					"--prefix", validRolePrefix,
					"-f",
					"--hosted-cp",
					"--permissions-boundary", permissionsBoundaryArn,
				)
				Expect(err).NotTo(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, validRolePrefix)
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Forcing creation of policies only works in auto mode"))

				By("Try to create account-roles with invalid permission boundary")
				output, err = ocmResourceService.CreateAccountRole("--mode", validModeAuto,
					"--prefix", validRolePrefix,
					"--permissions-boundary", invalidPermissionsBoundaryArn,
					"-y",
				)
				Expect(err).NotTo(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, validRolePrefix)
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Expected a valid policy ARN for permissions boundary: Invalid ARN: arn: invalid prefix"))

				By("Try to create account-roles with non-existing permission boundary")
				output, err = ocmResourceService.CreateAccountRole("--mode", validModeAuto,
					"--prefix", validRolePrefix,
					"--hosted-cp",
					"--permissions-boundary", nonExistingPermissionsBoundaryArn,
					"-y",
				)
				Expect(err).NotTo(BeNil())

				accountRolePrefixesNeedCleanup = append(accountRolePrefixesNeedCleanup, validRolePrefix)
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("There was an error creating the account roles"))
				Expect(textData).To(ContainSubstring("policy/non-existing does not exist or is not attachable"))
			})
	})

var _ = Describe("List IAM",
	labels.Day1,
	labels.FeatureRoles,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient         *rosacli.Client
			ocmResourceService rosacli.OCMResourceService
		)
		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
		})

		It("to list account-roles by rosa-cli - [id:44511]", labels.High, func() {

			accrolePrefix := "arPrefix44511"
			path := "/a/b/"

			By("Prepare a version for testing")
			var version string
			versionService := rosaClient.Version
			versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, false)
			Expect(err).To(BeNil())

			defaultVersion := versionList.DefaultVersion()
			Expect(defaultVersion).ToNot(BeNil())

			_, _, version, err = defaultVersion.MajorMinor()
			Expect(err).To(BeNil())

			By("Create account-roles")
			output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
				"--prefix", accrolePrefix,
				"--path", path,
				"--version", version,
				"-y")
			Expect(err).To(BeNil())
			defer func() {
				By("Delete the account-roles")
				output, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", accrolePrefix,
					"-y")

				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("Successfully deleted"))
			}()
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			Expect(textData).To(ContainSubstring("Created role"))

			By("List account-roles")
			arl, _, err := ocmResourceService.ListAccountRole()
			Expect(err).To(BeNil())

			ars := arl.AccountRoles(accrolePrefix)
			fmt.Println(ars)

			Expect(len(ars)).To(Equal(7))

			for _, v := range ars {
				Expect(v.OpenshiftVersion).To(Equal(version))
				Expect(v.RoleArn).NotTo(BeEmpty())
				if strings.Contains(v.RoleName, "HCP-ROSA") {
					Expect(v.AWSManaged).To(Equal("Yes"))
				} else {
					Expect(v.AWSManaged).To(Equal("No"))
				}
			}
		})
	})
