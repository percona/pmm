#!/bin/sh

shell_quote_string() {
  echo "$1" | sed -e 's,\([^a-zA-Z0-9/_.=-]\),\\\1,g'
}

usage () {
    cat <<EOF
Usage: $0 [OPTIONS]
    The following options may be given :
        --builddir=DIR      Absolute path to the dir where all actions will be performed
        --get_sources       Source will be downloaded from github
        --build_src_rpm     If it is 1 src rpm will be built
        --build_source_deb  If it is 1 source deb package will be built
        --build_rpm         If it is 1 rpm will be built
        --build_deb         If it is 1 deb will be built
        --build_tarball     If it is 1 tarball will be built
        --install_deps      Install build dependencies(root previlages are required)
        --branch            Branch from which submodules should be taken(default master)
        --help) usage ;;
Example $0 --builddir=/tmp/PMM_CLIENT --get_sources=1 --build_src_rpm=1 --build_rpm=1
EOF
        exit 1
}

append_arg_to_args () {
  args="$args "`shell_quote_string "$1"`
}

parse_arguments() {
    pick_args=
    if test "$1" = PICK-ARGS-FROM-ARGV
    then
        pick_args=1
        shift
    fi
  
    for arg do
        val=`echo "$arg" | sed -e 's;^--[^=]*=;;'`
        optname=`echo "$arg" | sed -e 's/^\(--[^=]*\)=.*$/\1/'`
        case "$arg" in
            # these get passed explicitly to mysqld
            --builddir=*) WORKDIR="$val" ;;
            --build_src_rpm=*) SRPM="$val" ;;
            --build_source_deb=*) SDEB="$val" ;;
            --build_rpm=*) RPM="$val" ;;
            --build_deb=*) DEB="$val" ;;
            --get_sources=*) SOURCE="$val" ;;
            --build_tarball=*) TARBALL="$val" ;;
            --branch=*) SUBMODULE_BRANCH="$val" ;;
            --install_deps=*) INSTALL="$val" ;;
            --help) usage ;;      
            *)
              if test -n "$pick_args"
              then
                  append_arg_to_args "$arg"
              fi
              ;;
        esac
    done
}

get_branches() {
    COMPONENT=$1
    if [ ! -e pmm-submodules ]; then
        git clone https://github.com/Percona-Lab/pmm-submodules.git
    fi
    cd pmm-submodules
      git reset --hard > /dev/null 2>&1
      git clean -xdf > /dev/null 2>&1
      git checkout $SUBMODULE_BRANCH > /dev/null 2>&1
      git submodule status | grep $COMPONENT | awk '{print $1}' | awk -F'-' '{print $2}'
    cd ../
}

get_repos() {
    COMPONENT=$1
    if [ ! -e pmm-submodules ]; then
        git clone https://github.com/Percona-Lab/pmm-submodules.git
    fi
    cd pmm-submodules
      git reset --hard > /dev/null 2>&1
      git clean -xdf > /dev/null 2>&1
      git checkout $SUBMODULE_BRANCH > /dev/null 2>&1
      grep -A 3 "\[submodule \"${COMPONENT}\"\]" .gitmodules | grep "url" | awk '{print $3}'
    cd ../
}

check_workdir(){
    if [ "x$WORKDIR" = "x$CURDIR" ]
    then
        echo >&2 "Current directory cannot be used for building!"
        exit 1
    else
        if ! test -d "$WORKDIR"
        then
            echo >&2 "$WORKDIR is not a directory."
            exit 1
        fi
    fi
    return
}

add_percona_yum_repo(){
    if [ ! -f /etc/yum.repos.d/percona-dev.repo ]
    then
        cat >/etc/yum.repos.d/percona-dev.repo <<EOL
[percona-dev-$basearch]
name=Percona internal YUM repository for build slaves \$releasever - \$basearch
baseurl=http://jenkins.percona.com/yum-repo/\$releasever/RPMS/\$basearch
gpgkey=http://jenkins.percona.com/yum-repo/PERCONA-PACKAGING-KEY
gpgcheck=0
enabled=1

[percona-dev-noarch]
name=Percona internal YUM repository for build slaves \$releasever - noarch
baseurl=http://jenkins.percona.com/yum-repo/\$releasever/RPMS/noarch
gpgkey=http://jenkins.percona.com/yum-repo/PERCONA-PACKAGING-KEY
gpgcheck=0
enabled=1
EOL
    fi
    return
}

