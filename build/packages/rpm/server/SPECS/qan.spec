# Go build id is not supported for now.
# https://github.com/rpm-software-management/rpm/issues/367
# https://bugzilla.redhat.com/show_bug.cgi?id=1295951
%undefine _missing_build_ids_terminate_build

%global repo            pmm
%global provider        github.com/percona/%{repo}
%global import_path     %{provider}
# The commit hash gets sed'ed by build-server-rpm script to set a correct version
# see: https://github.com/percona/pmm/blob/main/build/scripts/build-server-rpm#L58
%global commit          0000000000000000000000000000000000000000
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         1
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

# the line below is sed'ed by build/bin/build-server-rpm to set a correct version
%define full_pmm_version 3.0.0

Name:           percona-qan
Version:        %{version}
Release:        %{rpm_release}
Summary:        Query Analytics service for PMM

License:        AGPLv3
URL:            https://%{provider}
Source0:        https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

%description
Query Analytics (QAN) is part of Percona Monitoring and Management (PMM). This
package provides the qan service that replaces qan-api2.
See PMM docs for more information - https://docs.percona.com/percona-monitoring-and-management/3/use/qan/index.html.


%prep
%setup -T -c -n %{repo}-%{version}
%setup -q -c -a 0 -n %{repo}-%{version}
mkdir -p src/github.com/percona
mv %{repo}-%{commit} src/%{provider}


%build
export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}
export PMM_RELEASE_BRANCH=""

cd src/%{provider}/qan
make release


%install

install -d -p %{buildroot}%{_sbindir}
install -p -m 0755 src/%{provider}/bin/qan %{buildroot}%{_sbindir}/%{name}


%files
%attr(0755, root, root) %{_sbindir}/%{name}
%license src/%{provider}/LICENSE
%doc src/%{provider}/qan/README.md

%changelog
* Fri Jun  6 2026 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-1
- Initial qan package: composable rollups + DDSketch percentiles, replaces qan-api2.
