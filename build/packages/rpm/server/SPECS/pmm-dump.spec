%global _missing_build_ids_terminate_build 0

%global repo            pmm-dump
%global provider        github.com/percona/%{repo}
%global commit          f226dbb3afb62ac4b9b39032935b5694a48d526f
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define release         2
%define rpm_release     %{release}.%{shortcommit}%{?dist}

Name:     pmm-dump
Version:  3.0.0
Release:  %{rpm_release}
Summary:  Percona PMM Dump allows to export and import monitoring metrics and query analytics.

License:  AGPLv3
URL:      https://%{provider}
Source0:  https://%{provider}/archive/%{commit}.tar.gz

%description
%{summary}

%prep
%setup -q -n %{repo}-%{commit}

%build
make build

%install
install -d -p %{buildroot}%{_sbindir}
install -p -m 0755 pmm-dump %{buildroot}%{_sbindir}/pmm-dump

%files
%license LICENSE
%doc README.md
%{_sbindir}/pmm-dump


%changelog
* Thu Aug 8 2024 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-1
- PMM-13282 Migrate pmm-dump to v3 API

* Mon Apr 1 2024 Alex Demidoff <alexander.demidoff@percona.com> - 0.7.0-2
- PMM-12899 Use module and build cache

* Thu Nov 23 2023 Artem Gavrilov <artem.gavrilov@percona.com> - 0.7.0-ga
- PMM-12460 Update pmm-dump to v0.7.0-ga version

* Tue Mar 29 2022 Alex Tymchuk <alexander.tymchuk@percona.com> - 0.6.0-1
- Initial pmm-dump version
