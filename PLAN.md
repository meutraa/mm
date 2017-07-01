root/
  server/
    @account/
      accesstoken ?
      nextbatch   √
      !room/
        name
        → in      √
        @user/
          name      √
          avatar    √
          $message
          read        → eventId
          typing      → 0/1       (self √, other ×)

* sending read receipts				}
