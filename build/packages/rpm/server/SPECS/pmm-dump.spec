%undefine _missing_build_ids_terminate_build

%global repo            pmm-dump
%global provider        github.com/percona/%{repo}
%global commit          56cc93fe8bf811ee69ab39c0d1b22398e126a8a2
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         1
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

Name:		pmm-dump
Version:	3.4.0
Release:	%{rpm_release}
Summary:	Percona PMM Dump allows to export and import monitoring metrics and query analytics.

License:	AGPLv3
URL:		https://%{provider}
Source0:	https://%{provider}/archive/%{commit}.tar.gz

%description
%{summary}

%prep
%setup -q -n %{repo}-%{commit}

%build
make build BRANCH="main" COMMIT="%{shortcommit}" VERSION="%{version}"

%install
install -d -p %{buildroot}%{_sbindir}
install -p -m 0755 pmm-dump %{buildroot}%{_sbindir}/pmm-dump

%files
%license LICENSE
%doc README.md
%{_sbindir}/pmm-dump


%changelog
* Thu Jul 28 2025 Michael Okoko <michael.okoko@percona.com> - 3.4.0
- PMM-14215 Default to main branch for pmm-dump
- PMM-14085 Fix an issue where pmm-dump would not export OS metrics when used via the PMM GUI

* Thu Aug 8 2024 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-1
- PMM-13282 Migrate pmm-dump to v3 API

* Tue Nov 23 2023 Artem Gavrilov <artem.gavrilov@percona.com> - 0.7.0-ga
- PMM-12460 Update pmm-dump to v0.7.0-ga version

* Tue Mar 29 2022 Alex Tymchuk <alexander.tymchuk@percona.com> - 0.6.0-1
- Initial pmm-dump version
