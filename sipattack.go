package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/knakk/kbp/sip2"
)

type client struct {
	busyFactor float64 // how often it sends a request to SIP server
}

func newClient(b float64) *client {
	return &client{
		busyFactor: b,
	}
}

func (c *client) Run() {
	conn, err := net.Dial("tcp", sipHost)
	if err != nil {
		log.Println("error connecting to SIP server: " + err.Error())
		return
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		time.Sleep(time.Duration(rand.Float64()*10000/c.busyFactor) * time.Millisecond)
		if err := randomRequest().Encode(conn); err != nil {
			if err != io.EOF {
				log.Println("error writing SIP request: " + err.Error())
			}
			return
		}

		b, err := r.ReadBytes('\r')
		if err != nil {
			if err != io.EOF {
				log.Println("error reading SIP response: " + err.Error())
			}
			return
		}
		_, err = sip2.Decode(b)
		if err != nil {
			log.Println("error decoding SIP response: " + err.Error())
		}
	}
}

var (
	barcodes = make([]string, 0)
	patrons  = make([]string, 0)
	branches = make([]string, 0)
)

func randomBarcode() string { return barcodes[rand.Intn(len(barcodes)-1)] }
func randomPatron() string  { return patrons[rand.Intn(len(patrons)-1)] }
func randomBranch() string  { return branches[rand.Intn(len(branches)-1)] }

var mf = sip2.NewMessageFactory(
	sip2.Field{Type: sip2.FieldRenewalPolicy, Value: "Y"},
	sip2.Field{Type: sip2.FieldNoBlock, Value: "N"},
	sip2.Field{Type: sip2.FieldInstitutionID, Value: ""},
	sip2.Field{Type: sip2.FieldTerminalPassword, Value: ""},
	sip2.Field{Type: sip2.FieldSecurityMarker, Value: "01"},
	sip2.Field{Type: sip2.FieldFeeType, Value: "01"},
	sip2.Field{Type: sip2.FieldMagneticMedia, Value: "N"},
	sip2.Field{Type: sip2.FieldDesentisize, Value: "N"},
)

func randomRequest() sip2.Message {
	n := rand.Intn(100)
	switch {
	case n < 40:
		return mf.NewMessage(sip2.MsgReqCheckin).AddField(
			sip2.Field{Type: sip2.FieldItemIdentifier, Value: randomBarcode()},
			sip2.Field{Type: sip2.FieldCurrentLocation, Value: randomBranch()})
	case n < 80:
		return mf.NewMessage(sip2.MsgReqCheckout).AddField(
			sip2.Field{Type: sip2.FieldItemIdentifier, Value: randomBarcode()},
			sip2.Field{Type: sip2.FieldPatronIdentifier, Value: randomPatron()},
			sip2.Field{Type: sip2.FieldCurrentLocation, Value: randomBranch()})
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
		busyFactor  = flag.Float64("b", 0.1, "busyness factor (0-1)")
		barcodeFile = flag.String("barcodes", "barcodes.txt", "file with valid barcodes (one per line)")
		patronFile  = flag.String("patrons", "patrons.txt", "file with valid patrons IDs (one per line)")
		branchFile  = flag.String("branches", "branches.txt", "file with locations (one per line)")
	)
	rand.Seed(time.Now().UnixNano())

	flag.Parse()
	sipHost = *sipServer

	if *busyFactor < 0 || *busyFactor > 1 {
		flag.Usage()
	}

	readSamples(*barcodeFile, &barcodes)
	readSamples(*patronFile, &patrons)
	readSamples(*branchFile, &branches)

	for i := 0; i < *numClients; i++ {
		go newClient(*busyFactor).Run()
	}
	time.Sleep(time.Hour)
}

func readSamples(file string, dest *[]string) {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		*dest = append(*dest, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
