%global _missing_build_ids_terminate_build 0
%global _dwz_low_mem_die_limit 0

%global repo            pmm
%global provider        github.com/percona/%{repo}
%global commit          8f74cea10d85e441ee88ef4b12bc47bc05165ba9
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define release         2
%define rpm_release     %{release}.%{shortcommit}%{?dist}

# the line below is sed'ed by build/bin/build-server-rpm to set a correct version
%define full_pmm_version 2.0.0

Name:     vmproxy
Version:  %{full_pmm_version}
Release:  %{rpm_release}
Summary:  Percona VMProxy stateless reverse proxy for VictoriaMetrics

License:  AGPLv3
URL:      https://%{provider}
Source0:  https://%{provider}/archive/%{commit}.tar.gz

%description
VMProxy is a stateless reverse proxy which proxies requests to VictoriaMetrics and
optionally adds `extra_filters` query based on the provided configuration.


%prep
%setup -q -n %{repo}-%{commit}


%build
export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}

make -C vmproxy release


%install
install -d -p %{buildroot}%{_sbindir}
install -p -m 0755 ./bin/vmproxy %{buildroot}%{_sbindir}/vmproxy


%files
%license vmproxy/LICENSE
%doc vmproxy/README.md
%{_sbindir}/vmproxy


%changelog
* Mon Apr 1 2024 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-2
- PMM-12899 Use module and build cache

* Mon Dec 5 2022 Michal Kralik <michal.kralik@percona.com> - 2.34.0-1
- Initial release of VMProxy
