%global debug_package   %{nil}
%global commit          ae7b461382be0d9b9acd0022398369bf313cc6f8
%global shortcommit     %(c=%{commit}; echo ${c:0:7})
%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         107
%define grafana_version 11.1.8
%define full_pmm_version 2.0.0
%define full_version    v%{grafana_version}-%{full_pmm_version}
%define rpm_release     %{release}.%{build_timestamp}.%{shortcommit}%{?dist}

%if ! 0%{?gobuild:1}
%define gobuild(o:) go build -ldflags "${LDFLAGS:-} -B 0x$(head -c20 /dev/urandom|od -An -tx1|tr -d ' \\n')" -a -v -x %{?**};
%endif

Name:           percona-grafana
Version:        %{grafana_version}
Release:        %{rpm_release}
Summary:        Grafana is an open source, feature rich metrics dashboard and graph editor
License:        AGPLv3
URL:            https://github.com/percona/grafana
Source0:        https://github.com/percona/grafana/archive/%{commit}.tar.gz

BuildRequires: fontconfig
%if 0%{?rhel} < 9
BuildRequires: nodejs-grunt-cli
%endif

%description
Grafana is an open source, feature rich metrics dashboard and graph editor for
Graphite, InfluxDB & OpenTSDB.

%prep
%setup -q -n grafana-%{commit}
rm -rf Godeps
sed -i "s/unknown-dev/%{grafana_version}/" pkg/build/git.go
%if 0%{?rhel} >= 9
    sudo npm install -g grunt-cli
%endif

%build
mkdir -p _build/src
export GOPATH="$(pwd)/_build"

make build-go

make deps-js
make build-js

%install
install -d -p %{buildroot}%{_datadir}/grafana
cp -rpav conf %{buildroot}%{_datadir}/grafana
cp -rpav public %{buildroot}%{_datadir}/grafana
cp -rpav tools %{buildroot}%{_datadir}/grafana

install -d -p %{buildroot}%{_sbindir}
cp bin/linux-amd64/grafana-server %{buildroot}%{_sbindir}/
cp bin/linux-amd64/grafana %{buildroot}%{_sbindir}/
install -d -p %{buildroot}%{_bindir}
cp bin/linux-amd64/grafana-cli %{buildroot}%{_bindir}/

install -d -p %{buildroot}%{_sysconfdir}/grafana
cp conf/sample.ini %{buildroot}%{_sysconfdir}/grafana/grafana.ini
mv conf/ldap.toml %{buildroot}%{_sysconfdir}/grafana/
install -d -p %{buildroot}%{_sharedstatedir}/grafana

%files
%defattr(-, pmm, pmm, -)
%{_datadir}/grafana
%doc CHANGELOG.md README.md
%license LICENSE
%attr(0755, pmm, pmm) %{_sbindir}/grafana
%attr(0755, pmm, pmm) %{_sbindir}/grafana-server
%attr(0755, pmm, pmm) %{_bindir}/grafana-cli
%{_sysconfdir}/grafana/grafana.ini
%{_sysconfdir}/grafana/ldap.toml
%dir %{_sharedstatedir}/grafana

%pre
getent group pmm >/dev/null || echo "Group pmm does not exist. Please create it manually."
getent passwd pmm >/dev/null || echo "User pmm does not exist. Please create it manually."
exit 0

%changelog
* Thu Oct 24 2024 Yash Sartanpara <yash.sartanpara@ext.percona.com> - 11.1.8-1
- PMM-13449 Grafana 11.1.8

* Fri Sep 06 2024 Matej Kubinec <matej.kubinec@ext.percona.com> - 11.1.5-1
- PMM-13235 Grafana 11.1.5

* Thu Aug 15 2024 Matej Kubinec <matej.kubinec@ext.percona.com> - 11.1.4-1
- PMM-13235 Grafana 11.1.4

* Wed Jul 17 2024 Matej Kubinec <matej.kubinec@ext.percona.com> - 11.1.0-1
- PMM-13235 Grafana 11.1.0

* Tue Apr 16 2024 Matej Kubinec <matej.kubinec@ext.percona.com> - 10.4.2-1
- PMM-13059 Grafana 10.4.2

* Tue Mar 12 2024 Matej Kubinec <matej.kubinec@ext.percona.com> - 10.4.0-1
- PMM-12991 Grafana 10.4.0

