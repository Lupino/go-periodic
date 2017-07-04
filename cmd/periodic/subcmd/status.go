package subcmd

import (
	"fmt"
	"github.com/Lupino/go-periodic"
	"github.com/gosuri/uitable"
	"log"
	"strconv"
	"time"
)

// ShowStatus cli status
func ShowStatus(entryPoint string) {
	c := periodic.NewClient()
	if err := c.Connect(entryPoint); err != nil {
		log.Fatal(err)
	}
	stats, _ := c.Status()
	table := uitable.New()
	table.MaxColWidth = 50

	table.AddRow("FUNCTION", "WORKERS", "JOBS", "PROCESSING", "SCHEDAT")
	for _, stat := range stats {
		ut, _ := strconv.ParseInt(stat[4], 10, 0)
		t := time.Unix(ut, 0)
		table.AddRow(stat[0], stat[1], stat[2], stat[3], t.Format("2006-01-02 15:04:05"))
	}
	fmt.Println(table)
}
