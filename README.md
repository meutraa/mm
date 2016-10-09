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

###### Planned
* Syncing all message history without gaps.
* Sending messages through named pipes.

###### Unsure
* Different treatment for notice and emote type messages.
* Automatic file, image, and audio downloads.

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

#### Directory Structure
Symbolic links are created for each room with the name of the room, or the
participant of the private chat.
```
mm			(-d <dir>)
└── server.org		(-s <host>)
    ├── @roomName:server.org -> !roomID:server.org
    └── !roomID:server.org
        ├── @person1:server.org
        │   ├── message1
        │   └── message2
        └── @me:server.org
            ├── message1
            └── message2
```

#### Usage
Some commands that could prove useful. All of these print newest last.

Inside a room directory
```
List messages:
$ ls -tr1 */*		(without time)
$ ls -gGo */*		(with time)

Print room messages:
$ cat `ls -tr */*`	(without filenames)
$ tail */*		(with filenames)
```

Inside a server directory
```
List all messages:
cat `find -L -type f | grep -v ! | sort -t '/' -k 4,4`
cat `ls -rt1 \!*/*/*`

List last 10 messages with headers:
tail `find -L -type f | grep -v ! | sort -t '/' -k 4,4 | tail -n 10`
tail `ls -rt1 */*/* | grep -v '!' | tail -n 10`

```
