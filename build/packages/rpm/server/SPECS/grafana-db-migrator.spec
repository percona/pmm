%undefine _missing_build_ids_terminate_build

%global release_hash 27b37c412e257bd8a5bcdbcaf52befa3fe555a9d

Name:           grafana-db-migrator
Version:        1.0.8
Release:        2%{?dist}
Summary:        A tool for Grafana database migration
License:        MIT
URL:            https://github.com/percona/grafana-db-migrator
Source0:        https://github.com/percona/grafana-db-migrator/archive/%{release_hash}.tar.gz

%description
%{summary}

%prep
%setup -q -n grafana-db-migrator-%{release_hash}

%build
make

%install
mkdir -p %{buildroot}/usr/sbin/
install -m 755 dist/grafana-db-migrator %{buildroot}%{_sbindir}/


%files
%license LICENSE
%doc README.md
%{_sbindir}/grafana-db-migrator

%changelog
* Fri Oct 13 2023 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 1.0.8-2
- Fix issue with hexed values and folder fixes.

* Mon Feb 13 2023 Nikita Beletskii <2nikita.b@gmail.com> - 1.0.7-1
- Fix issue with convert_from()

* Fri Jan 27 2023 Nikita Beletskii <2nikita.b@gmail.com> - 1.0.6-1
- Fix build

* Tue Jan 17 2023 Nikita Beletskii <2nikita.b@gmail.com> - 1.0.5-1
- Upgrade grafana-db-migrator for Grafana 8 and Grafana 9

* Tue Mar 29 2022 Alex Tymchuk <alexander.tymchuk@percona.com> - 1.0.4-1
- Add README.md and LICENSE files

* Thu Jan 20 2022 Nikita Beletskii <nikita.beletskii@percona.com> - 1.0.3-2
- Add fixes for CHAR fields

* Tue Nov 02 2021 Nikita Beletskii <nikita.beletskii@percona.com> - 1.0.1-1
- Creating package for grafana-db-migrator