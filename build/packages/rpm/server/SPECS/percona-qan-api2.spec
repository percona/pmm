# Go build id is not supported for now.
# https://github.com/rpm-software-management/rpm/issues/367
# https://bugzilla.redhat.com/show_bug.cgi?id=1295951
%undefine _missing_build_ids_terminate_build

%global repo            qan-api2
%global provider        github.com/percona/%{repo}
%global import_path     %{provider}
%global commit          376dbed06e403faad1b444f99ab3e1e28ac7687e
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         16
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
Percona Query Analytics (QAN) API v2 is part of Percona Monitoring and Management.
See the PMM docs for more information.


%prep
%setup -T -c -n %{repo}-%{version}
%setup -q -c -a 0 -n %{repo}-%{version}
mkdir -p src/github.com/percona
mv %{repo}-%{commit} src/%{provider}


%build
export GOPATH=$(pwd)/

export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}
export PMM_RELEASE_BRANCH=""

cd src/github.com/percona/qan-api2
make release


%install
cd src/github.com/percona/qan-api2

install -d -p %{buildroot}%{_sbindir}
install -p -m 0755 bin/qan-api2 %{buildroot}%{_sbindir}/%{name}


%files
%attr(0755, root, root) %{_sbindir}/%{name}

%changelog
* Mon May 16 2022 Nikita Beletskii <nikita.beletskii@percona.com> - 2.0.0-16
- PMM-10027 remove useless packages

* Thu Jul  2 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 2.0.0-15
- PMM-5645 built using Golang 1.14

* Tue Mar 19 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 2.0.0-4
- PMM-3681 Remove old qan-api and move qan-api2 in feature builds.

* Wed Dec 19 2018 Andrii Skomorokhov <andrii.skomorokhov@percona.com> - 2.0.0-1
- Initial.
