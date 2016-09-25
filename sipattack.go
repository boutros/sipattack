package main

import (
	"bufio"
	"flag"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/knakk/kbp/sip2"
)

type client struct {
	conn net.Conn

	busyFactor float64 // how often it sends a request to SIP server
	failFactor float64 // how often it sends invalid SIP requests, or disconnectes
}

func newClient(b, f float64) *client {
	return &client{
		busyFactor: b,
		failFactor: f,
	}
}

func (c *client) Run() {
	conn, err := net.Dial("tcp", sipHost)
	if err != nil {
		log.Println("error connecting to SIP server: " + err.Error())
		return
	}

	r := bufio.NewReader(conn)
	for {
		if err := randomRequest().Encode(conn); err != nil {
			log.Println("error writing SIP request: " + err.Error())
			return
		}

		b, err := r.ReadBytes('\r')
		if err != nil {
			log.Println("error reading SIP response: " + err.Error())
			return
		}
		_, err = sip2.Decode(b)
		if err != nil {
			log.Println("error decoding SIP response: " + err.Error())
		}

		time.Sleep(time.Duration(math.Abs(rand.NormFloat64()*100+(10*c.busyFactor))) * time.Second)
	}
}

var (
	barcodes = make([]string, 0)
	patrons  = make([]string, 0)
)

func randomBarcode() string {
	return barcodes[rand.Intn(len(barcodes)-1)]
}

func randomPatron() string {
	return patrons[rand.Intn(len(patrons)-1)]
}

var mf = sip2.NewMessageFactory(
	sip2.Field{Type: sip2.FieldRenewalPolicy, Value: "Y"},
	sip2.Field{Type: sip2.FieldNoBlock, Value: "N"},
	sip2.Field{Type: sip2.FieldInstitutionID, Value: ""},
	sip2.Field{Type: sip2.FieldTerminalPassword, Value: ""},
	sip2.Field{Type: sip2.FieldSecurityMarker, Value: "01"},
	sip2.Field{Type: sip2.FieldFeeType, Value: "01"},
	sip2.Field{Type: sip2.FieldMagneticMedia, Value: "N"},
	sip2.Field{Type: sip2.FieldDesentisize, Value: "N"},
	sip2.Field{Type: sip2.FieldCurrentLocation, Value: "here"},
)

func randomRequest() sip2.Message {
	n := rand.Intn(100)
	switch {
	case n < 40:
		return mf.NewMessage(sip2.MsgReqCheckin).AddField(
			sip2.Field{Type: sip2.FieldItemIdentifier, Value: randomBarcode()})
	case n < 80:
		return mf.NewMessage(sip2.MsgReqCheckout).AddField(
			sip2.Field{Type: sip2.FieldItemIdentifier, Value: randomBarcode()},
			sip2.Field{Type: sip2.FieldPatronIdentifier, Value: randomPatron()})
	default:
		return mf.NewMessage(sip2.MsgReqItemInformation).AddField(
			sip2.Field{Type: sip2.FieldItemIdentifier, Value: randomBarcode()})
	}
}

var sipHost string

func main() {
	var (
		sipServer   = flag.String("s", "localhost:3333", "SIP server address")
		numClients  = flag.Int("n", 100, "number of SIP clients to create")
		busyFactor  = flag.Float64("b", 1, "busyness factor (0-1)")
		failFactor  = flag.Float64("f", 0.01, "failure factor (0-1)")
		barcodeFile = flag.String("barcodes", "barcodes.txt", "file with valid barcodes (one per line)")
		patronFile  = flag.String("patrons", "patrons.txt", "file with valid patrons IDs (one per line)")
	)

	flag.Parse()
	sipHost = *sipServer

	if *busyFactor < 0 || *busyFactor > 1 {
		flag.Usage()
	}
	if *failFactor < 0 || *failFactor > 1 {
		flag.Usage()
	}

	if *barcodeFile != "" {
		f, err := os.Open(*barcodeFile)
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			barcodes = append(barcodes, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		f.Close()
	}
	if *patronFile != "" {
		f, err := os.Open(*patronFile)
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			patrons = append(patrons, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		f.Close()
	}

	for i := 0; i < *numClients; i++ {
		c := newClient(*busyFactor, *failFactor)
		go c.Run()
	}
	time.Sleep(time.Hour)
}
