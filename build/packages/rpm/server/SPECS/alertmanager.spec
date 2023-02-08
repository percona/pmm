%undefine _missing_build_ids_terminate_build

%global repo            alertmanager
%global provider        github.com/prometheus/%{repo}
%global commit          258fab7cdd551f2cf251ed0348f0ad7289aee789
%global shortcommit     %(c=%{commit}; echo ${c:0:7})

Name:           percona-%{repo}
Version:        0.25.0
Release:        3%{?dist}
Summary:        The Prometheus monitoring system and time series database
License:        ASL 2.0
URL:            https://%{provider}
Source0:        https://%{provider}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

%description
%{summary}

%description
%{summary}

%prep
%setup -q -n %{repo}-%{commit}
mkdir -p ./build/src/github.com/prometheus
ln -s $(pwd) ./build/src/github.com/prometheus/alertmanager

%build
export GOPATH="$(pwd)/build"
export CGO_ENABLED=0
export USER=builder

cd build/src/github.com/prometheus/alertmanager
make build

%install
install -D -p -m 0755 ./%{repo}  %{buildroot}%{_sbindir}/%{repo}
install -D -p -m 0755 ./amtool %{buildroot}%{_bindir}/amtool
install -d %{buildroot}%{_datadir}/%{repo}
install -d %{buildroot}%{_sharedstatedir}/%{repo}

%files
%doc LICENSE CHANGELOG.md README.md NOTICE
%{_sbindir}/%{repo}
%{_bindir}/amtool
%{_datadir}/%{repo}
%dir %attr(-, nobody, nobody) %{_sharedstatedir}/%{repo}

%changelog
* Wed Feb  8 2023 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 0.25.0
- PMM-11497 Upgrade AlertManager to 0.25

* Tue May  4 2021 David Mikus <david.mikus.sde@gmail.com> - 0.21.0
- PMM-7302 Upgrade AlertManager to 0.21

* Thu Jul  2 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 0.20.0-3
- PMM-5645 built using Golang 1.14

* Fri Mar 27 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 0.20.0
- Init version
