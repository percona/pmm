%undefine _missing_build_ids_terminate_build
%global _dwz_low_mem_die_limit 0

# do not strip debug symbols
%global debug_package     %{nil}

# The commit hash gets sed'ed by build-server-rpm script to set a correct version
# see: https://github.com/percona/pmm/blob/main/build/scripts/build-server-rpm#L58
%global commit            0000000000000000000000000000000000000000
%define full_pmm_version 2.0.0

%global shortcommit       %(c=%{commit}; echo ${c:0:7})
%define build_timestamp   %(date -u +"%y%m%d%H%M")
%define release           1
%define rpm_release       %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

Name:       dbaas-controller
# the line below is sed'ed by build-server-rpm script to set a correct version
# see: https://github.com/Percona-Lab/pmm-submodules/blob/PMM-2.0/build/bin/build-server-rpm
Version:    %{version}
Release:    %{rpm_release}
Summary:    Simplified API for managing Percona Kubernetes Operators

License:    AGPLv3
URL:        https://github.com/percona/dbaas-controller
Source0:    https://github.com/percona/dbaas-controller/archive/%{commit}/dbaas-controller-%{shortcommit}.tar.gz

%description
dbaas-controller exposes a simplified API for managing Percona Kubernetes Operators
See the PMM docs for more information.


%prep
%setup -q -n dbaas-controller-%{commit}


%build
export COMPONENT_VERSION=%{full_pmm_version}
export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}
export PMM_RELEASE_BRANCH=""

make release

%install
install -d -p %{buildroot}%{_bindir}
install -d -p %{buildroot}%{_sbindir}
install -p -m 0755 bin/dbaas-controller %{buildroot}%{_sbindir}/dbaas-controller


%files
%license LICENSE
%doc README.md
%{_sbindir}/dbaas-controller


%changelog
* Tue Aug  4 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 2.10.0-1
- init version
