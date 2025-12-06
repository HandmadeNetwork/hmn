#!/bin/bash

if [ ! -e "cinera.conf" ]; then
    echo "Can't find cinera.conf"
    exit
fi
. cinera.conf

if [ ! -d $CINERA_REPO_PATH ]; then
    git clone --config core.sshCommand="ssh -i ~/.ssh/gitlab-annotation-system" git@git.handmade.network:Annotation-Pushers/Annotation-System.git $CINERA_REPO_PATH
fi

if [ ! -d $CINERA_REPO_PATH ]; then
    echo "Failed to clone cinera"
    exit
fi


cd $CINERA_REPO_PATH
git pull
if [[ -z "${CINERA_VERSION}" ]]; then
	git checkout master
else
	git checkout $CINERA_VERSION
fi
cd $CINERA_REPO_PATH/hmmlib
make
cp hmml.a hmmlib.h ../cinera
cd $CINERA_REPO_PATH/cinera
`$SHELL cinera.c`

if [ ! -d $CINERA_ASSETS_PATH ]; then
	mkdir $CINERA_ASSETS_PATH
fi

cp -av $CINERA_REPO_PATH/cinera/*.{css,js,png} $CINERA_ASSETS_PATH
