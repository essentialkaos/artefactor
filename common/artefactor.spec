################################################################################

%global crc_check pushd ../SOURCES ; sha512sum -c %{SOURCE100} ; popd

################################################################################

%define debug_package  %{nil}

################################################################################

%define _srv  /srv

################################################################################

Summary:        Utility for downloading artefacts from GitHub
Name:           artefactor
Version:        0.5.0
Release:        0%{?dist}
Group:          Applications/System
License:        Apache License, Version 2.0
URL:            https://kaos.sh/artefactor

Source0:        https://source.kaos.st/%{name}/%{name}-%{version}.tar.bz2

Source100:      checksum.sha512

BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:  golang >= 1.22

Provides:       %{name} = %{version}-%{release}

################################################################################

%description
Utility for downloading artefacts from GitHub.

################################################################################

%prep
%{crc_check}

%setup -q

%build
if [[ ! -d "%{name}/vendor" ]] ; then
  echo "This package requires vendored dependencies"
  exit 1
fi

pushd %{name}
  %{__make} %{?_smp_mflags} all
popd

%install
rm -rf %{buildroot}

install -dDm 755 %{buildroot}%{_srv}/artefactor/data

install -pDm 755 %{name}/%{name} \
                 %{buildroot}%{_bindir}/%{name}
install -pDm 644 %{name}/common/artefacts.yml \
                 %{buildroot}%{_srv}/%{name}/artefacts.yml

install -pDm 640 %{name}/common/%{name}.sysconfig \
                 %{buildroot}%{_sysconfdir}/sysconfig/%{name}
install -pDm 644 %{name}/common/%{name}.service \
                 %{buildroot}%{_unitdir}/%{name}.service
install -pDm 644 %{name}/common/%{name}.timer \
                 %{buildroot}%{_unitdir}/%{name}.timer

%clean
rm -rf %{buildroot}

################################################################################

%files
%defattr(-,root,root,-)
%doc %{name}/LICENSE
%dir %{_srv}/%{name}/data
%config(noreplace) %{_srv}/%{name}/artefacts.yml
%config(noreplace) %{_unitdir}/%{name}.service
%config(noreplace) %{_unitdir}/%{name}.timer
%{_bindir}/%{name}

################################################################################

%changelog
* Fri Mar 22 2024 Anton Novojilov <andy@essentialkaos.com> - 0.4.2-0
- Minor UI improvements

* Mon Mar 04 2024 Anton Novojilov <andy@essentialkaos.com> - 0.4.1-0
- Initial build for kaos-repo
