package agents

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/klauspost/compress/zstd"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/types"
)

/*
	CSVAgent reads an allocation trace from a file and replays it
	Rows in the CSV file should take the form of comma-separate integers: Time (seconds),Type(1=allocate,0=release),ID(unique across allocated IPs),TenantID
*/
type CSVAgent struct {
	instanceSlotIds map[uint64]types.IPAddress
	input           io.Reader
	scanner         *bufio.Scanner

	InputFilename string
	Zstd          bool

	BaseAgent
}

func (a *CSVAgent) Init(s types.Simulator, minID types.TenantId, maxID types.TenantId) {
	var err error
	a.input, err = os.Open(a.InputFilename)
	if err != nil {
		log.Fatal(err)
	}
	if a.Zstd {
		a.input, err = zstd.NewReader(a.input)
		if err != nil {
			log.Fatal(err)
		}
	}
	a.scanner = bufio.NewScanner(a.input)
	a.scanner.Scan() // Process assumes a scan has already happened
	a.BaseAgent.Init(s, minID, maxID)
	a.instanceSlotIds = make(map[uint64]types.IPAddress)
}

func (a *CSVAgent) Process(s types.Simulator) {
	t := s.GetTime()
	for {
		line := bytes.Split(a.scanner.Bytes(), []byte{','})
		if len(line) != 4 {
			log.Fatal("CSV trace lines must have 4 entries")
		}
		time, err := strconv.ParseUint(string(line[0]), 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		if types.Duration(time) < t {
			log.Fatal("CSV trace is not ordered")
		} else if types.Duration(time) > t {
			break
		}
		Type, err := strconv.ParseUint(string(line[1]), 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		instanceId, err := strconv.ParseUint(string(line[2]), 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		user, err := strconv.ParseUint(string(line[3]), 10, 64)
		if err != nil {
			log.Fatal(err)
		}

		if Type == 1 {
			ip := s.GetIP(a.minID + types.TenantId(user))
			a.instanceSlotIds[instanceId] = ip
		} else if Type == 0 {
			ip, ok := a.instanceSlotIds[instanceId]
			if !ok {
				log.Fatal("CSV contains released instance not allocated")
			}
			s.ReleaseIP(ip, a.minID+types.TenantId(user), true)
			delete(a.instanceSlotIds, instanceId)
		} else {
			log.Fatal("CSV trace type must be 1 or 0")
		}

		if !a.scanner.Scan() {
			s.Done()
			break
		}
	}
}
