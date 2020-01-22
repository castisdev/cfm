#!/bin/bash -e
#set -x #echo on

source ./scripts/build.sh

#${AppName}, ${AppVersion}, ${AppPreRelVer} 은 build.sh에서 구함
#${CapitalAppName}은 build.sh에서 구함
#${GoVer}은 build.sh에서 구함

BuildDir=build_${GoVer}
PackDir=package_${GoVer}
if [ -n "$AppVersion" ]
then
	VERSION=$AppVersion-$AppPreRelVer
else
	VERSION=$(./${BuildDir}/${AppName} -version | awk '{print $2}')
fi
PackAppName=${AppName}-v$VERSION
PackAppName64=${PackAppName}-x86_64-${GoVer}
PackAppName32=${PackAppName}-i386-${GoVer}
PackName=${AppName}-v$VERSION-${GoVer}

cp ${BuildDir}/${AppName} ${BuildDir}/${PackAppName64}
cp ${BuildDir}/${AppName}-i386 ${BuildDir}/${PackAppName32}

rm -rf ${PackDir}
mkdir -p ${PackDir}/bin
mkdir -p ${PackDir}/doc

mv ./${BuildDir}/${PackAppName}* ${PackDir}/bin
cp doc/${AppName}.yml ${PackDir}/doc
cp doc/ReleaseNote-${CapitalAppName}.txt ${PackDir}/doc
cp doc/*.png ${PackDir}/doc/.

scripts/md_to_pdf.py doc/API.md ${PackDir}/doc/API.pdf
scripts/md_to_pdf.py doc/CHANGELOG.md ${PackDir}/doc/CHANGELOG.pdf
scripts/md_to_pdf.py doc/SEQUENCE.md ${PackDir}/doc/SEQUENCE.pdf

scripts/md_to_html.py doc/API.md ${PackDir}/doc/API.html
scripts/md_to_html.py doc/CHANGELOG.md ${PackDir}/doc/CHANGELOG.html
scripts/md_to_html.py doc/SEQUENCE.md ${PackDir}/doc/SEQUENCE.html

mv ${PackDir} ${PackName}
tar cvzf ${PackName}.tar.gz ${PackName}
rm -rf ${PackName}

# fab -f scripts/fabfile.py -H d7@172.16.45.11 package_copy
#rm -f ${PackName}.tar.gz
