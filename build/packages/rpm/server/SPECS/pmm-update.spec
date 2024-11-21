%global _missing_build_ids_terminate_build  0
%define _binaries_in_noarch_packages_terminate_build  0
%define _unpackaged_files_terminate_build  0

%global repo            pmm
%global provider        github.com/percona/%{repo}
%global commit	        592eddf656bce32a11bd958af0a32c62bd5ea34c
%global shortcommit	    %(c=%{commit}; echo ${c:0:7})
%define release         68
%define rpm_release     %{release}.%{shortcommit}%{?dist}

# the line below is sed'ed by build/bin/build-server-rpm to set a correct version
%define full_pmm_version 2.0.0

Name:		  pmm-update
Version:	%{full_pmm_version}
Release:	%{rpm_release}
Summary:	Tool for updating packages and OS configuration for PMM Server

License:	AGPLv3
URL:		  https://%{provider}
Source0:	https://%{provider}/archive/%{commit}.tar.gz

BuildArch:	noarch

%description
%{summary}


%prep
%setup -q -n %{repo}-%{commit}

%build
export PMM_RELEASE_VERSION=%{full_pmm_version}
export PMM_RELEASE_FULLCOMMIT=%{commit}
export PMM_RELEASE_BRANCH=""

make -C update release


%install
install -d %{buildroot}%{_sbindir}
install -p -m 0755 update/bin/pmm-update %{buildroot}%{_sbindir}/


%files
%license update/LICENSE
%doc update/README.md
%{_sbindir}/pmm-update


%changelog
* Mon Apr 1 2024 Alex Demidoff <alexander.demidoff@percona.com> - 3.0.0-68
- PMM-12899 Use module and build cache

* Thu Dec 8 2022 Michal Kralik <michal.kralik@percona.com> - 2.34.0-67
- PMM-11207 Migrate pmm-update to monorepo

* Mon May 16 2022 Nikita Beletskii <nikita.beletskii@percona.com> - 2.29.0-1
- https://per.co.na/pmm/latest

* Tue Oct 19 2021 Nikita Beletskii <nikita.beletskii@percona.com> - 2.23.0-64
- https://per.co.na/pmm/latest

* Tue Oct 19 2021 Nikita Beletskii <nikita.beletskii@percona.com> - 2.23.0-63
- https://per.co.na/pmm/2.23.0

* Tue Sep 21 2021 Nikita Beletskii <nikita.beletskii@percona.com> - 2.22.0-62
- https://per.co.na/pmm/2.22.0

* Thu Aug 26 2021 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.21.0-61
- https://per.co.na/pmm/2.21.0

* Tue Jul 27 2021 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.20.0-60
- https://per.co.na/pmm/2.20.0

* Wed Jun 30 2021 Denys Kondratenko <denys.kondratenko@percona.com> - 2.19.0-59
- https://per.co.na/pmm/2.19.0

* Tue Jun 01 2021 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.18.0-57
- https://per.co.na/pmm/2.18.0

* Tue May 11 2021 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.17.0-56
- https://per.co.na/pmm/2.17.0

* Thu Apr 15 2021 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.16.0-54
- https://per.co.na/pmm/2.16.0

* Thu Mar 18 2021 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.15.1-53
- https://per.co.na/pmm/2.15.1

* Thu Jan 28 2021 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.14.0-52
- https://per.co.na/pmm/2.14.1

* Thu Jan 28 2021 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.14.0-51
- https://per.co.na/pmm/2.14.0

* Tue Dec 29 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.13.0-49
- https://per.co.na/pmm/2.13.0

* Tue Dec 01 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.12.0-46
- https://per.co.na/pmm/2.12.0

* Mon Oct 19 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.11.1-45
- https://per.co.na/pmm/2.11.1

* Wed Oct 14 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.11.0-44
- https://per.co.na/pmm/2.11.0

* Tue Sep 22 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.10.1-43
- https://per.co.na/pmm/2.10.1

* Tue Sep 15 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.10.0-41
- https://per.co.na/pmm/2.10.0

* Tue Aug 04 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.9.1-39
- https://per.co.na/pmm/2.9.1

* Tue Jul 14 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.9.0-37
- https://per.co.na/pmm/2.9.0

* Thu Jul  2 2020 Mykyta Solomko <mykyta.solomko@percona.com> - 2.6.1-35
- PMM-5645 built using Golang 1.14

* Thu Jun 25 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.8.0-34
- https://per.co.na/pmm/2.8.0

* Tue Jun 09 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.7.0-33
- https://per.co.na/pmm/2.7.0

* Mon May 18 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.6.1-32
- https://per.co.na/pmm/2.6.1

* Mon May 11 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.6.0-31
- https://per.co.na/pmm/2.6.0

* Tue Apr 14 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.5.0-30
- https://per.co.na/pmm/2.5.0

* Wed Mar 18 2020 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 2.4.0-29
- https://per.co.na/pmm/2.4.0

* Mon Feb 17 2020 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.3.0-28
- https://per.co.na/pmm/2.3.0

* Tue Feb 4 2020 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.2.2-27
- https://per.co.na/pmm/2.2.2

* Thu Jan 23 2020 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.2.1-26
- https://per.co.na/pmm/2.2.1

* Tue Dec 24 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.2.0-25
- https://per.co.na/pmm/2.2.0

* Mon Nov 11 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.1.0-23
- https://per.co.na/pmm/2.1.0

* Mon Sep 23 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.0.1-19
- https://per.co.na/pmm/2.0.1

* Wed Sep 18 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.0.0-18
- https://per.co.na/pmm/2.0.0

* Tue Sep 17 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.0.0-17.rc4
- https://per.co.na/pmm/2.0.0-rc4

* Mon Sep 16 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.0.0-16.rc3
- https://per.co.na/pmm/2.0.0-rc3

* Fri Sep 13 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.0.0-15.rc2
- https://per.co.na/pmm/2.0.0-rc2

* Wed Sep 11 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.0.0-14.rc1
- https://per.co.na/pmm/2.0.0-rc1

* Mon Sep  9 2019 Alexey Palazhchenko <alexey.palazhchenko@percona.com> - 2.0.0-12.beta7
- https://per.co.na/pmm/2.0.0-beta7
