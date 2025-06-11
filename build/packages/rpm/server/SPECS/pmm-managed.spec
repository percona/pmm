%undefine _missing_build_ids_terminate_build
%global _dwz_low_mem_die_limit 0

%global repo            pmm
%global provider        github.com/percona/%{repo}
%global commit          8f3d007617941033867aea6a134c48b39142427f
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         20
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

# the line below is sed'ed by build/bin/build-server-rpm to set a correct version
%define full_pmm_version 2.0.0

Name:		pmm-managed
Version:	%{version}
Release:	%{rpm_release}
Summary:	Percona Monitoring and Management management daemon

License:	AGPLv3
URL:		  https://%{provider}
Source0:	https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

%description
pmm-managed manages configuration of PMM server components (VictoriaMetrics,
Grafana, etc.) and exposes API for that. Those APIs are used by pmm-admin tool.
See PMM docs for more information.


%prep
%setup -q -n pmm-%{commit}
mkdir -p src/github.com/percona
ln -s $(pwd) src/%{provider}


%build

export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}
export PMM_RELEASE_BRANCH=""

cd src/github.com/percona/pmm/managed
make release

cd ../ui
make release

%install
install -d -p %{buildroot}%{_bindir}
install -d -p %{buildroot}%{_sbindir}
install -d -p %{buildroot}%{_datadir}/%{name}
install -d -p %{buildroot}%{_datadir}/pmm-ui
install -d -p %{buildroot}/usr/local/percona/pmm/advisors
install -d -p %{buildroot}/usr/local/percona/pmm/checks
install -p -m 0755 bin/pmm-managed %{buildroot}%{_sbindir}/pmm-managed
install -p -m 0755 bin/pmm-encryption-rotation %{buildroot}%{_sbindir}/pmm-encryption-rotation
install -p -m 0755 bin/pmm-managed-init %{buildroot}%{_sbindir}/pmm-managed-init
install -p -m 0755 bin/pmm-managed-starlark %{buildroot}%{_sbindir}/pmm-managed-starlark

cd src/github.com/percona/pmm
cp -pa ./api/swagger %{buildroot}%{_datadir}/%{name}
cp -pa ./ui/dist/. %{buildroot}%{_datadir}/pmm-ui
cp -pa ./managed/data/advisors/*.yaml %{buildroot}/usr/local/percona/pmm/advisors/
cp -pa ./managed/data/checks/*.yaml %{buildroot}/usr/local/percona/pmm/checks/

%files
%license src/%{provider}/LICENSE
%doc src/%{provider}/README.md
%{_sbindir}/pmm-managed
%{_sbindir}/pmm-encryption-rotation
%{_sbindir}/pmm-managed-init
%{_sbindir}/pmm-managed-starlark
%{_datadir}/%{name}
%{_datadir}/pmm-ui
%{buildroot}/local/percona/pmm/advisors/*.yaml
%{buildroot}/local/percona/pmm/checks/*.yaml

%changelog
* Wed Jun 11 2025 Michael Okoko <michael.okoko@percona.com> - 3.4.0-1
- PMM-14009 bundle advisors with PMM.

* Mon Sep 23 2024 Jiri Ctvrtka <jiri.ctvrtka@ext.percona.com> - 3.0.0-1
- PMM-13132 add PMM encryption rotation tool

* Fri Mar 22 2024 Matej Kubinec <matej.kubinec@ext.percona.com> - 3.0.0-1
- PMM-11231 add pmm ui

* Thu Jul 28 2022 Alex Tymchuk <alexander.tymchuk@percona.com> - 2.30.0-1
- PMM-10036 migrate to monorepo

* Fri Jun 17 2022 Anton Bystrov <anton.bystrov@simbirsoft.com> - 2.0.0-17
- PMM-10206 merge pmm-managed to monorepo pmm

* Thu Jul  2 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 2.0.0-17
- PMM-5645 built using Golang 1.14

* Tue May 12 2020 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.0.0-16
- added pmm-managed-starlark

* Tue Feb 11 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 2.0.0-14
- added pmm-managed-init

* Thu Sep  5 2019 Viacheslav Sarzhan <slava.sarzhan@percona.com> - 2.0.0-10
- init version
