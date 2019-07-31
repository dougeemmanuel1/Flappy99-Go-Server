package main

import (
    "log"
    "time"
    "net/http"
    "io/ioutil"
)

const (
    playersNeededToStart int = 2
)

//Will attempt to create matches of 99 and ship them off to the game server

type Matchmaker struct {
    //Parties currently being held to make a match
    partiesMatched  []*Party

    //Parties in queue which are pushed into partiesMatch, when full starts a match
    partiesInQueue []*Party

    //Buffered channel to receive parties that want to queue
    receive chan *Party
}

func newMatchmaker() *Matchmaker {
    m := &Matchmaker{
        partiesMatched: []*Party{},
        partiesInQueue: []*Party{},
        receive:        make(chan *Party, 20),
    }
    return m
}

//Tick at some interval and attempt to create matches from parties
//that are in queue
func (m *Matchmaker) run() {
    //ADD DEFER
    log.Println("Matchmaker go routine started...")
    ticker := time.NewTicker(1 * time.Second)
    maxPartyWait := time.NewTimer(10 * time.Second)

    go func() {
        for {
            select {
            case <-ticker.C:   //receive tick to try match more people
                m.attemptToCreateAMatch()
            case p := <-m.receive:     //add this party to in queue
                log.Println("Added party %d to queue", p.id)
                m.partiesInQueue = append(m.partiesInQueue, p)
            case <- maxPartyWait.C:
                m.sendToGameServer()
            }
        }
    }()
}

//This function will iterate over all the current parties in queue,
//and try to create a match from those parties.
//Once it creates a suitable match, it will ship them over to the game server.
func (m *Matchmaker) attemptToCreateAMatch() {
    // log.Println("\tAttempting to create match:")
    // log.Println("\tParties in queue:", len(m.partiesInQueue))
    // log.Println("\tParties matched:", len(m.partiesMatched))
    log.Println("\tParties pruned:", m.prune())
    //iterate over current parties to see if we can match them
    for _, p := range m.partiesInQueue {
        if(p.State == Ready) {   //add to matched parties
            //change state
            p.State = InQueue

            //remove from parties in queue
            m.partiesInQueue = removeParty(m.partiesInQueue, p)

            //add to matched parties
            m.partiesMatched = append(m.partiesMatched, p)
        }
        //after we add each party check if match is full
        if(m.isMatchFull()) { //start match
            //TODO HAVE BACK UP IN PLAC IF CANT REACH GAMESERVER
            m.sendToGameServer()
            break
        }
    }
}

//This function scans the matchmaker of arrays of parties and
//removes any parties that have no connected clients,
//it then returns the number of parties pruned if any

func (m *Matchmaker) prune() int {
    var pruneTotal int = 0

    //if the parties have 0 clients connected to them,
    //then all we had to do is remove them since they have
    //already disconnected
    for _, p := range m.partiesMatched {
        if(p.count() == 0) {
            m.partiesMatched = removeParty(m.partiesMatched, p)
            pruneTotal++
        }
    }

    for _, p := range m.partiesInQueue {
        if(p.count() == 0) {
            m.partiesInQueue = removeParty(m.partiesInQueue, p)
            pruneTotal++
        }
    }
    return pruneTotal
}

func (m *Matchmaker) isMatchFull() bool {
    var totalPlayers int = 0
    for _, p := range m.partiesMatched {
        totalPlayers += p.count()
    }
    return (playersNeededToStart == totalPlayers)
}

//Request a room from game room server for all of the
//players we created
func (m *Matchmaker) sendToGameServer() {
    log.Println("Match full sending to game server...")
    resp, err := http.Get("http://" + GS_SERVER_URL + "/requestRoom")

    if(err != nil) {
        //TODO MORE ROBUST HANLDING FOR FAILURE
        log.Println("Error making game room server request.")
        return
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if(err != nil) {
        log.Println("Error reading body:", err)
    }
    //TODO ADD PROPER EROR HANDLIG, CHECK IF THE STRING IS EQUAL TO -1,
    //IF SO DO PROPER FFAILURE TO ACQUIRE ROOM CHECKINGS
    //MAYBE TRY AGAIN
    msg := string(body)
    log.Println("Received gameroom ID:", msg)

    payload := newPayload()
    payload.GameRoomId = msg

    //tell all matched parties to connect to specific game room id
    for _, p := range m.partiesMatched {
        p.broadcast <- payload
    }
}
