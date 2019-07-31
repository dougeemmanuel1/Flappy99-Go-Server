package main

type Payload struct {
    Chat []string   `json:"Chat"`
    GameRoomId string `json:"GameRoomId"`
}

func newPayload() Payload {
    return Payload{
        Chat: []string{},
        GameRoomId: "",
    }
}
