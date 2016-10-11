# mm (19311931)

#### Features
* Less than 200 lines of code.
* Fetching last ten messages of each room on start.
* Receiving messages.
* Sending messages through named pipes.
* Marking latest event as read (the recommendation to only mark this
as read when the user has read the message seems iffy here. An IMAP type tagging
system could work, but would be as complex as the program itself). At least
this way, users of other clients will know your computer has recieved the
message.
* Online presence.
* Message modification time set to message timestamp (allows system ordering
by time).

###### Planned
* Syncing all message history without gaps.

###### Unsure
* Different treatment for notice and emote type messages.
* Automatic file, image, and audio downloads.
* Redact / edit messages somehow.

###### Not Planned
* More than 250 lines of code.
* Presence status other than online, such as idle.

#### Notes
I am not here to interperate trailing slashes and such so:
* -s <host> must include a protocol ( http:// or https:// ) and have no trailing
  slash. If the server includes a port, use server.name:port. For example,
	`-s https://matrix.org`
* -d <dir> will create that directory structure. In `-d $HOME/chat/matrix`, all
	directories in this path will be created if they do not exist, and
	servers are placed in the matrix folder. Do not include a trailing
	slash.

#### Install (from source)
```shell
go get gitlab.com/meutraa/mm
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

#### Example Scripts
**Edit the CONFIG sections for your account and setup.**

Script that displays a short history and all new messages.
```shell
#!/bin/bash
declare -A ROOMS
declare -A NICK

# START CONFIG
cd "$HOME/mm/server.org/@account:server.org" || exit
ROOMS=(
["roomId1:server.org"]="Room Name 1"
["roomId2:server.org"]="\033[1;31mRoom Name 2\033[0m"
)
# Do not include the :server.org in these.
NICK=(
["@contact1"]="nick1"
["@contact2"]="\033[0;31mnick2\033[0m"
)
# END CONFIG

function message {
        FILE="!"$(echo "$@" | cut -d'!' -f2)
        ROOM=${ROOMS[$(echo "$FILE" | cut -d'/' -f1)]}
        if [ "$ROOM" != "$CUR_ROOM" ]; then
                CUR_ROOM="$ROOM"
                echo -e "\n-- $CUR_ROOM -- "
        fi
        SENDER=${NICK[$(echo "$FILE" | grep -o '\@[^\:]*')]}
        UNIX_TIME=$(stat -c "%Y" "$FILE")
        TIME=$(date -d "@$UNIX_TIME" +%R)
        echo -e "\a$TIME  $SENDER\t $(cat "$FILE")"
}
CUR_ROOM=""
OLD_MSGS=($(ls -1rt \!*/@*/\$* | tail -n 40))
for i in "${OLD_MSGS[@]}"; do message "$i"; done
inotifywait -m -q -e close_write --exclude ".*\/in" -r --format '%w%f' ~/mm |
while read -r MESSAGE; do
        message "$MESSAGE"
done
```

And here is a dmenu script to send messages.
```shell
#!/bin/bash
declare -A ROOM

# START CONFIG
cd "$HOME/mm/server.org/@account:server.org" || exit
FONT="Inconsolata:size=28"
ROOM=(
["roomName1"]="!roomId1:lost.host"
["roomName2"]="!roomId2:lost.host"
)
# END CONFIG

NAMES=""
for k in "${!ROOM[@]}"; do
        NAMES+="$k\n"
done
NAME=$(echo -e "$NAMES" | dmenu -b -fn "$FONT")
if [ "$?" -ne 0 ]; then exit; fi
MESSAGE=$(echo "" | dmenu -b -fn "$FONT" -p "$NAME")
case $MESSAGE in
        (*[![:blank:]]*) echo "$MESSAGE" > "${ROOM[$NAME]}/in";;
        (*) exit
esac
```
