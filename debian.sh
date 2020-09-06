#!/bin/bash
version=$1
if [ -d debian ];then
    rm -rf debian
fi
dh_make -p lpmx_$1 --single --native --copyright apache --email jasonyangshadow@gmail.com
rm debian/*.ex debian/*.EX
perl -pi -e "s/unstable/$(lsb_release -cs)/" debian/changelog
perl -pi -e 's/^(Section:).*/$1 utils/' debian/control
perl -pi -e 's/^(Homepage:).*/$1 https:\/\/github.com\/jasonyangshadow\/lpmx/' debian/control
perl -pi -e 's/^#(Vcs-Browser:).*/$1 https:\/\/github.com\/jasonyangshadow\/lpmx/' debian/control
perl -pi -e 's/^#(Vcs-Git:).*/$1 https:\/\/github.com\/jasonyangshadow\/lpmx.git/' debian/control
perl -pi -e 's/^(Description:).*/$1 A rootless composable container system/' debian/control
perl -i -0777 -pe "s/(Copyright: ).+\n +.+/\${1}$(date +%Y) Xu Yang <jasonyangshadow\@gmail.com>/" debian/copyright

debuild -S -kjasonyangshadow@gmail.com
dput ppa:jasonyangshadow/lpmxppa ../lpmx_$1_source.changes
