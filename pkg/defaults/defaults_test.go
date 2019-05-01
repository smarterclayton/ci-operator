package defaults

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/rest"

	templateapi "github.com/openshift/api/template/v1"

	"github.com/openshift/ci-operator/pkg/api"
)

func addCloneRefs(cfg *api.SourceStepConfiguration) *api.SourceStepConfiguration {
	cfg.ClonerefsImage = api.ImageStreamTagReference{Cluster: "https://api.ci.openshift.org", Namespace: "ci", Name: "clonerefs", Tag: "latest"}
	cfg.ClonerefsPath = "/app/prow/cmd/clonerefs/app.binary.runfiles/io_k8s_test_infra/prow/cmd/clonerefs/linux_amd64_pure_stripped/app.binary"
	return cfg
}

func TestStepConfigsForBuild(t *testing.T) {
	var testCases = []struct {
		name    string
		input   *api.ReleaseBuildConfiguration
		jobSpec *api.JobSpec
		output  []api.StepConfiguration
	}{
		{
			name: "minimal information provided",
			input: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
				},
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},
			output: []api.StepConfiguration{{
				SourceStepConfiguration: addCloneRefs(&api.SourceStepConfiguration{
					From: api.PipelineImageStreamTagReferenceRoot,
					To:   api.PipelineImageStreamTagReferenceSource,
				}),
			}, {
				InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
					BaseImage: api.ImageStreamTagReference{
						Namespace: "base-1",
						Name:      "repo-test-base",
						Tag:       "manual",
					},
					To: api.PipelineImageStreamTagReferenceRoot,
				},
			}},
		},
		{
			name: "binary build requested",
			input: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
				},
				BinaryBuildCommands: "hi",
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},
			output: []api.StepConfiguration{{
				SourceStepConfiguration: addCloneRefs(&api.SourceStepConfiguration{
					From: api.PipelineImageStreamTagReferenceRoot,
					To:   api.PipelineImageStreamTagReferenceSource,
				}),
			}, {
				InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
					BaseImage: api.ImageStreamTagReference{
						Namespace: "base-1",
						Name:      "repo-test-base",
						Tag:       "manual",
					},
					To: api.PipelineImageStreamTagReferenceRoot,
				},
			}, {
				PipelineImageCacheStepConfiguration: &api.PipelineImageCacheStepConfiguration{
					From:     api.PipelineImageStreamTagReferenceSource,
					To:       api.PipelineImageStreamTagReferenceBinaries,
					Commands: "hi",
				},
			}},
		},
		{
			name: "binary and rpm build requested",
			input: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
				},
				BinaryBuildCommands: "hi",
				RpmBuildCommands:    "hello",
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},
			output: []api.StepConfiguration{{
				SourceStepConfiguration: addCloneRefs(&api.SourceStepConfiguration{
					From: api.PipelineImageStreamTagReferenceRoot,
					To:   api.PipelineImageStreamTagReferenceSource,
				}),
			}, {
				InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
					BaseImage: api.ImageStreamTagReference{
						Namespace: "base-1",
						Name:      "repo-test-base",
						Tag:       "manual",
					},
					To: api.PipelineImageStreamTagReferenceRoot,
				},
			}, {
				PipelineImageCacheStepConfiguration: &api.PipelineImageCacheStepConfiguration{
					From:     api.PipelineImageStreamTagReferenceSource,
					To:       api.PipelineImageStreamTagReferenceBinaries,
					Commands: "hi",
				},
			}, {
				PipelineImageCacheStepConfiguration: &api.PipelineImageCacheStepConfiguration{
					From:     api.PipelineImageStreamTagReferenceBinaries,
					To:       api.PipelineImageStreamTagReferenceRPMs,
					Commands: "hello; ln -s $( pwd )/_output/local/releases/rpms/ /srv/repo",
				},
			}, {
				RPMServeStepConfiguration: &api.RPMServeStepConfiguration{
					From: api.PipelineImageStreamTagReferenceRPMs,
				},
			}},
		},
		{
			name: "rpm but not binary build requested",
			input: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
				},
				RpmBuildCommands: "hello",
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},
			output: []api.StepConfiguration{{
				SourceStepConfiguration: addCloneRefs(&api.SourceStepConfiguration{
					From: api.PipelineImageStreamTagReferenceRoot,
					To:   api.PipelineImageStreamTagReferenceSource,
				}),
			}, {
				InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
					BaseImage: api.ImageStreamTagReference{
						Namespace: "base-1",
						Name:      "repo-test-base",
						Tag:       "manual",
					},
					To: api.PipelineImageStreamTagReferenceRoot,
				},
			}, {
				PipelineImageCacheStepConfiguration: &api.PipelineImageCacheStepConfiguration{
					From:     api.PipelineImageStreamTagReferenceSource,
					To:       api.PipelineImageStreamTagReferenceRPMs,
					Commands: "hello; ln -s $( pwd )/_output/local/releases/rpms/ /srv/repo",
				},
			}, {
				RPMServeStepConfiguration: &api.RPMServeStepConfiguration{
					From: api.PipelineImageStreamTagReferenceRPMs,
				},
			}},
		},
		{
			name: "rpm with custom output but not binary build requested",
			input: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
				},
				RpmBuildLocation: "testing",
				RpmBuildCommands: "hello",
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},
			output: []api.StepConfiguration{{
				SourceStepConfiguration: addCloneRefs(&api.SourceStepConfiguration{
					From: api.PipelineImageStreamTagReferenceRoot,
					To:   api.PipelineImageStreamTagReferenceSource,
				}),
			}, {
				InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
					BaseImage: api.ImageStreamTagReference{
						Namespace: "base-1",
						Name:      "repo-test-base",
						Tag:       "manual",
					},
					To: api.PipelineImageStreamTagReferenceRoot,
				},
			}, {
				PipelineImageCacheStepConfiguration: &api.PipelineImageCacheStepConfiguration{
					From:     api.PipelineImageStreamTagReferenceSource,
					To:       api.PipelineImageStreamTagReferenceRPMs,
					Commands: "hello; ln -s $( pwd )/testing /srv/repo",
				},
			}, {
				RPMServeStepConfiguration: &api.RPMServeStepConfiguration{
					From: api.PipelineImageStreamTagReferenceRPMs,
				},
			}},
		},
		{
			name: "explicit base image requested",
			input: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
					BaseImages: map[string]api.ImageStreamTagReference{
						"name": {
							Namespace: "namespace",
							Name:      "name",
							Tag:       "tag",
						},
					},
				},
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},
			output: []api.StepConfiguration{{
				SourceStepConfiguration: addCloneRefs(&api.SourceStepConfiguration{
					From: api.PipelineImageStreamTagReferenceRoot,
					To:   api.PipelineImageStreamTagReferenceSource,
				}),
			}, {
				InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
					BaseImage: api.ImageStreamTagReference{
						Namespace: "base-1",
						Name:      "repo-test-base",
						Tag:       "manual",
					},
					To: api.PipelineImageStreamTagReferenceRoot,
				},
			}, {
				InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
					BaseImage: api.ImageStreamTagReference{
						Namespace: "namespace",
						Name:      "name",
						Tag:       "tag",
						As:        "name",
					},
					To: api.PipelineImageStreamTagReference("name"),
				},
			}},
		},
		{
			name: "implicit base image from release configuration",
			input: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
					ReleaseTagConfiguration: &api.ReleaseTagConfiguration{
						Namespace: "test",
						Name:      "other",
					},
					BaseImages: map[string]api.ImageStreamTagReference{
						"name": {
							Tag: "tag",
						},
					},
				},
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},
			output: []api.StepConfiguration{
				{
					InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
						BaseImage: api.ImageStreamTagReference{
							Namespace: "base-1",
							Name:      "repo-test-base",
							Tag:       "manual",
						},
						To: api.PipelineImageStreamTagReferenceRoot,
					},
				},
				{
					SourceStepConfiguration: addCloneRefs(&api.SourceStepConfiguration{
						From: api.PipelineImageStreamTagReferenceRoot,
						To:   api.PipelineImageStreamTagReferenceSource,
					}),
				},
				{
					InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
						BaseImage: api.ImageStreamTagReference{
							Namespace: "test",
							Name:      "other",
							Tag:       "tag",
							As:        "name",
						},
						To: api.PipelineImageStreamTagReference("name"),
					},
				},
				{
					ReleaseImagesTagStepConfiguration: &api.ReleaseTagConfiguration{
						Namespace: "test",
						Name:      "other",
					},
				},
			},
		},
		{
			name: "rpm base image requested",
			input: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
					BaseRPMImages: map[string]api.ImageStreamTagReference{
						"name": {
							Namespace: "namespace",
							Name:      "name",
							Tag:       "tag",
						},
					},
				},
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},
			output: []api.StepConfiguration{{
				SourceStepConfiguration: addCloneRefs(&api.SourceStepConfiguration{
					From: api.PipelineImageStreamTagReferenceRoot,
					To:   api.PipelineImageStreamTagReferenceSource,
				}),
			}, {
				InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
					BaseImage: api.ImageStreamTagReference{
						Namespace: "base-1",
						Name:      "repo-test-base",
						Tag:       "manual",
					},
					To: api.PipelineImageStreamTagReferenceRoot,
				},
			}, {
				InputImageTagStepConfiguration: &api.InputImageTagStepConfiguration{
					BaseImage: api.ImageStreamTagReference{
						Namespace: "namespace",
						Name:      "name",
						Tag:       "tag",
						As:        "name",
					},
					To: api.PipelineImageStreamTagReference("name-without-rpms"),
				},
			}, {
				RPMImageInjectionStepConfiguration: &api.RPMImageInjectionStepConfiguration{
					From: api.PipelineImageStreamTagReference("name-without-rpms"),
					To:   api.PipelineImageStreamTagReference("name"),
				},
			}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if configs := stepConfigsForBuild(testCase.input, testCase.jobSpec); !stepListsEqual(configs, testCase.output) {
				t.Logf("%s", diff.ObjectReflectDiff(testCase.output, configs))
				t.Errorf("incorrect defaulted step configurations,\n\tgot:\n%s\n\texpected:\n%s", formatSteps(configs), formatSteps(testCase.output))
			}
		})
	}
}