add_percona_apt_repo(){
    if [ ! -f /etc/apt/sources.list.d/percona-dev.list ]
    then
        cat >/etc/apt/sources.list.d/percona-dev.list <<EOL
deb http://jenkins.percona.com/apt-repo/ @@DIST@@ main
deb-src http://jenkins.percona.com/apt-repo/ @@DIST@@ main
EOL
    sed -i "s:@@DIST@@:$OS_NAME:g" /etc/apt/sources.list.d/percona-dev.list
    fi
    return
}

get_sources(){
    cd $WORKDIR
    if [ $SOURCE = 0 ]
    then
        echo "Sources will not be downloaded"
        return 0
    fi
    git clone $REPO
    retval=$?
    if [ $retval != 0 ]
    then
        echo "There were some issues during repo cloning from github. Please retry one more time"
        exit 1
    fi
    cd pmm-client
    if [ ! -z $BRANCH ]
    then
        git reset --hard
        git clean -xdf
        git checkout $BRANCH
    fi
    REVISION=$(git rev-parse --short HEAD)
    git reset --hard
    #
    VERSION=$(cat VERSION)
    mv Makefile build/
    #cat VERSION > $VERSION_FILE
    echo "VERSION=${VERSION}" > $VERSION_FILE
    echo "REVISION=${REVISION}" >> $VERSION_FILE
    echo "RPM_RELEASE=${RPM_RELEASE}" >> $VERSION_FILE
    echo "DEB_RELEASE=${DEB_RELEASE}" >> $VERSION_FILE
    echo "GIT_REPO=${REPO}" >> $VERSION_FILE
    echo "BRANCH_NAME=${BRANCH}" >> $VERSION_FILE
    echo "NodeExp_BRANCH_NAME=${NodeExp_BRANCH_NAME}" >> $VERSION_FILE
    echo "MongoExp_BRANCH_NAME=${MongoExp_BRANCH_NAME}" >> $VERSION_FILE
    echo "MysqlExp_BRANCH_NAME=${MysqlExp_BRANCH_NAME}" >> $VERSION_FILE
    echo "ProxysqlExp_BRANCH_NAME=${ProxysqlExp_BRANCH_NAME}" >> $VERSION_FILE
    echo "QAN_BRANCH_NAME=${QAN_BRANCH_NAME}" >> $VERSION_FILE
    echo "TOOLKIT_REPO=${TOOLKIT_REPO}" >> $VERSION_FILE
    echo "TOOLKIT_BRANCH_NAME=${TOOLKIT_BRANCH_NAME}" >> $VERSION_FILE
    PRODUCT=pmm-client
    PRODUCT_NAME=pmm
    echo "PRODUCT=${PRODUCT}" >> $VERSION_FILE
    echo "PRODUCT_NAME=${PRODUCT_NAME}" >> $VERSION_FILE
    PRODUCT_FULL=${PRODUCT}-${VERSION}
    echo "PRODUCT_FULL=${PRODUCT_FULL}" >> $VERSION_FILE
    echo "BUILD_NUMBER=${BUILD_NUMBER}" >> $VERSION_FILE
    echo "BUILD_ID=${BUILD_ID}" >> $VERSION_FILE
    echo "UPLOAD=UPLOAD/experimental/BUILDS/${PRODUCT_NAME}/${VERSION}/${BRANCH_NAME}/${REVISION}/${BUILD_ID}" >> $VERSION_FILE
    echo "MongoExp_REPO=${MongoExp_REPO}" >> $VERSION_FILE
    echo "TOOLKIT_REPO=${TOOLKIT_REPO}" >> $VERSION_FILE
    echo "MysqlExp_REPO=${MysqlExp_REPO}" >> $VERSION_FILE
    echo "ProxysqlExp_REPO=${ProxysqlExp_REPO}" >> $VERSION_FILE
    echo "QAN_REPO=${QAN_REPO}" >> $VERSION_FILE
    echo "NodeExp_REPO=${NodeExp_REPO}" >> $VERSION_FILE
    cd ../
    mv ${PRODUCT} ${PRODUCT}-${VERSION}
    
    tar -zcvf ${PRODUCT}-${VERSION}.tar.gz ${PRODUCT}-${VERSION} --exclude=.bzr*
    mkdir $WORKDIR/source_tarball
    mkdir $CURDIR/source_tarball
    cp ${PRODUCT}-${VERSION}.tar.gz $WORKDIR/source_tarball
    cp ${PRODUCT}-${VERSION}.tar.gz $CURDIR/source_tarball
    cd $CURDIR
    rm -rf pmm-client
    return
}

