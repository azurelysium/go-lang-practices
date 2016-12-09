#!/bin/bash

GISA_BIN=/home/azurelysium/Bin/gisa
DATABASE=/home/azurelysium/Documents/articles.db
PAGE_SIZE=10
PAGE=0
UNREAD_ONLY=false

while true; do
    clear
    echo -e "PAGE: $PAGE\n"
    mapfile -t ARTICLES < <( $GISA_BIN list --database $DATABASE --pageSize $PAGE_SIZE --page $PAGE --unreadOnly=$UNREAD_ONLY)

    ARTICLE_IDS=()
    for i in "${!ARTICLES[@]}"
    do
        ARTICLE=${ARTICLES[$i]}

        ID=$( echo "$ARTICLE" | cut -f1)
        ARTICLE_IDS+=($ID)

        TITLE=$( echo "$ARTICLE" | cut -f3)
        echo "[$i] $TITLE"
    done

    echo -e -n "\n[n,p,0-9,a,q] >> "
    read -n 1 COMMAND

    RE_NUMBER='^[0-9]+$'
    if [[ $COMMAND =~ $RE_NUMBER ]]; then
        $GISA_BIN read --database $DATABASE --id ${ARTICLE_IDS[$COMMAND]}
        $GISA_BIN show --database $DATABASE --id ${ARTICLE_IDS[$COMMAND]} | less
    else
        if [ $COMMAND = "n" ]; then
            PAGE=$((PAGE + 1))
        elif [ $COMMAND = "p" ]; then
            PAGE=$((PAGE - 1))
        elif [ $COMMAND = "q" ]; then
            exit
        elif [ $COMMAND = "a" ]; then
            while true; do
                echo -e -n "\nSelect an article to be archived [0-9] >>"
                read -n 1 COMMAND
                if [[ $COMMAND =~ $RE_NUMBER ]]; then
                    $GISA_BIN archive --database $DATABASE --id ${ARTICLE_IDS[$COMMAND]}
                    echo -e " .. archived."
                    sleep 1
                    break
                fi
            done
        fi
    fi
done
clear
