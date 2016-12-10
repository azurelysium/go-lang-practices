#!/bin/bash

GISA_BIN=/home/azurelysium/Bin/gisa
DATABASE=/home/azurelysium/Documents/articles.db
PAGE_SIZE=10
PAGE=0
UNREAD_ONLY=false

RECENT_ID=''
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

        READ=$( echo "$ARTICLE" | cut -f3)
        TITLE=$( echo "$ARTICLE" | cut -f4)
        if [ $READ = "1" ]; then
            READ="*"
        else
            READ=" "
        fi
        echo "$i: [$READ] $TITLE"
    done

    echo -e -n "\n[n,p,0-9,i,a,q] >> "
    read -n 1 COMMAND

    RE_NUMBER='^[0-9]+$'    
    if [[ $COMMAND =~ $RE_NUMBER ]]; then
        RECENT_ID=${ARTICLE_IDS[$COMMAND]}
        $GISA_BIN read --database $DATABASE --id ${ARTICLE_IDS[$COMMAND]}
        $GISA_BIN show --database $DATABASE --id ${ARTICLE_IDS[$COMMAND]} | fold -w 120 -s | most -d -s-w
    else
        if [ $COMMAND = "n" ]; then
            PAGE=$((PAGE + 1))
        elif [ $COMMAND = "p" ]; then
            PAGE=$((PAGE - 1))
        elif [ $COMMAND = "q" ]; then
            break
        elif [ $COMMAND = "i" ]; then
            while true; do
                echo -e -n "\nSelect an article to be ignored [0-9] >> "
                read -n 1 COMMAND
                if [[ $COMMAND =~ $RE_NUMBER ]]; then
                    $GISA_BIN ignore --database $DATABASE --id ${ARTICLE_IDS[$COMMAND]}
                    echo -e " .. ignored."
                    sleep 1
                    break
                fi
            done
        elif [ $COMMAND = "a" ]; then
            if [ $COMMAND = "" ]; then
                echo -e " .. RECENT_ID is null."
            else
                $GISA_BIN archive --database $DATABASE --id $RECENT_ID
                echo -e " .. $RECENT_ID archived."
            fi
            sleep 1
        fi
    fi
done
clear