* Tue Jan 16 2024 Matej Kubinec <matej.kubinec@ext.percona.com> - 10.2.3-1
- PMM-12314 Grafana 10.2.3

* Mon Nov 27 2023 Alex Demidoff <alexander.demidoff@percona.com> - 9.2.20-2
- PMM-12693 Run Grafana as non-root user

* Tue Jun 27 2023 Matej Kubinec <matej.kubinec@ext.percona.com> - 9.2.20-1
- PMM-12254 Grafana 9.2.20

* Thu May 18 2023 Matej Kubinec <matej.kubinec@ext.percona.com> - 9.2.18-1
- PMM-12114 Grafana 9.2.18

* Fri Mar 10 2023 Matej Kubinec <matej.kubinec@ext.percona.com> - 9.2.13-1
- PMM-11762 Grafana 9.2.13

* Tue Nov 29 2022 Alex Tymchuk <alexander.tymchuk@percona.com> - 9.2.5-1
- PMM-10881 Grafana 9.2.5

* Mon May 16 2022 Nikita Beletskii <nikita.beletskii@percona.com> - 8.3.5-2
- PMM-10027 remove useless packages

* Mon Apr 11 2022 Nikita Beletskii <nikita.beletskii@percona.com> - 8.3.5-1
- PMM-7 Fix grafana version

* Fri Jan  14 2022 Nikita Beletskii <nikita.beletskii@percona.com> - 7.5.11-2
- PMM-9452 Remove useless operation in grafana spec

* Wed Oct  6 2021 Tiago Santos <tiago.mota@percona.com> - 7.5.11-1
- PMM-8967 Update grafana to version 7.5.11

* Sat Jun 12 2021 Alex Tymchuk <alexander.tymchuk@percona.com> - 7.5.7-1
- PMM-7809 Update grafana to version 7.5.7

* Thu Feb 18 2021 Roman Misyurin <roman.misyurin@percona.com> - 7.3.7-92
- PMM-6695 Update grafana to version 7.3.7

* Thu Feb 11 2021 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 7.1.3-88
- PMM-6693 Fix grafana build in FB

* Wed Feb 10 2021 Nicola Lamacchia <nicola.lamacchia@percona.com> - 7.1.3-87
- PMM-6924 Page breadcrumb component

* Wed Jan 20 2021 Tiago Santos <tiago.mota@percona.com> - 7.1.3-70
- PMM-7282 Create rule without channels and filters

* Mon Dec 28 2020 Tiago Santos <tiago.mota@percona.com> - 7.1.3-48
- PMM-7005 Alert rule enable disable

* Tue Nov 17 2020 Nicola Lamacchia <nicola.lamacchia@percona.com> - 7.1.3-7
- PMM-6872 add an Integrated Alerting section

* Tue Aug 18 2020 Vadim Yalovets <vadim.yalovets@percona.com> - 7.1.3-1
- PMM-6360 grafana upgrade to 7.1.x build changes

* Thu Jul  2 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 6.7.4-2
- PMM-5645 Built using Golang 1.14

* Thu Jun  4 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 6.7.4-1
- PMM-6009 Update Grafana to 6.7.4; Fixes CVE-2020-13379

* Thu May 21 2020 Vadim Yalovets <vadim.yalovets@percona.com> - 6.7.3-3
- PMM-5906 Remove Update page

* Wed May  6 2020 Vadim Yalovets <vadim.yalovets@percona.com> - 6.7.3-2
- PMM-5882 Delete Snapshot throws an Error

* Wed Apr 29 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 6.7.3-1
- PMM-5549 update Grafana v.6.7.3

* Mon Mar 23 2020 Alexander Tymchuk <alexander.tymchuk@percona.com> - 6.5.1-4
- PMM-4252 Better resolution favicon

* Wed Feb  5 2020 Vadim Yalovets  <vadim.yalovets@percona.com> - 6.5.1-2
- PMM-5251 Last two rows are not visible when scrolling data tables

* Mon Dec  9 2019 Vadim Yalovets  <vadim.yalovets@percona.com> - 6.5.1-1
- PMM-5087 update Grafana v.6.5.1

* Tue Nov 19 2019 Vadim Yalovets  <vadim.yalovets@percona.com> - 6.4.4-1
- PMM-4969 update Grafana v.6.4.4

* Wed Sep 18 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 6.3.5-2
- Remove old patches.

* Wed Sep  4 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.3.5-1
- PMM-4592 Grafana v6.3.5

