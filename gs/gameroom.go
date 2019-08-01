package main

import (
    "log"
    "time"
    // "encoding/json"
    "container/list"
)
const (
    //Ticks per second
    TICK_RATE = 10
)
type GameRoom struct {
    //game logic
    logic *GameLogic

    //connected clients
    clients []*Client

    //Player ids which start from room and are assigned upwards
    ids int

    //channel for inbound messages from clients which are funneled into a queue
    broadcast chan Payload

    //Linked list containing payload ptrs basically messages received from clients
    messages *list.List

    //channel for messages were sending out to clients
    outbound chan Payload

    //register requests from the clients
    register chan *Client

    //unregister requests from clients
    unregister chan *Client

    close chan bool

    //game room id - synonymou with index in global array of rooms
    id int
}

func newGameRoom(id int) *GameRoom {
    log.Println("Game room created.", TICK_RATE)
    return &GameRoom{
        logic:      newGameLogic(),
        clients:    []*Client{},
        ids:        1,
        broadcast:  make(chan Payload),
        messages:   list.New(),
        outbound:   make(chan Payload, 10), //remove?
        register:   make(chan *Client, 20),
        unregister: make(chan *Client, 20),
        close:      make(chan bool, 1),
        id:         id,
    }
}

//This entry phase is used as a started point for all clients to join the room,
//once the expected number of clients have joined OR the maximum amount of waiting time
//has elapsed then we start the match regardless
func (g *GameRoom) run() {
    connectionWaitTimer := time.NewTimer(5 * time.Second)
    defer func() {
        connectionWaitTimer.Stop() //stop timer
        g.closeRoom()
    }()
    outer:
        for {
            select {
            case client := <-g.register: //clients try to join
                g.clients = append(g.clients, client)
                // log.Println("Client registered, total clients:", len(g.clients))

                //get an id for this newly connected client
                client.id = g.createClientId()

                //send the newly connected player his assigned clientId
                client.send <- newIdPayload(client.id)
                // if(len(g.clients) == g.expectedClients) {
                //     log.Println("Client registered, total clients:", len(g.clients))
                //     log.Println("All clients connected succesfully.")
                //     break outer  // exit for and reuse goroutine for match
                // }
            case client := <-g.unregister: //clients disconnecting
                //TODO ADD MORE ROBUST RESPONSE TO DC
                g.disconnect(client)
            case <-connectionWaitTimer.C: //max client wait time reached
                break outer
            }
        }
        connectionWaitTimer.Stop()
        g.startMatch()
}

func (g *GameRoom) startMatch() {
    ticker := time.NewTicker(time.Second/TICK_RATE)
    defer func() {  //if for any reset this returns close all connections
        ticker.Stop()
    }()
    // log.Println("Starting match...")

    //have the game logic receive the starting list of entities with ids
    g.logic.initializeEntities(g.clients)

    startingState := g.logic.packageStartingState()

    // log.Println("Starting state had %d players.", len(startingState.Players))

    //send initial state of al clients
    g.dispatchToAllClients(&startingState)

    for {
        select {
        case <-ticker.C:
            // log.Printf("GR:%d Update Tick.", g.id)

            //current timestamp
            ts := time.Now().UnixNano() / 1000000

            //process all messages received from clients
            g.logic.processInput(g.messages, ts)

            //empty messages after theyve been processed from linked list
            g.messages.Init()

            //get the world state
            msg := g.logic.packageWorldState()

            // g.outbound <- msg

            //send new worldstate to all clients
            g.dispatchToAllClients(&msg)

            if(len(g.clients) == 0 || g.logic.matchEnded) { g.close <-true }
        case payload := <-g.broadcast: //funnell any messages from clients into an array in order
            g.messages.PushBack(&payload)
        case client := <-g.unregister: //clients disconnecting
            //TODO ADD MORE ROBUST RESPONSE TO DC
            // log.Println("Received unregister")
            g.disconnect(client)
        case <-g.close:
            g.closeRoom()
            return
        }
    }
}

func (g *GameRoom) createClientId() int {
    g.ids = g.ids + 1
    return g.ids
}
//Close the game room, this function is safe to be called multiple times
//Close room is effectively called twice but this is necessary, once in the
//defer of the run function should an error be thrown anywhere once the match starts, and 2
//should all the clients disconnect.
func (g *GameRoom) closeRoom() {
    log.Println("Closing GR:", g.id)
    for _, c := range g.clients { //disconnect all clients
        g.disconnect(c)
    }
    gameRooms = removeGameRoom(gameRooms, g)
}

func (g *GameRoom) dispatchToAllClients(msg *Payload) {
    // log.Println("Dispatch to all:", *msg)
    for _, client := range g.clients {
        client.send <- *msg
    }
}

func (g *GameRoom) disconnect(c *Client) {
    // log.Println("Client disconnected from room:", g.id)
    g.clients = removeClient(g.clients, c)
    close(c.send)       //close websocket
}

//Utility function to retrieve party index for a party in global parties array
func findGameRoom(rooms []*GameRoom, roomToFind *GameRoom) int {
    idx := -1            //assume we didnt find the party
    for i, room := range rooms {
        if(room == roomToFind) {
            return i
        }
    }
    return idx
}

//Utility function to remove a room from an array of parties,
//then return that new array
func removeGameRoom(rooms []*GameRoom, roomToRemove *GameRoom) []*GameRoom {
    //find index of room to remove first
    index := findGameRoom(rooms, roomToRemove)

    if(index == -1) { //coudlnt find it
        log.Println("Remove cancelled, couldn't find room.")
        return rooms //return original array
    }

    //replace partyToRemove with first element
    rooms[index] = rooms[0]

    //reslice from 2nd element to end
    return rooms[1:]
}
