package main
import (
  "fmt"
  "log"
  "net/http"
  "sync"
  // "time"
  "strconv"
  "github.com/gorilla/mux"  //eases route handling
)
//Port to listen on
const PORT = "8080"
const ROOM_CAP = 100
var gameRooms = make([]*GameRoom, 0, ROOM_CAP)
var mtx sync.Mutex

// const MM_SERVER_URL = "localhost:8080"
// const GS_SERVER_URL = "localhost:8081"
//

func main() {
    log.Println("Game Server starting, listening on PORT:", PORT)

    r := mux.NewRouter()

    //routes consist of a path and a handler function
    r.HandleFunc("/requestRoom", requestRoom).Methods("GET")

    //allow clients to connect to their game rooms
    r.HandleFunc("/ws", connectToGameRoom)

    //Bind listen to port and pass mux router in
    log.Fatal(http.ListenAndServe(":" + PORT, r))
}

//Since clients have to connect through a route to be upgraded to websocket,
//they will append their gameroom number that they received from the matchmaker,
//to their ws protocol request. ex: ws://localhost:8080/ws?id=100
//ROOM_CAP routes so we can serve them a websocket and they can connect to their respective rooms.
func connectToGameRoom(w http.ResponseWriter, r  *http.Request) {
    // log.Println("GET parameters for connection to game room were:", r.URL.Query())
    idParam := r.URL.Query().Get("id")    //get id
    if(idParam != "") { //not empty
        // log.Println("Client attempting to connect with id:", idParam)

        //convert id to int
        id, err := strconv.Atoi(idParam);
        if err != nil { // if converstion to int failed
            log.Println("Failed to convert id to int, denying connection to game room:", idParam)
            return
        }
        serveWs(gameRooms[id], w, r)
    }
}

func requestRoom(w http.ResponseWriter, r *http.Request) {
    // log.Println("Request for room received")
    _, id := createGameRoom()
    if(id == -1) {
        log.Println("Capacity for rooms reached.")
        //TODO ADD PROPER RESPONSE HANDLING TO MM SERVER WHEN CAP REACHED
        return
    }

    //write room id response to mathc making server
    fmt.Fprintf(w, strconv.Itoa(id))
}

func createGameRoom() (*GameRoom, int) {
    mtx.Lock()
    if(len(gameRooms) == ROOM_CAP) {
        log.Println("Unable to create gameroom, capped reached.")
        return nil, -1
    }
    //TODO CHANGE THIS TO ACTUALLY ACCEPT THE INCOMING AMOUNT OF PLAYRS
    g := newGameRoom(0)

    //append to global rooms
    gameRooms = append(gameRooms, g)
    roomId := findGameRoom(gameRooms, g)
    g.id = roomId
    log.Println(gameRooms[0].id,g.id)

    //start room go routine
    go g.run()
    mtx.Unlock()

    return g, g.id
}

//serve websocket to client and add them to a party
func wsEndpoint(w http.ResponseWriter, r *http.Request) {
    // serveWs(party, w, r)       //create ws and add to party of 1
}


//Serves a websocket to the connecting client and attaches it to a party
func serveWs(g *GameRoom, w http.ResponseWriter, r *http.Request) {
    //upgrade http requesto to web socket in accoradance with RFC for websockets
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return //exit early
    }

    //create client ptr
    client := &Client{room: g, conn: conn, send: make(chan Payload, 256)}

    //send client to party to be registered
    client.room.register <- client

    //begin go routines to listen and write
    go client.write()
    go client.read()
}
