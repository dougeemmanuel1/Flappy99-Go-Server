package main
import (
  "fmt"
  "log"
  "net/http"
  "flag"
  "sync"
  // "string"
  // "strconv"
  // "time"
  // "github.com/gorilla/websocket" //go ws lib eases ws use
  "github.com/gorilla/mux"  //eases route handling
)
//Port to listen on
const PORT = "8080"
var mtx sync.Mutex
//matchmaker which creates rooms for large groups of players
var matchMaker *Matchmaker = newMatchmaker()

//List of parties active on match making server
var parties = make([]*Party, 0, 100)

func main() {
    var dir string
    flag.StringVar(&dir, "dir", ".", "the directory to serve files from. Defaults to the current dir")
    flag.Parse()

    fmt.Println("Server starting, listening on PORT:", PORT)

    go matchMaker.run() //start match maker

    r := mux.NewRouter()
    //routes consist of a path and a handler function
    r.HandleFunc("/ws", wsEndpoint)

    // ticker := time.NewTicker(1 * time.Second)
    // go func() {
    //     for range ticker.C {
    //         log.Println("Main ticker.. pct:", len(parties))
    //     }
    // }()

    //Bind listen to port and pass mux router in
    log.Fatal(http.ListenAndServe(":" + PORT, r))
}

//serve websocket to client and add them to a party
func wsEndpoint(w http.ResponseWriter, r *http.Request) {
    mtx.Lock()
    party := newParty()    //create party of 1
    parties = append(parties, party)  //add to global list of parties
    mtx.Unlock()

    go party.run()          //start party go routine
    serveWs(party, w, r)       //create ws and add to party of 1

}
