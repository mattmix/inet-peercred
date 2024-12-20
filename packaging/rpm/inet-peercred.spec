Name:           inet-peercred
Version:        0.1.1
Release:        1%{?dist}
Summary:        A simple server to provide the peer credentials of an inet socket

License:        MIT License
Source0:        %{name}-%{version}.tar.gz
URL:            https://github.com/mattmix/inet-peercred

BuildRequires:  go-toolset
Requires:       systemd

%description

A simple server to provide the peer credentials of an inet socket

This server listens on port 411 and responds to queries to give information about the user/group that is using a given connection.

%global debug_package %{nil}

%prep
%autosetup

%build
go build -v -o %{name}

%install
install -Dpm 0755 %{name} %{buildroot}%{_sbindir}/%{name}
install -Dpm 0644 packaging/rpm/%{name}.service %{buildroot}%{_unitdir}/%{name}.service
install -Dpm 0644 packaging/rpm/%{name}.sysconfig %{buildroot}%{_sysconfdir}/sysconfig/%{name}

%clean
rm -rf %{buildroot}

%files
%{_sbindir}/%{name}
%{_unitdir}/%{name}.service
%config(noreplace) %{_sysconfdir}/sysconfig/%{name}

%changelog
* Fri Dec 11 2024 Initial RPM
- 