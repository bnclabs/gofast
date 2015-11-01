package main

import "bytes"
import "encoding/json"
import "sort"
import "strconv"
import "fmt"
import "os/exec"
import "strings"
import "time"
import "log"
import "os"

func main() {
	fmt.Println("===========================================")
	since := time.Now()
	conns, rtns, cnts := int64(16), int64(200), int64(10000)
	rv := verifyPost(conns, rtns, cnts)
	fmt.Printf("no. of msg %v in %v\n", conns*rtns*cnts, time.Since(since))

	fmt.Println("===========================================")
	since = time.Now()
	conns, rtns, cnts = int64(16), int64(200), int64(10000)
	rv += verifyRequest(conns, rtns, cnts)
	fmt.Printf("no. of msg %v in %v\n", conns*rtns*cnts, time.Since(since))

	fmt.Println("===========================================")
	since = time.Now()
	conns, rtns, cnts = int64(16), int64(200), int64(10000)
	rv += verifyStream(conns, rtns, cnts)
	fmt.Println()
	fmt.Printf("no. of msg %v in %v\n", conns*rtns*cnts, time.Since(since))

	fmt.Println("===========================================")
	//rv += verifyRandom()
	fmt.Println()

	os.Exit(rv)
}

func verifyPost(conns, rtns, cnts int64) (rv int) {
	var cargs = []string{
		"-do", "post", "-addr", "localhost:9900",
		"-conns", strconv.Itoa(int(conns)),
		"-routines", strconv.Itoa(int(rtns)),
		"-count", strconv.Itoa(int(cnts)),
	}
	var sargs = []string{"-addr", "localhost:9900"}

	cstat, sstat := runCmds(cargs, sargs)

	// common
	if x, y := sstat["n_dropped"], cstat["n_dropped"]; x != 0 || y != 0 {
		fmt.Println("n_dropped failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_mdrops"], cstat["n_mdrops"]; x != 0 || y != 0 {
		fmt.Println("n_mdrops failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxbeats"], cstat["n_rxbeats"]; x != 0 || y != 0 {
		fmt.Println("n_rxbeats failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txbeats"], cstat["n_txbeats"]; x != 0 || y != 0 {
		fmt.Println("n_txbeats failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxfin"], cstat["n_rxfin"]; x != 0 || y != 0 {
		fmt.Println("n_rxfin failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txfin"], cstat["n_txfin"]; x != 0 || y != 0 {
		fmt.Println("n_txfin failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxstart"], cstat["n_rxstart"]; x != 0 || y != 0 {
		fmt.Println("n_rxstart failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxstream"], cstat["n_rxstream"]; x != 0 || y != 0 {
		fmt.Println("n_rxstream failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txstart"], cstat["n_txstart"]; x != 0 || y != 0 {
		fmt.Println("n_txstart failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txstream"], cstat["n_txstream"]; x != 0 || y != 0 {
		fmt.Println("n_txstream failed ...", x, y)
		rv = 1
	}

	// client
	if y := cstat["n_rxreq"]; y != conns {
		fmt.Println("cstat n_rxreq failed ...", y)
		rv = 1
	} else if z := cstat["n_rxresp"]; z != conns*2 {
		fmt.Println("cstat n_rxresp failed ...", z)
		rv = 1
	} else if x := cstat["n_rx"]; x != (y + z) {
		fmt.Println("cstat n_rx failed ...", x, y, z)
		rv = 1
	} else if x = cstat["n_rxpost"]; x != 0 {
		fmt.Println("cstat n_rxpost failed ...", x)
		rv = 1
	} else if b := cstat["n_txreq"]; b != conns*2 {
		fmt.Println("cstat n_txreq failed ...", b)
		rv = 1
	} else if c := cstat["n_txresp"]; c != conns {
		fmt.Println("cstat n_txresp failed ...", c)
		rv = 1
	} else if d := cstat["n_txpost"]; d != conns*rtns*cnts {
		fmt.Println("cstat n_txpost failed ...", d)
		rv = 1
	} else if a := cstat["n_tx"]; a != (b + c + d) {
		fmt.Println("cstat n_tx failed ...", a, b, c, d)
		rv = 1
	}

	// server
	if y := sstat["n_txreq"]; y != conns {
		fmt.Println("sstat n_txreq failed ...", y)
		rv = 1
	} else if z := sstat["n_txresp"]; z != conns*2 {
		fmt.Println("sstat n_txresp failed ...", z)
		rv = 1
	} else if x := sstat["n_tx"]; x != (y + z) {
		fmt.Println("sstat n_tx failed ...", x, y, z)
		rv = 1
	} else if x = sstat["n_txpost"]; x != 0 {
		fmt.Println("sstat n_txpost failed ...", x)
		rv = 1
	} else if b := sstat["n_rxreq"]; b != conns*2 {
		fmt.Println("sstat n_rxreq failed ...", b)
		rv = 1
	} else if c := sstat["n_rxresp"]; c != conns {
		fmt.Println("sstat n_rxresp failed ...", c)
		rv = 1
	} else if d := sstat["n_rxpost"]; d != conns*rtns*cnts {
		fmt.Println("sstat n_rxpost failed ...", d)
		rv = 1
	} else if a := sstat["n_rx"]; a != (b + c + d) {
		fmt.Println("sstat n_rx failed ...", a, b, c, d)
		rv = 1
	}
	printstats(cstat, sstat)
	return
}

func verifyRequest(conns, rtns, cnts int64) (rv int) {
	var cargs = []string{
		"-do", "request", "-addr", "localhost:9900",
		"-conns", strconv.Itoa(int(conns)),
		"-routines", strconv.Itoa(int(rtns)),
		"-count", strconv.Itoa(int(cnts)),
	}
	var sargs = []string{"-addr", "localhost:9900"}

	cstat, sstat := runCmds(cargs, sargs)

	// common
	if x, y := sstat["n_dropped"], cstat["n_dropped"]; x != 0 || y != 0 {
		fmt.Println("n_dropped failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_mdrops"], cstat["n_mdrops"]; x != 0 || y != 0 {
		fmt.Println("n_mdrops failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxbeats"], cstat["n_rxbeats"]; x != 0 || y != 0 {
		fmt.Println("n_rxbeats failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txbeats"], cstat["n_txbeats"]; x != 0 || y != 0 {
		fmt.Println("n_txbeats failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxfin"], cstat["n_rxfin"]; x != 0 || y != 0 {
		fmt.Println("n_rxfin failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txfin"], cstat["n_txfin"]; x != 0 || y != 0 {
		fmt.Println("n_txfin failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxstart"], cstat["n_rxstart"]; x != 0 || y != 0 {
		fmt.Println("n_rxstart failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxstream"], cstat["n_rxstream"]; x != 0 || y != 0 {
		fmt.Println("n_rxstream failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txstart"], cstat["n_txstart"]; x != 0 || y != 0 {
		fmt.Println("n_txstart failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txstream"], cstat["n_txstream"]; x != 0 || y != 0 {
		fmt.Println("n_txstream failed ...", x, y)
		rv = 1
	}

	// client
	if y := cstat["n_rxreq"]; y != conns {
		fmt.Println("cstat n_rxreq failed ...", y)
		rv = 1
	} else if z := cstat["n_rxresp"]; z != conns*rtns*cnts+conns {
		fmt.Println("cstat n_rxresp failed ...", z)
		rv = 1
	} else if x := cstat["n_rx"]; x != (y + z) {
		fmt.Println("cstat n_rx failed ...", x, y, z)
		rv = 1
	} else if x = cstat["n_rxpost"]; x != 0 {
		fmt.Println("cstat n_rxpost failed ...", x)
		rv = 1
	} else if b := cstat["n_txreq"]; b != conns*rtns*cnts+conns {
		fmt.Println("cstat n_txreq failed ...", b)
		rv = 1
	} else if c := cstat["n_txresp"]; c != conns {
		fmt.Println("cstat n_txresp failed ...", c)
		rv = 1
	} else if d := cstat["n_txpost"]; d != 0 {
		fmt.Println("cstat n_txpost failed ...", d)
		rv = 1
	} else if a := cstat["n_tx"]; a != (b + c + d) {
		fmt.Println("cstat n_tx failed ...", a, b, c, d)
		rv = 1
	}

	// server
	if y := sstat["n_txreq"]; y != conns {
		fmt.Println("sstat n_txreq failed ...", y)
		rv = 1
	} else if z := sstat["n_txresp"]; z != conns*rtns*cnts+conns {
		fmt.Println("sstat n_txresp failed ...", z)
		rv = 1
	} else if x := sstat["n_tx"]; x != (y + z) {
		fmt.Println("sstat n_tx failed ...", x, y, z)
		rv = 1
	} else if x = sstat["n_txpost"]; x != 0 {
		fmt.Println("sstat n_txpost failed ...", x)
		rv = 1
	} else if b := sstat["n_rxreq"]; b != conns*rtns*cnts+conns {
		fmt.Println("sstat n_rxreq failed ...", b)
		rv = 1
	} else if c := sstat["n_rxresp"]; c != conns {
		fmt.Println("sstat n_rxresp failed ...", c)
		rv = 1
	} else if d := sstat["n_rxpost"]; d != 0 {
		fmt.Println("sstat n_rxpost failed ...", d)
		rv = 1
	} else if a := sstat["n_rx"]; a != (b + c + d) {
		fmt.Println("sstat n_rx failed ...", a, b, c, d)
		rv = 1
	}
	printstats(cstat, sstat)
	return
}

func verifyStream(conns, rtns, cnts int64) (rv int) {
	var cargs = []string{
		"-do", "stream", "-addr", "localhost:9900",
		"-conns", strconv.Itoa(int(conns)),
		"-routines", strconv.Itoa(int(rtns)),
		"-count", strconv.Itoa(int(cnts)),
	}
	var sargs = []string{"-addr", "localhost:9900"}

	cstat, sstat := runCmds(cargs, sargs)

	// common
	if x, y := sstat["n_dropped"], cstat["n_dropped"]; x != 0 || y != 0 {
		fmt.Println("n_dropped failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_mdrops"], 0; x != 0 || y != 0 {
		fmt.Println("n_mdrops failed ...", x, y)
		rv = 1
	} else if x, y = cstat["n_rxpost"], sstat["n_rxpost"]; x != 0 || y != 0 {
		fmt.Println("n_rxpost failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxbeats"], cstat["n_rxbeats"]; x != 0 || y != 0 {
		fmt.Println("n_rxbeats failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txbeats"], cstat["n_txbeats"]; x != 0 || y != 0 {
		fmt.Println("n_txbeats failed ...", x, y)
		rv = 1
	} else if x, y = 0, cstat["n_rxfin"]; x != 0 || y != 0 {
		fmt.Println("n_rxfin failed ...", x, y)
		rv = 1
	} else if x, y = 0, cstat["n_rxstart"]; x != 0 || y != 0 {
		fmt.Println("n_rxstart failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txstart"], 0; x != 0 || y != 0 {
		fmt.Println("n_txstart failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txpost"], cstat["n_txpost"]; x != 0 || y != 0 {
		fmt.Println("n_txpost failed ...", x, y)
		rv = 1
	}

	// client
	if a := cstat["n_rxreq"]; a != conns {
		fmt.Println("cstat n_rxreq failed ...", a)
		rv = 1
	} else if b := cstat["n_rxresp"]; b != conns {
		fmt.Println("cstat n_rxresp failed ...", b)
		rv = 1
	} else if c := cstat["n_mdrops"]; c != conns*rtns {
		fmt.Println("cstat n_mdrops failed ...", c)
		rv = 1
	} else if d := cstat["n_rxstream"]; d != (conns*rtns*cnts)+(conns*rtns) {
		fmt.Println("cstat n_mdrops failed ...", d)
		rv = 1
	} else if x := cstat["n_rx"]; x != (a + b + c + d) {
		fmt.Println("cstat n_rx failed ...", x, a, b, c, d)
		rv = 1
	} else if a = cstat["n_txreq"]; a != conns {
		fmt.Println("cstat n_txreq failed ...", a)
		rv = 1
	} else if b = cstat["n_txresp"]; b != conns {
		fmt.Println("cstat n_txresp failed ...", b)
		rv = 1
	} else if c = cstat["n_txstart"]; c != conns*rtns {
		fmt.Println("cstat n_txstart failed ...", c)
		rv = 1
	} else if d = cstat["n_txstream"]; d != conns*rtns*cnts {
		fmt.Println("cstat n_txstream failed ...", d)
		rv = 1
	} else if e := cstat["n_txfin"]; e != conns*rtns {
		fmt.Println("cstat n_txfin failed ...", e)
		rv = 1
	} else if x = cstat["n_tx"]; x != (a + b + c + d + e) {
		fmt.Println("cstat n_tx failed ...", x, a, b, c, d)
		rv = 1
	}

	// server
	if a := sstat["n_txreq"]; a != conns {
		fmt.Println("sstat n_txreq failed ...", a)
		rv = 1
	} else if b := sstat["n_txresp"]; b != conns {
		fmt.Println("sstat n_txresp failed ...", b)
		rv = 1
	} else if d := sstat["n_txstream"]; d != (conns*rtns*cnts)+(conns*rtns) {
		fmt.Println("sstat n_txstream failed ...", d)
		rv = 1
	} else if c := sstat["n_txfin"]; c != conns*rtns {
		fmt.Println("sstat n_txfin failed ...", c)
		rv = 1
	} else if x := sstat["n_tx"]; x != (a + b + c + d) {
		fmt.Println("sstat n_tx failed ...", x, a, b, c, d)
		rv = 1
	} else if a := sstat["n_rxreq"]; a != conns {
		fmt.Println("sstat n_rxreq failed ...", a)
		rv = 1
	} else if b := sstat["n_rxresp"]; b != conns {
		fmt.Println("sstat n_rxresp failed ...", b)
		rv = 1
	} else if c = sstat["n_rxfin"]; c != conns*rtns {
		fmt.Println("sstat n_txresp failed ...", c)
		rv = 1
	} else if d := sstat["n_rxstart"]; d != conns*rtns {
		fmt.Println("sstat n_rxstart failed ...", d)
		rv = 1
	} else if e := sstat["n_rxstream"]; e != conns*rtns*cnts {
		fmt.Println("sstat n_rxstream failed ...", e)
		rv = 1
	} else if x := sstat["n_rx"]; x != (a + b + c + d + e) {
		fmt.Println("sstat n_rx failed ...", x, a, b, c, d, e)
		rv = 1
	}
	printstats(cstat, sstat)
	return
}

func verifyRandom() (rv int) {
	var cargs = []string{
		"-do", "random", "-addr", "localhost:9900",
		"-conns", strconv.Itoa(rndargs[0]),
		"-routines", strconv.Itoa(rndargs[1]),
		"-count", strconv.Itoa(rndargs[2]),
	}
	var sargs = []string{"-addr", "localhost:9900"}

	cstat, sstat := runCmds(cargs, sargs)

	conns, rtns, cnts := int64(rndargs[0]), int64(rndargs[1]), int64(rndargs[2])

	// common
	if x, y := sstat["n_dropped"], cstat["n_dropped"]; x != 0 || y != 0 {
		fmt.Println("n_dropped failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_mdrops"], cstat["n_mdrops"]; x != 0 || y != 0 {
		fmt.Println("n_dropped failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxfin"], cstat["n_rxfin"]; x != 0 || y != 0 {
		fmt.Println("n_rxfin failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txfin"], cstat["n_txfin"]; x != 0 || y != 0 {
		fmt.Println("n_txfin failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxstart"], cstat["n_rxstart"]; x != 0 || y != 0 {
		fmt.Println("n_rxstart failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_rxstream"], cstat["n_rxstream"]; x != 0 || y != 0 {
		fmt.Println("n_rxstream failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txstart"], cstat["n_txstart"]; x != 0 || y != 0 {
		fmt.Println("n_txstart failed ...", x, y)
		rv = 1
	} else if x, y = sstat["n_txstream"], cstat["n_txstream"]; x != 0 || y != 0 {
		fmt.Println("n_txstream failed ...", x, y)
		rv = 1
	}

	// client
	if y := cstat["n_rxreq"]; y != conns {
		fmt.Println("cstat n_rxreq failed ...", y)
		rv = 1
	} else if z := cstat["n_rxresp"]; z != conns*2 {
		fmt.Println("cstat n_rxresp failed ...", z)
		rv = 1
	} else if x := cstat["n_rx"]; x != (y + z) {
		fmt.Println("cstat n_rx failed ...", x, y, z)
		rv = 1
	} else if x = cstat["n_rxpost"]; x != 0 {
		fmt.Println("cstat n_rxpost failed ...", x)
		rv = 1
	} else if b := cstat["n_txreq"]; b != conns*2 {
		fmt.Println("cstat n_txreq failed ...", b)
		rv = 1
	} else if c := cstat["n_txresp"]; c != conns {
		fmt.Println("cstat n_txresp failed ...", c)
		rv = 1
	} else if d := cstat["n_txpost"]; d != conns*rtns*cnts {
		fmt.Println("cstat n_txpost failed ...", d)
		rv = 1
	} else if a := cstat["n_tx"]; a != (b + c + d) {
		fmt.Println("cstat n_tx failed ...", a, b, c, d)
		rv = 1
	}

	// server
	if y := sstat["n_txreq"]; y != conns {
		fmt.Println("sstat n_txreq failed ...", y)
		rv = 1
	} else if z := sstat["n_txresp"]; z != conns*2 {
		fmt.Println("sstat n_txresp failed ...", z)
		rv = 1
	} else if x := sstat["n_tx"]; x != (y + z) {
		fmt.Println("sstat n_tx failed ...", x, y, z)
		rv = 1
	} else if x = sstat["n_txpost"]; x != 0 {
		fmt.Println("sstat n_txpost failed ...", x)
		rv = 1
	} else if b := sstat["n_rxreq"]; b != conns*2 {
		fmt.Println("sstat n_rxreq failed ...", b)
		rv = 1
	} else if c := sstat["n_rxresp"]; c != conns {
		fmt.Println("sstat n_rxresp failed ...", c)
		rv = 1
	} else if d := sstat["n_rxpost"]; d != conns*rtns*cnts {
		fmt.Println("sstat n_rxpost failed ...", d)
		rv = 1
	} else if a := sstat["n_rx"]; a != (b + c + d) {
		fmt.Println("sstat n_rx failed ...", a, b, c, d)
		rv = 1
	}
	printstats(cstat, sstat)
	return
}

func runclient(cargs []string) map[string]int64 {
	cmd := exec.Command("./client/client", cargs...)
	fmt.Printf("starting client: %s ...\n", strings.Join(cmd.Args, " "))
	var in, out, er bytes.Buffer
	cmd.Stdin, cmd.Stdout, cmd.Stderr = &in, &out, &er
	err := cmd.Run()
	if err != nil {
		fmt.Println("client out...")
		fmt.Println(out.String())
		fmt.Println("client err...")
		fmt.Println(er.String())
		return nil
	}
	return getstat(out.String())
}

func getstat(str string) map[string]int64 {
	for _, line := range strings.Split(str, "\n") {
		stat := make(map[string]int64)
		if strings.HasPrefix(line, "stats ") {
			if err := json.Unmarshal([]byte(line[6:]), &stat); err != nil {
				panic(err)
			}
			return stat
		}
	}
	return nil
}

func printstats(cstat, sstat map[string]int64) {
	keys := []string{}
	for key := range cstat {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Println("statistics:")
	for _, key := range keys {
		v1, v2 := int(cstat[key]), int(sstat[key])
		fmt.Printf("  %20s: %10d %10d\n", key, v1, v2)
	}
}

func runCmds(cargs, sargs []string) (cstat, sstat map[string]int64) {
	cmd := exec.Command("./server/server", sargs...)
	in, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	var out, er bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &er
	fmt.Printf("starting server: %s ...\n", strings.Join(cmd.Args, " "))
	go func() {
		err := cmd.Run()
		if err != nil {
			fmt.Println("server out...")
			fmt.Println(out.String())
			fmt.Println("server err...")
			fmt.Println(er.String())
		}
	}()

	time.Sleep(100 * time.Millisecond)

	cstat = runclient(cargs)
	fmt.Println("... done")

	time.Sleep(300 * time.Millisecond)

	fmt.Println("shutting down server ...")
	in.Close()
	time.Sleep(100 * time.Millisecond)
	sstat = getstat(out.String())
	return
}