// stepListsEqual determines if the two lists of step configs
// contain the same elements, but is not interested
// in ordering
func stepListsEqual(first, second []api.StepConfiguration) bool {
	if len(first) != len(second) {
		return false
	}

	for _, item := range first {
		otherContains := false
		for _, other := range second {
			if reflect.DeepEqual(item, other) {
				otherContains = true
			}
		}
		if !otherContains {
			return false
		}
	}

	return true
}

func formatSteps(steps []api.StepConfiguration) string {
	output := bytes.Buffer{}
	for _, step := range steps {
		output.WriteString(formatStep(step))
		output.WriteString("\n")
	}
	return output.String()
}

func formatStep(step api.StepConfiguration) string {
	if step.InputImageTagStepConfiguration != nil {
		return fmt.Sprintf("Tag %s to pipeline:%s", formatReference(step.InputImageTagStepConfiguration.BaseImage), step.InputImageTagStepConfiguration.To)
	}

	if step.PipelineImageCacheStepConfiguration != nil {
		return fmt.Sprintf("Run %v in pipeline:%s to cache in pipeline:%s", step.PipelineImageCacheStepConfiguration.Commands, step.PipelineImageCacheStepConfiguration.From, step.PipelineImageCacheStepConfiguration.To)
	}

	if step.SourceStepConfiguration != nil {
		return fmt.Sprintf("Clone source into pipeline:%s to cache in pipline:%s", step.SourceStepConfiguration.From, step.SourceStepConfiguration.To)
	}

	if step.ProjectDirectoryImageBuildStepConfiguration != nil {
		return fmt.Sprintf("Build project image from %s in pipeline:%s to cache in pipline:%s", step.ProjectDirectoryImageBuildStepConfiguration.ContextDir, step.ProjectDirectoryImageBuildStepConfiguration.From, step.ProjectDirectoryImageBuildStepConfiguration.To)
	}

	if step.RPMImageInjectionStepConfiguration != nil {
		return fmt.Sprintf("Inject RPM repos into pipeline:%s to cache in pipline:%s", step.RPMImageInjectionStepConfiguration.From, step.RPMImageInjectionStepConfiguration.To)
	}

	if step.RPMServeStepConfiguration != nil {
		return fmt.Sprintf("Serve RPMs from pipeline:%s", step.RPMServeStepConfiguration.From)
	}

	return ""
}

