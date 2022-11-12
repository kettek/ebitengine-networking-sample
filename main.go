package main

import (
	"bytes"
	"encoding/gob"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"image"
	_ "image/png"
	"log"
	"net"
	"os"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	playerImage *ebiten.Image
)

// MoveMessage is used to represent either a movement direction request from the client or an exact player position update from the server.
type MoveMessage struct {
	Id int
	X  int
	Y  int
}

// Game contains our networking structures as well as a very simple container for player state.
type Game struct {
	conn     net.Conn
	listener net.Listener // Populated if we are a server.
	encoder  *gob.Encoder
	decoder  *gob.Decoder
	netChan  chan interface{}
	address  string
	isServer bool
	players  [2]struct {
		x, y int
	}
}

// Update updates our game world, processing network messages, updating player positions, and converting arrow keys to movement updates.
func (g *Game) Update() error {
	// Check for any network data. We read this in a loop in case more than one message is sent.
	for done := false; !done; {
		select {
		case msg := <-g.netChan:
			switch msg := msg.(type) {
			case MoveMessage:
				if g.isServer {
					// If we're the server, incorporate the movement and tell the client to move their player.
					g.players[1].x += msg.X
					g.players[1].y += msg.Y
					// Send the move as an exact coordinate.
					g.NetSend(MoveMessage{
						Id: 1, // Note: player 2 has an ID of 1.
						X:  g.players[1].x,
						Y:  g.players[1].y,
					})
				} else {
					// If we're the client, adjust the given player to the exact coordinate sent by the server.
					g.players[msg.Id].x = msg.X
					g.players[msg.Id].y = msg.Y
				}
			}

		default:
			done = true
		}
	}

	// Check if we want to do any movement.
	var move MoveMessage
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		move.X = -1
	} else if ebiten.IsKeyPressed(ebiten.KeyRight) {
		move.X = 1
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		move.Y = 1
	} else if ebiten.IsKeyPressed(ebiten.KeyUp) {
		move.Y = -1
	}
	// Send our message if we actually want to move.
	if move.X != 0 || move.Y != 0 {
		if g.isServer {
			// If we're the server, update our position and send the client our exact coordinate.
			g.players[0].x += move.X
			g.players[0].y += move.Y
			g.NetSend(MoveMessage{
				Id: 0,
				X:  g.players[0].x,
				Y:  g.players[0].y,
			})
		} else {
			// Send a directional movement request to the server.
			g.NetSend(MoveMessage{
				X: move.X,
				Y: move.Y,
			})
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw our players.
	for _, pl := range g.players {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(pl.x), float64(pl.y))
		screen.DrawImage(playerImage, op)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// NetLoop is a method that is run as a goroutine. It controls reading from the network connection.
func (g *Game) NetLoop() {
	// Read our received messages and store them in our network channel.
	for {
		var msg interface{}
		if err := g.decoder.Decode(&msg); err != nil {
			log.Fatal(err)
		}
		g.netChan <- msg
	}
}

// NetSend sends a given message to the connection.
func (g *Game) NetSend(msg interface{}) {
	g.encoder.Encode(&msg)
}

func main() {
	// Create our game instance.
	g := Game{}
	// Set our player starting positions.
	g.players[0].x = 100
	g.players[0].y = 100
	g.players[1].x = 200
	g.players[1].y = 200

	// Load our character image.
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}

	playerImage = ebiten.NewImageFromImage(img)

	// Initialize our network types to be sent. Any additional message types to be sent across the wire should be registered here.
	gob.Register(MoveMessage{})

	// Check if we're hosting or joining.
	if len(os.Args) < 3 {
		log.Printf("Usage: '%s host <address>' or '%s join <address>'\n", os.Args[0], os.Args[0])
		return
	}
	if os.Args[1] == "host" {
		g.address = os.Args[2]
		g.isServer = true
	} else if os.Args[1] == "join" {
		g.address = os.Args[2]
	}

	// Set up our network connection.
	if g.isServer {
		g.listener, err = net.Listen("tcp", g.address)
		if err != nil {
			log.Fatal(err)
		}
		// Wait for a single client to connect.
		g.conn, err = g.listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		g.encoder = gob.NewEncoder(g.conn)
		g.decoder = gob.NewDecoder(g.conn)
		g.netChan = make(chan interface{})
		go g.NetLoop()
		log.Println("client connected :)")
	} else {
		g.conn, err = net.Dial("tcp", g.address)
		if err != nil {
			log.Fatal(err)
		}
		g.encoder = gob.NewEncoder(g.conn)
		g.decoder = gob.NewDecoder(g.conn)
		g.netChan = make(chan interface{})
		go g.NetLoop()
		log.Println("connected to server :)")
	}

	// Create our ebitengine window and start the game.
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	if g.isServer {
		ebiten.SetWindowTitle("Multiplayer Server (Ebiten Demo)")
	} else {
		ebiten.SetWindowTitle("Multiplayer Client (Ebiten Demo)")
	}
	if err := ebiten.RunGame(&g); err != nil {
		log.Fatal(err)
	}
}
