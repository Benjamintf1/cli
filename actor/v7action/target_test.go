package v7action_test

import (
	"errors"

	. "code.cloudfoundry.org/cli/actor/v7action"
	"code.cloudfoundry.org/cli/actor/v7action/v7actionfakes"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Targeting", func() {
	var (
		actor                     *Actor
		fakeCloudControllerClient *v7actionfakes.FakeCloudControllerClient
		fakeConfig                *v7actionfakes.FakeConfig

		settings          TargetSettings
		skipSSLValidation bool
		targetedURL       string
	)

	BeforeEach(func() {
		actor, fakeCloudControllerClient, fakeConfig, _, _, _ = NewTestActor()

	})

	Describe("SetTarget", func() {
		var (
			expectedAPI        string
			expectedAPIVersion string
			expectedAuth       string
			expectedDoppler    string
			expectedRouting    string

			err      error
			warnings Warnings
		)

		BeforeEach(func() {
			expectedAPI = "https://api.foo.com"
			expectedAPIVersion = "3.81.0"
			expectedAuth = "https://login.foo.com"
			expectedDoppler = "wss://doppler.foo.com"
			expectedRouting = "https://api.foo.com/routing"

			skipSSLValidation = true
			targetedURL = expectedAPI
			var meta struct {
				Version            string `json:"version"`
				HostKeyFingerprint string `json:"host_key_fingerprint"`
				OAuthClient        string `json:"oath_client"`
			}
			meta.Version = expectedAPIVersion

			rootResponse := ccv3.Info{
				Links: ccv3.InfoLinks{
					CCV3: ccv3.APILink{
						Meta: meta,
					},
					Logging: ccv3.APILink{
						HREF: expectedDoppler,
					},
					Routing: ccv3.APILink{
						HREF: expectedRouting,
					},
					UAA: ccv3.APILink{
						HREF: expectedAuth,
					},
				},
			}
			fakeCloudControllerClient.TargetCFReturns(rootResponse, ccv3.Warnings{"info-warning"}, nil)
		})

		JustBeforeEach(func() {
			settings = TargetSettings{
				SkipSSLValidation: skipSSLValidation,
				URL:               targetedURL,
			}
			warnings, err = actor.SetTarget(settings)
		})

		It("targets CF with the expected arguments", func() {
			Expect(fakeCloudControllerClient.TargetCFCallCount()).To(Equal(1))
			connectionSettings := fakeCloudControllerClient.TargetCFArgsForCall(0)
			Expect(connectionSettings.URL).To(Equal(expectedAPI))
			Expect(connectionSettings.SkipSSLValidation).To(BeTrue())
		})

		When("targeting CF fails", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.TargetCFReturns(ccv3.Info{}, ccv3.Warnings{"info-warning"}, errors.New("target-error"))
			})

			It("returns an error and all warnings", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("target-error"))
				Expect(warnings).To(ConsistOf(Warnings{"info-warning"}))

				Expect(fakeCloudControllerClient.TargetCFCallCount()).To(Equal(1))
			})
		})

		It("sets all the target information", func() {
			Expect(fakeConfig.SetTargetInformationCallCount()).To(Equal(1))
			api, apiVersion, auth, _, doppler, routing, sslDisabled := fakeConfig.SetTargetInformationArgsForCall(0)

			Expect(api).To(Equal(expectedAPI))
			Expect(apiVersion).To(Equal(expectedAPIVersion))
			Expect(auth).To(Equal(expectedAuth))
			Expect(doppler).To(Equal(expectedDoppler))
			Expect(routing).To(Equal(expectedRouting))
			Expect(sslDisabled).To(Equal(skipSSLValidation))
		})

		It("clears all the token information", func() {
			Expect(fakeConfig.SetTokenInformationCallCount()).To(Equal(1))
			accessToken, refreshToken, sshOAuthClient := fakeConfig.SetTokenInformationArgsForCall(0)

			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
			Expect(sshOAuthClient).To(BeEmpty())
		})

		It("succeeds and returns all warnings", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(ConsistOf(Warnings{"info-warning"}))
		})
	})

	Describe("ClearTarget", func() {
		It("clears all the target information", func() {
			actor.ClearTarget()

			Expect(fakeConfig.SetTargetInformationCallCount()).To(Equal(1))
			api, apiVersion, auth, minCLIVersion, doppler, routing, sslDisabled := fakeConfig.SetTargetInformationArgsForCall(0)

			Expect(api).To(BeEmpty())
			Expect(apiVersion).To(BeEmpty())
			Expect(auth).To(BeEmpty())
			Expect(minCLIVersion).To(BeEmpty())
			Expect(doppler).To(BeEmpty())
			Expect(routing).To(BeEmpty())
			Expect(sslDisabled).To(BeFalse())
		})

		It("clears all the token information", func() {
			actor.ClearTarget()

			Expect(fakeConfig.SetTokenInformationCallCount()).To(Equal(1))
			accessToken, refreshToken, sshOAuthClient := fakeConfig.SetTokenInformationArgsForCall(0)

			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
			Expect(sshOAuthClient).To(BeEmpty())
		})
	})
})
