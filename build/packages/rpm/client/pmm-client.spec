%define debug_package %{nil}

Name:           pmm-client
Summary:        Percona Monitoring and Management Client (pmm-agent)
Version:        %{version}
Release:        %{release}%{?dist}
Group:          Applications/Databases
License:        ASL 2.0
Vendor:         Percona LLC
URL:            https://percona.com
Source:         pmm-client-%{version}.tar.gz
BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root

BuildRequires:    systemd
BuildRequires:    pkgconfig(systemd)
%if 0%{?rhel} && 0%{?rhel} >= 9
Requires:         perl-interpreter
%endif
Requires(post):   systemd
Requires(preun):  systemd
Requires(postun): systemd

AutoReq:        no
Conflicts:      pmm-client
Obsoletes:      pmm2-client < 3.0.0

%description
Percona Monitoring and Management (PMM) is an open-source platform for managing and monitoring MySQL and MongoDB
performance. It is developed by Percona in collaboration with experts in the field of managed database services,
support and consulting.
PMM is a free and open-source solution that you can run in your own environment for maximum security and reliability.
It provides thorough time-based analysis for MySQL and MongoDB servers to ensure that your data works as efficiently
as possible.


%prep
%setup -q

%pretrans
if [ -f /usr/local/percona/pmm2/config/pmm-agent.yaml ]; then
    cp -a /usr/local/percona/pmm2/config/pmm-agent.yaml /usr/local/percona/pmm2/config/pmm-agent.yaml.bak
fi

%posttrans
if [ -f /usr/local/percona/pmm2/config/pmm-agent.yaml.bak ]; then
    mv /usr/local/percona/pmm/config/pmm-agent.yaml /usr/local/percona/pmm/config/pmm-agent.yaml.new
    # Take a backup of pmm-agent.yaml and then modify it to remove paths properties
    mv /usr/local/percona/pmm2/config/pmm-agent.yaml.bak /usr/local/percona/pmm/config/pmm-agent.yaml.bak
    cp /usr/local/percona/pmm/config/pmm-agent.yaml.bak /usr/local/percona/pmm/config/pmm-agent.yaml
    sed '/^paths:/,/^[^[:space:]]/ {
        /^paths:/d
        /^[^[:space:]]/!d
    }' "/usr/local/percona/pmm/config/pmm-agent.yaml" > "/usr/local/percona/pmm/config/pmm-agent.yaml.tmp" && mv "/usr/local/percona/pmm/config/pmm-agent.yaml.tmp" "/usr/local/percona/pmm/config/pmm-agent.yaml"

    if [ -d /usr/local/percona/pmm2/config ] && [ -z "$(ls -A /usr/local/percona/pmm2/config)" ]; then
       rmdir /usr/local/percona/pmm2/config
    fi

    if [ -d /usr/local/percona/pmm2 ] && [ -z "$(ls -A /usr/local/percona/pmm2)" ]; then
       rmdir /usr/local/percona/pmm2
    fi

    if ! getent passwd pmm-agent > /dev/null 2>&1; then
       /usr/sbin/groupadd -r pmm-agent
       /usr/sbin/useradd -M -r -g pmm-agent -d /usr/local/percona/ -s /bin/false -c "PMM Agent User" pmm-agent
       chown -R pmm-agent:pmm-agent /usr/local/percona/pmm
    fi
    /usr/bin/systemctl enable pmm-agent >/dev/null 2>&1 || :
    /usr/bin/systemctl daemon-reload
    /usr/bin/systemctl start pmm-agent.service
fi

%build

%install
install -m 0755 -d $RPM_BUILD_ROOT/usr/sbin
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/bin
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/tools
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/exporters
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/config
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/textfile-collector
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/textfile-collector/low-resolution
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/textfile-collector/medium-resolution
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/textfile-collector/high-resolution
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/mysql
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/mysql/low-resolution
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/mysql/medium-resolution
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/mysql/high-resolution
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/postgresql
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/postgresql/low-resolution
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/postgresql/medium-resolution
install -m 0755 -d $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/postgresql/high-resolution

