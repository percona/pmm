%undefine _missing_build_ids_terminate_build
%global _dwz_low_mem_die_limit 0

%global repo            pmm
%global provider        github.com/percona/%{repo}
%global commit          8f3d007617941033867aea6a134c48b39142427f
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         1
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

# the line below is sed'ed by build/bin/build-server-rpm to set a correct version
%define full_pmm_version 2.0.0

Name:		aichat-backend
Version:	%{full_pmm_version}
Release:	%{rpm_release}
Summary:	Percona Monitoring and Management AI Chat Backend

License:	AGPLv3
URL:		https://%{provider}
Source0:	https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

%description
AI Chat Backend for Percona Monitoring and Management (PMM). This service provides
AI-powered chat functionality with support for multiple LLM providers (OpenAI, 
Google Gemini, Anthropic Claude, Ollama) and Model Context Protocol (MCP) tools
for database monitoring and management tasks.

%prep
%setup -q -n pmm-%{commit}
mkdir -p src/github.com/percona
ln -s $(pwd) src/%{provider}

%build
export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}
export PMM_RELEASE_BRANCH=""

cd src/github.com/percona/pmm/aichat-backend
make build

%install
install -d -p %{buildroot}%{_sbindir}
install -d -p %{buildroot}%{_sysconfdir}/%{name}

# Install binary
install -p -m 0755 src/github.com/percona/pmm/aichat-backend/bin/aichat-backend %{buildroot}%{_sbindir}/aichat-backend

# Install configuration files
install -p -m 0644 src/github.com/percona/pmm/aichat-backend/config.yaml %{buildroot}%{_sysconfdir}/%{name}/config.yaml



%files
%license src/%{provider}/LICENSE
%doc src/%{provider}/aichat-backend/README.md
%doc src/%{provider}/aichat-backend/ARCHITECTURE.md
%{_sbindir}/aichat-backend
%dir %{_sysconfdir}/%{name}
%config(noreplace) %{_sysconfdir}/%{name}/config.yaml

%changelog
* Wed Dec 18 2024 AI Assistant <ai@percona.com> - 1.0.0-1
- Initial RPM package for PMM AI Chat Backend
- Support for multiple LLM providers (OpenAI, Google Gemini, Anthropic Claude, Ollama)
- Model Context Protocol (MCP) integration for database tools
- RESTful API with streaming chat support
- File upload and processing capabilities
- Comprehensive logging and monitoring 