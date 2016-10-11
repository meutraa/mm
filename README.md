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
These will require you to edit the highlighted variables to work with your
setup.

Simple script that displays a short history and all new messages.
```shell
#/bin/sh
# Change this to your account directory.
cd "$HOME/mm/server.org/@account:server.org"

function friendly {
	# These are optional if you want to be confused who you are talking to.
	case "$@" in
		#!roomId:server.org)	echo name;;
		*)			echo "$@"
	esac
}

# Nicknames
function senders {
	# Contact here is the fully qualified username without the server
	# So @contact:server.org would be just @contact
	case "$@" in
		#@contact)	echo name;;
		*)		echo "$@"
	esac
}

function message {
	FILE="!$(echo $@ | cut -d'!' -f2)"
	ROOM=`friendly $(echo "$FILE" | cut -d'/' -f1)`
	if [ "$ROOM" != "$CUR_ROOM" ]; then
		CUR_ROOM="$ROOM"
		echo -e "\n-- $CUR_ROOM -- "
	fi
	SENDER=`senders "$(echo $FILE | grep -o '\@[^\:]*')"`
	UNIX_TIME=`stat -c "%Y" "$FILE"`
	TIME=`date -d "@$UNIX_TIME" +%R`
	echo -e "$TIME " "$SENDER\t" `cat "$FILE"`
}

clear
CUR_ROOM=""
OLD_MSGS=(`ls -1rt */@*/* | tail -n 20`)
for i in "${OLD_MSGS[@]}"; do message "$i"; done
inotifywait -m -q -e close_write --exclude ".*\/in" -r --format '%w%f' ~/mm | \
while read MESSAGE; do
        message "$MESSAGE"
	echo -e "\a"
done
```

And here is a dmenu script to send messages.
```shell
#!/bin/sh
# Change these to your preferences.
cd "$HOME/mm/server.org/@account:server.org"
FONT="Inconsalata:size=28"

function friendly {
	# Change these to your room name mappings.
	case "$@" in
		#roomName1)	echo !roomId1:server.org;;
		#roomName2)	echo !roomId2:server.org;;
		*)		echo "$@"
	esac
}

# Add in any rooms here that you gave a friendly name.
REC=`echo -e "roomName1\nroomName2" | dmenu -b -fn "$FONT"`
if [ "$?" -ne 0 ]; then exit; fi
MESSAGE=`echo "" | dmenu -b -fn "$FONT" -p "$REC"`
case $MESSAGE in
	(*[![:blank:]]*) echo "$MESSAGE" > "`friendly $REC`/in";;
	(*) exit
esac
```