func formatReference(ref api.ImageStreamTagReference) string {
	return fmt.Sprintf("%s/%s:%s (as:%s)", ref.Namespace, ref.Name, ref.Tag, ref.As)
}

func TestFromConfig(t *testing.T) {
	tests := []struct {
		name string

		config          *api.ReleaseBuildConfiguration
		jobSpec         *api.JobSpec
		templates       []*templateapi.Template
		paramFile       string
		artifactDir     string
		promote         bool
		clusterConfig   *rest.Config
		requiredTargets []string

		wantGraph bool
		want      []string
		wantPost  []string
		wantErr   bool
	}{
		{
			config: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
					BaseRPMImages: map[string]api.ImageStreamTagReference{
						"name": {
							Namespace: "namespace",
							Name:      "name",
							Tag:       "tag",
						},
					},
				},
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},
			want: []string{"[input:root]", "src", "[input:name-without-rpms]", "name", "[output-images]", "[images]"},
		},

		{
			name: "a test referencing an image and a root defines a valid graph",

			config: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
				},
				Images: []api.ProjectDirectoryImageBuildStepConfiguration{
					{
						From: api.PipelineImageStreamTagReference("root"),
						To:   api.PipelineImageStreamTagReference("name"),
					},
				},
				Tests: []api.TestStepConfiguration{
					{
						As: "e2e-aws",
						ContainerTestConfiguration: &api.ContainerTestConfiguration{
							From: "name",
						},
					},
				},
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},

			wantGraph: true,
			want:      []string{"[input:root]", "[output-images]", "src", "name", "[output:stable:name]", "e2e-aws", "[images]"},
		},

		{
			name: "specifying a template overrides the step from the config",

			config: &api.ReleaseBuildConfiguration{
				InputConfiguration: api.InputConfiguration{
					BuildRootImage: &api.BuildRootImageConfiguration{
						ImageStreamTagReference: &api.ImageStreamTagReference{Tag: "manual"},
					},
				},
				Images: []api.ProjectDirectoryImageBuildStepConfiguration{
					{
						From: api.PipelineImageStreamTagReference("root"),
						To:   api.PipelineImageStreamTagReference("name"),
					},
				},
				Tests: []api.TestStepConfiguration{
					{
						As: "e2e-aws",
						ContainerTestConfiguration: &api.ContainerTestConfiguration{
							From: "name",
						},
					},
				},
			},
			jobSpec: &api.JobSpec{
				Refs: &api.Refs{
					Org:  "org",
					Repo: "repo",
				},
				BaseNamespace: "base-1",
			},
			templates: []*templateapi.Template{
				{
					ObjectMeta: meta.ObjectMeta{Name: "e2e-aws"},
				},
			},

			wantGraph: true,
			want:      []string{"[input:root]", "e2e-aws", "[output-images]", "src", "name", "[output:stable:name]", "[images]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := FromConfig(tt.config, tt.jobSpec, tt.templates, tt.paramFile, tt.artifactDir, tt.promote, tt.clusterConfig, tt.requiredTargets)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var names []string
			if tt.wantGraph {
				// verify we can build a graph from the result
				graph := api.BuildGraph(got)
				sorted, err := api.TopologicalSort(graph)
				if err != nil {
					t.Fatalf("unexpected error sorting steps: %v", err)
				}
				for _, node := range sorted {
					names = append(names, node.Step.Name())
				}
			} else {
				for _, step := range got {
					names = append(names, step.Name())
				}
			}
			if !reflect.DeepEqual(names, tt.want) {
				t.Errorf("\n%v\n%v", names, tt.want)
			}

			names = nil
			for _, step := range got1 {
				names = append(names, step.Name())
			}
			if !reflect.DeepEqual(names, tt.wantPost) {
				t.Errorf("\n%v\n%v", names, tt.wantPost)
			}
		})
	}
}
