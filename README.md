# mm (19311931)
The [Matrix](https://matrix.org/) client that requires you to read to the bottom
of this README.

#### Features
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

###### Unsure
These are all up for discussion.

* Different treatment for notice and emote type messages.
* Automatic file, image, and audio downloads.
* Redact / edit messages somehow.
* Automatically accepting room invites.
* A seperate program for sending room invites. Maybe.
* Using the filter API to limit data sent.

###### Not Planned
* Presence status other than online, such as idle.

#### Notes
* -s <host> `[scheme://]host[:port][/path]`
* Scheme, port, and path are optional.
* Will assume https if no scheme is provided.
* -d <dir> will create that directory structure. In `-d $HOME/chat/matrix`, all
	directories in this path will be created if they do not exist, and
	servers are placed in the matrix folder. Do not include a trailing
	slash.

#### Install (or update)
```shell
go get -u gitlab.com/meutraa/mm
```

###### Cross Compiling
See https://golang.org/doc/install/source#environment for GOOS and GOARCH combinations.
```shell
git clone git@gitlab.com:meutraa/mm.git
cd mm
GOOS=linux GOARCH=arm go build
```

#### Directory Structure
Structure works with multiple servers and accounts.
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

#### Usage
ls, tail, cat, find, and echo are your best friends.

```shell
mm [-d dir] -s https://host[:port] -u user -p password

Examples:
mm -s https://matrix.org -u bob -p 1234
mm -d "$HOME/chat/mm" -s http://localhost:8008 -u "$USER" -p pass
```

Send message to room
```shell
echo "message" > in
```

View all messages in room (newest last)
```shell
cat `ls -1rt @*/*`
```

#### Example POSIX shell scripts
**Make sure to read and edit the config blocks.**

Script that displays a short history and all new messages.
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

CUR_ROOM=""
message() {
    while read -r MSG; do
        FILE="!"$(echo "$MSG" | cut -d'!' -f2)
        ROOMID=$(echo "$FILE" | cut -d'/' -f1)
        ROOM=$(echo "$ROOMS" | grep "$ROOMID" | cut -d'=' -f2)
        if [ -z "$ROOM" ]; then ROOM="$ROOMID"; fi
        if [ "$ROOM" != "$CUR_ROOM" ]; then
            CUR_ROOM="$ROOM"
            printf "\n-- $ROOM --\n"
        fi
        SENDER=$(echo "$FILE" | cut -d'/' -f2)
        NICK=$(echo "$NICKS" | grep "$SENDER" | cut -d'=' -f2)
        if [ -z "$NICK" ]; then NICK="$SENDER"; fi
        TIME=$(ls -l "$FILE" | awk '{ print $8 }')
        printf "\a$TIME  $NICK\t $(cat "$FILE")\n"
    done
}

ls -1rt \!*/@*/\$* | tail -n 40 | message
tail -n 0 -f "$MMOUT" | message
```

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
