package subcmd

import (
	"bytes"
	"fmt"
	"github.com/Lupino/go-periodic"
	"github.com/Lupino/go-periodic/protocol"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Run cli run
func Run(entryPoint string, param protocol.RSAConnParam, funcName, cmd string, n int, auth ...protocol.ClientAuth) {
	w := periodic.NewWorker(n)
	if len(auth) > 0 {
		if err := w.SetAuth(auth[0].Name, auth[0].Token); err != nil {
			log.Fatal(err)
		}
	}
	if err := w.Connect(entryPoint, param); err != nil {
		log.Fatalf("Error: %s\n", err.Error())
	}
	w.AddFunc(funcName, func(job periodic.Job) {
		handleWorker(job, cmd)
	})
	w.Work()
}

func handleWorker(job periodic.Job, cmd string) {
	var err error
	realCmd := strings.Split(cmd, " ")
	realCmd = append(realCmd, job.Name)
	c := exec.Command(realCmd[0], realCmd[1:]...)
	c.Stdin = strings.NewReader(job.Args)
	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = os.Stderr
	err = c.Run()
	var schedLater int
	var fail = false
	var line string
	for {
		line, err = out.ReadString([]byte("\n")[0])
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "SCHEDLATER") {
			parts := strings.SplitN(line[:len(line)-1], " ", 2)
			later := strings.Trim(parts[1], " ")
			schedLater, _ = strconv.Atoi(later)
		} else if strings.HasPrefix(line, "FAIL") {
			fail = true
		} else {
			fmt.Print(line)
		}
	}

	if (err != nil && err != io.EOF) || fail {
		job.Fail()
	} else if schedLater > 0 {
		job.SchedLater(schedLater)
	} else {
		job.Done()
	}
}
