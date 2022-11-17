%undefine _missing_build_ids_terminate_build

%global repo            VictoriaMetrics
%global provider        github.com/VictoriaMetrics/%{repo}
%global commit          v1.82.1

Name:           percona-victoriametrics-gateway
Version:        1.82.1
Release:        2%{?dist}
Summary:        VictoriaMetrics gateway solution
License:        AGPL-3
URL:            https://%{provider}
Source0:        https://%{provider}/releases/download/%{commit}/vmutils-linux-amd64-%{commit}-enterprise.tar.gz
Source1:        https://raw.githubusercontent.com/percona/pmm/main/LICENSE

%description
%{summary}


%prep
%setup -q -c
cp -p %{SOURCE1} LICENSE

%install
install -D -p -m 0755 ./vmgateway-prod %{buildroot}%{_sbindir}/vmgateway
install -D -p -m 0755 ./LICENSE %{buildroot}%{_sbindir}/LICENSE


%files
%license LICENSE
%{_sbindir}/vmgateway


%changelog
* Mon Nov 14 2022 Michal Kralik <michal.kralik@percona.com> - 1.82.1-2
- AGPL-3 license

* Mon Oct 24 2022 Michal Kralik <michal.kralik@percona.com> - 1.82.1-1
- VictoraMetrics Gateway v1.82.1
