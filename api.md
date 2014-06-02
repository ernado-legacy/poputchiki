Попутчики/Документация REST API
=========================== 

## Аутентификация
Используются токены. Токен должен быть либо в куках `token`, либо в url `/api/query/.../?token=token`
Получить токен можно послав запрос на `/api/auth/login`, либо он выдается после регистрации `/api/auth/register`

## API v0.1




```
#format
/base
    /dir1
        ...
            /dirn - method(input type) -> output type
```

```
# root
/api
    /auth
        /register - post(form)
        /login - post(form)
        /logout - post()

    /user/:id
        - get() -> user
        - put(user)

        # get current status of user
        /status - get() -> status

        # messaging system
        /messages - get() -> message[] # get messages from user :id for current user
        /messages - put(message)       # send message from current user to user :id

        # favorites
        /fav
            - post(id)
            - delete(id)
            - get() -> user[]

        /blacklist - post(id)
        /blacklist - delete(id)

        # add to guests 
        /guests - post(id)
        /guests - get() -> user[]

        # not implemented
        # get list of user photo ???
        /photo - get() -> photo[]
        # get list of users video
        /video - get() -> video[]

    # not implemented
    /photo ???
        - put(photo)
        /:id
            - get()
            - delete()

    # not implemented
    /album ???
        / - put()
        /:id
            - get()
            - delete()
            /photo
                - get()
                - put(photo)
                - delete()
                - post(form) -> photo

    # not implemented
    /audio
        - put(audio)
        /:id
            - get() -> audio
            - delete()

    # not implemented
    /video
        - put(video)
        /:id
            - get() -> video
            - delete()

    # statuses system
    /status - put(status)
        /:id
            - get() -> status
            - put(status)
            - delete()

    # not implemented
    /stripe
        - put(stipeitem)
        /:id - get(stipeitem)
            /comments
                - get() -> comment[]
                - put(comment) -> comment
                /:id - put(comment) -> comment
                /:id - delete()

    /message/:id - delete()

    /upload
        /image - post(form) -> file
        /video - post(form) -> file
        # not implemented
        /audio - post(form) -> file

    /realtime - get()->[ws protocol upgrade]
```
# models and types

```
#format
typename {
    key1 type1
    ...
    keyn typen
}
```


```
guest {
    id      objectId
    user    objectId
    guest   objectId
    time    time.Time
}

message {
    id          objectId
    user        objectId
    origin      objectId
    destination objectId
    time        time.Time
    text        string
}

# поля origin, destination, time заполняются на бекенде

realtimeevent {
    type string
    body Object
    time time.Time
}

progressmessage {
    progress float32
}

messagesendblacklisted {
    id objectId
}

comment {
    id    objectId
    user  objectId
    text  string
    time  time.Time
}

statusupdate {
    id        objectId
    user      objectId
    time      time.Time
    text      string
    comments  comment[]
}

stripeitem {
    id         objectId
    user       objectId 
    image      objectId
    image_url  string
    countries  sttring[]
    time       time.Time
}

file {
    id   objectId
    fid  string
    user objectId
    time time.Time
    type string
}

photo {
    id            objectId
    user          objectId
    image         objectId
    image_url     string
    thumbnail     objectId
    thumbnail_url string
    description   string 
    time          time.Time
    comments      comment[]
}
```