install -m 0755 bin/pmm-admin $RPM_BUILD_ROOT/usr/local/percona/pmm/bin
install -m 0755 bin/pmm-agent $RPM_BUILD_ROOT/usr/local/percona/pmm/bin
install -m 0755 bin/pmm-agent-entrypoint $RPM_BUILD_ROOT/usr/local/percona/pmm/bin
install -m 0755 bin/node_exporter $RPM_BUILD_ROOT/usr/local/percona/pmm/exporters
install -m 0755 bin/mysqld_exporter $RPM_BUILD_ROOT/usr/local/percona/pmm/exporters
install -m 0755 bin/postgres_exporter $RPM_BUILD_ROOT/usr/local/percona/pmm/exporters
install -m 0755 bin/mongodb_exporter $RPM_BUILD_ROOT/usr/local/percona/pmm/exporters
install -m 0755 bin/proxysql_exporter $RPM_BUILD_ROOT/usr/local/percona/pmm/exporters
install -m 0755 bin/rds_exporter $RPM_BUILD_ROOT/usr/local/percona/pmm/exporters
install -m 0755 bin/azure_exporter $RPM_BUILD_ROOT/usr/local/percona/pmm/exporters
install -m 0755 bin/valkey_exporter $RPM_BUILD_ROOT/usr/local/percona/pmm/exporters
install -m 0755 bin/vmagent $RPM_BUILD_ROOT/usr/local/percona/pmm/exporters
install -m 0755 bin/pt-summary $RPM_BUILD_ROOT/usr/local/percona/pmm/tools
install -m 0755 bin/pt-mysql-summary $RPM_BUILD_ROOT/usr/local/percona/pmm/tools
install -m 0755 bin/pt-mongodb-summary $RPM_BUILD_ROOT/usr/local/percona/pmm/tools
install -m 0755 bin/pt-pg-summary $RPM_BUILD_ROOT/usr/local/percona/pmm/tools
install -m 0755 bin/nomad $RPM_BUILD_ROOT/usr/local/percona/pmm/tools
install -m 0660 example.prom $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/textfile-collector/low-resolution/
install -m 0660 example.prom $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/textfile-collector/medium-resolution/
install -m 0660 example.prom $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/textfile-collector/high-resolution/
install -m 0660 queries-mysqld.yml $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/mysql/low-resolution/
install -m 0660 queries-mysqld.yml $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/mysql/medium-resolution/
install -m 0660 queries-mysqld.yml $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/mysql/high-resolution/
install -m 0660 queries-mysqld-group-replication.yml $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/mysql/high-resolution/
install -m 0660 example-queries-postgres.yml $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/postgresql/low-resolution/
install -m 0660 example-queries-postgres.yml $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/postgresql/medium-resolution/
install -m 0660 example-queries-postgres.yml $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/postgresql/high-resolution/
install -m 0660 queries-postgres-uptime.yml $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/postgresql/high-resolution/
install -m 0660 queries-mr.yaml $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/postgresql/medium-resolution/
install -m 0660 queries-lr.yaml $RPM_BUILD_ROOT/usr/local/percona/pmm/collectors/custom-queries/postgresql/low-resolution/
install -m 0755 -d $RPM_BUILD_ROOT/%{_unitdir}
install -m 0644 config/pmm-agent.service %{buildroot}/%{_unitdir}/pmm-agent.service


%clean
rm -rf $RPM_BUILD_ROOT

%pre
if [ $1 -eq 1 ]; then
  if ! getent passwd pmm-agent > /dev/null 2>&1; then
    /usr/sbin/groupadd -r pmm-agent
    /usr/sbin/useradd -M -r -g pmm-agent -d /usr/local/percona/ -s /bin/false -c pmm-agent pmm-agent > /dev/null 2>&1
  fi
fi
if [ $1 -eq 2 ]; then
    /usr/bin/systemctl stop pmm-agent.service >/dev/null 2>&1 ||:
fi

%post
for file in pmm-admin pmm-agent
do
  %{__ln_s} -f /usr/local/percona/pmm/bin/$file /usr/bin/$file
  %{__ln_s} -f /usr/local/percona/pmm/bin/$file /usr/sbin/$file
