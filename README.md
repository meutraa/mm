# mm

The [Matrix](https://matrix.org/) client that requires you to read to the bottom of this README.

### Features

* Around 400 lines of code.
* Sending messages through named pipes.
* Online presence & sending typing notifications.

### Install (or update)

```shell
go get -u gitlab.com/meutraa/mm
```

##### Cross Compiling

See https://golang.org/doc/install/source#environment for GOOS and GOARCH combinations.
```shell
git clone git@gitlab.com:meutraa/mm.git
cd mm
GOOS=linux GOARCH=arm go build
```

### Directory Structure

```
.
└── server.org
    └── @account1:server.org
        └─── !roomId:server.org
            ├─> in
            ├── @account1:server.org
            │   ├─> typing
            │   └── $messageId:server.org
            └── @contact1:server.org
                └── $messageId:server.org
```

### Usage

ls, tail, cat, find, and echo are your best friends.

```shell
Usage of mm:
  -c string
        mm configuration file (default "~/.config/mm/config.yml")
```

First run will generate a configuration file for you. 
Update the values and rerun.

Send message to room
```shell
echo "message" > in
```

## mchat

This is an example POSIX mm client. For a much faster client,
have a look at mm-client.

**Make sure to read and edit the config blocks.**
```shell
#!/bin/sh
# mm outputs all newly written messages to stdout. Write stdout to  a file and
# set $MMOUT to that file and this script will print new messages.
# mm 2>> ~/mm/log 1>> ~/mm/out &

# START CONFIG
MMOUT="$HOME/mm/out"
cd "$HOME/mm/server.org/@account1:server.org" || exit

# Filling ROOMS and NICKS in with your account and contact detail is optional
# but who does not want short room names and nicknames?
ROOMS="!roomId1:server.org=roomName 1
!roomId2:server.org=\033[1;31mroomName 2\033[0m"

NICKS="@contact1:server.org=nickname1
@account1:server.org=\033[0;37mme\033[0m"
# END CONFIG

message() {
    while read -r MSG; do
        if [ $(echo "$MSG" | grep -c "$SINGLE") -eq 0 ]; then continue; fi
        FILE="!"$(echo "$MSG" | cut -d'!' -f2)
        ROOMID=$(echo "$FILE" | cut -d'/' -f1)
        ROOM=$(echo "$ROOMS" | grep "$ROOMID" | cut -d'=' -f2)
        if [ -z "$ROOM" ]; then ROOM="$ROOMID"; fi
        SENDER=$(echo "$FILE" | cut -d'/' -f2)
        NICK=$(echo "$NICKS" | grep "$SENDER" | cut -d'=' -f2)
        if [ -z "$NICK" ]; then NICK="$SENDER"; fi
        DATE=$(ls -l "$FILE" | awk '{ print $6" "$7 }')
        if [ "$DATE" != "$CDATE" ]; then printf "\n-- $ROOM ($DATE) --\n";
        elif [ "$ROOM" != "$CROOM" ]; then printf "\n-- $ROOM --\n"; fi
        CDATE="$DATE"
        CROOM="$ROOM"
        TIME=$(ls -l "$FILE" | awk '{ print $8 }')
        printf "\a%s  $NICK\t %s\n" "$TIME" "$(cat "$FILE")"
    done
}

if [ -n "$1" ]; then
    SINGLE=$(echo "$ROOMS" | grep "$1" | cut -d'=' -f1)
fi

ls -1rt $SINGLE*/@*/\$* | tail -n 40 | message
tail -n 0 -f "$MMOUT" | message
```

## dmmsg

And here is a dmenu script to send messages.
```shell
#!/bin/sh

# START CONFIG
cd "$HOME/mm/server.org/@account:server.org" || exit
FONT="Inconsolata:size=28"

# To add a room to dmenu add it with a name to this variable. Required this time.
ALIASES="roomName1=!roomId1:server.org
roomName2=!roomId2:server.org"
# END CONFIG

NAMES=$(echo "$ALIASES" | cut -d'=' -f1)
NAME=$(echo "$NAMES" | dmenu -b -fn "$FONT")
if [ "$?" -ne 0 ]; then exit; fi
ROOMID=$(echo "$ALIASES" | grep "$NAME" | cut -d'=' -f2)
MESSAGE=$(echo "" | dmenu -b -fn "$FONT" -p "$NAME")
case $MESSAGE in
    (*[![:blank:]]*) echo "$MESSAGE" > "$ROOMID/in";;
    (*) exit
esac
```