get_system(){
    if [ -f /etc/redhat-release ]; then
        RHEL=$(rpm --eval %rhel)
        ARCH=$(echo $(uname -m) | sed -e 's:i686:i386:g')
        OS_NAME="el$RHEL"
        OS="rpm"
    else
        ARCH=$(uname -m)
        OS_NAME="$(lsb_release -sc)"
        OS="deb"
    fi
    return
}

install_golang() {
    wget http://jenkins.percona.com/downloads/golang/go1.9.4.linux-amd64.tar.gz -O /tmp/golang1.9.4.tar.gz
    tar --transform=s,go,go1.9, -zxf /tmp/golang1.9.4.tar.gz
    rm -rf /usr/local/go /usr/local/go1.8 /usr/local/go1.9
    mv go1.9 /usr/local/
    ln -s /usr/local/go1.9 /usr/local/go
}

install_deps() {
    if [ $INSTALL = 0 ]
    then
        echo "Dependencies will not be installed"
        return;
    fi
    if [ ! $( id -u ) -eq 0 ]
    then
        echo "It is not possible to instal dependencies. Please run as root"
        exit 1
    fi
    install_golang
    CURPLACE=$(pwd)
    if [ "x$OS" = "xrpm" ]
    then
        add_percona_yum_repo
        yum -y install git wget
        yum -y install epel-release rpmdevtools bison
        cd $WORKDIR
        link="https://raw.githubusercontent.com/percona/pmm-client/master/build/rpm.spec"
        if [ ! -z $BRANCH ]
        then
            sed "s:master:$BRANCH:" $link
        fi
        wget $link
        yum-builddep -y $WORKDIR/rpm.spec
    else
        add_percona_apt_repo
        apt-get update
        apt-get -y install devscripts equivs
        CURPLACE=$(pwd)
        cd $WORKDIR
        link="https://raw.githubusercontent.com/percona/pmm-client/master/build/deb/control"
        if [ ! -z $BRANCH ]
        then
            sed "s:master:$BRANCH:" $link
        fi
        wget $link
        cd $CURPLACE
        sed -i 's:apt-get :apt-get -y --force-yes :g' /usr/bin/mk-build-deps
        mk-build-deps --install $WORKDIR/control
    fi
    return;
}

get_tar(){
    TARBALL=$1
    TARFILE=$(basename $(find $WORKDIR/$TARBALL -name 'pmm-client*.tar.gz' | sort | tail -n1))
    if [ -z $TARFILE ]
    then
        TARFILE=$(basename $(find $CURDIR/$TARBALL -name 'pmm-client*.tar.gz' | sort | tail -n1))
        if [ -z $TARFILE ]
        then
            echo "There is no $TARBALL for build"
            exit 1
        else
            cp $CURDIR/$TARBALL/$TARFILE $WORKDIR/$TARFILE
        fi
    else
        cp $WORKDIR/$TARBALL/$TARFILE $WORKDIR/$TARFILE
    fi
    return
}

get_deb_sources(){
    param=$1
    echo $param
    FILE=$(basename $(find $WORKDIR/source_deb -name "pmm-client*.$param" | sort | tail -n1))
    if [ -z $FILE ]
    then
        FILE=$(basename $(find $CURDIR/source_deb -name "pmm-client*.$param" | sort | tail -n1))
        if [ -z $FILE ]
        then
            echo "There is no sources for build"
            exit 1
        else
            cp $CURDIR/source_deb/$FILE $WORKDIR/
        fi
    else
        cp $WORKDIR/source_deb/$FILE $WORKDIR/
    fi
    return
}

