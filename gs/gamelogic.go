package main

import (
    "log"
    "container/list"
    "math/rand"
)

const (
    FAKE_PLAYRERS = 0
)

type GameLogic struct {
    //players
    players map[int]*Player

    //timestamp on the server renewed on each update
    timestamp int64

    //bool denoting whether or not the match ended
    matchEnded bool
}

func (l *GameLogic) initializeEntities(clients []*Client) {
    for _, client := range clients {
        l.players[client.id] = newPlayer(client.id, "Default")
    }

    //create fake players
    for i := 0; i < FAKE_PLAYRERS; i++ {
        randomId := rand.Intn(100)
        l.players[randomId] = newPlayer(randomId, "Default")
        l.players[randomId].isSimulated = true
    }


    log.Printf("Initialized %d starting entities.", len(clients))
}
func newGameLogic() *GameLogic {
    l := &GameLogic{
        players: make(map[int]*Player),
    }
    log.Println("Game logic created")
    return l
}

//loop over all messages and apply them to the world
func (l *GameLogic) processInput(messages *list.List, ts int64) {
    log.Println("Processing input...")
    for m := messages.Front(); m != nil; m = m.Next() {
        //cast from list element to payload so we can read it
        payload := m.Value.(*Payload)
        for _, p := range payload.Players {
            l.players[p.Id].apply(&p, ts)    //apply player update to server side entity
        }
    }

    //simulates fake movements for those server simulated entites if its on
    for _, p := range l.players {
        if(p.isSimulated) {
            p.simulateFakeMovement(ts)
        }
    }
}

func (l *GameLogic) packageWorldState() Payload {
    p := newPayload()
    numDead := 0        //int describing number of players dead
    for _, v := range l.players {
        //When packaging the data from a player, it will return a boolean as
        //well indicating whether or not the data changed, if it hasnt then we wont
        //send it to the clients to be updated.
        data, changed := v.packageData();
        if changed {
            p.Players = append(p.Players, data)
        }

        //Increment number of players dead if this player is dead
        if(v.isDead) {
            numDead = numDead + 1
        }
    }

    //If theres players-1 dead players, then there is one plyaer alive.
    //He is the winner figure out who it is and set the winner on payload
    if(len(l.players)-1 == numDead) {
        log.Println("Match should be over")
        l.matchEnded = true 
        for _, player := range l.players {
            if(!player.isDead) {
                p.Winner = player.id
                break
            } else {
                log.Printf("id:%d Player is dead, total players: %d, dead: %d", player.id, len(l.players), numDead)
            }
        }
        log.Printf("Winner has id: %d", p.Winner)
    }


    return p
}

func (l *GameLogic) packageStartingState() Payload {
    p := newPayload()

    for _, v := range l.players {
        data, _ := v.packageData();
        p.Players = append(p.Players, data)
    }
    log.Println("Starting state:", len(p.Players))
    return p
}
