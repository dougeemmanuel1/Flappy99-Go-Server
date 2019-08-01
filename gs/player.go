package main

import (
    // "log"
    "math/rand"
)

type Player struct {
    //Id of the player
    id int

    //Skin for the player
    skin string

    isDead bool

    //structure containing sim information like x,y,state, direction
    data PlayerData

    //keep track of the previous state update sent, this allows us to only send the changes
    //between the last update
    previous PlayerData

    //Update the general position of the player occasionally
    lastCourtesyUpdate int64

    shouldSendCourtesyUpdate bool

    //TESTING ONLY
    //bool determining whether or not this player simulates fake movements
    isSimulated bool

    //last time we jumped
    lastJump int64
}


func newPlayer(pid int, s string) *Player {
    p := &Player{
        id:     pid,
        skin:   s,
    }
    return p
}

func (p *Player) apply(d *PlayerData, ts int64) {
    if(d.D) { //IF THE PLAYER IS DEAD UPDARE HIM
        p.kill()
        p.data = *d
    } else if(p.data.D) {
        p.kill()
         //If im dead dont update anything
    } else if(p.data.J && d.J) {
        //if we already jumped and the new update has a jump too, overwrite it!
        //apply new update
        p.data = *d
    } else if(p.data.J && !d.J) {
        //if we already got a jump update and the new update doesnt have one,
        //then dont update it. we want to make sure to communicate all jumps

    } else if(!p.data.J && d.J) {
        //if he didnt jump yet and theyre trying to, let them.
        p.data = *d
    } else {
        p.data = *d
    }

    // if(p.lastCourtesyUpdate + 5000 < ts) {
        // log.Println("Sending courtesy update...")
    //     p.shouldSendCourtesyUpdate = true
    //     p.lastCourtesyUpdate = ts
    // } else {
    //     p.shouldSendCourtesyUpdate = false
    // }
}

func (p *Player) kill() {
    p.isDead = true
    // log.Println("A player died.")
}

//Jiggles some of the X and y values around each update
func (p *Player) simulateFakeMovement(ts int64) {
    fake := PlayerData{}
    fake.X = p.data.X + rand.Intn(5) + -(rand.Intn(5))
    fake.Y = p.data.Y + rand.Intn(5) + -(rand.Intn(5))

    if(p.lastJump + 1000 < ts) { //jump every 50 milliseconds
        fake.J = true
        p.lastJump = ts
    }
    p.apply(&fake, ts)
}

func (p *Player) packageData() (PlayerData, bool) {
    var shouldSend bool = false
    data, _ := calculateDelta(p.previous, p.data)

    //If we have a state change that we have to communicate the player,
    // and the position that the jump starts at is always sent
    //with a state change.
    if(p.data.J || p.data.D || p.shouldSendCourtesyUpdate) {
        //Send this data
        shouldSend = true

        //override delta which only sends players coordinates that change
        data.X = p.data.X
        data.Y = p.data.Y

        //save last update sent
        p.previous = data

        //if were sending this as a courtesy update, and we just sent an Update
        //for this player because he was jumping or the player is dead, dont send it
        if(p.shouldSendCourtesyUpdate && p.previous.J || p.isDead) {
            shouldSend = false
        }

        //reset jump back to false
        p.data.J = false
    }

    //append id to data
    data.Id = p.id

    return data, shouldSend
}

func calculateDelta(previous PlayerData, current PlayerData) (PlayerData, bool) {
    delta := PlayerData{}
    var changed bool = false
    if(previous.X != current.X) { //generally dont include the X unless dash
        changed = true
        delta.X = current.X
    }
    if(previous.Y != current.Y) {
        changed = true
        delta.Y = current.Y
    }
    if(current.J) {
        delta.J = true
        changed = true
    }

    if(previous.D != current.D) {
        delta.D = current.D
    }


    return delta, changed
}
