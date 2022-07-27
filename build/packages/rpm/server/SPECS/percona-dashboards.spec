%global debug_package   %{nil}

%global repo		grafana-dashboards
%global provider	github.com/percona/%{repo}
%global import_path	%{provider}
%global commit		ad4af6808bcd361284e8eb8cd1f36b1e98e32bce
%global shortcommit	%(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         16
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

Name:		percona-dashboards
Version:	%{version}
Release:	%{rpm_release}
Summary:	Grafana dashboards for MySQL and MongoDB monitoring using Prometheus

License:	AGPLv3
URL:		https://%{provider}
Source0:	https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

BuildRequires:	nodejs
Requires:	percona-grafana
Provides:	percona-grafana-dashboards = %{version}-%{release}

%description
This is a set of Grafana dashboards for database and system monitoring
using VictoriaMetrics datasource.
This package is part of Percona Monitoring and Management.


%prep
%setup -q -n %{repo}-%{commit}


%build
node -v
npm version
make release


%install
install -d %{buildroot}%{_datadir}/%{name}/panels/pmm-app
cp -pa ./panels %{buildroot}%{_datadir}/%{name}
cp -pa ./pmm-app/dist %{buildroot}%{_datadir}/%{name}/panels/pmm-app
echo %{version} > %{buildroot}%{_datadir}/%{name}/VERSION


%files
%license LICENSE
%doc README.md LICENSE
%{_datadir}/%{name}


%changelog
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
