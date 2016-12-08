#!/bin/bash

GISA_BIN=/home/azurelysium/Bin/gisa
DATABASE=/home/azurelysium/articles.db
PAGE_SIZE=10
PAGE=0

while true; do
    clear
    echo -e "PAGE: $PAGE\n"
    mapfile -t ARTICLES < <( $GISA_BIN list --database $DATABASE --pageSize $PAGE_SIZE --page $PAGE )

    ARTICLE_IDS=()
    for i in "${!ARTICLES[@]}"
    do
        ARTICLE=${ARTICLES[$i]}

        ID=$( echo "$ARTICLE" | cut -f1)
        ARTICLE_IDS+=($ID)

        TITLE=$( echo "$ARTICLE" | cut -f3)
        echo "[$i] $TITLE"
    done

    echo -e -n "\n[n,p,0-9,q] >> "
    read -n 1 COMMAND

    RE_NUMBER='^[0-9]+$'
    if [[ $COMMAND =~ $RE_NUMBER ]]; then
        $GISA_BIN print --database $DATABASE --id ${ARTICLE_IDS[$COMMAND]} | less
    else
        if [ $COMMAND = "n" ]; then
            PAGE=$((PAGE + 1))
        elif [ $COMMAND = "p" ]; then
            PAGE=$((PAGE - 1))
        elif [ $COMMAND = "q" ]; then
            exit
        fi
    fi
done
