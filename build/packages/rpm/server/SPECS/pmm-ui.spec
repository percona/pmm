%global debug_package   %{nil}

%global repo            pmm
%global provider        github.com/percona/%{repo}
%global commit          840c96176c13a533b4fb3cc6788767e736f01279
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         1
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

Name:		pmm-ui
Version:	%{version}
Release:	%{rpm_release}
Summary:	Percona Monitoring and Management web UI

License:	Apache-2.0
URL:	    https://%{provider}
Source0:	https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

%description
This package provides the PMM web UI (the pmm application) and the pmm-compat
Grafana panel plugin, both built from the PMM monorepo UI workspace.


%prep
%setup -q -n %{repo}-%{commit}


%build
node -v
make -C ui release


%install
install -d -p %{buildroot}%{_datadir}/pmm-ui
install -d -p %{buildroot}%{_datadir}/percona-dashboards/panels/pmm-compat-app
cp -pa ./ui/apps/pmm/dist/. %{buildroot}%{_datadir}/pmm-ui
cp -pa ./ui/apps/pmm-compat/dist/. %{buildroot}%{_datadir}/percona-dashboards/panels/pmm-compat-app


%files
%license ./LICENSE
%doc ./README.md
%attr(-, pmm, root) %{_datadir}/pmm-ui
%attr(-, pmm, root) %{_datadir}/percona-dashboards/panels/pmm-compat-app


%changelog
* Wed Jul 22 2026 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-1
- PMM-13776 Split the PMM UI into its own package to cache its artifacts
