package clusterinstall

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclientset "k8s.io/client-go/kubernetes/typed/core/v1"

	templateapi "github.com/openshift/api/template/v1"

	"github.com/openshift/ci-operator/pkg/api"
	"github.com/openshift/ci-operator/pkg/junit"
	"github.com/openshift/ci-operator/pkg/steps"
)

type e2eTestStep struct {
	config     api.OpenshiftInstallerClusterTestConfiguration
	testConfig api.TestStepConfiguration

	secretClient coreclientset.SecretsGetter
	jobSpec      *api.JobSpec

	step api.Step
	nestedSubTests
}

type nestedSubTests interface {
	SubTests() []*junit.TestCase
}

// E2ETestStep installs a cluster and then runs end-to-end tests against it.
func E2ETestStep(
	config api.OpenshiftInstallerClusterTestConfiguration,
	testConfig api.TestStepConfiguration,
	params api.Parameters,
	podClient steps.PodClient,
	templateClient steps.TemplateClient,
	secretClient coreclientset.SecretsGetter,
	artifactDir string,
	jobSpec *api.JobSpec,
) (api.Step, error) {
	var template *templateapi.Template
	if err := yaml.Unmarshal([]byte(installTemplateE2E), &template); err != nil {
		panic(fmt.Errorf("the embedded template is invalid: %v", err))
	}

	template.Name = testConfig.As

	overrides := make(map[string]string)
	for i := range template.Parameters {
		p := &template.Parameters[i]
		switch p.Name {
		case "JOB_NAME_SAFE":
			if !params.HasInput(p.Name) {
				overrides[p.Name] = testConfig.As
			}
		case "TEST_COMMAND":
			p.Value = testConfig.Commands
		case "CLUSTER_TYPE":
			p.Value = strings.Split(string(config.ClusterProfile), "-")[0]
		}
	}

	if config.Upgrade {
		// ensure we depend on the release image
		name := "RELEASE_IMAGE_INITIAL"
		template.Parameters = append(template.Parameters, templateapi.Parameter{
			Required: true,
			Name:     name,
		})

		// ensure the installer image points to the initial state
		name = "IMAGE_INSTALLER"
		if !params.HasInput(name) {
			overrides[name] = "stable-initial:installer"
		}
		template.Parameters = append(template.Parameters, templateapi.Parameter{
			Required: true,
			Name:     name,
		})

		// set install initial release true for use in the template
		name = "INSTALL_INITIAL_RELEASE"
		template.Parameters = append(template.Parameters, templateapi.Parameter{
			Required: true,
			Name:     name,
			Value:    "true",
		})
	}

	params = api.NewOverrideParameters(params, overrides)

	step := steps.TemplateExecutionStep(template, params, podClient, templateClient, artifactDir, jobSpec)
	subTests, ok := step.(nestedSubTests)
	if !ok {
		return nil, fmt.Errorf("unexpected %T", step)
	}

	return &e2eTestStep{
		config:     config,
		testConfig: testConfig,

		secretClient: secretClient,
		jobSpec:      jobSpec,

		step:           step,
		nestedSubTests: subTests,
	}, nil
}

func (s *e2eTestStep) checkPrereqs() error {
	return nil
}

func (s *e2eTestStep) Inputs(ctx context.Context, dry bool) (api.InputDefinition, error) {
	return nil, nil
}

func (s *e2eTestStep) Run(ctx context.Context, dry bool) error {
	if dry {
		return nil
	}
	if _, err := s.secretClient.Secrets(s.jobSpec.Namespace).Get(fmt.Sprintf("%s-cluster-profile", s.testConfig.As), meta.GetOptions{}); err != nil {
		return fmt.Errorf("could not find required secret: %v", err)
	}
	return s.step.Run(ctx, dry)
}

func (s *e2eTestStep) Done() (bool, error) {
	return false, nil
}

func (s *e2eTestStep) Requires() []api.StepLink {
	links := s.step.Requires()
	if s.config.Upgrade {
		links = append([]api.StepLink{api.ReleasePayloadImageLink(api.PipelineImageStreamTagReference("initial"))}, links...)
	}
	return links
}

func (s *e2eTestStep) Creates() []api.StepLink {
	return nil
}

func (s *e2eTestStep) Provides() (api.ParameterMap, api.StepLink) {
	return nil, nil
}

func (s *e2eTestStep) Name() string { return s.testConfig.As }

func (s *e2eTestStep) Description() string {
	if s.config.Upgrade {
		return fmt.Sprintf("Run cluster install and upgrade %s", s.testConfig.As)
	}
	return fmt.Sprintf("Run cluster install %s", s.testConfig.As)
}
