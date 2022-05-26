#!/bin/bash
go build main.go
mv main sag-bot
chmod +x sag-bot
while :
do
    ./sag-bot
done