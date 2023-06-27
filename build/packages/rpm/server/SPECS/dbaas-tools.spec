%undefine _missing_build_ids_terminate_build
%define debug_package %{nil}

%global commit_aws          ea9bcaeb5e62c110fe326d1db58b03a782d4bdd6
%global shortcommit_aws     %(c=%{commit_aws}; echo ${c:0:7})

%global commit_k8s          ef70d260f3d036fc22b30538576bbf6b36329995
%global shortcommit_k8s     %(c=%{commit_k8s}; echo ${c:0:7})
%global version_k8s         v1.24.12

%global install_golang 1
%global debug_package %{nil}

%define build_timestamp %(date -u +"%y%m%d%H%M")
%define release         2
%define rpm_release     %{release}.%{build_timestamp}%{?dist}

Name:           dbaas-tools
Version:        0.6.10
Release:        %{rpm_release}
Summary:        A set of tools for Percona DBaaS
License:        ASL 2.0
URL:            https://github.com/kubernetes-sigs/aws-iam-authenticator
# Git tag can be moved and pointed to different commit hash which may brake reproducibility of the build
# As by using an exact commit hash, we can ensure that each time source will be identical
Source0:        https://github.com/kubernetes-sigs/aws-iam-authenticator/archive/%{commit_aws}/aws-iam-authenticator-%{shortcommit_aws}.tar.gz
Source1:        https://github.com/kubernetes/kubernetes/archive/%{commit_k8s}/kubernetes-%{shortcommit_k8s}.tar.gz

BuildRequires: which

%description
%{summary}

%prep
%setup -T -c -n aws-iam-authenticator-%{commit_aws}
%setup -q -c -a 0 -n aws-iam-authenticator-%{commit_aws}
mkdir -p src/github.com/kubernetes-sigs/
mv aws-iam-authenticator-%{commit_aws} src/github.com/kubernetes-sigs/aws-iam-authenticator-%{commit_aws}

%setup -T -c -n kubernetes-%{commit_k8s}
%setup -q -c -a 1 -n kubernetes-%{commit_k8s}
mkdir -p src/github.com/kubernetes/
mv kubernetes-%{commit_k8s} src/github.com/kubernetes/kubernetes-%{commit_k8s}

%build
cd %{_builddir}/aws-iam-authenticator-%{commit_aws}
export GOPATH="$(pwd)"
export CGO_ENABLED=0
export USER=builder

cd src/github.com/kubernetes-sigs/aws-iam-authenticator-%{commit_aws}
sed -i '/- darwin/d;/- windows/d;/- arm64/d;/dockers:/,+23d' .goreleaser.yaml
make goreleaser

cd %{_builddir}/kubernetes-%{commit_k8s}/
export GOPATH="$(pwd)"

cd src/github.com/kubernetes/kubernetes-%{commit_k8s}
make WHAT="cmd/kubectl"

%install
cd %{_builddir}/aws-iam-authenticator-%{commit_aws}/src/github.com/kubernetes-sigs/aws-iam-authenticator-%{commit_aws}
install -D -p -m 0755 dist/aws-iam-authenticator_linux_amd64_v1/aws-iam-authenticator %{buildroot}/opt/dbaas-tools/bin/aws-iam-authenticator

cd %{_builddir}/kubernetes-%{commit_k8s}/src/github.com/kubernetes/kubernetes-%{commit_k8s}
install -D -p -m 0775 _output/local/go/bin/kubectl %{buildroot}/opt/dbaas-tools/bin/kubectl-1.23


%files
/opt/dbaas-tools/bin/aws-iam-authenticator
/opt/dbaas-tools/bin/kubectl-1.23

%changelog

* Mon Jun 12 2023 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 0.6.10-1
- Update versions of aws-iam-authenticator

* Mon Jun 05 2023 Andrew Minkin <andrew.minkin@percona.com> - 0.6.2-1
- Update versions of kubectl and aws-iam-authenticator

* Mon Nov 21 2022 Alex Tymchuk <alexander.tymchuk@percona.com> - 0.5.7-2
- Fix the double description warning

* Wed May 04 2022 Nurlan Moldomurov <nurlan.moldomurov@percona.com> - 0.5.7-1
- Update versions of dbaas-tools

* Thu Aug 27 2020 Illia Pshonkin <illia.pshonkin@percona.com> - 0.5.1-1
- Initial packaging for dbaas-tools

