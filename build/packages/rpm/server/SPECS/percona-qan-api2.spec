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
%define release         17
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

# the line below is sed'ed by build/bin/build-server-rpm to set a correct version
%define full_pmm_version 2.0.0

Name:           percona-qan-api2
Version:        %{version}
Release:        %{rpm_release}
Summary:        Query Analytics API v2 for PMM

License:        AGPLv3
URL:            https://%{provider}
Source0:        https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

%description
Percona Query Analytics (QAN) API v2 is part of Percona Monitoring and Management (PMM).
See PMM docs for more information - https://docs.percona.com/percona-monitoring-and-management/using/query-analytics.html.


%prep
%setup -T -c -n %{repo}-%{version}
%setup -q -c -a 0 -n %{repo}-%{version}
mkdir -p src/github.com/percona
mv %{repo}-%{commit} src/%{provider}


%build
export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}
export PMM_RELEASE_BRANCH=""

cd src/%{provider}/qan-api2
make release


%install

install -d -p %{buildroot}%{_sbindir}
install -p -m 0755 src/%{provider}/bin/qan-api2 %{buildroot}%{_sbindir}/%{name}


%files
%attr(0755, root, root) %{_sbindir}/%{name}
%license src/%{provider}/qan-api2/LICENSE
%doc src/%{provider}/qan-api2/README.md

%changelog
* Mon Nov  7 2022 Alexander Tymchuk <alexander.tymchuk@percona.com> - 2.0.0-17
- PMM-10117 migrate QAN API to monorepo

* Mon May 16 2022 Nikita Beletskii <nikita.beletskii@percona.com> - 2.0.0-16
- PMM-10027 remove useless packages

* Thu Jul  2 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 2.0.0-15
- PMM-5645 built using Golang 1.14

* Tue Mar 19 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 2.0.0-4
- PMM-3681 Remove old qan-api and move qan-api2 in feature builds.

* Wed Dec 19 2018 Andrii Skomorokhov <andrii.skomorokhov@percona.com> - 2.0.0-1
- Initial.
