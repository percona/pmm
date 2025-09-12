%global repo            pmm
%global provider        github.com/percona/%{repo}
%global import_path     %{provider}
# The commit hash gets sed'ed by build-server-rpm script
%global commit          0000000000000000000000000000000000000000
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define release         18
%define rpm_release     %{release}.%{shortcommit}%{?dist}

# The line below is sed'ed by build-server-rpm
%define full_pmm_version 2.0.0

Name:           pmm-qan-api
Version:        %{version}
Release:        %{rpm_release}
Summary:        Query Analytics API for PMM

License:        AGPLv3
URL:            https://%{provider}
Source0:        https://%{provider}/archive/%{commit}.tar.gz

%description
Percona Query Analytics (QAN) API is part of Percona Monitoring and Management (PMM).
Refer to PMM docs for more information - https://docs.percona.com/percona-monitoring-and-management/get-started/query-analytics.html.


%prep
%setup -q -n %{repo}-%{commit}


%build
export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}

make -C qan-api2 release


%install

install -d -p %{buildroot}%{_sbindir}
install -p -m 0755 ./bin/qan-api2 %{buildroot}%{_sbindir}/%{name}


%files
%attr(0755, root, root) %{_sbindir}/%{name}
%license qan-api2/LICENSE
%doc qan-api2/README.md

%changelog
* Mon Apr 1 2024 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-18
- PMM-12899 Use module and build cache

* Mon Nov 7 2022 Alexander Tymchuk <alexander.tymchuk@percona.com> - 2.0.0-17
- PMM-10117 migrate QAN API to monorepo

* Mon May 16 2022 Nikita Beletskii <nikita.beletskii@percona.com> - 2.0.0-16
- PMM-10027 remove useless packages

* Thu Jul 2 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 2.0.0-15
- PMM-5645 built using Golang 1.14

* Tue Mar 19 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 2.0.0-4
- PMM-3681 Remove old qan-api and move qan-api2 in feature builds.

* Wed Dec 19 2018 Andrii Skomorokhov <andrii.skomorokhov@percona.com> - 2.0.0-1
- Initial.
