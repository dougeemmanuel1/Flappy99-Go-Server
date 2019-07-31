package main

import (
    // "fmt"
    // "bytes"
    "log"
    "net/http"
    "time"
    "github.com/gorilla/websocket"
)

const (
    //Time allowed to write a message to the peer
    writeWait = 500 * time.Millisecond

    //Time allowed to read the next pong message from the peer
    pongWait = 15 * time.Second

    //Send pings to peer with this period. Must be less than pongWait
    pingPeriod = (pongWait * 9) / 10

    //Maximum message size allowed from peer
    maxMesasgeSize = 512
)

var (
    newline = []byte{'\n'}
    space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
    //allow same origin so we can host frontend and backend on same machien for test
    CheckOrigin:    func(r *http.Request) bool { return true },
}

//Client is a middleman between websocket connection and party
type Client struct {
    //Party player is a part of
    party *Party

    //The websocket connection
    conn *websocket.Conn

    //Buffered channel of outbound messages
    send chan Payload

    //player id
    id int

    //player name
    name string
}

//Writes message from the party to the client side socket
//A go routine running write is started for each connection. The
//application ensures that there is at most one writer to a connection
//by executign all writes from this goroutine.
func (c *Client) write() {
    ticker := time.NewTicker(pingPeriod) //ping this often

    //defer means this function will be called upon completion of
    //surrounding fucntion regardless of error/successful termination
    defer func() {
        ticker.Stop()   //stop the heartbeat ticker
        c.conn.Close()  //close the web socket
    }()

    //handle messages from channels
    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                //party closed the channel
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return  //end go routine
            }

            log.Println("Sending message to client:", message)
            c.conn.WriteJSON(message)
        }
    }
}

//Read reads messages from the client side websocket
//The applications run read in a per-connection goroutine. The application
//ensures that there is at most one reader on a connection by executing all
//reads form this goroutine.
func (c *Client) read() {
    log.Println("Starting cli read")
    defer func() {
        c.party.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(maxMesasgeSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })
    for {
        // _, message, err := c.conn.ReadMessage()
        message := Payload{}
        err := c.conn.ReadJSON(&message)
        if err != nil {
            log.Printf("Error reading JSON:", err)
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway,
                                                     websocket.CloseAbnormalClosure,
                                                     websocket.CloseNormalClosure) {
                log.Printf("Socket closed error: %v", err)
                // c.conn.Close()
            }
            break //quit read
        }

        log.Println("Message received from client:", message)
        // message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
        //send messsage we received from client to the party
        c.party.broadcast <- message
    }

}

//Serves a websocket to the connecting client and attaches it to a party
func serveWs(party *Party, w http.ResponseWriter, r *http.Request) {
    //upgrade http requesto to web socket in accoradance with RFC for websockets
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return //exit early
    }

    //create client ptr
    client := &Client{party: party, conn: conn, send: make(chan Payload, 256)}

    //send client to party to be registered
    client.party.register <- client

    //begin go routines to listen and write
    go client.write()
    go client.read()
}