* Thu Aug 22 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.3.3-1
- PMM-4560 Update to Grafana v.6.3.3

* Fri Aug 09 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.3.2-1
- PMM-4491 Grafana v6.3.2

* Fri Jul 05 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.2.5-1
- PMM-4303 Grafana v6.2.5

* Tue Jun 25 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.2.4-1
- PMM-4248 Grafana v6.2.4

* Thu Jun 13 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.2.2-1
- PMM-4141 Grafana v6.2.1

* Wed May  1 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.1.6-1
- PMM-3969 Grafana 6.1.6

* Fri Apr 26 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.1.4-1
- PMM-3936 Grafana v6.1.4

* Wed Apr 10 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.1.3-1
- PMM-3806 Grafana 6.1.2 update

* Tue Apr  9 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.1.2-1
- PMM-3806 Grafana 6.1.2 update

* Thu Apr  4 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.1.0-1
- PMM-3771 Grafana 6.1.0

* Thu Feb 28 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 6.0.0-1
- PMM-3561 grafana update for 6.0

* Mon Jan  7 2019 Vadim Yalovets <vadim.yalovets@percona.com> - 5.4.2-1
- PMM-2685 Grafana 5.4.2

* Thu Nov 15 2018 Vadim Yalovets <vadim.yalovets@percona.com> - 5.3.3-1
- PMM-2685 Grafana 5.3

* Wed Nov 14 2018 Vadim Yalovets <vadim.yalovets@percona.com> - 5.1.3-7
- PMM-3257 Apply Patch from Grafana 5.3.3 to latest PMM version

* Mon Nov 5 2018 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 5.1.3-5
- PMM-2837 Fix image rendering

* Mon Oct 8 2018 Daria Lymanska <daria.lymanska@percona.com> - 5.1.3-4
- PMM-2880 add change-icon patch

* Mon Jun 18 2018 Mykola Marzhan <mykola.marzhan@percona.com> - 5.1.3-3
- PMM-2625 fix share-panel patch

* Mon Jun 18 2018 Mykola Marzhan <mykola.marzhan@percona.com> - 5.1.3-2
- PMM-2625 add share-panel patch

* Mon May 21 2018 Vadim Yalovets <vadim.yalovets@percona.com> - 5.1.3-1
- PMM-2561 update to 5.1.3

* Thu Mar 29 2018 Mykola Marzhan <mykola.marzhan@percona.com> - 5.0.4-1
- PMM-2319 update to 5.0.4

* Mon Jan  8 2018 Mykola Marzhan <mykola.marzhan@percona.com> - 4.6.3-1
- PMM-1895 update to 4.6.3

* Mon Nov  6 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.6.1-1
- PMM-1652 update to 4.6.1

* Tue Oct 31 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.6.0-1
- PMM-1652 update to 4.6.0

* Fri Oct  6 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.5.2-1
- PMM-1521 update to 4.5.2

* Tue Sep 19 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.4.3-2
- fix HOME variable in unit file

* Wed Aug  2 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.4.3-1
- PMM-1221 update to 4.4.3

* Wed Aug  2 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.4.2-1
- PMM-1221 update to 4.4.2

* Wed Jul 19 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.4.1-1
- PMM-1221 update to 4.4.1

* Thu Jul 13 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.3.2-2
- PMM-1208 install fontconfig freetype urw-fonts

* Thu Jun  1 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.3.2-1
- update to 4.3.2

* Wed Mar 29 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.2.0-2
- up to 4.2.0
- PMM-708 rollback tooltip position

* Tue Mar 14 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.1.2-1
- up to 4.1.2

* Thu Jan 26 2017 Mykola Marzhan <mykola.marzhan@percona.com> - 4.1.1-1
- up to 4.1.1

* Thu Dec 29 2016 Mykola Marzhan <mykola.marzhan@percona.com> - 4.0.2-2
- use fixed grafana-server.service

* Thu Dec 15 2016 Mykola Marzhan <mykola.marzhan@percona.com> - 4.0.2-1
- up to 4.0.2

* Fri Jul 31 2015 Graeme Gillies <ggillies@redhat.com> - 2.0.2-3
- Unbundled phantomjs from grafana

* Tue Jul 28 2015 Lon Hohberger <lon@redhat.com> - 2.0.2-2
- Change ownership for grafana-server to root

* Tue Apr 14 2015 Graeme Gillies <ggillies@redhat.com> - 2.0.2-1
- First package for Fedora
