package subcmd

import (
	"github.com/Lupino/go-periodic"
	"github.com/Lupino/go-periodic/protocol"
	"log"
)

// RemoveJob cli remove
func RemoveJob(entryPoint string, param protocol.RSAConnParam, funcName, name string) {
	c := periodic.NewClient()
	if err := c.Connect(entryPoint, param); err != nil {
		log.Fatal(err)
	}
	if err := c.RemoveJob(funcName, name); err != nil {
		log.Fatal(err)
	}
	log.Printf("Remove Job[%s] success.\n", name)
}
