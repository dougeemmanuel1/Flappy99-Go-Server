package main

/* Maintains a group of clients and provides these features:
    -Grouping of clients in a 'party'
    -Chatting between all players within party
    -Allows parties to "fuse" with one another for matchmaking
    -Hand off to game server
*/

import (
    "log"
    "math/rand"
)

type PartyState uint32
const (
    NotReady PartyState = iota //0 auto increment
    Ready
    InQueue
)

type Party struct {
    //Registered clients
    clients map[*Client]bool

    //Inbound messages from clients
    broadcast chan Payload

    //Register requests from the clients.
    register chan *Client

    //Unregister requests from clients.
    unregister chan *Client

    //Party's id
    id int

    State PartyState
}

func newParty() *Party {
    p := &Party {
        clients:    make(map[*Client]bool),
        broadcast:  make(chan Payload),
        register:   make(chan *Client, 20),
        unregister: make(chan *Client, 20),
        id:         rand.Intn(100),
        State:      NotReady,
    }
    log.Println("New party created with id:", p.id)
    return p
}

func (p *Party) disconnect(c *Client) {
    delete(p.clients, c) //remove from map
    close(c.send)       //close websocket
    if(p.count() == 0) { //party is empty
        //remove from all matchmaker arrays
        //match maker arrays should prune itself of
        //empty parties automatically

        //remove from global party array
        parties = removeParty(parties, p)
    }
}

func (p *Party) run() {
    defer func() {
        for c := range p.clients {
            p.disconnect(c)
        }
    }()
    for {
        select {
        case client := <-p.register:
            log.Println("Client registered")
            p.clients[client] = true
            //Send dummy message to connected clients to test JSON
            // client.send <- Payload{ Chat: []string{"HELLO", "TEST"}}
            matchMaker.receive <- p //request to find match
            parties = removeParty(parties, p)          //remove p from global parties list
            p.State = Ready
        //someone disconnected from the party 
        case client := <-p.unregister:
            log.Println("Client unregistered")
            if _, ok := p.clients[client]; ok {
                p.disconnect(client)
            }
        //broadcast the game room these clients should go connect to everyone in the party
        case message := <-p.broadcast:
            for client := range p.clients {
                client.send <- message
            }
        }
    }
}


//counts connected clients to a party
func (p *Party) count() int {
    return len(p.clients)
}

//Utility function to retrieve party index for a party in global parties array
func findParty(pArr []*Party, partyToFind *Party) int {
    idx := -1            //assume we didnt find the party
    for i, party := range pArr {
        if(party == partyToFind) {
            return i
        }
    }
    return idx
}

//Utility function to remove a party from an array of parties,
//then return that new array
func removeParty(pArr []*Party, partyToRemove *Party) []*Party {
    //find index of party to remove first
    index := findParty(pArr, partyToRemove)

    if(index == -1) { //coudlnt find it
        log.Println("Remove cancelled, couldn't find party.")
        return pArr //return original array
    }

    //replace partyToRemove with first element
    pArr[index] = pArr[0]

    //reslice from 2nd element to end
    return pArr[1:]
}
