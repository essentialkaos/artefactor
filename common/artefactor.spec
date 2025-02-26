################################################################################

%global crc_check pushd ../SOURCES ; sha512sum -c %{SOURCE100} ; popd

################################################################################

%define debug_package  %{nil}

################################################################################

%define _srv  /srv

################################################################################

Summary:        Utility for downloading artefacts from GitHub
Name:           artefactor
Version:        0.6.1
Release:        0%{?dist}
Group:          Applications/System
License:        Apache License, Version 2.0
URL:            https://kaos.sh/artefactor

Source0:        https://source.kaos.st/%{name}/%{name}-%{version}.tar.bz2

Source100:      checksum.sha512

BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:  golang >= 1.23

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
  echo -e "----\nThis package requires vendored dependencies\n----"
  exit 1
elif [[ -f "%{name}/%{name}" ]] ; then
  echo -e "----\nSources must not contain precompiled binaries\n----"
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

%post
if [[ $1 -eq 0 ]] ; then
  systemctl --no-reload disable %{name}.service &>/dev/null || :
  systemctl stop %{name}.service &>/dev/null || :
fi

if [[ -d %{_sysconfdir}/bash_completion.d ]] ; then
  %{name} --completion=bash 1> %{_sysconfdir}/bash_completion.d/%{name} 2>/dev/null
fi

if [[ -d %{_datarootdir}/fish/vendor_completions.d ]] ; then
  %{name} --completion=fish 1> %{_datarootdir}/fish/vendor_completions.d/%{name}.fish 2>/dev/null
fi

if [[ -d %{_datadir}/zsh/site-functions ]] ; then
  %{name} --completion=zsh 1> %{_datadir}/zsh/site-functions/_%{name} 2>/dev/null
fi

%postun
if [[ $1 -ge 1 ]] ; then
  systemctl daemon-reload &>/dev/null || :
fi

if [[ $1 == 0 ]] ; then
  if [[ -f %{_sysconfdir}/bash_completion.d/%{name} ]] ; then
    rm -f %{_sysconfdir}/bash_completion.d/%{name} &>/dev/null || :
  fi

  if [[ -f %{_datarootdir}/fish/vendor_completions.d/%{name}.fish ]] ; then
    rm -f %{_datarootdir}/fish/vendor_completions.d/%{name}.fish &>/dev/null || :
  fi

  if [[ -f %{_datadir}/zsh/site-functions/_%{name} ]] ; then
    rm -f %{_datadir}/zsh/site-functions/_%{name} &>/dev/null || :
  fi
fi

################################################################################

%files
%defattr(-,root,root,-)
%doc %{name}/LICENSE
%dir %{_srv}/%{name}/data
%config(noreplace) %{_sysconfdir}/sysconfig/%{name}
%config(noreplace) %{_srv}/%{name}/artefacts.yml
%config(noreplace) %{_unitdir}/%{name}.service
%config(noreplace) %{_unitdir}/%{name}.timer
%{_bindir}/%{name}

################################################################################

%changelog
* Tue Jan 14 2025 Anton Novojilov <andy@essentialkaos.com> - 0.6.1-0
- Fixed bug with creating symlink to the latest version for the version 'latest'
- Redownload version if it has been updated on GitHub
- Dependencies update

* Wed Sep 11 2024 Anton Novojilov <andy@essentialkaos.com> - 0.6.0-0
- Migrate to v13 version of ek package
- Code refactoring

* Mon Jun 17 2024 Anton Novojilov <andy@essentialkaos.com> - 0.5.1-0
- Dependencies update

* Wed Jun 05 2024 Anton Novojilov <andy@essentialkaos.com> - 0.5.0-0
- Full code refactoring
- Added index generation
- Added download command for downloading artefacts
- Added list command with support of remote HTTP storage
- Added cleanup command for removing outdated versions of artefacts
- Added get command for downloading artefacts files
- Improved systemd service file

* Fri Mar 22 2024 Anton Novojilov <andy@essentialkaos.com> - 0.4.2-0
- Minor UI improvements

* Mon Mar 04 2024 Anton Novojilov <andy@essentialkaos.com> - 0.4.1-0
- Initial build for kaos-repo
