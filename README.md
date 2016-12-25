# mm (19311931)
The [Matrix](https://matrix.org/) client that requires you to read to the bottom
of this README.

If you do not want a gitlab account but have problems or suggestions,
send an email to my gmail address: bendypauldron

### Features
* Less than 250 lines of code.
* Fetching last ten messages of each room on start.
* Receiving messages by long polling sync call.
* Sending messages through named pipes.
* Marking latest event as read (the recommendation to only mark this
as read when the user has read the message seems iffy here. An IMAP type tagging
system could work, but would be as complex as the program itself). At least
this way, users of other clients will know your computer has recieved the
message.
* Online presence.
* Message modification time set to message timestamp minus five seconds.
* List of new messages (file paths) written to stdout

###### Planned
* Syncing all message history without gaps.

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
            ├── in
            ├── @account1:server.org
            │   └── $messageId:server.org
            └── @contact1:server.org
                └── $messageId:server.org
```

### Usage
ls, tail, cat, find, and echo are your best friends.

```shell
mm [-d dir] -s [scheme://]host[:port][/path] -u user -p password [-c path]

Examples:
mm -s matrix.org -u bob -p 1234
mm -d "$HOME/chat/mm" -s http://localhost:8008 -u "$USER" -p pass -c "$HOME/chat/mm/cert.pem"
```

Send message to room
```shell
echo "message" > in
```

## mchat
This is a POSIX system mm client, the one I use.

**Make sure to read and edit the config blocks.**
```shell
#!/bin/sh
# mm outputs all newly written messages to stdout. Write stdout to  a file and
# set $MMOUT to that file and this script will print new messages.
# mm -s https://server.org -u user -p pass 2>> ~/mm/log 1>> ~/mm/out &

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
And here is a dmenu script to send messages. Obviously dmenu is not POSIX.
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
