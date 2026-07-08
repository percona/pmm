%global debug_package   %{nil}
%global __strip         /bin/true

%global repo		        pmm
%global provider	      github.com/percona/%{repo}
%global commit		      ad4af6808bcd361284e8eb8cd1f36b1e98e32bce
%global shortcommit	    %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         25
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

%define clickhouse_datasource_version 4.17.0
%define polystat_panel_version        2.1.16

%ifarch x86_64
%define plugin_platform linux_amd64
%else
%define plugin_platform linux_arm64
%endif

Name:		  percona-dashboards
Version:	%{version}
Release:	%{rpm_release}
Summary:	Percona dashboards for monitoring

License:	AGPLv3
URL:		  https://%{provider}

BuildRequires:	nodejs
BuildRequires:	unzip
Requires:	percona-grafana

Source0:	https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz
Source1:	https://github.com/grafana/clickhouse-datasource/releases/download/v%{clickhouse_datasource_version}/grafana-clickhouse-datasource-%{clickhouse_datasource_version}.%{plugin_platform}.zip
Source2:	https://github.com/grafana/grafana-polystat-panel/releases/download/v%{polystat_panel_version}/grafana-polystat-panel-%{polystat_panel_version}.zip

%description
This package provides a set of PMM dashboards for database and system monitoring
using VictoriaMetrics datasource.


%prep
%setup -q -n %{repo}-%{commit}


%build
node -v
npm version
make -C dashboards release


%install
install -d %{buildroot}%{_datadir}/%{name}/panels/pmm-app

# cp -a ./dashboards/panels %{buildroot}%{_datadir}/%{name}
cp -a ./dashboards/pmm-app/dist %{buildroot}%{_datadir}/%{name}/panels/pmm-app
unzip -q %{SOURCE1} -d %{buildroot}%{_datadir}/%{name}/panels
unzip -q %{SOURCE2} -d %{buildroot}%{_datadir}/%{name}/panels
echo %{version} > %{buildroot}%{_datadir}/%{name}/VERSION


%files
%license ./dashboards/LICENSE
%doc ./dashboards/README.md
%attr(-,pmm,root) %{_datadir}/%{name}


%changelog
* Mon May 11 2026 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-25
- PMM-15044 Bump clickhouse datasource plugin to 4.17.0

* Sat Apr 18 2026 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-24
- PMM-14944 Bump clickhouse datasource plugin to 4.15.0

* Tue Mar 17 2026 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-23
- PMM-14837 Move dashboards to the monorepo

* Tue Jul 23 2024 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 3.0.0-22
- PMM-13053 Remove /setup page

* Wed Nov 29 2023 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-21
- PMM-12693 Run Grafana as non-root user

* Wed Jul 12 2023 Alex Tymchuk <alexander.tymchuk@percona.com> - 2.39.0-20
- PMM-12231 Set grafana user as owner of plugins directory

* Tue May 16 2023 Oleksii Kysil <oleksii.kysil@ext.percona.com> - 2.38.0-1
- PMM-12118 Skip stripping of plugin binaries

* Thu Jul 28 2022 Alex Tymchuk <alexander.tymchuk@percona.com> - 2.30.0-1
- PMM-10036 migrate to monorepo, part 2

* Mon May 16 2022 Nikita Beletskii <nikita.beletskii@percona.com> - 2.28.0-1
- PMM-10027 remove useless packages

* Sat Nov 06 2021 Nikita Beletskii <nikita.beletskii@percona.com> - 2.25.0-1
- Migrate to grafana provisioning

* Tue Jan 26 2021 Alex Tymchuk <alexander.tymchuk@percona.com> - 2.15.0-15
- PMM-6766 remove qan-app

* Wed Apr 08 2020 Vadim Yalovets <vadim.yalovets@percona.com> - 2.5.0-14
- PMM-5655 remove leftovers of Grafana plugins

* Tue Oct 29 2019 Roman Misyurin <roman.misyurin@percona.com> - 1.9.0-7
- build process fix

* Mon Feb  4 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 1.9.0-6
- PMM-3488 Add some plugins into PMM

* Wed Mar 14 2018 Mykola Marzhan <mykola.marzhan@percona.com> - 1.9.0-5
- use more new node_modules

* Tue Feb 13 2018 Mykola Marzhan <mykola.marzhan@percona.com> - 1.7.0-4
- PMM-2034 compile grafana app

* Mon Nov 13 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 1.5.1-1
- PMM-1771 keep QAN Plugin in dashboards dir

* Mon Nov 13 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 1.5.0-1
- PMM-1680 Include QAN Plugin into PMM

* Thu Feb  2 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 1.1.0-1
- add build_timestamp to Release value

* Thu Dec 15 2016 Mykola Marzhan <mykola.marzhan@percona.com> - 1.0.7-1
- init version
