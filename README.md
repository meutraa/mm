This file represents the current functionality, which is not yet complete.

##### Directory Structure
Symbolic links are created for each room with the name of the room, or the participant of the private chat.
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

##### Usage
Inside a room directory
```
List messages:
$ ls -tr1 */*		(without time)
$ ls -gGo */*		(with time)

Print room messages:
$ cat `ls -tr */*`	(without filenames)
$ tail */*		(with filenames)
```

