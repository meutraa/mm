package main

type auth struct {
	Type string `json:"type"`
	User string `json:"user"`
	Pass string `json:"password"`
}

type data struct {
	NextBatch string `json:"next_batch"`
	Rooms     rooms  `json:"rooms"`
}

type rooms struct {
	Join map[string]joinedRooms `json:"join"`
}

type joinedRooms struct {
	Timeline timeline `json:"timeline"`
}

type timeline struct {
	Events    []event `json:"events"`
	PrevBatch string  `json:"prev_batch"`
}

type event struct {
	Timestamp int64   `json:"origin_server_ts"`
	EventId   string  `json:"event_id"`
	Type      string  `json:"type"`
	Content   content `json:"content"`
	Sender    string  `json:"sender"`
}

type message struct {
	Body string `json:"body"`
	Type string `json:"msgtype"`
}

type content struct {
	message
	Name   string `json:"name"`
	Url    string `json:"url"`
	GeoUri string `json:"geo_uri"`
	FileInfo info `json:"info"`
}

type info struct {
	Height int `json:"h"`
	Width int `json:"w"`
	Size int `json:"size"`
	MimeType string `json:"mimetype"`
}

type session struct {
	Token        string `json:"access_token"`
	Homeserver   string `json:"home_server"`
	UserId       string `json:"user_id"`
	DeviceId     string `json:"device_id"`
	CurrentBatch string
}
