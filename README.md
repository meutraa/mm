This file represents the current functionality, which is not yet complete.

#### Features
* Fetching last ten messages of each room on start.
* Receiving messages.
* Marking latest event as read (the recommendation to only mark this
as read when the user has read the message seems iffy here. An IMAP type tagging
system could work, but would be as complex as the program itself). At least
this way, users of other clients will know your computer has recieved the
message.
* Online presence.
* Message modification time set to message timestamp (allows system ordering
by time).
* Sending messages through named pipes.

###### Planned
* Syncing all message history without gaps.
* Swap exchange token for access token when access token expires.
* Message redaction from other clients.

###### Unsure
* Should messages be sent by creating files under an in directory instead of a
fifo pipe?
* Different treatment for notice and emote type messages.
* Automatic file, image, and audio downloads.
* Redact / edit messages somehow.

###### Not Planned
* Presence status other than online, such as idle.

#### Notes
I am not here to interperate trailing slashes and such so:
* If your homeserver/username/password contains funky characters, enclose it
	in quotation marks. For example, `-p "\"ha\?!$t"`. Remember to escape
	any quotation marks with a backslash as in the example.
* -s <host> must include a protocol (http:// or https://) and have no trailing
  slash. If the server includes a port, use server.name:port. For example,
	`-s https://matrix.org`
* -d <dir> will create that directory structure. In `-d $HOME/chat/matrix`, all
	directories in this path will be created if they do not exist, and
	servers are placed in the matrix folder. Do not include a trailing
	slash.

#### Install (from source)
```
go get gitlab.com/meutraa/mm

```
###### Cross Compiling
See (doc/install/source)[https://golang.org/doc/install/source#environment] for
GOOS and GOARCH combinations.
```
git clone git@gitlab.com:meutraa/mm.git
cd mm
GOOS=linux GOARCH=arm go build
```

#### Directory Structure
```
Structure should work with multiple servers and accounts.
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
```
Send message to room
$ echo "message" > in

View all message in room (newest last)
$ cat `ls -1rt @*/*`

Simple script that displays all new messages with time and sender prefixed.
#/bin/sh
while true; do
        FILE=`inotifywait -q -e close_write --exclude ".*\/in" -r --format '%w%f' ~/mm`
        SENDER=`echo "$FILE" | grep -o '@\([^\/]*\)' | tail -n 1`
	UNIX_TIME=`stat -c "%Y" "$FILE"`
        TIME=`date -d "@$UNIX_TIME" +%R`
        echo "$TIME " "$SENDER:" `cat "$FILE"`
done
```
