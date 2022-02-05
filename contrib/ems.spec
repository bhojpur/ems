%define name ems
%define version 1.0.0-alpha
%define release 1
%define path usr/local
%define group Database/Applications
%define __os_install_post %{nil}

Summary:    ems
Name:       %{name}
Version:    %{version}
Release:    %{release}
Group:      %{group}
Packager:   Bhojpur Consulting <info@bhojpur-consulting.com>
License:    MIT
BuildRoot:  %{_tmppath}/%{name}-%{version}-%{release}
AutoReqProv: no
# we just assume you have Go installed. You may or may not have an RPM to depend on.
# BuildRequires: go

%description 
Bhojpur EMS - A realtime distributed messaging platform
https://github.com/bhojpur/ems

%prep
mkdir -p $RPM_BUILD_DIR/%{name}-%{version}-%{release}
cd $RPM_BUILD_DIR/%{name}-%{version}-%{release}
git clone git@github.com:bhojpur/ems.git

%build
cd $RPM_BUILD_DIR/%{name}-%{version}-%{release}/ems
make PREFIX=/%{path}

%install
export DONT_STRIP=1
rm -rf $RPM_BUILD_ROOT
cd $RPM_BUILD_DIR/%{name}-%{version}-%{release}/ems
make PREFIX=/${path} DESTDIR=$RPM_BUILD_ROOT install

%files
/%{path}/bin/emsadmin
/%{path}/bin/emsd
/%{path}/bin/emslookupd
/%{path}/bin/ems_to_file
/%{path}/bin/ems_to_http
/%{path}/bin/ems_to_ems
/%{path}/bin/ems_tail
/%{path}/bin/ems_stat
/%{path}/bin/to_ems