package main

type Payload struct {
    Chat []string   `json:"Chat,omitempty"`
    GameRoomId string `json:"GameRoomId,omitempty"`
    Players []PlayerData `json:"Players,omitempty"`
    ClientId int       `json:"ClientId,omitempty"`
    Winner int        `json:"Winner,omitempty"` 
}

type PlayerData struct {
    Id int `json:"Id,omitempty"`        //players id sent with each data
    X int `json:"X,omitempty"`          //players X
    Y int `json:"Y,omitempty"`          //players Y
    J bool `json:"J,omitempty"`         //bool denoted if they jumped or not
    D bool `json:"D,omitempty"`         //bool denoted if the player is dead
}

func newPayload() Payload {
    return Payload{
        Chat: []string{},
        GameRoomId: "",
        Players:    []PlayerData{},
    }
}

func newIdPayload(id int) Payload {
    return Payload{
        ClientId:  id,
    }
}
