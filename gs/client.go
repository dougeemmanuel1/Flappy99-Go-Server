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
    pongWait = 60 * time.Second

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

//Client is a middleman between websocket connection and room
type Client struct {
    //Gameroom player is a part of
    room *GameRoom

    //player id
    id int


    //The websocket connection
    conn *websocket.Conn

    //Buffered channel of outbound messages
    send chan Payload

    //player name
    name string
}

//Writes message from the room to the client side socket
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
                //room closed the channel
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return  //end go routine
            }

            //log.Println("Sending message to client:", message)
            c.conn.WriteJSON(message)
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
        }
    }
}

//Read reads messages from the client side websocket
//The applications run read in a per-connection goroutine. The application
//ensures that there is at most one reader on a connection by executing all
//reads form this goroutine.
func (c *Client) read() {
    // log.Println("Starting cli read")
    defer func() {
        c.room.unregister <- c
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
            log.Printf("Error Reading JSON Client:", err)
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway,
                                                     websocket.CloseAbnormalClosure) {
                log.Printf("error: %v", err)
            }
            break //quit read
        }
        // log.Println("Message received in client:", message)
        c.room.broadcast <- message
    }

}

//Utility function to retrieve room index for a room in global parties array
func findClient(clients []*Client, clientToFind *Client) int {
    idx := -1            //assume we didnt find the room
    for i, client := range clients {
        if(client == clientToFind) {
            return i
        }
    }
    return idx
}

//Utility function to remove a client from an array of parties,
//then return that new array
func removeClient(clients []*Client, clientToRemove *Client) []*Client {
    //find index of client to remove first
    index := findClient(clients, clientToRemove)

    if(index == -1) { //coudlnt find it
        log.Println("Remove cancelled, couldn't find client.")
        return clients //return original array
    }

    //replace roomToRemove with first element
    clients[index] = clients[0]

    //reslice from 2nd element to end
    return clients[1:]
}
