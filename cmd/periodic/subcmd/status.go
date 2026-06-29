package subcmd

import (
	"fmt"
	"github.com/Lupino/go-periodic"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/gosuri/uitable"
	"log"
	"strconv"
	"time"
)

// ShowStatus cli status
func ShowStatus(entryPoint string, param protocol.RSAConnParam, auth ...protocol.ClientAuth) {
	c := periodic.NewClient()
	if len(auth) > 0 {
		if err := c.SetAuth(auth[0].Name, auth[0].Token); err != nil {
			log.Fatal(err)
		}
	}
	if err := c.Connect(entryPoint, param); err != nil {
		log.Fatal(err)
	}
	stats, _ := c.Status()
	table := uitable.New()
	table.MaxColWidth = 50

	table.AddRow("FUNCTION", "WORKERS", "JOBS", "PROCESSING", "LOCK", "SCHEDAT")
	for _, stat := range stats {
		ut, _ := strconv.ParseInt(stat[5], 10, 0)
		t := time.Unix(ut, 0)
		table.AddRow(stat[0], stat[1], stat[2], stat[3], stat[4], t.Format("2006-01-02 15:04:05"))
	}
	fmt.Println(table)
}