build_srpm(){
    if [ $SRPM = 0 ]
    then
        echo "SRC RPM will not be created"
        return;
    fi
    if [ "x$OS" = "xdeb" ]
    then
        echo "It is not possible to build src rpm here"
        exit 1
    fi
    cd $WORKDIR
    get_tar "tarball"
    #
    rm -fr rpmbuild
    ls | grep -v tar.gz | xargs rm -rf
    #
    TARFILE=$(basename $(find . -name 'pmm-client-*.tar.gz' | sort | tail -n1))
    NAME=$(echo ${TARFILE}| awk -F '-' '{print $1"-"$2}')
    VERSION_TMP=$(echo ${TARFILE}| awk -F '-' '{print $3}')
    VERSION=${VERSION_TMP%.tar.gz}
    #
    
    mkdir -vp rpmbuild/{SOURCES,SPECS,BUILD,SRPMS,RPMS}
    #
    
    git clone $REPO  pmm-client-$VERSION
    pushd pmm-client-$VERSION
        git fetch origin
        if [ ! -z ${BRANCH} ]; then
            git reset --hard
            git clean -xdf
            git checkout ${BRANCH}
        fi
    popd
    pushd pmm-client-$VERSION
    REVISION=$(git rev-parse --short HEAD)
    popd
    #
    cd ${WORKDIR}/rpmbuild/SPECS
    cp -ap ${WORKDIR}/${NAME}-${VERSION}/build/*.spec .
    cp -ap ${WORKDIR}/${TARFILE} ../SOURCES/
    cd ${WORKDIR}
    rpmbuild -bs --define "_topdir ${WORKDIR}/rpmbuild" --define "version $VERSION" --define "release $RPM_RELEASE" --define "dist .generic" rpmbuild/SPECS/rpm.spec
    mkdir -p ${WORKDIR}/srpm
    mkdir -p ${CURDIR}/srpm
    cp rpmbuild/SRPMS/*.src.rpm ${CURDIR}/srpm
    cp rpmbuild/SRPMS/*.src.rpm ${WORKDIR}/srpm
    #

}

build_rpm(){
    if [ $RPM = 0 ]
    then
        echo "RPM will not be created"
        return;
    fi
    if [ "x$OS" = "xdeb" ]
    then
        echo "It is not possible to build rpm here"
        exit 1
    fi
    SRC_RPM=$(basename $(find $WORKDIR/srpm -name 'pmm-client*.src.rpm' | sort | tail -n1))
    if [ -z $SRC_RPM ]
    then
        SRC_RPM=$(basename $(find $CURDIR/srpm -name 'pmm-client*.src.rpm' | sort | tail -n1))
        if [ -z $SRC_RPM ]
        then
            echo "There is no src rpm for build"
            echo "You can create it using key --build_src_rpm=1"
            exit 1
        else
            cp $CURDIR/srpm/$SRC_RPM $WORKDIR
        fi
    else
        cp $WORKDIR/srpm/$SRC_RPM $WORKDIR
    fi
    cd $WORKDIR
    rm -fr rpmbuild
    mkdir -vp rpmbuild/{SOURCES,SPECS,BUILD,SRPMS,RPMS}
    cp $SRC_RPM rpmbuild/SRPMS/
    rpmbuild --define "_topdir ${WORKDIR}/rpmbuild" --define "dist .$OS_NAME" --rebuild rpmbuild/SRPMS/$SRC_RPM

    return_code=$?
    if [ $return_code != 0 ]; then
        exit $return_code
    fi
    mkdir -p ${WORKDIR}/rpm
    mkdir -p ${CURDIR}/rpm
    cp rpmbuild/RPMS/*/*.rpm ${WORKDIR}/rpm
    cp rpmbuild/RPMS/*/*.rpm ${CURDIR}/rpm
    
}

build_source_deb(){
    if [ $SDEB = 0 ]
    then
        echo "source deb package will not be created"
        return;
    fi
    if [ "x$OS" = "xrmp" ]
    then
        echo "It is not possible to build source deb here"
        exit 1
    fi
    rm -rf pmm-client*
    get_tar "tarball"
    rm -f *.dsc *.orig.tar.gz *.debian.tar.gz *.changes
    #
    TARFILE=$(basename $(find . -name 'pmm-client-*.tar.gz' | sort | tail -n1))
    NAME=$(echo ${TARFILE}| awk -F '-' '{print $1"-"$2}')
    VERSION_TMP=$(echo ${TARFILE}| awk -F '-' '{print $3}')
    VERSION=${VERSION_TMP%.tar.gz}
    
    rm -fr ${NAME}-${VERSION}
    #
    NEWTAR=${NAME}_${VERSION}.orig.tar.gz
    mv ${TARFILE} ${NEWTAR}
    #
    
    git clone $REPO ${NAME}-${VERSION}_all
    push ${NAME}-${VERSION}_all
        git fetch origin
        if [ ! -z ${BRANCH} ]; then
            git reset --hard
            git clean -xdf
            git checkout ${BRANCH}
        fi
    popd
    pushd ${NAME}-${VERSION}_all
    REVISION=$(git rev-parse --short HEAD)
    mkdir distro
    popd
    
    tar xzf ${NEWTAR}
    cd ${NAME}-${VERSION}
    mv bin/* ../${NAME}-${VERSION}_all/distro/
    cd ../
    
    rm -rf ${NAME}-${VERSION}
    mv ${NAME}-${VERSION}_all ${NAME}-${VERSION}
    rm -rf *.tar.gz 
    tar -zcvf ${NEWTAR} ${NAME}-${VERSION}
    cd ${NAME}-${VERSION}
    mkdir debian
    cp build/deb/* debian/
    mv Makefile build/
    sed -i "s/%{version}/$VERSION-$DEB_RELEASE/" debian/control
    
    cd debian
    echo "${NAME} (${VERSION}) unstable; urgency=low" >> changelog
    echo "  * Initial Release." >> changelog
    echo " -- EvgeniyPatlan <evgeniy.patlan@percona.com> $(date -R)" >> changelog
    
    cd ../
    
    dch -D unstable --force-distribution -v "${VERSION}-${DEB_RELEASE}" "Update to new upstream release PMM-client ${VERSION}-${DEB_RELEASE}"
    dpkg-buildpackage -S
    cd ../
    mkdir -p $WORKDIR/source_deb
    mkdir -p $CURDIR/source_deb
    cp *.diff.gz $WORKDIR/source_deb
    cp *_source.changes $WORKDIR/source_deb
    cp *.dsc $WORKDIR/source_deb
    cp *.orig.tar.gz $WORKDIR/source_deb
    cp *.diff.gz $CURDIR/source_deb
    cp *_source.changes $CURDIR/source_deb
    cp *.dsc $CURDIR/source_deb
    cp *.orig.tar.gz $CURDIR/source_deb
}

build_deb(){
    if [ $DEB = 0 ]
    then
        echo "source deb package will not be created"
        return;
    fi
    if [ "x$OS" = "xrmp" ]
    then
        echo "It is not possible to build source deb here"
        exit 1
    fi
    for file in 'dsc' 'orig.tar.gz' 'changes' 'diff.gz'
    do
        get_deb_sources $file
    done
    cd $WORKDIR
    rm -fv *.deb
    export DEBIAN_VERSION="$(lsb_release -sc)"
    DSC=$(basename $(find . -name '*.dsc' | sort | tail -n 1))
    DIRNAME=$(echo ${DSC} | sed -e 's:_:-:g' | awk -F'-' '{print $1"-"$2}')
    VERSION=$(echo ${DSC} | sed -e 's:_:-:g' | awk -F'-' '{print $3}')
    ARCH=$(uname -m)
    echo "DEBIAN_VERSION=${DEBIAN_VERSION}" >> $VERSION_FILE
    echo "ARCH=${ARCH}" >> $VERSION_FILE
    #
    dpkg-source -x ${DSC}
    cd ${DIRNAME}-${VERSION}
    mv Makefile build/
    dch -b -m -D "$DEBIAN_VERSION" --force-distribution -v "${VERSION}-${DEB_RELEASE}.${DEBIAN_VERSION}" 'Update distribution'
    
    dpkg-buildpackage -rfakeroot -uc -us -b
    mkdir -p $CURDIR/deb
    mkdir -p $WORKDIR/deb
    cp $WORKDIR/*.deb $WORKDIR/deb
    cp $WORKDIR/*.deb $CURDIR/deb
}

build_tarball(){
    if [ $TARBALL = 0 ]
    then
        echo "Binary tarball will not be created"
        return;
    fi
    get_tar "source_tarball"
    cd $WORKDIR
    TARFILE=$(basename $(find . -name 'pmm-client-*.tar.gz' | sort | tail -n1))
    NAME=$(echo ${TARFILE}| awk -F '-' '{print $1"-"$2}')
    VERSION_TMP=$(echo ${TARFILE}| awk -F '-' '{print $3}')
    VERSION=${VERSION_TMP%.tar.gz}
    
    #
    tar -zxvf ${TARFILE}
    rm -rf ${TARFILE}
    cd ${NAME}-${VERSION}
    export GOROOT="/usr/local/go/"
    export GOPATH=$(pwd)/
    export PATH="/usr/local/go/bin:$PATH:$GOPATH"
    export GOBINPATH="/usr/local/go/bin"
    
    mkdir -p $GOPATH/src/github.com/percona
    cd $GOPATH/src/github.com/percona
        git clone $MongoExp_REPO
        cd mongodb_exporter
            git fetch origin
            if [ ! -z ${MongoExp_BRANCH_NAME} ]; then
                git reset --hard
                git clean -xdf
                git checkout ${MongoExp_BRANCH_NAME}
            fi
        cd ../
    #
        git clone $MysqlExp_REPO
        cd mysqld_exporter
            git fetch origin
            if [ ! -z ${MysqlExp_BRANCH_NAME} ]; then
                git reset --hard
                git clean -xdf
                git checkout ${MysqlExp_BRANCH_NAME}
            fi
        cd ../
    #
        git clone $ProxysqlExp_REPO
        cd proxysql_exporter
            git fetch origin
            if [ ! -z ${ProxysqlExp_BRANCH_NAME} ]; then
                git reset --hard
                git clean -xdf
                git checkout ${ProxysqlExp_BRANCH_NAME}
            fi
        cd ../
    #
      git clone $REPO
        cd pmm-client
            git fetch origin
            if [ ! -z ${BRANCH} ]; then
                git reset --hard
                git clean -xdf
                git checkout ${BRANCH}
            fi
        cd ../
    #
      git clone $QAN_REPO
        cd qan-agent
            git fetch origin
            if [ ! -z ${QAN_BRANCH_NAME} ]; then
                git reset --hard
                git clean -xdf
                git checkout ${QAN_BRANCH_NAME}
            fi
        cd ../
    #
    cd ../
    
    mkdir -p $GOPATH/src/github.com/prometheus
    cd $GOPATH/src/github.com/prometheus
        git clone $NodeExp_REPO
        cd node_exporter
            git fetch origin
            if [ ! -z ${NodeExp_BRANCH_NAME} ]; then
                git reset --hard
                git clean -xdf
                git checkout ${NodeExp_BRANCH_NAME}
            fi
        cd ../
    cd ../
    cd $GOPATH
    
    cd src/github.com/percona/pmm-client
    sed -i 's:make:make build:' scripts/build
    export DEV=no
    sh -x scripts/build
    
    cp distro/${NAME}-${VERSION}-*.tar.gz ${WORKDIR}/${NAME}-${VERSION}.tar.gz
    cd ${WORKDIR}
    rm -rf `ls | grep -v tar.gz| grep -v $VERSION_FILE`
    cd ${WORKDIR}
    tar -xvzf ${NAME}-${VERSION}.tar.gz
    
    mv ${NAME}-${VERSION}-x86_64 ${NAME}-${VERSION}
    tar -cvzf ${NAME}-${VERSION}.tar.gz ${NAME}-${VERSION}
    EXPORTED_TAR=$(basename $(find . -type f -name *.tar.gz | sort | tail -n 1))
    #
    CLIENT_DIR=${EXPORTED_TAR%.tar.gz}
    rm -fr ${CLIENT_DIR}
    tar xzf ${EXPORTED_TAR}
    rm -f ${EXPORTED_TAR}
    
    mkdir -p golang/src/github.com/percona
    export GOPATH=$(pwd)/golang
    cd golang/src/github.com/percona
    export GOROOT="/usr/local/go/"
    export PATH="/usr/local/go/bin:$PATH:$GOPATH"
    export GOBINPATH="/usr/local/go/bin"
    
    git clone $TOOLKIT_REPO
    cd percona-toolkit
        git fetch origin
        if [ ! -z ${TOOLKIT_BRANCH_NAME} ]; then
            git reset --hard
            git clean -xdf
            git checkout ${TOOLKIT_BRANCH_NAME}
        fi
    cd ../
    
    cd percona-toolkit
        export BUILD_TAR=1
        export BUILD_RPM=0
        export BUILD_DEB=0
        export BUILD_GO=1
        export QUIET=1
        export UPDATE=0
        export CHECK=0
        
        
        go get -u github.com/golang/dep/cmd/dep
        go install github.com/golang/dep/cmd/dep
        rm -rf bin/govendor
        
        bash -x util/build-packages ${VERSION} docs/release_notes.rst
        cp release/* ${WORKDIR}/
    cd ../
    
    cd  ${WORKDIR}
    TOOLKIT_TAR=$(basename $(find . -type f -name percona-toolkit*.tar.gz | sort | tail -n 1))
    TOOLKIT_DIR=${TOOLKIT_TAR%.tar.gz}
    rm -rf ${TOOLKIT_DIR}
    tar xvzf ${TOOLKIT_TAR}
    rm -rf ${TOOLKIT_TAR}
    
    cp ${TOOLKIT_DIR}/bin/pt-summary ${CLIENT_DIR}/bin/
    cp ${TOOLKIT_DIR}/bin/pt-mysql-summary ${CLIENT_DIR}/bin/
    cp ${TOOLKIT_DIR}/bin/pt-mongodb-summary ${CLIENT_DIR}/bin/
    
    rm -rf ${TOOLKIT_DIR}
    tar -cvzf ${CLIENT_DIR}.tar.gz ${CLIENT_DIR}
    mkdir -p ${WORKDIR}/tarball
    mkdir -p ${CURDIR}/tarball
    cp ${CLIENT_DIR}.tar.gz ${WORKDIR}/tarball
    cp ${CLIENT_DIR}.tar.gz ${CURDIR}/tarball
}

#main

CURDIR=$(pwd)
VERSION_FILE=$CURDIR/pmm-client.properties
args=
WORKDIR=
SRPM=0
SDEB=0
RPM=0
DEB=0
SOURCE=0
TARBALL=0
OS_NAME=
ARCH=
OS=
SUBMODULE_BRANCH="master"
INSTALL=0
RPM_RELEASE=1
DEB_RELEASE=1
REVISION=0
parse_arguments PICK-ARGS-FROM-ARGV "$@"

BRANCH=$(get_branches "pmm-client" "branch")
REPO=$(get_repos "pmm-client")
MongoExp_BRANCH_NAME=$(get_branches "mongodb_exporter")
MongoExp_REPO=$(get_repos "mongodb_exporter")
TOOLKIT_BRANCH_NAME=$(get_branches "percona-toolkit")
TOOLKIT_REPO=$(get_repos "percona-toolkit")
MysqlExp_BRANCH_NAME=$(get_branches "mysqld_exporter")
MysqlExp_REPO=$(get_repos "mysqld_exporter")
ProxysqlExp_BRANCH_NAME=$(get_branches "proxysql_exporter")
ProxysqlExp_REPO=$(get_repos "proxysql_exporter")
QAN_BRANCH_NAME=$(get_branches "qan-agent")
QAN_REPO=$(get_repos "qan-agent")
NodeExp_BRANCH_NAME=$(get_branches "node_exporter")
NodeExp_REPO=$(get_repos "node_exporter")

check_workdir
get_system
install_deps
get_sources
build_tarball
build_srpm
build_source_deb
build_rpm
build_deb