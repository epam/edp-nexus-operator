package nexus

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/epam/edp-nexus-operator/api/v1alpha1"
)

func TestGetRepoData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		repo    *v1alpha1.NexusRepositorySpec
		want    *RepoData
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "apt proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Apt: &v1alpha1.AptSpec{
					Proxy: &v1alpha1.AptProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "apt-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatApt,
				Type:   TypeProxy,
				Name:   "apt-proxy",
				Data: &v1alpha1.AptProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "apt-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "apt hosted repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Apt: &v1alpha1.AptSpec{
					Hosted: &v1alpha1.AptHostedRepository{
						HostedSpec: v1alpha1.HostedSpec{
							Name: "apt-hosted",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatApt,
				Type:   TypeHosted,
				Name:   "apt-hosted",
				Data: &v1alpha1.AptHostedRepository{
					HostedSpec: v1alpha1.HostedSpec{
						Name: "apt-hosted",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "bower proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Bower: &v1alpha1.BowerSpec{
					Proxy: &v1alpha1.BowerProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "bower-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatBower,
				Type:   TypeProxy,
				Name:   "bower-proxy",
				Data: &v1alpha1.BowerProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "bower-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "bower hosted repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Bower: &v1alpha1.BowerSpec{
					Hosted: &v1alpha1.BowerHostedRepository{
						HostedSpec: v1alpha1.HostedSpec{
							Name: "bower-hosted",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatBower,
				Type:   TypeHosted,
				Name:   "bower-hosted",
				Data: &v1alpha1.BowerHostedRepository{
					HostedSpec: v1alpha1.HostedSpec{
						Name: "bower-hosted",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "bower group repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Bower: &v1alpha1.BowerSpec{
					Group: &v1alpha1.BowerGroupRepository{
						GroupSpec: v1alpha1.GroupSpec{
							Name: "bower-group",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatBower,
				Type:   TypeGroup,
				Name:   "bower-group",
				Data: &v1alpha1.BowerGroupRepository{
					GroupSpec: v1alpha1.GroupSpec{
						Name: "bower-group",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "cocoapods proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Cocoapods: &v1alpha1.CocoapodsSpec{
					Proxy: &v1alpha1.CocoapodsProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "cocoapods-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatCocoapods,
				Type:   TypeProxy,
				Name:   "cocoapods-proxy",
				Data: &v1alpha1.CocoapodsProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "cocoapods-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "conan proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Conan: &v1alpha1.ConanSpec{
					Proxy: &v1alpha1.ConanProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "conan-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatConan,
				Type:   TypeProxy,
				Name:   "conan-proxy",
				Data: &v1alpha1.ConanProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "conan-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "conda proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Conda: &v1alpha1.CondaSpec{
					Proxy: &v1alpha1.CondaProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "conda-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatConda,
				Type:   TypeProxy,
				Name:   "conda-proxy",
				Data: &v1alpha1.CondaProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "conda-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "docker proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Docker: &v1alpha1.DockerSpec{
					Proxy: &v1alpha1.DockerProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "docker-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatDocker,
				Type:   TypeProxy,
				Name:   "docker-proxy",
				Data: &v1alpha1.DockerProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "docker-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "docker hosted repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Docker: &v1alpha1.DockerSpec{
					Hosted: &v1alpha1.DockerHostedRepository{
						HostedSpec: v1alpha1.HostedSpec{
							Name: "docker-hosted",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatDocker,
				Type:   TypeHosted,
				Name:   "docker-hosted",
				Data: &v1alpha1.DockerHostedRepository{
					HostedSpec: v1alpha1.HostedSpec{
						Name: "docker-hosted",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "docker group repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Docker: &v1alpha1.DockerSpec{
					Group: &v1alpha1.DockerGroupRepository{
						Name: "docker-group",
					},
				},
			},
			want: &RepoData{
				Format: FormatDocker,
				Type:   TypeGroup,
				Name:   "docker-group",
				Data: &v1alpha1.DockerGroupRepository{
					Name: "docker-group",
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "gitlfs hosted repository",
			repo: &v1alpha1.NexusRepositorySpec{
				GitLfs: &v1alpha1.GitLfsSpec{
					Hosted: &v1alpha1.GitLfsHostedRepository{
						HostedSpec: v1alpha1.HostedSpec{
							Name: "gitlfs-hosted",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatGitLfs,
				Type:   TypeHosted,
				Name:   "gitlfs-hosted",
				Data: &v1alpha1.GitLfsHostedRepository{
					HostedSpec: v1alpha1.HostedSpec{
						Name: "gitlfs-hosted",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "go proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Go: &v1alpha1.GoSpec{
					Proxy: &v1alpha1.GoProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "go-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatGo,
				Type:   TypeProxy,
				Name:   "go-proxy",
				Data: &v1alpha1.GoProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "go-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "go group repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Go: &v1alpha1.GoSpec{
					Group: &v1alpha1.GoGroupRepository{
						GroupSpec: v1alpha1.GroupSpec{
							Name: "go-group",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatGo,
				Type:   TypeGroup,
				Name:   "go-group",
				Data: &v1alpha1.GoGroupRepository{
					GroupSpec: v1alpha1.GroupSpec{
						Name: "go-group",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "helm hosted repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Helm: &v1alpha1.HelmSpec{
					Hosted: &v1alpha1.HelmHostedRepository{
						HostedSpec: v1alpha1.HostedSpec{
							Name: "helm-hosted",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatHelm,
				Type:   TypeHosted,
				Name:   "helm-hosted",
				Data: &v1alpha1.HelmHostedRepository{
					HostedSpec: v1alpha1.HostedSpec{
						Name: "helm-hosted",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "helm proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Helm: &v1alpha1.HelmSpec{
					Proxy: &v1alpha1.HelmProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "helm-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatHelm,
				Type:   TypeProxy,
				Name:   "helm-proxy",
				Data: &v1alpha1.HelmProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "helm-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "maven proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Maven: &v1alpha1.MavenSpec{
					Proxy: &v1alpha1.MavenProxyRepository{
						Name: "maven-proxy",
					},
				},
			},
			want: &RepoData{
				Format: FormatMaven,
				Type:   TypeProxy,
				Name:   "maven-proxy",
				Data: &v1alpha1.MavenProxyRepository{
					Name: "maven-proxy",
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "maven hosted repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Maven: &v1alpha1.MavenSpec{
					Hosted: &v1alpha1.MavenHostedRepository{
						HostedSpec: v1alpha1.HostedSpec{
							Name: "maven-hosted",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatMaven,
				Type:   TypeHosted,
				Name:   "maven-hosted",
				Data: &v1alpha1.MavenHostedRepository{
					HostedSpec: v1alpha1.HostedSpec{
						Name: "maven-hosted",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "maven group repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Maven: &v1alpha1.MavenSpec{
					Group: &v1alpha1.MavenGroupRepository{
						GroupSpec: v1alpha1.GroupSpec{
							Name: "maven-group",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatMaven,
				Type:   TypeGroup,
				Name:   "maven-group",
				Data: &v1alpha1.MavenGroupRepository{
					GroupSpec: v1alpha1.GroupSpec{
						Name: "maven-group",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "npm proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Npm: &v1alpha1.NpmSpec{
					Proxy: &v1alpha1.NpmProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "npm-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatNpm,
				Type:   TypeProxy,
				Name:   "npm-proxy",
				Data: &v1alpha1.NpmProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "npm-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "npm hosted repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Npm: &v1alpha1.NpmSpec{
					Hosted: &v1alpha1.NpmHostedRepository{
						HostedSpec: v1alpha1.HostedSpec{
							Name: "npm-hosted",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatNpm,
				Type:   TypeHosted,
				Name:   "npm-hosted",
				Data: &v1alpha1.NpmHostedRepository{
					HostedSpec: v1alpha1.HostedSpec{
						Name: "npm-hosted",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "npm group repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Npm: &v1alpha1.NpmSpec{
					Group: &v1alpha1.NpmGroupRepository{
						GroupSpec: v1alpha1.GroupSpec{
							Name: "npm-group",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatNpm,
				Type:   TypeGroup,
				Name:   "npm-group",
				Data: &v1alpha1.NpmGroupRepository{
					GroupSpec: v1alpha1.GroupSpec{
						Name: "npm-group",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "nuget proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Nuget: &v1alpha1.NugetSpec{
					Proxy: &v1alpha1.NugetProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "nuget-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatNuget,
				Type:   TypeProxy,
				Name:   "nuget-proxy",
				Data: &v1alpha1.NugetProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "nuget-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "nuget hosted repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Nuget: &v1alpha1.NugetSpec{
					Hosted: &v1alpha1.NugetHostedRepository{
						HostedSpec: v1alpha1.HostedSpec{
							Name: "nuget-hosted",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatNuget,
				Type:   TypeHosted,
				Name:   "nuget-hosted",
				Data: &v1alpha1.NugetHostedRepository{
					HostedSpec: v1alpha1.HostedSpec{
						Name: "nuget-hosted",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "nuget group repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Nuget: &v1alpha1.NugetSpec{
					Group: &v1alpha1.NugetGroupRepository{
						GroupSpec: v1alpha1.GroupSpec{
							Name: "nuget-group",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatNuget,
				Type:   TypeGroup,
				Name:   "nuget-group",
				Data: &v1alpha1.NugetGroupRepository{
					GroupSpec: v1alpha1.GroupSpec{
						Name: "nuget-group",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "p2 proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				P2: &v1alpha1.P2Spec{
					Proxy: &v1alpha1.P2ProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "p2-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatP2,
				Type:   TypeProxy,
				Name:   "p2-proxy",
				Data: &v1alpha1.P2ProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "p2-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "pypi proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Pypi: &v1alpha1.PypiSpec{
					Proxy: &v1alpha1.PypiProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "pypi-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatPypi,
				Type:   TypeProxy,
				Name:   "pypi-proxy",
				Data: &v1alpha1.PypiProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "pypi-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "r proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				R: &v1alpha1.RSpec{
					Proxy: &v1alpha1.RProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "r-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatR,
				Type:   TypeProxy,
				Name:   "r-proxy",
				Data: &v1alpha1.RProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "r-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "raw proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Raw: &v1alpha1.RawSpec{
					Proxy: &v1alpha1.RawProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "raw-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatRaw,
				Type:   TypeProxy,
				Name:   "raw-proxy",
				Data: &v1alpha1.RawProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "raw-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "rubygems proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				RubyGems: &v1alpha1.RubyGemsSpec{
					Proxy: &v1alpha1.RubyGemsProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "rubygems-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatRubyGems,
				Type:   TypeProxy,
				Name:   "rubygems-proxy",
				Data: &v1alpha1.RubyGemsProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "rubygems-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "yum proxy repository",
			repo: &v1alpha1.NexusRepositorySpec{
				Yum: &v1alpha1.YumSpec{
					Proxy: &v1alpha1.YumProxyRepository{
						ProxySpec: v1alpha1.ProxySpec{
							Name: "yum-proxy",
						},
					},
				},
			},
			want: &RepoData{
				Format: FormatYum,
				Type:   TypeProxy,
				Name:   "yum-proxy",
				Data: &v1alpha1.YumProxyRepository{
					ProxySpec: v1alpha1.ProxySpec{
						Name: "yum-proxy",
					},
				},
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRepoData(tt.repo)

			require.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}
