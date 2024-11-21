%global _missing_build_ids_terminate_build 0
%global _dwz_low_mem_die_limit 0

%global repo            pmm
%global provider        github.com/percona/%{repo}
%global commit          8f3d007617941033867aea6a134c48b39142427f
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define release         1
%define rpm_release     %{release}.%{shortcommit}%{?dist}

# the line below is sed'ed by build/bin/build-server-rpm to set a correct version
%define full_pmm_version 3.0.0

Name:     pmm-ui
Version:  %{version}
Release:  %{rpm_release}
Summary:  Percona Monitoring and Management UI

License:  AGPLv3
URL:      https://%{provider}
Source0:  https://%{provider}/archive/%{commit}.tar.gz

%description
pmm-ui is the frontend application for pmm-managed.


%prep
%setup -q -n %{repo}-%{commit}

%build
export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}
export PMM_RELEASE_BRANCH=""

make -C ui release

%install
install -d -p %{buildroot}%{_datadir}/%{name}
cp -pa ui/dist/. %{buildroot}%{_datadir}/%{name}

%files
%license ui/LICENSE
%doc ui/README.md
%{_datadir}/%{name}

%changelog
* Fri Mar 22 2024 Matej Kubinec <matej.kubinec@ext.percona.com> - 3.0.0-1
- PMM-11231 add pmm ui
