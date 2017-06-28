root/
  server/
    account/
      accesstoken
      nextbatch
      room/
        user/
          message/
            message → string
            receipt → 0 or 1 → whether the server has confirmed this message.
            read → 0 or 1 → whether the user has read the message.

read for 3rd party messages will be an open pipe until
