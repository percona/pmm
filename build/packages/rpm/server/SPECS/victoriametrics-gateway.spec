%undefine _missing_build_ids_terminate_build

%define copying() \
%if 0%{?fedora} >= 21 || 0%{?rhel} >= 7 \
%license %{*} \
%else \
%doc %{*} \
%endif

%global repo            VictoriaMetrics
%global provider        github.com/VictoriaMetrics/%{repo}
%global commit          v1.82.1

Name:           percona-victoriametrics-gateway
Version:        1.82.1
Release:        1%{?dist}
Summary:        VictoriaMetrics gateway solution
License:        Apache-2.0
URL:            https://%{provider}
Source0:        https://%{provider}/releases/download/%{commit}/vmutils-linux-amd64-%{commit}-enterprise.tar.gz


%description
%{summary}


%prep
%setup -q -c


%install
install -D -p -m 0755 ./vmgateway-prod %{buildroot}%{_sbindir}/vmgateway


%files
%{_sbindir}/vmgateway


%changelog
* Mon Oct 24 2022 Michal Kralik <michal.kralik@percona.com> - 1.82.1
- VictoraMetrics Gateway v1.82.1
