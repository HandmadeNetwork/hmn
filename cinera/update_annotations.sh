#!/bin/bash

if [ ! -e "cinera.conf" ]; then
    echo "Can't find cinera.conf"
    exit
fi
. cinera.conf

if [ ! -d $CINERA_HMML_PATH ]; then
    git clone --config core.sshCommand="ssh -i ~/.ssh/gitlab-hmml" git@gitssh.handmade.network:Annotation-Pushers/cinera_handmade.network.git $CINERA_HMML_PATH
fi

if [ ! -d $CINERA_HMML_PATH ]; then
    echo "Failed to clone annotation repo"
    exit
fi

cd $CINERA_HMML_PATH
git pull
chown -R $ANNOTATIONS_USER:$ANNOTATIONS_USER_GROUP $CINERA_HMML_PATH

cp -av $CINERA_HMML_PATH/cmuratori/hero/cinera__hero.css $CINERA_ASSETS_PATH/
cp -av $CINERA_HMML_PATH/miotatsu/riscy/cinera__riscy.css $CINERA_ASSETS_PATH/
cp -av $CINERA_HMML_PATH/pervognsen/bitwise/cinera__bitwise.css $CINERA_ASSETS_PATH/
