package nexus

import (
	"errors"

	"github.com/epam/edp-nexus-operator/api/v1alpha1"
)

const (
	TypeHosted = "hosted"
	TypeProxy  = "proxy"
	TypeGroup  = "group"

	FormatApt       = "apt"
	FormatBower     = "bower"
	FormatCocoapods = "cocoapods"
	FormatConan     = "conan"
	FormatConda     = "conda"
	FormatDocker    = "docker"
	FormatGitLfs    = "gitlfs"
	FormatGo        = "go"
	FormatHelm      = "helm"
	FormatMaven     = "maven"
	FormatNpm       = "npm"
	FormatNuget     = "nuget"
	FormatP2        = "p2"
	FormatPypi      = "pypi"
	FormatR         = "r"
	FormatRaw       = "raw"
	FormatRubyGems  = "rubygems"
	FormatYum       = "yum"
)

type RepoData struct {
	Type   string
	Format string
	Name   string
	Data   interface{}
}

// GetRepoData returns repository data based on NexusRepositorySpec.
// nolint:funlen,cyclop  // We can skip this some rules here because we have a lot of simple conditions.
func GetRepoData(repo *v1alpha1.NexusRepositorySpec) (*RepoData, error) {
	if repo.Apt != nil {
		if repo.Apt.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatApt,
				Name:   repo.Apt.Hosted.Name,
				Data:   repo.Apt.Hosted,
			}, nil
		}

		if repo.Apt.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatApt,
				Name:   repo.Apt.Proxy.Name,
				Data:   repo.Apt.Proxy,
			}, nil
		}

		return nil, errors.New("no apt repository set")
	}

	if repo.Bower != nil {
		if repo.Bower.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatBower,
				Name:   repo.Bower.Hosted.Name,
				Data:   repo.Bower.Hosted,
			}, nil
		}

		if repo.Bower.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatBower,
				Name:   repo.Bower.Proxy.Name,
				Data:   repo.Bower.Proxy,
			}, nil
		}

		if repo.Bower.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatBower,
				Name:   repo.Bower.Group.Name,
				Data:   repo.Bower.Group,
			}, nil
		}

		return nil, errors.New("no bower repository set")
	}

	if repo.Cocoapods != nil {
		if repo.Cocoapods.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatCocoapods,
				Name:   repo.Cocoapods.Proxy.Name,
				Data:   repo.Cocoapods.Proxy,
			}, nil
		}

		return nil, errors.New("no cocoapods repository set")
	}

	if repo.Conan != nil {
		if repo.Conan.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatConan,
				Name:   repo.Conan.Proxy.Name,
				Data:   repo.Conan.Proxy,
			}, nil
		}

		return nil, errors.New("no conan repository set")
	}

	if repo.Conda != nil {
		if repo.Conda.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatConda,
				Name:   repo.Conda.Proxy.Name,
				Data:   repo.Conda.Proxy,
			}, nil
		}

		return nil, errors.New("no conda repository set")
	}

	if repo.Docker != nil {
		if repo.Docker.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatDocker,
				Name:   repo.Docker.Hosted.Name,
				Data:   repo.Docker.Hosted,
			}, nil
		}

		if repo.Docker.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatDocker,
				Name:   repo.Docker.Proxy.Name,
				Data:   repo.Docker.Proxy,
			}, nil
		}

		if repo.Docker.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatDocker,
				Name:   repo.Docker.Group.Name,
				Data:   repo.Docker.Group,
			}, nil
		}

		return nil, errors.New("no docker repository set")
	}

	if repo.GitLfs != nil {
		if repo.GitLfs.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatGitLfs,
				Name:   repo.GitLfs.Hosted.Name,
				Data:   repo.GitLfs.Hosted,
			}, nil
		}

		return nil, errors.New("no gitlfs repository set")
	}

	if repo.Go != nil {
		if repo.Go.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatGo,
				Name:   repo.Go.Proxy.Name,
				Data:   repo.Go.Proxy,
			}, nil
		}

		if repo.Go.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatGo,
				Name:   repo.Go.Group.Name,
				Data:   repo.Go.Group,
			}, nil
		}

		return nil, errors.New("no go repository set")
	}

	if repo.Helm != nil {
		if repo.Helm.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatHelm,
				Name:   repo.Helm.Hosted.Name,
				Data:   repo.Helm.Hosted,
			}, nil
		}

		if repo.Helm.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatHelm,
				Name:   repo.Helm.Proxy.Name,
				Data:   repo.Helm.Proxy,
			}, nil
		}

		return nil, errors.New("no helm repository set")
	}

	if repo.Maven != nil {
		if repo.Maven.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatMaven,
				Name:   repo.Maven.Hosted.Name,
				Data:   repo.Maven.Hosted,
			}, nil
		}

		if repo.Maven.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatMaven,
				Name:   repo.Maven.Proxy.Name,
				Data:   repo.Maven.Proxy,
			}, nil
		}

		if repo.Maven.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatMaven,
				Name:   repo.Maven.Group.Name,
				Data:   repo.Maven.Group,
			}, nil
		}

		return nil, errors.New("no maven repository set")
	}

	if repo.Npm != nil {
		if repo.Npm.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatNpm,
				Name:   repo.Npm.Hosted.Name,
				Data:   repo.Npm.Hosted,
			}, nil
		}

		if repo.Npm.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatNpm,
				Name:   repo.Npm.Proxy.Name,
				Data:   repo.Npm.Proxy,
			}, nil
		}

		if repo.Npm.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatNpm,
				Name:   repo.Npm.Group.Name,
				Data:   repo.Npm.Group,
			}, nil
		}

		return nil, errors.New("no npm repository set")
	}

	if repo.Nuget != nil {
		if repo.Nuget.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatNuget,
				Name:   repo.Nuget.Hosted.Name,
				Data:   repo.Nuget.Hosted,
			}, nil
		}

		if repo.Nuget.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatNuget,
				Name:   repo.Nuget.Proxy.Name,
				Data:   repo.Nuget.Proxy,
			}, nil
		}

		if repo.Nuget.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatNuget,
				Name:   repo.Nuget.Group.Name,
				Data:   repo.Nuget.Group,
			}, nil
		}

		return nil, errors.New("no nuget repository set")
	}

	if repo.P2 != nil {
		if repo.P2.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatP2,
				Name:   repo.P2.Proxy.Name,
				Data:   repo.P2.Proxy,
			}, nil
		}

		return nil, errors.New("no p2 repository set")
	}

	if repo.Pypi != nil {
		if repo.Pypi.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatPypi,
				Name:   repo.Pypi.Proxy.Name,
				Data:   repo.Pypi.Proxy,
			}, nil
		}

		if repo.Pypi.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatPypi,
				Name:   repo.Pypi.Hosted.Name,
				Data:   repo.Pypi.Hosted,
			}, nil
		}

		if repo.Pypi.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatPypi,
				Name:   repo.Pypi.Group.Name,
				Data:   repo.Pypi.Group,
			}, nil
		}

		return nil, errors.New("no pypi repository set")
	}

	if repo.R != nil {
		if repo.R.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatR,
				Name:   repo.R.Hosted.Name,
				Data:   repo.R.Hosted,
			}, nil
		}

		if repo.R.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatR,
				Name:   repo.R.Proxy.Name,
				Data:   repo.R.Proxy,
			}, nil
		}

		if repo.R.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatR,
				Name:   repo.R.Group.Name,
				Data:   repo.R.Group,
			}, nil
		}

		return nil, errors.New("no r repository set")
	}

	if repo.Raw != nil {
		if repo.Raw.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatRaw,
				Name:   repo.Raw.Hosted.Name,
				Data:   repo.Raw.Hosted,
			}, nil
		}

		if repo.Raw.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatRaw,
				Name:   repo.Raw.Proxy.Name,
				Data:   repo.Raw.Proxy,
			}, nil
		}

		if repo.Raw.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatRaw,
				Name:   repo.Raw.Group.Name,
				Data:   repo.Raw.Group,
			}, nil
		}

		return nil, errors.New("no raw repository set")
	}

	if repo.RubyGems != nil {
		if repo.RubyGems.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatRubyGems,
				Name:   repo.RubyGems.Hosted.Name,
				Data:   repo.RubyGems.Hosted,
			}, nil
		}

		if repo.RubyGems.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatRubyGems,
				Name:   repo.RubyGems.Proxy.Name,
				Data:   repo.RubyGems.Proxy,
			}, nil
		}

		if repo.RubyGems.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatRubyGems,
				Name:   repo.RubyGems.Group.Name,
				Data:   repo.RubyGems.Group,
			}, nil
		}

		return nil, errors.New("no rubygems repository set")
	}

	if repo.Yum != nil {
		if repo.Yum.Hosted != nil {
			return &RepoData{
				Type:   TypeHosted,
				Format: FormatYum,
				Name:   repo.Yum.Hosted.Name,
				Data:   repo.Yum.Hosted,
			}, nil
		}

		if repo.Yum.Proxy != nil {
			return &RepoData{
				Type:   TypeProxy,
				Format: FormatYum,
				Name:   repo.Yum.Proxy.Name,
				Data:   repo.Yum.Proxy,
			}, nil
		}

		if repo.Yum.Group != nil {
			return &RepoData{
				Type:   TypeGroup,
				Format: FormatYum,
				Name:   repo.Yum.Group.Name,
				Data:   repo.Yum.Group,
			}, nil
		}

		return nil, errors.New("no yum repository set")
	}

	return nil, errors.New("no repository set")
}
