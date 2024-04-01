%global repo            pmm-dump
%global provider        github.com/percona/%{repo}
%global commit          0d49b27729506dc62950f9fa59147d63df194db2
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define release         2
%define rpm_release     %{release}.%{shortcommit}%{?dist}

%if ! 0%{?gobuild:1}
# https://github.com/rpm-software-management/rpm/issues/367
# https://fedoraproject.org/wiki/PackagingDrafts/Go#Build_ID
%define gobuild(o:) go build -ldflags "${LDFLAGS:-} -B 0x$(head -c20 /dev/urandom | od -An -tx1 | tr -d ' \\n')" -a -v -x %{?**};
%endif

Name:		pmm-dump
Version:	0.7.0
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
make build

%install
install -d -p %{buildroot}%{_sbindir}
install -p -m 0755 pmm-dump %{buildroot}%{_sbindir}/pmm-dump

%files
%license LICENSE
%doc README.md
%{_sbindir}/pmm-dump


%changelog
* Mon Apr 1 2024 Alex Demidoff <alexander.demidoff@percona.com> - 0.7.0-2
- PMM-12899 Use module and build cache

* Tue Nov 23 2023 Artem Gavrilov <artem.gavrilov@percona.com> - 0.7.0-ga
- PMM-12460 Update pmm-dump to v0.7.0-ga version

* Tue Mar 29 2022 Alex Tymchuk <alexander.tymchuk@percona.com> - 0.6.0-1
- Initial pmm-dump version
