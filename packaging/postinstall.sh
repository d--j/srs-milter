#!/bin/sh

set -e

use_systemctl="True"
#systemd_version=0
if ! command -V systemctl >/dev/null 2>&1; then
  use_systemctl="False"
#else
#    systemd_version=$(systemctl --version | head -1 | sed 's/systemd //g')
fi

fixSystemdUnit() {
  if ! getent group nogroup >/dev/null 2>&1; then
    sed -i -e "s/^SupplementaryGroups=nogroup/SupplementaryGroups=nobody/" /lib/systemd/system/srs-milter.service || printf "\033[31m Could not fix srs-milter.service\033[0m\n"
  fi
}

errorNoSrsKey() {
  printf "\033[31m Could not generate secret. Replace __SRS_KEY__ with random data in /etc/srs-milter/srs-milter.yml\033[0m\n"
}

seedConfig() {
  if grep -q __SRS_KEY__ /etc/srs-milter/srs-milter.yml; then
    if [ -f /etc/postsrsd.secret ] && SECRET_KEY=$(cat /etc/postsrsd.secret); then
      printf "\033[32m Re-Using SRS secret from /etc/postsrsd.secret\033[0m\n"
      sed -i -e "s/__SRS_KEY__/$SECRET_KEY/" /etc/srs-milter/srs-milter.yml || errorNoSrsKey
    else
      printf "\033[32m Generating random SRS secret key\033[0m\n"
      if [ -c /dev/urandom ]; then
        INPUT=/dev/urandom
      else
        INPUT=/dev/random
      fi
      if [ -c "$INPUT" ] && SECRET_KEY=$(LC_ALL=C tr -dc 'A-Za-z0-9' 2>/dev/null <"$INPUT" | head -c 64); then
        sed -i -e "s/__SRS_KEY__/$SECRET_KEY/" /etc/srs-milter/srs-milter.yml || errorNoSrsKey
      else
        errorNoSrsKey
      fi
    fi
  fi
  if grep -q __SRS_DOMAIN__ /etc/srs-milter/srs-milter.yml; then
    if [ -f /etc/postsrsd.srs-domain ] && DOMAIN=$(cat /etc/postsrsd.srs-domain); then
      printf "\033[32m Re-Using SRS domain from /etc/postsrsd.srs-domain (%s)\033[0m\n" "$DOMAIN"
    else
      # start with the fully qualified domain name of this machine
      DOMAIN=$(hostname -f)
      # If Postfix is installed set the SRS domain to mydomain
      if command -V postconf >/dev/null 2>&1; then
        DOMAIN=$(postconf -h mydomain 2>/dev/null)
      # If Sendmail is installed and the j macro defined, use this
      elif [ -f /etc/mail/sendmail.cf ] && grep -q '^Dj' /etc/mail/sendmail.cf; then
        DOMAIN=$(awk '/^Dj/ { print substr($0, 3) }' /etc/mail/sendmail.cf)
      fi
      printf "\033[32m Setting SRS domain to %s\033[0m\n" "$DOMAIN"
    fi
    if ! sed -i -e "s/__SRS_DOMAIN__/$DOMAIN/" /etc/srs-milter/srs-milter.yml; then
      printf "\033[31m Could not set SRS domain Replace __SRS_DOMAIN__ with the SRS domain in /etc/srs-milter/srs-milter.yml\033[0m\n"
    fi
  fi
}

cleanInstall() {
  if getent group nogroup >/dev/null 2>&1; then
    chgrp nogroup /etc/srs-milter /etc/srs-milter/srs-milter.yml || :
  else
    chgrp nobody /etc/srs-milter /etc/srs-milter/srs-milter.yml || :
  fi
  chmod 0750 /etc/srs-milter || :
  chmod 0640 /etc/srs-milter/srs-milter.yml || :
  seedConfig
  fixSystemdUnit
  if [ "${use_systemctl}" = "False" ]; then
    printf "\033[31m srs-milter does not support your init system. You need to setup daemon starting on your own.\033[0m\n"
  else
    printf "\033[32m Reload the service unit from disk\033[0m\n"
    systemctl daemon-reload || :
    printf "\033[32m Unmask the service\033[0m\n"
    systemctl unmask srs-milter.service || :
    printf "\033[32m Set the preset flag for the service unit\033[0m\n"
    systemctl preset srs-milter.service || :
    printf "\033[32m Set the enabled flag for the service unit\033[0m\n"
    systemctl enable srs-milter.service || :
    systemctl restart srs-milter.service || :
  fi
}

upgrade() {
  seedConfig
  fixSystemdUnit
  systemctl daemon-reload || :
  systemctl restart srs-milter.service || :
}

action="$1"
if [ "$1" = "configure" ] && [ -z "$2" ]; then
  # Alpine linux does not pass args, and deb passes $1=configure
  action="install"
elif [ "$1" = "configure" ] && [ -n "$2" ]; then
  # deb passes $1=configure $2=<current version>
  action="upgrade"
fi

case "$action" in
"1" | "install")
  cleanInstall
  ;;
"2" | "upgrade")
  upgrade
  ;;
*)
  # $1 == version being installed
  cleanInstall
  ;;
esac