done
%systemd_post pmm-agent.service
if [ $1 -eq 1 ]; then
    if [ ! -f /usr/local/percona/pmm/config/pmm-agent.yaml ]; then
        install -d -m 0755 /usr/local/percona/pmm/config
        install -m 0660 -o pmm-agent -g pmm-agent /dev/null /usr/local/percona/pmm/config/pmm-agent.yaml
    fi
    /usr/bin/systemctl enable pmm-agent >/dev/null 2>&1 || :
    /usr/bin/systemctl daemon-reload
    /usr/bin/systemctl start pmm-agent.service
fi

if [ $1 -eq 2 ]; then
    /usr/bin/systemctl daemon-reload
    /usr/bin/systemctl start pmm-agent.service
fi

%preun
%systemd_preun pmm-agent.service

if [ -f /usr/local/percona/pmm/config/pmm-agent.yaml.new ]; then
    rm -f /usr/local/percona/pmm/config/pmm-agent.yaml.new
fi

%postun
case "$1" in
   1) # This is a dnf upgrade.
      %systemd_postun_with_restart pmm-agent.service
   ;;
esac
if [ $1 -eq 0 ]; then
  %systemd_postun_with_restart pmm-agent.service
  if /usr/bin/id -g pmm-agent > /dev/null 2>&1; then
    /usr/sbin/userdel pmm-agent > /dev/null 2>&1
    /usr/sbin/groupdel pmm-agent > /dev/null 2>&1 || true
    if [ -f /usr/local/percona/pmm/config/pmm-agent.yaml ]; then
        rm -r /usr/local/percona/pmm/config/pmm-agent.yaml
    fi
    if [ -f /usr/local/percona/pmm/config/pmm-agent.yaml.bak ]; then
        rm -r /usr/local/percona/pmm/config/pmm-agent.yaml.bak
    fi
    if [ -d /usr/local/percona/pmm/config ] && [ -z "$(ls -A /usr/local/percona/pmm/config)" ]; then
       rmdir /usr/local/percona/pmm/config
    fi

    if [ -d /usr/local/percona/pmm ] && [ -z "$(ls -A /usr/local/percona/pmm)" ]; then
       rmdir /usr/local/percona/pmm
    fi

    for file in pmm-admin pmm-agent
    do
      if [ -L /usr/sbin/$file ]; then
        rm -rf /usr/sbin/$file
      fi
      if [ -L /usr/bin/$file ]; then
        rm -rf /usr/bin/$file
      fi
    done
  fi
fi

%files
%config %{_unitdir}/pmm-agent.service
%attr(0660,pmm-agent,pmm-agent) %ghost /usr/local/percona/pmm/config/pmm-agent.yaml
%attr(-,pmm-agent,pmm-agent) /usr/local/percona/pmm

%changelog
* Wed May 21 2025 Talha Bin Rizwan <talha.rizwan@percona.com>
- PKG-521 include valkey_exporter into pmm client

* Fri Nov 8 2024 Nurlan Moldomurov <nurlan.moldomurov@percona.com>
- PMM-13399 include nomad into pmm client

* Tue Jun 21 2022 Nikita Beletskii <nikita.beletskii@percona.com>
- PMM-7 remove support for RHEL older then 7

* Tue Aug 24 2021 Vadim Yalovets <vadim.yalovets@percona.com>
- PMM-8618 ship default PG queries in PMM.

* Tue Oct 13 2020 Nikolay Khramchikhin <nik@victoriametrics.com>
- PMM-6396 added vmagent binary.

* Tue Aug 25 2020 Vadim Yalovets <vadim.yalovets@percona.com>
- PMM-2045 MySQL Group Replication Dashboard.

* Fri Jul 31 2020 Vadim Yalovets <vadim.yalovets@percona.com>
- PMM-5701 DB_Uptime in Home Dashboard shows wrong metric.

* Thu Aug 29 2019 Evgeniy Patlan <evgeniy.patlan@percona.com>
- Rework file structure.
